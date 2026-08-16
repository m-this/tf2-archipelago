// Package state holds everything the bridge must not lose: which checks the
// plugin has reported, and which items Archipelago has sent.
//
// Both lists are written to disk before anything else happens with them. A
// wave clear costs ten minutes of play, and an Archipelago server that has
// been down for an hour must not cost a single one of them, so the order is
// always: record, acknowledge, then send upstream.
//
// Everything the bridge serves is derived from those two lists. There is no
// second copy of the unlock set to fall out of step with them.
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
// we hold. The caller answers it with a Sync, which makes the server resend
// the whole inventory.
var ErrDesync = errors.New("received items do not continue the held list")

// Store is the durable state, and the only place in the bridge where anything
// is remembered.
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

// BindSeed ties the state to one multiworld. A different seed means the old
// run is gone, so its checks and items are dropped rather than replayed into a
// world where those ids mean something else. Reports whether it wiped.
//
// Learning the seed for the first time keeps what is already held: the bridge
// answers the plugin before it has ever reached Archipelago, so a run can
// legitimately have checks before it has a seed name.
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

// AddCheck records a location the plugin reported. Reports whether it was new;
// the plugin retries on timeout and cannot know whether its first attempt
// landed, so a repeat has to be a no-op.
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

// Checks is every location reported so far. The whole set is what gets sent on
// every reconnect: Archipelago ignores repeats, and 210 ids is not worth
// tracking acknowledgements for.
func (s *Store) Checks() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.data.Checks)
}

// ApplyItems takes a ReceivedItems payload. index is where Archipelago thinks
// the list continues from: zero means it is sending the whole inventory,
// anything else has to line up with what we hold or the two have diverged.
func (s *Store) ApplyItems(index int, itemIDs []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case index == 0:
		s.data.Items = slices.Clone(itemIDs)
	case index > len(s.data.Items):
		return ErrDesync
	default:
		// A replay of items we already hold, plus possibly some new ones.
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

// Unlocks is what should be true right now.
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
