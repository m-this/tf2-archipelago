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
	"time"

	"github.com/m-this/tf2-archipelago/gamedata"
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

	// The last check, kept in memory only. It answers "did my wave count",
	// which is a question about the last five minutes, not about the run.
	lastCheckName string
	lastCheckAt   time.Time
}

// Stats is what an operator needs to answer "did my wave count", and nothing else.
type Stats struct {
	Seed        string     `json:"seed"`
	Checks      int        `json:"checks"`
	Items       int        `json:"items"`
	AckedSeq    int        `json:"acked_seq"`
	GoalSent    bool       `json:"goal_sent"`
	LastCheck   string     `json:"last_check,omitempty"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`
}

// Open loads the state file, or starts an empty one if there is none.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	data, wasVersion, err := readSnapshot(path)
	if err != nil {
		return nil, err
	}
	// Reading an older file promotes it, and the first check written rewrites it
	// in the current shape, which the binary that wrote it refuses to read back.
	// A copy at the version it arrived in is what keeps rolling the image back
	// an option.
	if wasVersion != FormatVersion {
		if err := archiveFormat(path, wasVersion); err != nil {
			return nil, err
		}
		if err := writeSnapshot(path, data); err != nil {
			return nil, err
		}
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

// Stats reports the run at a glance.
func (s *Store) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	stats := Stats{
		Seed:      s.data.Seed,
		Checks:    len(s.data.Checks),
		Items:     len(s.data.Items),
		AckedSeq:  s.data.AckedSeq,
		GoalSent:  s.data.GoalSent,
		LastCheck: s.lastCheckName,
	}
	if !s.lastCheckAt.IsZero() {
		at := s.lastCheckAt
		stats.LastCheckAt = &at
	}
	return stats
}

// BindSeed ties the state to one multiworld. It reports the file it set the
// previous run aside in, empty when it kept what it held.
//
// A different seed drops the old run rather than replay ids that now mean
// something else, and keeps a copy of the file it dropped. Learning the seed for
// the first time keeps what is held: the bridge answers the plugin before it has
// ever reached Archipelago.
func (s *Store) BindSeed(seed string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Seed == seed {
		return "", nil
	}
	if s.data.Seed == "" {
		s.data.Seed = seed
		if err := s.persist(); err != nil {
			s.data.Seed = ""
			return "", err
		}
		return "", nil
	}

	archive, err := archiveSnapshot(s.path, s.data.Seed)
	if err != nil {
		return "", fmt.Errorf("cannot set the previous run aside: %w", err)
	}
	previous, previousGrants := s.data, s.grants
	s.data = snapshot{FormatVersion: FormatVersion, Seed: seed}
	s.grants = nil
	if err := s.persist(); err != nil {
		// The run is still whole in the archive and in memory. Putting the file
		// back is what keeps the two agreeing.
		s.data, s.grants = previous, previousGrants
		if renamed := os.Rename(archive, s.path); renamed != nil {
			return archive, fmt.Errorf("%w, and the run stayed in %s: %w", err, archive, renamed)
		}
		return "", err
	}
	// The sequence just went back to zero. A plugin blocked on a long poll has
	// to hear that now rather than at the poll timeout.
	s.broadcast()
	return archive, nil
}

// AddCheck records a location the plugin reported and reports whether it was
// new. The plugin retries on timeout, so a repeat has to be a no-op.
func (s *Store) AddCheck(id int64) (bool, error) {
	location, known := gamedata.LocationByID(id)
	if !known {
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
	s.lastCheckName, s.lastCheckAt = location.Name, time.Now()
	s.broadcast()
	return true, nil
}

// AdoptChecks takes the locations Archipelago says this slot has already
// checked and adds whatever the state file is missing, reporting how many were
// new. It is what makes a lost or rolled-back state file survivable: the server
// holds the same list, and the run comes back on the next connect instead of
// asking a team to replay an evening.
//
// Ids the tables do not know are skipped. That is a seed from a newer gamedata,
// and the run is better off missing one check than refusing to start.
func (s *Store) AdoptChecks(ids []int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	held := len(s.data.Checks)
	for _, id := range ids {
		if _, known := gamedata.LocationByID(id); !known {
			continue
		}
		if slices.Contains(s.data.Checks, id) {
			continue
		}
		s.data.Checks = append(s.data.Checks, id)
	}
	adopted := len(s.data.Checks) - held
	if adopted == 0 {
		return 0, nil
	}
	if err := s.persist(); err != nil {
		s.data.Checks = s.data.Checks[:held]
		return 0, err
	}
	s.broadcast()
	return adopted, nil
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
	previousItems, previousGrants, previousAcked := s.data.Items, s.grants, s.data.AckedSeq
	s.data.Items = append(slices.Clone(s.data.Items[:index]), itemIDs...)
	s.grants = grantsFrom(s.data.Items)

	// A resend can be shorter than what we held. An acknowledgement left past
	// the end would then suppress the fresh effects that land in those slots,
	// which is the one thing it must never do.
	s.data.AckedSeq = min(s.data.AckedSeq, len(s.data.Items))

	if err := s.persist(); err != nil {
		s.data.Items, s.grants, s.data.AckedSeq = previousItems, previousGrants, previousAcked
		return err
	}
	s.broadcast()
	return nil
}

// GrantsSince returns everything past the sequence the plugin last applied,
// and how far the item list itself reaches. Both come from one look at the
// store: read separately, a seed wipe landing between them answers a plugin
// with the old run's grants and the new run's sequence, and nothing downstream
// can tell.
//
// Sequences count items, not grants, so this cannot index the slice.
//
// State grants are sent whenever they are asked for: applying one twice is the
// same as applying it once. An effect is held back once acknowledged, because
// the plugin asking from a lower sequence means it restarted, not that the
// effect should happen again.
func (s *Store) GrantsSince(seq int) ([]Grant, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fresh := make([]Grant, 0, len(s.grants))
	for _, grant := range s.grants {
		if grant.Seq <= seq {
			continue
		}
		if grant.OneShot && grant.Seq <= s.data.AckedSeq {
			continue
		}
		fresh = append(fresh, grant)
	}
	if len(fresh) == 0 {
		return nil, len(s.data.Items)
	}
	return fresh, len(s.data.Items)
}

// Ack records how far the plugin has applied one-shot grants. It only ever
// moves forward: an ack from a plugin that restarted mid-run would otherwise
// hand out every effect in the run again.
func (s *Store) Ack(seq int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if seq > len(s.data.Items) {
		return fmt.Errorf("cannot acknowledge sequence %d, only %d items exist", seq, len(s.data.Items))
	}
	if seq <= s.data.AckedSeq {
		return nil
	}
	previous := s.data.AckedSeq
	s.data.AckedSeq = seq
	if err := s.persist(); err != nil {
		s.data.AckedSeq = previous
		return err
	}
	return nil
}

// Unlocks is everything that should be true right now, and the sequence the
// plugin resumes polling from.
func (s *Store) Unlocks() Unlocks {
	s.mu.Lock()
	defer s.mu.Unlock()
	return unlocksFrom(s.grants, s.data.AckedSeq)
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
