package apclient

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

// What a player may say to the multiworld from inside the game.
//
// Not every Archipelago command is reversible. `!release` hands this slot's
// remaining items to the other players and nothing in Archipelago undoes it, so
// one line typed in game chat ends the run for everyone in the multiworld. The
// game server has no accounts and no trust model: anyone who can join can type.
//
// So commands are an allowlist of the ones that only read. Plain chat is never
// a command and is always passed through.
var sayCommandsAllowed = []string{
	"checked",
	"help",
	"hint",
	"hint_location",
	"license",
	"missing",
	"options",
	"players",
	"remaining",
	"status",
}

const (
	// sayBurst is how many lines can go out at once, and sayRefill is how fast
	// the allowance comes back. A player pasting a wall of text is a nuisance
	// for the whole multiworld, and the slot doing it is this game server.
	sayBurst  = 5
	sayRefill = 3 * time.Second

	// sayLengthMax bounds one line. Archipelago has its own limit and the
	// plugin's chat buffer is smaller still.
	sayLengthMax = 300
)

// ErrCommandRefused is a command that could end the run, sent from a place
// where anyone on the game server can type it.
var ErrCommandRefused = errors.New("that multiworld command cannot be sent from the game")

// ErrSayTooLong is a line past what the multiworld will take.
var ErrSayTooLong = errors.New("that line is too long for the multiworld")

// ErrSaidTooMuch is the flood guard.
var ErrSaidTooMuch = errors.New("too many lines to the multiworld, wait a moment")

// sayCommand reads the command out of a line, if it is one at all.
func sayCommand(text string) (string, bool) {
	if !strings.HasPrefix(text, "!") {
		return "", false
	}
	command := strings.TrimPrefix(text, "!")
	if space := strings.IndexAny(command, " \t"); space != -1 {
		command = command[:space]
	}
	return strings.ToLower(command), true
}

// checkSayable is the whole policy on a line: refused commands first, because
// being told the command is not allowed beats being told to slow down.
func checkSayable(text string) error {
	if len(text) > sayLengthMax {
		return fmt.Errorf("%w: %d characters, %d allowed", ErrSayTooLong, len(text), sayLengthMax)
	}
	command, isCommand := sayCommand(text)
	if isCommand && !slices.Contains(sayCommandsAllowed, command) {
		return fmt.Errorf("%w: !%s", ErrCommandRefused, command)
	}
	return nil
}

// bucket is a token bucket: sayBurst lines at once, then one per sayRefill.
type bucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
	now    func() time.Time
}

func newBucket(now func() time.Time) *bucket {
	return &bucket{tokens: sayBurst, last: now(), now: now}
}

func (b *bucket) take() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	at := b.now()
	b.tokens = min(sayBurst, b.tokens+at.Sub(b.last).Seconds()/sayRefill.Seconds())
	b.last = at
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
