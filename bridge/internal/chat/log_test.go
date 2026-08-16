package chat

import "testing"

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
	// The sequence keeps counting even though the message is gone, so a
	// listener that missed it does not receive it twice under a new number.
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
	if messages, latest := log.Since(0); len(messages) != 0 || latest != 0 {
		t.Fatalf("an empty line was kept: %+v, latest %d", messages, latest)
	}
}
