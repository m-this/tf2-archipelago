package apclient

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/fakeroom"
	"github.com/m-this/tf2-archipelago/gamedata"
)

// The offline room the launcher serves for a play-test is only worth anything if
// this client accepts it, so the test drives the real client against the real
// room rather than against a stand-in.
func TestFakeRoomServesThisClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	room, address, err := fakeroom.Start(ctx, fakeroom.Options{
		SlotName:     "tester",
		Goal:         "final_boss",
		MissionCount: 3,
		Log:          func(text string) { t.Log(text) },
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = room.Close() }()

	store := newStore(t)
	client := New(Options{
		URL:      address,
		SlotName: "tester",
		Store:    store,
		Logger:   slog.New(slog.DiscardHandler),
	})
	go func() { _ = client.Run(ctx) }()

	// The handshake, including the made-up slot data: without missions the
	// bridge has no seed and records nothing.
	waitFor(t, "the handshake", func() bool {
		health := client.Health()
		return health.Connected && len(health.Missions) == 3
	})

	// A cleared wave, the way the bridge records one, and the unlock the room
	// sends back for it.
	mission, ok := gamedata.MissionByPopFile(client.Health().Missions[0])
	if !ok {
		t.Fatalf("the room named a mission the game data does not have")
	}
	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatalf("cannot record the check: %v", err)
	}
	waitFor(t, "an unlock to arrive", func() bool {
		return store.Stats().Items > 0
	})
}
