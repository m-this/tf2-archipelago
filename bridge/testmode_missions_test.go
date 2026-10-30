package bridge

import (
	"context"
	"log/slog"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/bridge/config"
	"github.com/m-this/tf2-archipelago/bridge/internal/apclient"
	"github.com/m-this/tf2-archipelago/bridge/internal/state"
	"github.com/m-this/tf2-archipelago/fakeroom"
	"github.com/m-this/tf2-archipelago/gamedata"
)

// The Docker setting can name more missions than the unmodded pool contains.
// Exercise the path from its environment through the room and bridge client:
// this is the mission list and inventory the launcher ultimately displays.
func TestDockerTestModeServesEverySigModMission(t *testing.T) {
	mods := []string{"sigsegv-mvm"}
	eligible := dockerTestServer(mods).Playable()
	if len(eligible) <= len(gamedata.PlayableMissions()) {
		t.Fatal("fixture no longer has more SigMod missions than the unmodded pool")
	}
	const sigmodMission = "mvm_bronx_rc2_adv_point_of_impact"
	t.Setenv("TF2AP_TEST_MODE", "1")
	t.Setenv("MVM_MISSION_COUNT", strconv.Itoa(len(eligible)))
	t.Setenv("MVM_COMMUNITY_MISSIONS", "true")
	t.Setenv("MVM_DIFFICULTY", "normal")
	t.Setenv("MVM_EXCLUDED_MISSIONS", "")
	t.Setenv("MVM_START_MISSION", sigmodMission)
	t.Setenv("SRCDS_MODS", mods[0])

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load Docker settings: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	room, address, err := fakeroom.Start(ctx, dockerTestOptions(ctx, cfg, slog.New(slog.DiscardHandler)))
	if err != nil {
		cancel()
		t.Fatalf("start test room: %v", err)
	}
	var stopped chan error
	t.Cleanup(func() {
		cancel()
		_ = room.Close(context.Background())
		if stopped == nil {
			return
		}
		if err := <-stopped; err != nil {
			t.Errorf("Run after cancel: %v", err)
		}
	})

	store, err := state.Open(filepath.Join(t.TempDir(), "bridge.json"))
	if err != nil {
		t.Fatalf("open bridge state: %v", err)
	}
	client := apclient.New(apclient.Options{
		URL: address, SlotName: cfg.SlotName, Store: store,
		Logger: slog.New(slog.DiscardHandler),
	})
	stopped = make(chan error, 1)
	go func() { stopped <- client.Run(ctx) }()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		health := client.Health()
		if health.Connected && len(health.Missions) == len(eligible) &&
			len(store.Unlocks().Of(gamedata.ItemMissionTicket)) == len(eligible) {
			if health.Missions[0] != sigmodMission {
				t.Fatalf("SigMod start mission missing from room: %v", health.Missions)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	health := client.Health()
	t.Fatalf("Docker test room served %d/%d missions and %d tickets (connected=%t, error=%q)",
		len(health.Missions), len(eligible), len(store.Unlocks().Of(gamedata.ItemMissionTicket)),
		health.Connected, health.LastError)
}
