package apclient

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/bridge/internal/deathlink"
	"github.com/m-this/tf2-archipelago/bridge/internal/state"
)

func deathLinkSlotData() map[string]any {
	slot := slotDataFor("final_boss", "mvm_decoy", "mvm_decoy")
	slot["death_link"] = true
	return slot
}

func runDeathLinkClient(t *testing.T, room *fakeRoom) (*Client, *deathlink.Feed) {
	t.Helper()
	feed := deathlink.New(8)
	store, err := state.Open(t.TempDir() + "/bridge.json")
	if err != nil {
		t.Fatal(err)
	}
	client := New(Options{
		URL:      room.start(t),
		SlotName: "tf2",
		Store:    store,
		Deaths:   feed,
		Logger:   slog.New(slog.DiscardHandler),
	})
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- client.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-stopped
	})
	return client, feed
}

func TestASeedWithDeathLinkClaimsTheTag(t *testing.T) {
	room := &fakeRoom{seed: "seed-1", slotData: deathLinkSlotData()}
	client, _ := runDeathLinkClient(t, room)

	update := awaitCommand(t, room, "ConnectUpdate")
	var tags []string
	if err := json.Unmarshal(update["tags"], &tags); err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0] != deathLinkTag {
		t.Fatalf("tags = %v", tags)
	}
	waitFor(t, "health to say so", func() bool { return client.Health().DeathLink })
}

func TestASeedWithoutDeathLinkStaysQuiet(t *testing.T) {
	room := &fakeRoom{seed: "seed-1", slotData: slotDataFor("final_boss", "mvm_decoy", "mvm_decoy")}
	client, _ := runClient(t, room)
	waitFor(t, "the handshake", func() bool { return client.Health().Connected })

	if err := client.Die(t.Context(), "tf2 lost a wave"); !errors.Is(err, ErrDeathLinkOff) {
		t.Fatalf("Die = %v, want ErrDeathLinkOff", err)
	}
	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case message := <-room.heard:
			var cmd string
			_ = json.Unmarshal(message["cmd"], &cmd)
			if cmd == "ConnectUpdate" || cmd == "Bounce" {
				t.Fatalf("the bridge sent %s for a seed without DeathLink", cmd)
			}
		case <-deadline:
			return
		}
	}
}

func TestADeathGoesOutAsABounce(t *testing.T) {
	room := &fakeRoom{seed: "seed-1", slotData: deathLinkSlotData()}
	client, _ := runDeathLinkClient(t, room)
	waitFor(t, "the handshake", func() bool { return client.Health().Connected })

	if err := client.Die(t.Context(), "tf2 lost wave 3 of Decoy"); err != nil {
		t.Fatal(err)
	}
	bounce := awaitCommand(t, room, "Bounce")
	var tags []string
	if err := json.Unmarshal(bounce["tags"], &tags); err != nil {
		t.Fatal(err)
	}
	var data deathLinkData
	if err := json.Unmarshal(bounce["data"], &data); err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0] != deathLinkTag || data.Source != "tf2" || data.Cause != "tf2 lost wave 3 of Decoy" || data.Time == 0 {
		t.Fatalf("bounce = %v %+v", tags, data)
	}

	if err := client.Die(t.Context(), "again"); !errors.Is(err, ErrDiedTooMuch) {
		t.Fatalf("a second death right away = %v, want ErrDiedTooMuch", err)
	}
}

func TestADeathFromElsewhereReachesTheFeed(t *testing.T) {
	room := &fakeRoom{seed: "seed-1", slotData: deathLinkSlotData()}
	client, feed := runDeathLinkClient(t, room)
	waitFor(t, "the handshake", func() bool { return client.Health().Connected })

	room.bounce(t, "Ana", "Ana fell into lava")
	waitFor(t, "the death to land", func() bool {
		deaths, _ := feed.Since(0)
		return len(deaths) == 1 && deaths[0].Source == "Ana" && deaths[0].Cause == "Ana fell into lava"
	})
}

func TestOurOwnDeathIsNotAppliedTwice(t *testing.T) {
	room := &fakeRoom{seed: "seed-1", slotData: deathLinkSlotData()}
	client, feed := runDeathLinkClient(t, room)
	waitFor(t, "the handshake", func() bool { return client.Health().Connected })

	room.bounce(t, "tf2", "tf2 lost a wave")
	room.bounce(t, "Bram", "")
	waitFor(t, "the other death to land", func() bool {
		deaths, _ := feed.Since(0)
		return len(deaths) == 1 && deaths[0].Source == "Bram"
	})
}
