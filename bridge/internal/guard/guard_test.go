package guard

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// A panic on the pump used to take the launcher with it, and the player lost the
// server they were on. It has to become a line instead.
func TestRunContainsAPanicAndLogsIt(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&out, nil))

	var wg sync.WaitGroup
	wg.Go(func() { Run("the pump", logger, func() { panic("a closed channel") }) })
	wg.Wait()

	for _, want := range []string{"the pump", "a closed channel", "guard_test.go"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the line does not carry %q: %s", want, out.String())
		}
	}
}

// Work that returns says nothing, or every run carries a line about the thing
// that did not go wrong.
func TestRunIsSilentWhenNothingPanics(t *testing.T) {
	var out bytes.Buffer
	ran := false

	Run("the pump", slog.New(slog.NewTextHandler(&out, nil)), func() { ran = true })

	if !ran {
		t.Error("the work did not run")
	}
	if out.Len() != 0 {
		t.Errorf("it said %q", out.String())
	}
}
