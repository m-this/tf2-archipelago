package apclient

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestACommandThatCanEndTheRunIsRefused(t *testing.T) {
	// Anyone who can join the game server can type these. !release hands this
	// slot's remaining items to the other players and nothing undoes it.
	for _, text := range []string{
		"!release",
		"!RELEASE",
		"!collect",
		"!getitem Slippery Sausage",
		"!admin /exit",
		"!alias someone",
		"!countdown 600",
	} {
		if err := checkSayable(text); !errors.Is(err, ErrCommandRefused) {
			t.Errorf("%q was allowed through, err = %v", text, err)
		}
	}
}

func TestReadingTheMultiworldIsAllowed(t *testing.T) {
	for _, text := range []string{
		"!hint Scout",
		"!missing",
		"!status",
		"!checked",
		"!remaining",
		"!players",
		"!help",
		"hello from the game server",
		"nobody said ! anything",
	} {
		if err := checkSayable(text); err != nil {
			t.Errorf("%q was refused: %v", text, err)
		}
	}
}

func TestALineTooLongIsRefused(t *testing.T) {
	if err := checkSayable(strings.Repeat("a", sayLengthMax+1)); !errors.Is(err, ErrSayTooLong) {
		t.Fatalf("err = %v, want ErrSayTooLong", err)
	}
}

func TestTheFloodGuardRefillsOverTime(t *testing.T) {
	at := time.Unix(0, 0)
	guard := newBucket(func() time.Time { return at })

	for line := range sayBurst {
		if !guard.take() {
			t.Fatalf("line %d of the burst was refused", line+1)
		}
	}
	if guard.take() {
		t.Fatal("the burst was not bounded")
	}

	at = at.Add(sayRefill)
	if !guard.take() {
		t.Fatal("the allowance did not come back")
	}
	if guard.take() {
		t.Fatal("more came back than one refill's worth")
	}

	// However long it has been quiet, the allowance stops at the burst.
	at = at.Add(time.Hour)
	for line := range sayBurst {
		if !guard.take() {
			t.Fatalf("line %d after a quiet hour was refused", line+1)
		}
	}
	if guard.take() {
		t.Fatal("a quiet hour banked more than a burst")
	}
}
