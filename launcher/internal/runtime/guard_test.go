package runtime

import (
	"strings"
	"sync"
	"testing"
)

// A panic on a goroutine nobody waits on takes the whole launcher down, and
// with it the server somebody is playing on. It has to become a line instead.
func TestGuardContainsAPanicAndSaysSo(t *testing.T) {
	var (
		mu   sync.Mutex
		said []string
	)
	say := func(text string) {
		mu.Lock()
		defer mu.Unlock()
		said = append(said, text)
	}

	// Closed after guard returns, not inside the work: the work's own defers
	// run while the panic is still unwinding, so signalling there reads the
	// result before the recover has written it.
	done := make(chan struct{})
	go func() {
		defer close(done)
		Guard("the bridge", say, func() { panic("a nil map in the log watcher") })
	}()
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(said) != 1 {
		t.Fatalf("said %d lines, want 1: %v", len(said), said)
	}
	for _, want := range []string{"the bridge", "a nil map in the log watcher", "guard_test.go"} {
		if !strings.Contains(said[0], want) {
			t.Errorf("the line does not carry %q: %s", want, said[0])
		}
	}
}

// Work that returns normally says nothing, or every run would carry a line
// about the thing that did not go wrong.
func TestGuardIsSilentWhenNothingPanics(t *testing.T) {
	var said []string
	ran := false

	Guard("the bridge", func(text string) { said = append(said, text) }, func() { ran = true })

	if !ran {
		t.Error("the work did not run")
	}
	if len(said) != 0 {
		t.Errorf("said %v, want nothing", said)
	}
}
