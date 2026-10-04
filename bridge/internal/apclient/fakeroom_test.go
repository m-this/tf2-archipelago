package apclient

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/bridge/internal/state"
	"github.com/m-this/tf2-archipelago/fakeroom"
	"github.com/m-this/tf2-archipelago/gamedata"
)

// The offline room the launcher serves for a play-test is only worth anything if
// this client accepts it, so the test drives the real client against the real
// room rather than against a stand-in.
func TestFakeRoomServesThisClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// The room logs from its own goroutines, and one of them can fire after
	// this function returns, which the testing package treats as a failure.
	// So the room's log goes to a channel-free sink that stops caring once the
	// test is done.
	var done atomic.Bool
	room, address, err := fakeroom.Start(ctx, fakeroom.Options{
		SlotName:     "tester",
		Goal:         "final_boss",
		MissionCount: 3,
		MissionModifiers: map[string][]fakeroom.MissionModifier{
			"mvm_decoy": {{
				Key: "low_gravity", Name: "Low Gravity", Kind: "environment",
				Description: "World gravity is halved.",
			}},
		},
		Log: func(text string) {
			if !done.Load() {
				t.Log(text)
			}
		},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		done.Store(true)
		cancel()
		_ = room.Close(context.Background())
	})

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
		return health.Connected && len(health.Missions) == 3 &&
			len(health.MissionModifiers["mvm_decoy"]) == 1
	})

	// The starting inventory, which the plugin needs before it can enforce
	// anything: a run that starts with no class and no weapon slot leaves
	// every class pickable and every weapon in hand.
	waitFor(t, "the starting inventory", func() bool {
		unlocks := store.Unlocks()
		return len(unlocks.Of(gamedata.ItemClass)) == 1 &&
			len(unlocks.Of(gamedata.ItemWeaponSlot)) == 1 &&
			len(unlocks.Of(gamedata.ItemMissionTicket)) == 1
	})

	// A cleared wave, the way the bridge records one, and the unlock the room
	// sends back for it.
	mission, ok := gamedata.MissionByPopFile(client.Health().Missions[0])
	if !ok {
		t.Fatalf("the room named a mission the game data does not have")
	}
	held := store.Stats().Items
	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatalf("cannot record the check: %v", err)
	}
	waitFor(t, "an unlock to arrive", func() bool {
		return store.Stats().Items > held
	})
}

// Test mode binds its seed through the same handshake as a real room, so moving
// the bind to Connected has to leave it binding, and switching between test mode
// and a real room has to keep setting the other run aside.
func TestTestModeBindsItsOwnSeed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bridge.json")
	store, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindSeed("ours"); err != nil {
		t.Fatal(err)
	}
	mission, _ := gamedata.MissionByPopFile("mvm_decoy")
	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}

	connect := func(address string) {
		t.Helper()
		ctx, cancel := context.WithCancel(t.Context())
		client := New(Options{
			URL: address, SlotName: "tester", Store: store, Logger: slog.New(slog.DiscardHandler),
		})
		stopped := make(chan error, 1)
		go func() { stopped <- client.Run(ctx) }()
		waitFor(t, "the handshake", func() bool { return client.Health().Connected })
		cancel()
		<-stopped
	}

	connect(startTestRoom(t))
	testSeed := store.Stats().Seed
	if !strings.HasPrefix(testSeed, "test-mode-") || store.Stats().Checks != 0 {
		t.Fatalf("after test mode the store holds %+v", store.Stats())
	}
	if _, err := os.Stat(filepath.Join(dir, "bridge.ours.json")); err != nil {
		t.Fatalf("the real run was not set aside: %v", err)
	}

	connect(startTestRoom(t))
	if seed := store.Stats().Seed; seed == testSeed || !strings.HasPrefix(seed, "test-mode-") {
		t.Fatalf("a second test room bound %q after %q", seed, testSeed)
	}
	if _, err := os.Stat(filepath.Join(dir, "bridge."+testSeed+".json")); err != nil {
		t.Fatalf("the first test run was not set aside: %v", err)
	}

	ours := &fakeRoom{seed: "ours", slotData: slotDataFor("final_boss", "mvm_decoy", "mvm_decoy")}
	connect(ours.start(t))
	if stats := store.Stats(); stats.Seed != "ours" || stats.Checks != 1 {
		t.Fatalf("back in the real room the store holds %+v", stats)
	}
}

func startTestRoom(t *testing.T) string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	var done atomic.Bool
	room, address, err := fakeroom.Start(ctx, fakeroom.Options{
		SlotName: "tester", Goal: "final_boss", MissionCount: 3,
		Log: func(text string) {
			if !done.Load() {
				t.Log(text)
			}
		},
	})
	if err != nil {
		cancel()
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		done.Store(true)
		cancel()
		_ = room.Close(context.Background())
	})
	return address
}
