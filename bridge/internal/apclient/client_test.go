package apclient

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/state"
	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

// fakeRoom does the handshake and records what the bridge says to it.
type fakeRoom struct {
	seed     string
	slotData any
	items    []int64
	heard    chan map[string]json.RawMessage
	refuse   []string

	// dropFirst hangs up right after the first handshake, the way a server restarting mid-run does.
	dropFirst bool

	mu       sync.Mutex
	sessions int
}

func (f *fakeRoom) accept() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions++
	return f.sessions
}

func (f *fakeRoom) start(t *testing.T) string {
	t.Helper()
	f.heard = make(chan map[string]json.RawMessage, 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()
		f.serve(r.Context(), conn)
	}))
	t.Cleanup(server.Close)
	return "ws" + strings.TrimPrefix(server.URL, "http")
}

func (f *fakeRoom) serve(ctx context.Context, conn *websocket.Conn) {
	session := f.accept()
	if err := writeAll(ctx, conn, map[string]any{"cmd": "RoomInfo", "seed_name": f.seed}); err != nil {
		return
	}
	for {
		_, body, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var messages []map[string]json.RawMessage
		if err := json.Unmarshal(body, &messages); err != nil {
			return
		}
		for _, message := range messages {
			var cmd string
			if err := json.Unmarshal(message["cmd"], &cmd); err != nil {
				return
			}
			f.heard <- message
			if cmd != "Connect" {
				continue
			}
			if len(f.refuse) > 0 {
				_ = writeAll(ctx, conn, map[string]any{
					"cmd": "ConnectionRefused", "errors": f.refuse,
				})
				return
			}
			if err := f.acknowledge(ctx, conn); err != nil {
				return
			}
			if f.dropFirst && session == 1 {
				return
			}
		}
	}
}

func (f *fakeRoom) acknowledge(ctx context.Context, conn *websocket.Conn) error {
	items := make([]map[string]any, len(f.items))
	for i, id := range f.items {
		items[i] = map[string]any{"item": id, "location": 1, "player": 1, "flags": 1}
	}
	return writeAll(ctx, conn,
		map[string]any{
			"cmd":               "Connected",
			"team":              0,
			"slot":              1,
			"checked_locations": []int64{},
			"slot_data":         f.slotData,
		},
		map[string]any{"cmd": "ReceivedItems", "index": 0, "items": items},
	)
}

func writeAll(ctx context.Context, conn *websocket.Conn, messages ...any) error {
	body, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, body)
}

func slotDataFor(goal, goalMission string, missions ...string) map[string]any {
	return map[string]any{
		"format_version":       gamedata.FormatVersion,
		"missions":             missions,
		"goal":                 goal,
		"goal_mission":         goalMission,
		"missionsanity_target": 1,
		"death_link":           false,
	}
}

func newStore(t *testing.T) *state.Store {
	t.Helper()
	store, err := state.Open(filepath.Join(t.TempDir(), "bridge.json"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func (f *fakeRoom) sessionCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sessions
}

func runClient(t *testing.T, room *fakeRoom) (*Client, *state.Store) {
	t.Helper()
	store := newStore(t)
	return runClientWith(t, room, store), store
}

func runClientWith(t *testing.T, room *fakeRoom, store *state.Store) *Client {
	t.Helper()
	client := New(Options{
		URL:      room.start(t),
		SlotName: "tf2",
		Store:    store,
		Logger:   slog.New(slog.DiscardHandler),
	})
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- client.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-stopped
	})
	return client
}

// waitFor polls: the session runs in goroutines, so assertions about it are eventually-true.
func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// awaitCommand drains what the room heard until the named command shows up.
func awaitCommand(t *testing.T, room *fakeRoom, want string) map[string]json.RawMessage {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case message := <-room.heard:
			var cmd string
			if err := json.Unmarshal(message["cmd"], &cmd); err != nil {
				t.Fatal(err)
			}
			if cmd == want {
				return message
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s", want)
		}
	}
}

func TestSessionConnectsAndAppliesItems(t *testing.T) {
	mission, _ := gamedata.MissionByPopFile("mvm_coaltown")
	room := &fakeRoom{
		seed:     "seed-1",
		slotData: slotDataFor("final_boss", "mvm_coaltown", "mvm_coaltown"),
		items:    []int64{mission.TicketItemID(), gamedata.Classes[0].ItemID()},
	}
	client, store := runClient(t, room)

	connect := awaitCommand(t, room, "Connect")
	var game, name string
	if err := json.Unmarshal(connect["game"], &game); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(connect["name"], &name); err != nil {
		t.Fatal(err)
	}
	if game != gamedata.GameName || name != "tf2" {
		t.Fatalf("Connect carried game %q and name %q", game, name)
	}

	waitFor(t, "the session to report itself connected", func() bool {
		return client.Health().Connected
	})
	waitFor(t, "the received items to be applied", func() bool {
		unlocks := store.Unlocks()
		return len(unlocks.Missions) == 1 && len(unlocks.Classes) == 1
	})
}

func TestChecksReachTheServer(t *testing.T) {
	mission, _ := gamedata.MissionByPopFile("mvm_coaltown")
	room := &fakeRoom{
		seed:     "seed-1",
		slotData: slotDataFor("final_boss", "mvm_decoy", "mvm_coaltown", "mvm_decoy"),
	}
	client, store := runClient(t, room)
	waitFor(t, "the handshake", func() bool { return client.Health().Connected })

	if _, err := store.AddCheck(mission.WaveLocationID(2)); err != nil {
		t.Fatal(err)
	}
	message := awaitCommand(t, room, "LocationChecks")
	var locations []int64
	if err := json.Unmarshal(message["locations"], &locations); err != nil {
		t.Fatal(err)
	}
	if len(locations) != 1 || locations[0] != mission.WaveLocationID(2) {
		t.Fatalf("locations = %v", locations)
	}
}

func TestGoalIsAnnouncedOnce(t *testing.T) {
	goalMission, _ := gamedata.MissionByPopFile("mvm_decoy")
	room := &fakeRoom{
		seed:     "seed-1",
		slotData: slotDataFor("final_boss", "mvm_decoy", "mvm_decoy"),
	}
	client, store := runClient(t, room)
	waitFor(t, "the handshake", func() bool { return client.Health().Connected })

	if _, err := store.AddCheck(goalMission.ClearLocationID()); err != nil {
		t.Fatal(err)
	}
	message := awaitCommand(t, room, "StatusUpdate")
	var status int
	if err := json.Unmarshal(message["status"], &status); err != nil {
		t.Fatal(err)
	}
	if status != statusGoal {
		t.Fatalf("status = %d, want %d", status, statusGoal)
	}
	waitFor(t, "the win to be recorded", store.GoalSent)

	if _, err := store.AddCheck(goalMission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}
	awaitCommand(t, room, "LocationChecks")
	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case message := <-room.heard:
			var cmd string
			if err := json.Unmarshal(message["cmd"], &cmd); err != nil {
				t.Fatal(err)
			}
			if cmd == "StatusUpdate" {
				t.Fatal("the win was announced twice")
			}
		case <-deadline:
			return
		}
	}
}

func TestChecksTakenWhileDownAreReplayed(t *testing.T) {
	mission, _ := gamedata.MissionByPopFile("mvm_decoy")
	room := &fakeRoom{
		seed:     "seed-1",
		slotData: slotDataFor("final_boss", "mvm_decoy", "mvm_decoy"),
	}

	// The wave was cleared with nothing upstream to tell, and the plugin was answered anyway.
	store := newStore(t)
	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}

	runClientWith(t, room, store)
	message := awaitCommand(t, room, "LocationChecks")
	var locations []int64
	if err := json.Unmarshal(message["locations"], &locations); err != nil {
		t.Fatal(err)
	}
	if len(locations) != 1 || locations[0] != mission.WaveLocationID(1) {
		t.Fatalf("locations = %v", locations)
	}
}

func TestTheSessionReconnects(t *testing.T) {
	room := &fakeRoom{
		seed:      "seed-1",
		slotData:  slotDataFor("final_boss", "mvm_decoy", "mvm_decoy"),
		dropFirst: true,
	}
	client, _ := runClient(t, room)

	waitFor(t, "a second session after the server hung up", func() bool {
		return room.sessionCount() >= 2 && client.Health().Connected
	})
}

func TestMissionsanityCountsClearedMissions(t *testing.T) {
	first, _ := gamedata.MissionByPopFile("mvm_decoy")
	second, _ := gamedata.MissionByPopFile("mvm_coaltown")
	slot := slotDataFor("missionsanity", "", "mvm_decoy", "mvm_coaltown")
	slot["missionsanity_target"] = 2
	room := &fakeRoom{seed: "seed-1", slotData: slot}

	client, store := runClient(t, room)
	waitFor(t, "the handshake", func() bool { return client.Health().Connected })

	if _, err := store.AddCheck(first.ClearLocationID()); err != nil {
		t.Fatal(err)
	}
	awaitCommand(t, room, "LocationChecks")
	if store.GoalSent() {
		t.Fatal("the run was won one mission early")
	}

	if _, err := store.AddCheck(second.ClearLocationID()); err != nil {
		t.Fatal(err)
	}
	awaitCommand(t, room, "StatusUpdate")
	waitFor(t, "the win to be recorded", store.GoalSent)
}

func TestARefusedConnectionStopsTheClient(t *testing.T) {
	room := &fakeRoom{seed: "seed-1", refuse: []string{"InvalidSlot"}}
	store, err := state.Open(filepath.Join(t.TempDir(), "bridge.json"))
	if err != nil {
		t.Fatal(err)
	}
	client := New(Options{
		URL:      room.start(t),
		SlotName: "wrong",
		Store:    store,
		Logger:   slog.New(slog.DiscardHandler),
	})

	stopped := make(chan error, 1)
	go func() { stopped <- client.Run(t.Context()) }()
	select {
	case err := <-stopped:
		if err == nil || !strings.Contains(err.Error(), "InvalidSlot") {
			t.Fatalf("err = %v, want the refusal", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the client kept retrying a refused connection")
	}
}

func TestSlotDataFromAnotherFormatIsFatal(t *testing.T) {
	slotData := slotDataFor("final_boss", "mvm_decoy", "mvm_decoy")
	slotData["format_version"] = gamedata.FormatVersion + 1
	room := &fakeRoom{seed: "seed-1", slotData: slotData}

	store, err := state.Open(filepath.Join(t.TempDir(), "bridge.json"))
	if err != nil {
		t.Fatal(err)
	}
	client := New(Options{
		URL:      room.start(t),
		SlotName: "tf2",
		Store:    store,
		Logger:   slog.New(slog.DiscardHandler),
	})

	stopped := make(chan error, 1)
	go func() { stopped <- client.Run(t.Context()) }()
	select {
	case err := <-stopped:
		if err == nil || !strings.Contains(err.Error(), "data format version") {
			t.Fatalf("err = %v, want a format complaint", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the client accepted a seed it cannot read")
	}
}
