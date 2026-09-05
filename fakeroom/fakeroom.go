// Package fakeroom is a multiworld of one, for play-testing without
// Archipelago.
//
// Generating a seed needs the Archipelago app, and a room needs somewhere to
// host it. Neither is any use to somebody who just wants to see whether the
// plugin locks the right classes and whether a cleared wave sends a check. This
// serves the handful of messages the bridge needs, makes up a seed from the
// game data, and hands back an unlock every time a wave is cleared.
//
// It is not Archipelago. Only the draw is random, no other game is in the room,
// and no progress leaves the machine.
package fakeroom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// Room is a running fake multiworld. Close it to stop it.
type Room struct {
	server    *http.Server
	listener  net.Listener
	log       func(string)
	deathLink bool

	// start is what the run holds before it clears anything. A real generator
	// precollects it, so the plugin can enforce from the first wave; a room
	// that started everybody with nothing left every class pickable and every
	// weapon in hand until the first slot happened to arrive.
	start []int64

	// seed is unique per process start, not per test session. The bridge's
	// checks are durable and keyed by seed name: a constant name here would
	// have it treat every restart as the same run, replay a check list this
	// room has no memory of granting, and open everything at once.
	seed string

	mu      sync.Mutex
	items   []int64        // what the run hands out, in order
	next    int            // how many of them are gone
	checked map[int64]bool // locations already rewarded, so a resend hands out nothing
	given   int            // starting inventory plus everything handed out since
}

// Options say what the made-up seed looks like. Missions and Goal come from the
// player's own run settings, so a test run has the shape they asked for.
type Options struct {
	SlotName     string
	Missions     []string
	Goal         string
	Log          func(string)
	MissionCount int

	// Excluded is the popfiles a test run leaves out, the way the real
	// generator's excluded_missions does.
	Excluded []string

	// Difficulty is the easiest tier the run may draw, the difficulty_pool
	// key. Empty draws from every tier. Without it a test run played the
	// first missions of the table whatever the player asked for, so the one
	// setting an evening is shaped around did nothing.
	Difficulty string

	// StartMission is the popfile the run begins on, empty for the first one
	// drawn. The same option the real generator takes.
	StartMission string

	// StartClass is the mercenary the run begins with, by name, empty or
	// unknown for the first in the table. The same option the real generator
	// takes, and for the same reason the difficulty is here: a test run that
	// hands out a Scout whatever the player asked for is not a test of the
	// evening they set up.
	StartClass string

	// DeathLink makes the made-up seed ask for it, so the deaths this room
	// invents take the team down the way a real multiworld's would.
	DeathLink bool
}

// Start serves a fake room on loopback and returns it with the address the
// bridge should dial. Port 0 asks the operating system for a free one, so it
// never fights the real Archipelago on 38281.
func Start(ctx context.Context, options Options) (*Room, string, error) {
	var config net.ListenConfig
	listener, err := config.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("cannot open a port for the test room: %w", err)
	}
	logf := options.Log
	if logf == nil {
		logf = func(string) {}
	}

	missions := options.Missions
	if len(missions) == 0 {
		missions = defaultMissions(options.MissionCount, options.Excluded,
			options.Difficulty, options.StartMission)
	}
	start := startingInventory(missions[0], options.StartClass)
	room := &Room{
		listener:  listener,
		log:       logf,
		items:     unlockOrder(start),
		start:     start,
		checked:   make(map[int64]bool),
		deathLink: options.DeathLink,
		seed:      fmt.Sprintf("test-mode-%x", rand.Uint64()),
	}
	goal := options.Goal
	if goal == "" {
		goal = "final_boss"
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()
		room.serve(r.Context(), conn, missions, goal)
	})
	room.server = &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		if err := room.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logf("the test room stopped: " + err.Error())
		}
	}()
	address := "ws://" + listener.Addr().String()
	logf("test mode: a multiworld of one is serving at " + address)
	return room, address, nil
}

// Close stops serving. The context bounds the wait for connections to go.
func (r *Room) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, closeGrace)
	defer cancel()
	return r.server.Shutdown(ctx)
}

// closeGrace is how long a connection has to finish once the room is asked to
// stop. It serves one bridge on loopback, so this is generous.
const closeGrace = 2 * time.Second

func (r *Room) serve(ctx context.Context, conn *websocket.Conn, missions []string, goal string) {
	if err := write(ctx, conn, map[string]any{
		"cmd": "RoomInfo", "seed_name": r.seed, "password": false,
	}); err != nil {
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
			if err := r.handle(ctx, conn, cmd, message, missions, goal); err != nil {
				return
			}
		}
	}
}

// handle answers the five messages that matter. Anything else is read and
// dropped, which is what the bridge does with the rest of the protocol too.
func (r *Room) handle(ctx context.Context, conn *websocket.Conn, cmd string,
	message map[string]json.RawMessage, missions []string, goal string,
) error {
	switch cmd {
	case "Connect":
		r.log("test mode: the bridge connected")
		// The starting inventory is counted before the traffic goroutine can
		// send anything of its own, or the two race for the same index.
		start := r.startItems()
		go r.traffic(ctx, conn)
		return write(ctx, conn,
			map[string]any{
				"cmd":               "Connected",
				"team":              0,
				"slot":              1,
				"checked_locations": []int64{},
				"slot_data": map[string]any{
					"format_version":       gamedata.FormatVersion,
					"missions":             missions,
					"goal":                 goal,
					"goal_mission":         missions[len(missions)-1],
					"missionsanity_target": len(missions),
					"death_link":           r.deathLink,
				},
			},
			// The starting inventory, the way a generated seed precollects
			// it. Everything else arrives as waves are cleared.
			map[string]any{"cmd": "ReceivedItems", "index": 0, "items": start},
		)

	case "LocationChecks":
		var checks struct {
			Locations []int64 `json:"locations"`
		}
		if err := json.Unmarshal(mustRaw(message), &checks); err != nil {
			// A message this room cannot read is not worth dropping the
			// connection over: it answers what it understands.
			r.log("test mode: could not read a LocationChecks message")
			return nil //nolint:nilerr // the room keeps serving
		}
		return r.reward(ctx, conn, checks.Locations)

	case "StatusUpdate":
		r.log("test mode: the run reported its status")
		return nil

	case "Bounce":
		r.log("test mode: the team lost a wave, and every Death Link player in a real room would die now")
		return nil

	case "Say":
		return r.answer(ctx, conn, message, missions)
	}
	return nil
}

// reward hands out one unlock per newly checked location, in a fixed order, so
// a tester sees the classes and the weapon slots open up as they clear waves.
//
// The bridge resends its whole check list on every report, by design (a
// reconnect mid-wave must be a non-event), the way a real Archipelago server's
// checks are idempotent too. This room has to do that deduplication itself:
// counting the resend instead of the new locations in it hands out the same
// wave's reward again on every ping, until the pool the run started with is
// gone in minutes.
func (r *Room) reward(ctx context.Context, conn *websocket.Conn, locations []int64) error {
	r.mu.Lock()
	fresh := 0
	for _, location := range locations {
		if !r.checked[location] {
			r.checked[location] = true
			fresh++
		}
	}
	if fresh == 0 {
		r.mu.Unlock()
		return nil
	}
	index := r.given
	var handed []map[string]any
	for range fresh {
		if r.next >= len(r.items) {
			break
		}
		handed = append(handed, map[string]any{
			"item": r.items[r.next], "location": int64(1), "player": 1, "flags": 1,
		})
		r.next++
		r.given++
	}
	r.mu.Unlock()

	if len(handed) == 0 {
		r.log("test mode: a check arrived, and the run has handed out everything it has")
		return nil
	}
	r.log(fmt.Sprintf("test mode: %d check(s) arrived, sending %d unlock(s)", fresh, len(handed)))
	return write(ctx, conn, map[string]any{
		"cmd": "ReceivedItems", "index": index, "items": handed,
	})
}

// traffic makes the room feel inhabited: the other players find things, send
// things over, and die.
//
// A real evening is not a quiet room that only answers when a wave ends. The
// plugin has to show an item that arrives out of nowhere, and a death that
// comes from somebody else's game, and neither path is exercised by clearing
// waves alone. Every line goes to the launcher log and, through the plugin, to
// the chat in the game.
func (r *Room) traffic(ctx context.Context, conn *websocket.Conn) {
	// Long enough that a tester can read the chat, short enough to see several
	// in one wave.
	const (
		trafficMin = 25 * time.Second
		trafficMax = 70 * time.Second
	)
	players := []string{"Ana", "Bram", "Chika", "Dinesh"}
	games := []string{"Ocarina of Time", "Hollow Knight", "Factorio", "Slay the Spire"}

	for turn := 0; ; turn++ {
		wait := trafficMin + time.Duration(rand.Int64N(int64(trafficMax-trafficMin)))
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}

		who := players[rand.IntN(len(players))]
		game := games[rand.IntN(len(games))]

		switch turn % 3 {
		case 0:
			// Somebody else found one of their own things.
			r.say(ctx, conn, fmt.Sprintf("%s found their Hookshot (%s)", who, game))
		case 1:
			// Somebody sent us something, with no wave cleared for it.
			if err := r.gift(ctx, conn, who); err != nil {
				return
			}
		case 2:
			// A death somewhere else in the multiworld.
			r.log("test mode: " + who + " died in " + game)
			if err := write(ctx, conn, map[string]any{
				"cmd":  "Bounced",
				"tags": []string{"DeathLink"},
				"data": map[string]any{
					"time":   float64(time.Now().Unix()),
					"source": who,
					"cause":  who + " walked into a saw in " + game,
				},
			}); err != nil {
				return
			}
			r.say(ctx, conn, who+" died in "+game)
		}
	}
}

// startItems is the starting inventory as ReceivedItems carries it, and it
// counts against the same index the rest of the run uses.
func (r *Room) startItems() []map[string]any {
	items := make([]map[string]any, 0, len(r.start))
	for _, id := range r.start {
		items = append(items, map[string]any{
			"item": id, "location": int64(0), "player": 0, "flags": 1,
		})
	}
	r.mu.Lock()
	r.given = len(items)
	r.mu.Unlock()
	return items
}

// gift sends one item as if another player had found it for us.
//
// Filler, never an unlock. A real multiworld does hand over progression, but
// once a minute it empties the pool by the clock rather than by play: a
// play-test had every class and every weapon slot inside twenty minutes
// without clearing a wave, which reads as a randomizer that does not work.
func (r *Room) gift(ctx context.Context, conn *websocket.Conn, who string) error {
	r.mu.Lock()
	index := r.given
	r.given++
	r.mu.Unlock()

	r.log("test mode: " + who + " sent us something")
	return write(ctx, conn,
		map[string]any{"cmd": "ReceivedItems", "index": index, "items": []map[string]any{
			{"item": fillerItem(), "location": int64(1), "player": 2, "flags": 0},
		}},
	)
}

// fillerItem is what a gift carries: the cash bundle, the one filler the
// tables hold.
func fillerItem() int64 {
	for _, item := range gamedata.Items {
		if item.Classification == gamedata.Filler {
			return item.ID
		}
	}
	return gamedata.Items[0].ID
}

// startingInventory is what a normal-tier run starts with, matching the
// apworld's own rule: the first mission's ticket, one class, one weapon slot.
// Without the ticket the plugin has no mission it may play.
func startingInventory(popFile, startClass string) []int64 {
	var start []int64
	if mission, known := gamedata.MissionByPopFile(popFile); known {
		for _, item := range gamedata.Items {
			if item.Kind == gamedata.ItemMissionTicket && item.Mission == mission.ID {
				start = append(start, item.ID)
				break
			}
		}
	}
	if class, found := classItem(startClass); found {
		start = append(start, class)
	} else {
		start = append(start, firstItemOfKind(gamedata.ItemClass)...)
	}
	start = append(start, firstItemOfKind(gamedata.ItemWeaponSlot)...)
	return start
}

// classItem is the item that grants the named mercenary. A name nothing
// matches is not an error here: the settings offer "random", and a test run
// asked for a class it cannot place starts on the first one rather than on
// none at all.
func classItem(name string) (int64, bool) {
	if name == "" {
		return 0, false
	}
	for _, class := range gamedata.Classes {
		if !strings.EqualFold(class.Name, name) {
			continue
		}
		for _, item := range gamedata.Items {
			if item.Kind == gamedata.ItemClass && item.Class == class.ID {
				return item.ID, true
			}
		}
	}
	return 0, false
}

func firstItemOfKind(kind gamedata.ItemKind) []int64 {
	for _, item := range gamedata.Items {
		if item.Kind == kind {
			return []int64{item.ID}
		}
	}
	return nil
}

// say sends a chat line the way the server does, which the plugin repeats in the
// game's chat as an [AP] line.
func (r *Room) say(ctx context.Context, conn *websocket.Conn, text string) {
	r.log("test mode: " + text)
	_ = write(ctx, conn, map[string]any{
		"cmd":  "PrintJSON",
		"type": "Chat",
		"data": []map[string]any{{"text": text}},
	})
}

// unlockOrder is what a test run hands out: the classes, then the weapon slots,
// then the rest. Classes first because the interesting case is a run that
// starts with one class and one slot and has to open up.
//
// The weapon slot item is progressive, so its copies are handed out one at a
// time and the count comes from the pool.
func unlockOrder(held []int64) []int64 {
	var classes, slots, rest []int64
	taken := make(map[int64]int, len(held))
	for _, id := range held {
		taken[id]++
	}
	for _, item := range gamedata.Items {
		if item.Classification == gamedata.Filler {
			continue
		}
		copies := max(int(item.Count), 1)
		for range copies {
			// One copy per item already held: the progressive weapon slot has
			// several, and only the granted one is off the list.
			if taken[item.ID] > 0 {
				taken[item.ID]--
				continue
			}
			switch item.Kind {
			case gamedata.ItemClass:
				classes = append(classes, item.ID)
			case gamedata.ItemWeaponSlot:
				slots = append(slots, item.ID)
			default:
				rest = append(rest, item.ID)
			}
		}
	}
	return append(append(classes, slots...), promoteBuffs(rest)...)
}

/*
	promoteBuffs brings a handful of weapon buffs to the front of the tail.

Test mode hands the run out in one order: the classes, then the weapon slots,
then everything else. Everything else is fifty mission tickets and then sixteen
thousand buffs, so the first buff arrives about sixty waves in and nobody
testing them ever sees one.

Spread rather than the first few, because the first few are sixteen levels of
one effect on one weapon. A stride across the list gives different weapons and
different effects, which is what somebody trying the feature wants to see.

Test mode only. A real run draws its own order from the multiworld and never
comes through here.
*/
func promoteBuffs(rest []int64) []int64 {
	// The set is built once. Asking gamedata per id is a scan of seventeen
	// thousand items inside a loop over seventeen thousand ids.
	isBuff := make(map[int64]bool, len(gamedata.Items))
	for _, item := range gamedata.Items {
		if item.Kind == gamedata.ItemWeaponBuff {
			isBuff[item.ID] = true
		}
	}

	var buffs, others []int64
	for _, id := range rest {
		if isBuff[id] {
			buffs = append(buffs, id)
			continue
		}
		others = append(others, id)
	}
	if len(buffs) <= buffsUpFront {
		return append(others, buffs...)
	}
	stride := len(buffs) / buffsUpFront

	front := make([]int64, 0, buffsUpFront)
	taken := make(map[int]bool, buffsUpFront)
	for i := range buffsUpFront {
		at := i * stride
		front = append(front, buffs[at])
		taken[at] = true
	}
	tail := make([]int64, 0, len(buffs)-len(front))
	for at, id := range buffs {
		if !taken[at] {
			tail = append(tail, id)
		}
	}
	return append(append(front, others...), tail...)
}

// buffsUpFront is how many arrive before the mission tickets. Enough to try the
// feature on several weapons, few enough that a test run still unlocks missions.
const buffsUpFront = 8

// defaultMissions is the pool a test run draws, shaped by the same three
// options the real generator takes: the easiest tier, the exclusions, and how
// many. The run starts on the first of them, so a named start mission goes to
// the front.
//
// Drawn at random from the pool, the way a seed is. It used to take the first
// of the table, which read from a player's chair as a randomiser that does not
// randomise: eight missions in the order of the settings list, and the next
// eight when those were unticked.
//
// The floor is a floor, not a filter: picking intermediate draws intermediate
// and everything harder, which is what difficulty_pool means.
func defaultMissions(count int, excluded []string, difficulty, startMission string) []string {
	floor, known := gamedata.DifficultyByKey(difficulty)

	playable := gamedata.PlayableMissions()
	pool := make([]string, 0, len(playable))
	for _, mission := range playable {
		if known && mission.Difficulty < floor {
			continue
		}
		if slices.Contains(excluded, mission.PopFile) {
			continue
		}
		pool = append(pool, mission.PopFile)
	}
	// A tier that excludes everything is the player's mistake, not a reason to
	// serve a room with no mission in it.
	if len(pool) == 0 {
		pool = append(pool, playable[0].PopFile)
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	// The run begins on the first mission drawn, so this is how a start
	// mission is honoured. One outside the pool is put back into it: the
	// player asked for it by name, and it beats the tier they asked for by key.
	if startMission != "" {
		if at := slices.Index(pool, startMission); at > 0 {
			pool = slices.Insert(slices.Delete(slices.Clone(pool), at, at+1), 0, startMission)
		} else if at == -1 {
			if mission, known := gamedata.MissionByPopFile(startMission); known && gamedata.IsPlayableMission(mission.ID) {
				pool = append([]string{startMission}, pool...)
			}
		}
	}

	if count <= 0 || count > len(pool) {
		count = min(8, len(pool))
	}
	return pool[:count]
}

func write(ctx context.Context, conn *websocket.Conn, messages ...any) error {
	body, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, body)
}

// mustRaw re-marshals a message so it can be read into a typed struct.
func mustRaw(message map[string]json.RawMessage) []byte {
	body, err := json.Marshal(message)
	if err != nil {
		return []byte("{}")
	}
	return body
}

/* What a room of one says back to !ap.
 *
 * A real Archipelago server answers !missing, !checked and the rest, and the
 * plugin forwards them for a player who has no Archipelago client open. This
 * room used to take the command and say nothing at all, so every one of those
 * documented commands was silence, and silence reads as a broken server rather
 * than as a test room that never had an answer.
 */
func (r *Room) answer(ctx context.Context, conn *websocket.Conn,
	message map[string]json.RawMessage, missions []string,
) error {
	var text string
	if raw, ok := message["text"]; ok {
		_ = json.Unmarshal(raw, &text)
	}

	// A copy, because the answer is built outside the lock and the traffic
	// goroutine goes on checking things while it is.
	r.mu.Lock()
	checked := maps.Clone(r.checked)
	r.mu.Unlock()

	// One command the room answers with an item rather than a sentence, and
	// the only one that changes the run.
	if command, _, _ := strings.Cut(strings.TrimPrefix(strings.TrimSpace(text), "!"), " "); strings.EqualFold(command, "unlock") {
		return r.unlockNextMission(ctx, conn)
	}

	for _, line := range replyLines(text, checked, missions) {
		r.say(ctx, conn, line)
	}
	return nil
}

/* !ap unlock mission, which is a test room's answer to a long evening.
 *
 * A mission is unlocked by receiving its ticket, and the ticket is an item the
 * seed placed somewhere. In a test run it is behind every class and every
 * weapon slot, so the first one lands on the tenth check and a mission worth
 * nine checks cannot reach it: the run reloads the same mission instead, which
 * is correct and reads as broken.
 *
 * This hands the next ticket over, and hands it over the way a multiworld
 * would: a ReceivedItems with the ticket in it, through the bridge, into the
 * plugin's unlock set. Nothing here reaches around the path being tested,
 * which is the point of a test room.
 *
 * The rest of the order is left alone. The ticket is moved to the front of
 * what has not been handed out yet rather than added, so the run still holds
 * exactly what the seed said it would.
 */
func (r *Room) unlockNextMission(ctx context.Context, conn *websocket.Conn) error {
	r.mu.Lock()
	ticket, found := takeNextTicket(r.items, r.next)
	if !found {
		r.mu.Unlock()
		r.say(ctx, conn, "Every mission this run holds is already unlocked.")
		return nil
	}
	at := r.given
	r.next++
	r.given++
	r.mu.Unlock()

	name := "a mission"
	if item, known := gamedata.ItemByID(ticket); known {
		name = item.Name
	}
	r.say(ctx, conn, "Unlocking "+name+".")
	return write(ctx, conn, map[string]any{
		"cmd": "ReceivedItems", "index": at,
		"items": []map[string]any{{"item": ticket, "location": int64(1), "player": 1, "flags": 1}},
	})
}

// takeNextTicket moves the next mission ticket to position next, so the next
// item handed out is that ticket and everything after it keeps its order.
func takeNextTicket(items []int64, next int) (int64, bool) {
	if next < 0 || next >= len(items) {
		return 0, false
	}
	for index := next; index < len(items); index++ {
		item, known := gamedata.ItemByID(items[index])
		if !known || item.Kind != gamedata.ItemMissionTicket {
			continue
		}
		ticket := items[index]
		// Right by one, from next up to the ticket. copy handles the overlap.
		copy(items[next+1:index+1], items[next:index])
		items[next] = ticket
		return ticket, true
	}
	return 0, false
}

// replyMax caps a reply. Chat carries one line at a time and a run holds a
// couple of hundred locations, so the whole list would scroll the game's chat
// off the screen and tell nobody anything.
const replyMax = 8

/* replyLines is what the room says to one line of chat, and nothing else.
 *
 * Pure, and separate from sending it: what this room knows how to answer is
 * the part worth testing, and a websocket is not.
 *
 * Only the commands whose answer this room actually holds. Anything else says
 * so, because a wrong answer from a test room is worse than no answer.
 */
func replyLines(text string, checked map[int64]bool, missions []string) []string {
	text = strings.TrimSpace(text)
	// Ordinary chat is not this room's business.
	if !strings.HasPrefix(text, "!") {
		return nil
	}
	command, _, _ := strings.Cut(strings.TrimPrefix(text, "!"), " ")

	switch strings.ToLower(command) {
	case "missing":
		return locationLines(checked, missions, false)
	case "checked":
		return locationLines(checked, missions, true)
	case "players":
		return []string{"This is a room of one: the test room plays every other slot."}
	case "hint":
		return []string{"Every item in a test run is in this world, and they arrive in order as checks land."}
	}
	return []string{"The test room does not answer !" + command + ". A real Archipelago room does."}
}

// locationLines lists what has been checked, or what has not.
func locationLines(checked map[int64]bool, missions []string, want bool) []string {
	var names []string
	total := 0
	for _, popFile := range missions {
		mission, known := gamedata.MissionByPopFile(popFile)
		if !known {
			continue
		}
		for _, location := range gamedata.Locations {
			if location.Mission != mission.ID || checked[location.ID] != want {
				continue
			}
			total++
			if len(names) < replyMax {
				names = append(names, location.Name)
			}
		}
	}

	word := "still to check"
	if want {
		word = "checked"
	}
	if total == 0 {
		return []string{fmt.Sprintf("Nothing %s.", word)}
	}
	lines := []string{fmt.Sprintf("%d %s, starting with:", total, word)}
	for _, name := range names {
		lines = append(lines, "  "+name)
	}
	if total > len(names) {
		lines = append(lines, fmt.Sprintf("  and %d more.", total-len(names)))
	}
	return lines
}
