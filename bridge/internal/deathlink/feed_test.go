package deathlink

import "testing"

func TestSinceReturnsOnlyWhatIsPastTheSequence(t *testing.T) {
	feed := New(8)
	feed.Append("Ana", "fell")
	feed.Append("Bram", "drowned")

	deaths, latest := feed.Since(1)
	if latest != 2 || len(deaths) != 1 || deaths[0].Source != "Bram" || deaths[0].Seq != 2 {
		t.Fatalf("deaths = %v, latest = %d", deaths, latest)
	}
}

func TestANegativeSequenceStartsFromNow(t *testing.T) {
	feed := New(8)
	feed.Append("Ana", "fell")

	deaths, latest := feed.Since(-1)
	if latest != 1 || len(deaths) != 0 {
		t.Fatalf("deaths = %v, latest = %d", deaths, latest)
	}
}

func TestTheRingDropsTheOldest(t *testing.T) {
	feed := New(2)
	feed.Append("Ana", "")
	feed.Append("Bram", "")
	feed.Append("Chika", "")

	deaths, latest := feed.Since(0)
	if latest != 3 || len(deaths) != 2 || deaths[0].Source != "Bram" {
		t.Fatalf("deaths = %v, latest = %d", deaths, latest)
	}
}

func TestWatchWakesOnAppend(t *testing.T) {
	feed := New(2)
	woken := feed.Watch()
	feed.Append("Ana", "")
	select {
	case <-woken:
	default:
		t.Fatal("Watch did not wake")
	}
}
