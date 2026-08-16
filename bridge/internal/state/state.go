// Package state holds everything the bridge must not lose: the checks the
// plugin reported and the items Archipelago sent.
//
// A wave clear costs ten minutes of play, and an Archipelago server down for an
// hour must not cost one of them, so the order is always record, acknowledge,
// then send upstream. Everything the bridge serves is derived from those two
// lists, so there is no second copy to fall out of step.
package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

// ErrDesync means Archipelago sent an item list that does not continue the one
// we hold. The caller answers with a Sync to have the inventory resent.
var ErrDesync = errors.New("received items do not continue the held list")

// Store is the durable state, and the only place in the bridge where anything is remembered.
type Store struct {
	path string

	mu      sync.Mutex
	data    snapshot
	grants  []Grant
	updated chan struct{}
}

// Open loads the state file, or starts an empty one if there is none.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	data, err := readSnapshot(path)
	if err != nil {
		return nil, err
	}
	return &Store{
		path:    path,
		data:    data,
		grants:  grantsFrom(data.Items),
		updated: make(chan struct{}),
	}, nil
}

// Watch returns a channel closed the next time anything changes. Callers take
// a fresh one after every wake-up.
func (s *Store) Watch() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updated
}

// BindSeed ties the state to one multiworld and reports whether it wiped. A
// different seed drops the old run rather than replay ids that now mean
// something else. Learning the seed for the first time keeps what is held: the
// bridge answers the plugin before it has ever reached Archipelago.
func (s *Store) BindSeed(seed string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Seed == seed {
		return false, nil
	}
	if s.data.Seed == "" {
		s.data.Seed = seed
		return false, s.persist()
	}
	s.data = snapshot{FormatVersion: FormatVersion, Seed: seed}
	s.grants = nil
	if err := s.persist(); err != nil {
		return true, err
	}
	// The sequence just went back to zero. A plugin blocked on a long poll has
	// to hear that now rather than at the poll timeout.
	s.broadcast()
	return true, nil
}

// AddCheck records a location the plugin reported and reports whether it was
// new. The plugin retries on timeout, so a repeat has to be a no-op.
func (s *Store) AddCheck(id int64) (bool, error) {
	if _, known := gamedata.LocationByID(id); !known {
		return false, fmt.Errorf("location id %d is not in the tables", id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Contains(s.data.Checks, id) {
		return false, nil
	}
	s.data.Checks = append(s.data.Checks, id)
	if err := s.persist(); err != nil {
		s.data.Checks = s.data.Checks[:len(s.data.Checks)-1]
		return false, err
	}
	s.broadcast()
	return true, nil
}

// Checks is every location reported so far. The whole set goes upstream every
// time; Archipelago ignores repeats, so nothing tracks acknowledgements.
func (s *Store) Checks() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.data.Checks)
}

// ApplyItems takes a ReceivedItems payload. index is where Archipelago thinks
// the list continues from: zero is the whole inventory, anything past what we
// hold is a desync.
func (s *Store) ApplyItems(index int, itemIDs []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index > len(s.data.Items) {
		return ErrDesync
	}
	// Index zero is Archipelago restating the whole inventory; anything else
	// continues the list from that point.
	previousItems, previousGrants := s.data.Items, s.grants
	s.data.Items = append(slices.Clone(s.data.Items[:index]), itemIDs...)
	s.grants = grantsFrom(s.data.Items)

	if err := s.persist(); err != nil {
		s.data.Items, s.grants = previousItems, previousGrants
		return err
	}
	s.broadcast()
	return nil
}

// GrantsSince returns everything past the sequence the plugin last applied.
// Sequences count items, not grants, so this cannot index the slice.
func (s *Store) GrantsSince(seq int) []Grant {
	s.mu.Lock()
	defer s.mu.Unlock()
	fresh := make([]Grant, 0, len(s.grants))
	for _, grant := range s.grants {
		if grant.Seq > seq {
			fresh = append(fresh, grant)
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	return fresh
}

// Unlocks is everything that should be true right now.
func (s *Store) Unlocks() Unlocks {
	s.mu.Lock()
	defer s.mu.Unlock()
	return unlocksFrom(s.grants, len(s.data.Items))
}

// GoalSent reports whether CLIENT_GOAL has already gone out for this run.
func (s *Store) GoalSent() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.GoalSent
}

// MarkGoalSent records the win so a reconnect does not announce it twice.
func (s *Store) MarkGoalSent() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.GoalSent {
		return nil
	}
	s.data.GoalSent = true
	if err := s.persist(); err != nil {
		s.data.GoalSent = false
		return err
	}
	s.broadcast()
	return nil
}

// persist and broadcast are called with the lock held.

func (s *Store) persist() error {
	return writeSnapshot(s.path, s.data)
}

func (s *Store) broadcast() {
	close(s.updated)
	s.updated = make(chan struct{})
}
