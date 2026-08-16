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
	return true, s.persist()
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

	switch {
	case index == 0:
		s.data.Items = slices.Clone(itemIDs)
	case index > len(s.data.Items):
		return ErrDesync
	default:
		// A replay of items we already hold, plus possibly new ones.
		s.data.Items = append(s.data.Items[:index], itemIDs...)
	}

	previous := s.grants
	s.grants = grantsFrom(s.data.Items)
	if err := s.persist(); err != nil {
		s.grants = previous
		return err
	}
	s.broadcast()
	return nil
}

// GrantsSince returns everything past the sequence the plugin last applied.
func (s *Store) GrantsSince(seq int) []Grant {
	s.mu.Lock()
	defer s.mu.Unlock()
	if seq < 0 || seq >= len(s.grants) {
		return nil
	}
	return slices.Clone(s.grants[seq:])
}

// Unlocks is everything that should be true right now.
func (s *Store) Unlocks() Unlocks {
	s.mu.Lock()
	defer s.mu.Unlock()
	return unlocksFrom(s.grants)
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
