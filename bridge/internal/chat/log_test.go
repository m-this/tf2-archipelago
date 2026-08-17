package chat

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSinceReturnsWhatIsNew(t *testing.T) {
	log := New(8)
	log.Append("first")
	log.Append("second")

	messages, latest := log.Since(0)
	if len(messages) != 2 || latest != 2 {
		t.Fatalf("since 0: %+v, latest %d", messages, latest)
	}
	if messages[0].Text != "first" || messages[0].Seq != 1 {
		t.Fatalf("first message is %+v", messages[0])
	}

	messages, _ = log.Since(1)
	if len(messages) != 1 || messages[0].Text != "second" {
		t.Fatalf("since 1: %+v", messages)
	}
	if messages, _ := log.Since(2); len(messages) != 0 {
		t.Fatalf("since 2: %+v", messages)
	}
}

func TestANegativeSequenceSkipsTheBacklog(t *testing.T) {
	log := New(8)
	log.Append("said before anyone was listening")

	messages, latest := log.Since(-1)
	if len(messages) != 0 {
		t.Fatalf("a negative sequence returned %d message(s)", len(messages))
	}
	if latest != 1 {
		t.Fatalf("latest = %d, want 1", latest)
	}
}

func TestTheRingDropsTheOldest(t *testing.T) {
	log := New(2)
	log.Append("one")
	log.Append("two")
	log.Append("three")

	messages, latest := log.Since(0)
	if len(messages) != 2 || messages[0].Text != "two" || messages[1].Text != "three" {
		t.Fatalf("held %+v", messages)
	}
	// The sequence counts past a dropped message, so nothing is served twice under a new number.
	if latest != 3 || messages[1].Seq != 3 {
		t.Fatalf("latest = %d, last seq = %d", latest, messages[1].Seq)
	}
}

func TestWatchWakesOnAMessage(t *testing.T) {
	log := New(4)
	waiting := log.Watch()
	select {
	case <-waiting:
		t.Fatal("woken before anything was said")
	default:
	}

	log.Append("something")
	select {
	case <-waiting:
	default:
		t.Fatal("a message did not wake the watcher")
	}
}

func TestEmptyLinesAreDropped(t *testing.T) {
	log := New(4)
	log.Append("")
	log.Append("   \n \n  ")
	if messages, latest := log.Since(0); len(messages) != 0 || latest != 0 {
		t.Fatalf("an empty line was kept: %+v, latest %d", messages, latest)
	}
}

// The engine drops a user message over 255 bytes silently, so a line the plugin
// cannot print is a line nobody knows was said.
func TestEveryLineFitsInTheChat(t *testing.T) {
	log := New(64)
	log.Append(strings.Repeat("bot ", 400))
	log.Append("hint: " + strings.Repeat("x", LineMax*2))

	messages, _ := log.Since(0)
	if len(messages) == 0 {
		t.Fatal("nothing was kept")
	}
	for _, message := range messages {
		if len(message.Text) > LineMax {
			t.Fatalf("line %d is %d bytes: %q", message.Seq, len(message.Text), message.Text)
		}
	}
}

func TestMessagesSplitOnTheirOwnLineBreaks(t *testing.T) {
	log := New(8)
	log.Append("!help \n    Returns the help listing\n!players \n    Who is connected")

	messages, latest := log.Since(0)
	if latest != 4 {
		t.Fatalf("latest = %d, want 4", latest)
	}
	if messages[0].Text != "!help" || messages[1].Text != "Returns the help listing" {
		t.Fatalf("split into %+v", messages)
	}
}

func TestALongMessageIsCutOffRatherThanFlooding(t *testing.T) {
	log := New(64)
	log.Append(strings.Repeat("a line of the multiworld's help\n", LinesMax*3))

	messages, _ := log.Since(0)
	if len(messages) != LinesMax+1 {
		t.Fatalf("kept %d line(s), want %d", len(messages), LinesMax+1)
	}
	if last := messages[len(messages)-1].Text; last != truncatedNote {
		t.Fatalf("last line is %q, want the note", last)
	}
}

func TestWrappingKeepsWordsAndRunesWhole(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("bomb carrier ", 40))
	if rejoined := strings.Join(wrap(long), " "); rejoined != long {
		t.Fatalf("wrapping changed the text:\n%q", rejoined)
	}

	// No spaces to break on, so the cut lands mid-word and must still land
	// between runes.
	for _, line := range wrap(strings.Repeat("é", LineMax*2)) {
		if !utf8.ValidString(line) {
			t.Fatalf("cut a rune in half: %q", line)
		}
	}
}
