package apclient

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/m-this/tf2-archipelago/bridge/internal/state"
	"github.com/m-this/tf2-archipelago/gamedata"
)

// fakeRoom does the handshake and records what the bridge says to it.
type fakeRoom struct {
	seed     string
	slotData any
	items    []int64
	heard    chan map[string]json.RawMessage
	refuse   []string

	// bounces is what the room pushes down as Bounced, the way another
	// player's death arrives.
	bounces chan map[string]any

	// dropFirst hangs up right after the first handshake, the way a server restarting mid-run does.
	dropFirst bool

	// games is what the other slots in the room play, one slot per entry.
	// The room answers a GetDataPackage for any of them with an empty table
	// for that game, and hangs up after the first session's last answer when
	// dropAfterPackages is set.
	games             []string
	dropAfterPackages bool

	// stallOutboundFirst keeps sending room traffic after the first handshake
	// but never reads again, so the client's ping cannot receive a pong.
	stallOutboundFirst bool

	// checkedLocations is what the room claims this slot has already checked,
	// the way another player's !collect checks a location that holds this
	// slot's item.
	checkedLocations []int64

	mu       sync.Mutex
	sessions int
	answered int
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
	f.bounces = make(chan map[string]any, 8)
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
			if cmd == "GetDataPackage" {
				var ask getDataPackage
				if err := json.Unmarshal(message["games"], &ask.Games); err != nil {
					return
				}
				if err := f.answerPackage(ctx, conn, ask.Games); err != nil {
					return
				}
				if f.dropAfterPackages && session == 1 && f.packagesAnswered() == len(f.games) {
					return
				}
				continue
			}
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
			go f.pushBounces(ctx, conn)
			if f.dropFirst && session == 1 {
				return
			}
			if f.stallOutboundFirst && session == 1 {
				f.pushRoomTraffic(ctx, conn)
				return
			}
		}
	}
}

func (f *fakeRoom) pushRoomTraffic(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := writeAll(ctx, conn, map[string]any{"cmd": "RoomUpdate"}); err != nil {
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
	slotInfo := map[string]any{}
	for i, game := range f.games {
		slotInfo[strconv.Itoa(i+2)] = map[string]any{"name": "other" + strconv.Itoa(i), "game": game}
	}
	return writeAll(ctx, conn,
		map[string]any{
			"cmd":               "Connected",
			"team":              0,
			"slot":              1,
			"checked_locations": f.checkedLocations,
			"slot_data":         f.slotData,
			"slot_info":         slotInfo,
		},
		map[string]any{"cmd": "ReceivedItems", "index": 0, "items": items},
	)
}

// answerPackage is the room's DataPackage for the games asked, tables empty.
func (f *fakeRoom) answerPackage(ctx context.Context, conn *websocket.Conn, games []string) error {
	tables := map[string]any{}
	for _, game := range games {
		tables[game] = map[string]any{"item_name_to_id": map[string]int64{}, "location_name_to_id": map[string]int64{}}
	}
	f.mu.Lock()
	f.answered += len(games)
	f.mu.Unlock()
	return writeAll(ctx, conn, map[string]any{"cmd": "DataPackage", "data": map[string]any{"games": tables}})
}

func (f *fakeRoom) packagesAnswered() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.answered
}

func (f *fakeRoom) pushBounces(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case bounce := <-f.bounces:
			if err := writeAll(ctx, conn, bounce); err != nil {
				return
			}
		}
	}
}

// bounce is a DeathLink from another slot in the room.
func (f *fakeRoom) bounce(t *testing.T, source, cause string) {
	t.Helper()
	f.bounces <- map[string]any{
		"cmd":  "Bounced",
		"tags": []string{deathLinkTag},
		"data": map[string]any{"time": 1.0, "source": source, "cause": cause},
	}
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
		return len(unlocks.Of(gamedata.ItemMissionTicket)) == 1 &&
			len(unlocks.Of(gamedata.ItemClass)) == 1
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

func TestTheSessionReconnectsWhenOnlyTheOutboundPumpStops(t *testing.T) {
	mission, _ := gamedata.MissionByPopFile("mvm_decoy")
	room := &fakeRoom{
		seed:               "seed-1",
		slotData:           slotDataFor("final_boss", "mvm_decoy", "mvm_decoy"),
		stallOutboundFirst: true,
	}
	store := newStore(t)
	client := New(Options{
		URL:      room.start(t),
		SlotName: "tf2",
		Store:    store,
		Logger:   slog.New(slog.DiscardHandler),
	})
	client.pingEvery = 10 * time.Millisecond
	client.pingTimeout = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- client.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-stopped
	})
	waitFor(t, "the first handshake", func() bool { return client.Health().Connected })
	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}

	waitFor(t, "a replacement session after the outbound pump stopped", func() bool {
		return room.sessionCount() >= 2 && client.Health().Connected
	})
	message := awaitCommand(t, room, "LocationChecks")
	var locations []int64
	if err := json.Unmarshal(message["locations"], &locations); err != nil {
		t.Fatal(err)
	}
	if len(locations) != 1 || locations[0] != mission.WaveLocationID(1) {
		t.Fatalf("locations = %v", locations)
	}
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

// TestAnAdoptedGoalIsNotAnnouncedByItself covers the report a play-tester
// filed: another player finishing their game and running !collect checks
// every location that holds their item, a mission clear among them, and the
// room hands that back on the very first handshake. Adopting it must not
// read as this server having beaten the mission.
func TestAnAdoptedGoalIsNotAnnouncedByItself(t *testing.T) {
	goalMission, _ := gamedata.MissionByPopFile("mvm_decoy")
	room := &fakeRoom{
		seed:             "seed-1",
		slotData:         slotDataFor("final_boss", "mvm_decoy", "mvm_decoy"),
		checkedLocations: []int64{goalMission.ClearLocationID()},
	}
	client, store := runClient(t, room)
	waitFor(t, "the handshake", func() bool { return client.Health().Connected })
	waitFor(t, "the adopted check to be held", func() bool {
		return len(store.Checks()) == 1
	})

	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case message := <-room.heard:
			var cmd string
			if err := json.Unmarshal(message["cmd"], &cmd); err != nil {
				t.Fatal(err)
			}
			if cmd == "StatusUpdate" {
				t.Fatal("the goal was announced from a check the room adopted, not one this server played")
			}
		case <-deadline:
			goto reported
		}
	}
reported:
	if store.GoalSent() {
		t.Fatal("goal_sent was set without this server ever playing the goal mission")
	}

	// The same mission, actually cleared here, is what the run's own win looks like.
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
}

/*
A public room holds many games, and one data package for all of them went past
the read limit: EMann's bundle showed the session dropping a second after every
connect, fifty-nine times, with nothing reaching the room. The names are asked
for one game at a time now, and a reconnect asks only for what is missing.
*/
func TestNamesAreAskedForOneGameAtATimeAndOnlyOnce(t *testing.T) {
	room := &fakeRoom{
		seed:              "seed-1",
		slotData:          slotDataFor("final_boss", "mvm_decoy", "mvm_decoy"),
		games:             []string{"A Link to the Past", "Hollow Knight", "Pokemon Emerald"},
		dropAfterPackages: true,
	}
	client, _ := runClient(t, room)

	waitFor(t, "a second session after the server hung up", func() bool {
		return room.sessionCount() >= 2 && client.Health().Connected
	})
	// Give the second session time to ask again if it were going to.
	time.Sleep(100 * time.Millisecond)

	var asked [][]string
	for {
		select {
		case message := <-room.heard:
			var cmd string
			_ = json.Unmarshal(message["cmd"], &cmd)
			if cmd != "GetDataPackage" {
				continue
			}
			var games []string
			_ = json.Unmarshal(message["games"], &games)
			asked = append(asked, games)
			continue
		default:
		}
		break
	}
	if len(asked) != 3 {
		t.Fatalf("asked %d times, want once per game and nothing on the reconnect: %v", len(asked), asked)
	}
	for _, games := range asked {
		if len(games) != 1 {
			t.Errorf("one request carried %v, want one game", games)
		}
	}
}
