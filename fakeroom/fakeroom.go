// Package fakeroom is a multiworld of one, for play-testing without
// Archipelago.
//
// Generating a seed needs the Archipelago app, and a room needs somewhere to
// host it. Neither is any use to somebody who just wants to see whether the
// plugin locks the right classes and whether a cleared wave sends a check. This
// serves the handful of messages the bridge needs, makes up a seed from the
// game data, and hands back an unlock every time a wave is cleared.
//
// It is not Archipelago. Nothing here is random, no other game is in the room,
// and no progress leaves the machine.
package fakeroom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// Room is a running fake multiworld. Close it to stop it.
type Room struct {
	server   *http.Server
	listener net.Listener
	log      func(string)

	mu    sync.Mutex
	items []int64 // what the run hands out, in order
	next  int     // how many of them are gone
}

// Options say what the made-up seed looks like. Missions and Goal come from the
// player's own run settings, so a test run has the shape they asked for.
type Options struct {
	SlotName     string
	Missions     []string
	Goal         string
	Log          func(string)
	MissionCount int
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

	room := &Room{listener: listener, log: logf, items: unlockOrder()}
	missions := options.Missions
	if len(missions) == 0 {
		missions = defaultMissions(options.MissionCount)
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

// Close stops serving.
func (r *Room) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return r.server.Shutdown(ctx)
}

func (r *Room) serve(ctx context.Context, conn *websocket.Conn, missions []string, goal string) {
	if err := write(ctx, conn, map[string]any{
		"cmd": "RoomInfo", "seed_name": "test-mode", "password": false,
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

// handle answers the four messages that matter. Anything else is read and
// dropped, which is what the bridge does with the rest of the protocol too.
func (r *Room) handle(ctx context.Context, conn *websocket.Conn, cmd string,
	message map[string]json.RawMessage, missions []string, goal string,
) error {
	switch cmd {
	case "Connect":
		r.log("test mode: the bridge connected")
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
					"death_link":           false,
				},
			},
			// An empty first batch: the run starts with nothing, and the
			// unlocks arrive as waves are cleared.
			map[string]any{"cmd": "ReceivedItems", "index": 0, "items": []any{}},
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
		return r.reward(ctx, conn, len(checks.Locations))

	case "StatusUpdate":
		r.log("test mode: the run reported its status")
		return nil

	case "Say":
		return nil
	}
	return nil
}

// reward hands out one unlock per check, in a fixed order, so a tester sees the
// classes and the weapon slots open up as they clear waves.
func (r *Room) reward(ctx context.Context, conn *websocket.Conn, checks int) error {
	if checks <= 0 {
		return nil
	}
	r.mu.Lock()
	index := r.next
	var handed []map[string]any
	for range checks {
		if r.next >= len(r.items) {
			break
		}
		handed = append(handed, map[string]any{
			"item": r.items[r.next], "location": int64(1), "player": 1, "flags": 1,
		})
		r.next++
	}
	r.mu.Unlock()

	if len(handed) == 0 {
		r.log("test mode: a check arrived, and the run has handed out everything it has")
		return nil
	}
	r.log(fmt.Sprintf("test mode: %d check(s) arrived, sending %d unlock(s)", checks, len(handed)))
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

// gift sends one item as if another player had found it for us.
func (r *Room) gift(ctx context.Context, conn *websocket.Conn, who string) error {
	r.mu.Lock()
	if r.next >= len(r.items) {
		r.mu.Unlock()
		return nil
	}
	index, item := r.next, r.items[r.next]
	r.next++
	r.mu.Unlock()

	r.log("test mode: " + who + " sent us something")
	return write(ctx, conn,
		map[string]any{"cmd": "ReceivedItems", "index": index, "items": []map[string]any{
			{"item": item, "location": int64(1), "player": 2, "flags": 1},
		}},
	)
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
func unlockOrder() []int64 {
	var classes, slots, rest []int64
	for _, item := range gamedata.Items {
		copies := max(int(item.Count), 1)
		for range copies {
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
	return append(append(classes, slots...), rest...)
}

// defaultMissions picks the first missions the game data lists, which is the
// normal tier first: the gentlest thing to test against.
func defaultMissions(count int) []string {
	if count <= 0 || count > len(gamedata.Missions) {
		count = 8
	}
	missions := make([]string, 0, count)
	for _, mission := range gamedata.Missions[:count] {
		missions = append(missions, mission.PopFile)
	}
	return missions
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
