package fakeroom

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// client drives the room's websocket directly, the way the bridge does, so the
// dedup and the seed can be checked without a whole apclient session.
type client struct {
	t    *testing.T
	conn *websocket.Conn
}

func dial(t *testing.T, address string) *client {
	t.Helper()
	conn, handshake, err := websocket.Dial(t.Context(), address, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if handshake != nil && handshake.Body != nil {
		_ = handshake.Body.Close()
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return &client{t: t, conn: conn}
}

func (c *client) send(message any) {
	c.t.Helper()
	body, err := json.Marshal([]any{message})
	if err != nil {
		c.t.Fatal(err)
	}
	if err := c.conn.Write(c.t.Context(), websocket.MessageText, body); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

// await reads messages until one with the wanted cmd shows up.
func (c *client) await(want string) map[string]json.RawMessage {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(c.t.Context(), 3*time.Second)
	defer cancel()
	for {
		_, body, err := c.conn.Read(ctx)
		if err != nil {
			c.t.Fatalf("waiting for %s: %v", want, err)
		}
		var messages []map[string]json.RawMessage
		if err := json.Unmarshal(body, &messages); err != nil {
			c.t.Fatal(err)
		}
		for _, message := range messages {
			var cmd string
			if err := json.Unmarshal(message["cmd"], &cmd); err != nil {
				c.t.Fatal(err)
			}
			if cmd == want {
				return message
			}
		}
	}
}

func startRoom(t *testing.T) string {
	t.Helper()
	room, address, err := Start(t.Context(), Options{
		SlotName:     "tester",
		MissionCount: 2,
		Log:          func(string) {},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = room.Close(context.Background()) })
	return address
}

// The starting inventory is what a generated seed precollects. Without it the
// plugin has no class, no weapon slot and no mission it may play, so it
// enforces nothing: a play-test spent four waves able to pick any class.
func TestTheRunStartsWithSomethingToPlay(t *testing.T) {
	// Not connect(): Connected and the starting inventory ride in one frame,
	// and await returns the first match in it, dropping the rest.
	c := dial(t, startRoom(t))
	c.await("RoomInfo")
	c.send(map[string]any{"cmd": "Connect"})
	received := c.await("ReceivedItems")

	var payload struct {
		Index int `json:"index"`
		Items []struct {
			Item int64 `json:"item"`
		} `json:"items"`
	}
	if err := json.Unmarshal(mustRaw(received), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Index != 0 {
		t.Errorf("the starting inventory came at index %d", payload.Index)
	}

	kinds := map[gamedata.ItemKind]int{}
	for _, held := range payload.Items {
		item, known := gamedata.ItemByID(held.Item)
		if !known {
			t.Fatalf("the room handed out item %d, which the tables do not know", held.Item)
		}
		kinds[item.Kind]++
	}
	for _, kind := range []gamedata.ItemKind{
		gamedata.ItemMissionTicket, gamedata.ItemClass, gamedata.ItemWeaponSlot,
	} {
		if kinds[kind] != 1 {
			t.Errorf("the starting inventory holds %d of %s, want 1", kinds[kind], kind.Key())
		}
	}
}

// A gift is filler. The room used to hand out the run's own unlocks once a
// minute, which emptied the pool by the clock rather than by play.
func TestGiftsAreFillerRatherThanProgression(t *testing.T) {
	item, known := gamedata.ItemByID(fillerItem())
	if !known || item.Classification != gamedata.Filler {
		t.Fatalf("a gift carries %+v, want filler", item)
	}
}

// What the starting inventory holds is off the list the checks draw from, so
// nothing is handed out twice.
func TestTheUnlockOrderDropsWhatTheRunAlreadyHolds(t *testing.T) {
	start := startingInventory("mvm_decoy", "")
	order := unlockOrder(start)

	slots := 0
	for _, id := range order {
		item, _ := gamedata.ItemByID(id)
		if item.Kind == gamedata.ItemWeaponSlot {
			// The progressive slot has several copies and the run holds one.
			slots++
			continue
		}
		if item.Classification == gamedata.Filler {
			t.Error("filler is in the unlock pool")
		}
		if slices.Contains(start, id) {
			t.Errorf("%s is in the starting inventory and still in the pool", item.Name)
		}
	}
	held := 0
	for _, id := range start {
		if item, _ := gamedata.ItemByID(id); item.Kind == gamedata.ItemWeaponSlot {
			held++
		}
	}
	for _, item := range gamedata.Items {
		if item.Kind != gamedata.ItemWeaponSlot {
			continue
		}
		if want := int(item.Count) - held; slots != want {
			t.Errorf("the pool holds %d weapon slots, want %d", slots, want)
		}
	}
}

func TestDefaultMissionsSkipTheExcluded(t *testing.T) {
	for range 20 {
		got := defaultMissions(2, []string{"mvm_decoy"}, "", "")
		if len(got) != 2 || slices.Contains(got, "mvm_decoy") {
			t.Errorf("missions = %v", got)
		}
	}
}

// Eight missions in the order of the settings list, and the next eight once
// those were unticked, read from a player's chair as a randomiser that does
// not randomise. The draw is a draw.
func TestDefaultMissionsAreDrawnAtRandom(t *testing.T) {
	first := defaultMissions(8, nil, "", "")
	for range 30 {
		if !slices.Equal(defaultMissions(8, nil, "", ""), first) {
			return
		}
	}
	t.Errorf("thirty draws of eight all came out as %v", first)
}

// The start mission is honoured whatever the draw did.
func TestDefaultMissionsPutTheStartMissionFirst(t *testing.T) {
	for range 20 {
		if got := defaultMissions(3, nil, "", "mvm_decoy_advanced"); got[0] != "mvm_decoy_advanced" {
			t.Errorf("missions = %v", got)
		}
	}
}

// A test run is what a player shapes an evening with before generating a real
// seed. It drew the first missions of the table whatever tier they picked, so
// the one setting that decides how hard the evening is did nothing.
func TestDefaultMissionsRespectTheTier(t *testing.T) {
	for _, key := range []string{"intermediate", "advanced", "expert"} {
		floor, known := gamedata.DifficultyByKey(key)
		if !known {
			t.Fatalf("%s is not a tier", key)
		}
		for _, popFile := range defaultMissions(8, nil, key, "") {
			mission, ok := gamedata.MissionByPopFile(popFile)
			if !ok {
				t.Fatalf("drew %s, which is not a mission", popFile)
			}
			// A floor, not a filter: the tier and everything harder.
			if mission.Difficulty < floor {
				t.Errorf("%s drew %s, which is %s", key, mission.Name, mission.Difficulty.Key())
			}
		}
	}
	// An unknown key is a typo in a settings file, and draws the whole pool
	// rather than nothing.
	if got := defaultMissions(3, nil, "nonsense", ""); len(got) != 3 {
		t.Errorf("an unknown tier drew %v", got)
	}
}

// The run begins on the first mission drawn, so a named start mission has to
// come first even when the tier would not have drawn it at all.
func TestDefaultMissionsStartWhereAsked(t *testing.T) {
	got := defaultMissions(4, nil, "normal", "mvm_coaltown_advanced")
	if len(got) == 0 || got[0] != "mvm_coaltown_advanced" {
		t.Fatalf("missions = %v", got)
	}
	if len(got) != 4 {
		t.Errorf("drew %d missions, want 4: %v", len(got), got)
	}
	// Named but outside the tier: the player asked for it by name, which is
	// more specific than the tier they asked for by key.
	got = defaultMissions(2, nil, "expert", "mvm_decoy")
	if len(got) == 0 || got[0] != "mvm_decoy" {
		t.Errorf("missions = %v", got)
	}
	// Not a mission at all: ignored rather than served as one.
	got = defaultMissions(2, nil, "normal", "mvm_nowhere")
	if slices.Contains(got, "mvm_nowhere") {
		t.Errorf("served a mission that does not exist: %v", got)
	}
}

func connect(t *testing.T, address string) *client {
	t.Helper()
	c := dial(t, address)
	c.await("RoomInfo")
	c.send(map[string]any{"cmd": "Connect"})
	c.await("Connected")
	return c
}

func itemCount(t *testing.T, message map[string]json.RawMessage) int {
	t.Helper()
	var items []json.RawMessage
	if err := json.Unmarshal(message["items"], &items); err != nil {
		t.Fatal(err)
	}
	return len(items)
}

// This is the bug: the bridge resends its whole check list on every report,
// which is correct against a real server because a repeat check there hands
// out nothing new. The room has to make that true itself, or a resend floods
// unlocks nobody earned.
func TestASecondReportOfTheSameChecksGrantsNothingMore(t *testing.T) {
	address := startRoom(t)
	c := connect(t, address)

	c.send(map[string]any{"cmd": "LocationChecks", "locations": []int64{111, 222}})
	first := c.await("ReceivedItems")
	if got := itemCount(t, first); got != 2 {
		t.Fatalf("first report granted %d item(s), want 2", got)
	}

	// The same two locations again, the way a reconnect or a ping-triggered
	// report resends them. A third, genuinely new one is mixed in so a
	// wrongly-silent room cannot be mistaken for one that dropped the message.
	c.send(map[string]any{"cmd": "LocationChecks", "locations": []int64{111, 222, 333}})
	second := c.await("ReceivedItems")
	if got := itemCount(t, second); got != 1 {
		t.Fatalf("the resend granted %d item(s), want 1 for the one new location", got)
	}
}

// Test mode's state on the bridge side is durable and keyed by seed name
// (BindSeed). A constant seed name would have every restart treat the
// bridge's already-recorded checks as still due a reward, and this
// memory-only room has no record of ever having paid them.
func TestEachStartUsesADifferentSeed(t *testing.T) {
	addressA := startRoom(t)
	addressB := startRoom(t)

	seedA := seedOf(t, addressA)
	seedB := seedOf(t, addressB)
	if seedA == seedB {
		t.Fatalf("two rooms shared the seed %q", seedA)
	}
}

func seedOf(t *testing.T, address string) string {
	t.Helper()
	c := dial(t, address)
	roomInfo := c.await("RoomInfo")
	var seed string
	if err := json.Unmarshal(roomInfo["seed_name"], &seed); err != nil {
		t.Fatal(err)
	}
	return seed
}

// The class a test run starts on is the one the player asked for. It used to be
// whichever came first in the tables, which is the Scout, so a run set up to
// start on a Pyro started on a Scout and nothing said why.
func TestTheStartingInventoryHoldsTheClassAskedFor(t *testing.T) {
	for _, name := range []string{"Pyro", "Medic", "Spy"} {
		start := startingInventory("mvm_decoy", name)

		found := ""
		for _, id := range start {
			item, known := gamedata.ItemByID(id)
			if known && item.Kind == gamedata.ItemClass {
				found = item.Name
			}
		}
		if !strings.Contains(found, name) {
			t.Errorf("a run starting on %s holds %q", name, found)
		}
	}
}

// Random is what the settings offer, and an unknown name is what an edited
// config file holds. Neither may leave the run with no class at all.
func TestTheStartingInventoryFallsBackToAClass(t *testing.T) {
	for _, name := range []string{"", "random", "Saxton Hale"} {
		start := startingInventory("mvm_decoy", name)

		classes := 0
		for _, id := range start {
			if item, known := gamedata.ItemByID(id); known && item.Kind == gamedata.ItemClass {
				classes++
			}
		}
		if classes != 1 {
			t.Errorf("%q started the run with %d classes, want 1", name, classes)
		}
	}
}

// The plugin forwards !ap commands to the room for a player with no
// Archipelago client open. This room took them and said nothing at all, which
// reads as a broken server rather than as a test room with no answer.
func TestTheRoomAnswersTheChatCommands(t *testing.T) {
	missions := []string{"mvm_coaltown"}
	checked := map[int64]bool{}

	// Coaltown is nine checks and none of them are done.
	missing := strings.Join(locationLines(checked, missions, false), "\n")
	if !strings.Contains(missing, "9 still to check") {
		t.Errorf("!missing did not count the run's locations:\n%s", missing)
	}
	if lines := replyLines("!checked", checked, missions); !strings.Contains(lines[0], "Nothing checked") {
		t.Errorf("!checked said %q", lines[0])
	}
	for _, command := range []string{"!players", "!hint Scout", "!nonsense"} {
		if len(replyLines(command, checked, missions)) == 0 {
			t.Errorf("%s was answered with silence", command)
		}
	}
	if lines := replyLines("!nonsense", checked, missions); !strings.Contains(lines[0], "!nonsense") {
		t.Errorf("an unknown command was not named back: %q", lines[0])
	}
}

// A run holds a couple of hundred locations and chat carries one line at a
// time, so the list is capped and says how much it left out.
func TestTheRoomCapsTheList(t *testing.T) {
	lines := locationLines(map[int64]bool{}, []string{"mvm_decoy", "mvm_coaltown"}, false)

	if len(lines) > replyMax+2 {
		t.Errorf("a reply of %d lines is too long for chat", len(lines))
	}
	if !strings.Contains(lines[len(lines)-1], "more") {
		t.Errorf("the reply does not say what it left out: %q", lines[len(lines)-1])
	}
}

// A line that is not a command is chat, and chat is not this room's business.
func TestTheRoomIgnoresOrdinaryChat(t *testing.T) {
	if lines := replyLines("hello everyone", map[int64]bool{}, []string{"mvm_coaltown"}); lines != nil {
		t.Errorf("the room answered ordinary chat with %v", lines)
	}
}

// !ap unlock mission hands over the next ticket. A test run holds its tickets
// behind every class and every weapon slot, so the first one lands on the tenth
// check and a mission worth nine cannot reach it.
func TestTakeNextTicketMovesItToTheFront(t *testing.T) {
	items := unlockOrder(startingInventory("mvm_decoy", ""))

	firstTicket := -1
	for index, id := range items {
		if item, known := gamedata.ItemByID(id); known && item.Kind == gamedata.ItemMissionTicket {
			firstTicket = index
			break
		}
	}
	if firstTicket <= 0 {
		t.Fatalf("the pool has its first ticket at %d, so there is nothing to move", firstTicket)
	}

	before := slices.Clone(items)
	ticket, found := takeNextTicket(items, 0)
	if !found {
		t.Fatal("no ticket in the pool")
	}
	if items[0] != ticket {
		t.Errorf("the ticket is at %v, not at the front", slices.Index(items, ticket))
	}

	// Everything else keeps its order: the run still holds what the seed said.
	moved := slices.DeleteFunc(slices.Clone(items), func(id int64) bool { return id == ticket })
	kept := slices.DeleteFunc(before, func(id int64) bool { return id == ticket })
	if !slices.Equal(moved, kept) {
		t.Error("moving the ticket reordered the rest of the pool")
	}
}

// A run whose tickets have all been handed out says so rather than moving
// something that is not a ticket.
func TestTakeNextTicketWithNoneLeft(t *testing.T) {
	items := []int64{}
	for _, item := range gamedata.Items {
		if item.Kind == gamedata.ItemClass {
			items = append(items, item.ID)
		}
	}
	if _, found := takeNextTicket(items, 0); found {
		t.Error("a pool of classes gave up a ticket")
	}
	if _, found := takeNextTicket(items, len(items)); found {
		t.Error("a pool with nothing left gave up a ticket")
	}
}
