// Package deathlink carries the multiworld's deaths to the game.
//
// Nothing here is durable, like chat and unlike a check. A death that arrives
// while the game server is restarting has nobody to kill, and killing them ten
// minutes later would be a different death.
package deathlink

import (
	"slices"
	"sync"
)

// Death is one DeathLink bounce from somewhere else in the multiworld.
type Death struct {
	Seq    int    `json:"seq"`
	Source string `json:"source"`
	Cause  string `json:"cause,omitempty"`
}

// Feed is a bounded ring of recent deaths.
type Feed struct {
	capacity int

	mu      sync.Mutex
	deaths  []Death
	latest  int
	updated chan struct{}
}

func New(capacity int) *Feed {
	if capacity < 1 {
		capacity = 1
	}
	return &Feed{
		capacity: capacity,
		deaths:   make([]Death, 0, capacity),
		updated:  make(chan struct{}),
	}
}

// Watch returns a channel closed the next time a death arrives.
func (f *Feed) Watch() <-chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.updated
}

// Append records a death and wakes anyone waiting. The oldest falls off the end.
func (f *Feed) Append(source, cause string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.latest++
	f.deaths = append(f.deaths, Death{Seq: f.latest, Source: source, Cause: cause})
	if len(f.deaths) > f.capacity {
		f.deaths = slices.Delete(f.deaths, 0, len(f.deaths)-f.capacity)
	}
	close(f.updated)
	f.updated = make(chan struct{})
}

// Since returns the deaths past a sequence, and the sequence to ask from next.
// A negative sequence means "nothing behind me", which is what the plugin sends
// on load: a death from before it was listening is not one it should apply.
func (f *Feed) Since(seq int) ([]Death, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if seq < 0 {
		return nil, f.latest
	}
	fresh := make([]Death, 0, len(f.deaths))
	for _, death := range f.deaths {
		if death.Seq > seq {
			fresh = append(fresh, death)
		}
	}
	return fresh, f.latest
}
