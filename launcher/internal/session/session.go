// Package session reads the state of the run off the bridge, for the window's
// Session tab: whether the multiworld is connected, how many checks the run
// holds, and which missions are unlocked and cleared. The bridge already
// serves all of it to the plugin on loopback; this is the same two requests.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// BridgeURL is where the launcher's own bridge listens.
const BridgeURL = "http://127.0.0.1:24680"

// requestTimeout bounds each request. The bridge is on loopback, so anything
// slower is a bridge that is not there.
const requestTimeout = 3 * time.Second

var client = &http.Client{Timeout: requestTimeout}

// Health is the bridge's /healthz, the fields the tab shows.
type Health struct {
	Connected bool   `json:"connected"`
	Slot      string `json:"slot"`
	Seed      string `json:"seed"`
	Checks    int    `json:"checks"`
	Items     int    `json:"items"`
	DeathLink bool   `json:"death_link"`
	GoalSent  bool   `json:"goal_sent"`
	LastCheck string `json:"last_check"`
	LastError string `json:"last_error"`
}

// Mission is one mission of the run, as /missions lists them.
type Mission struct {
	PopFile  string `json:"popfile"`
	Name     string `json:"name"`
	Map      string `json:"map"`
	Waves    int    `json:"waves"`
	Source   string `json:"source"`
	Unlocked bool   `json:"unlocked"`
	Cleared  bool   `json:"cleared"`

	// Played is this server having cleared it, where Cleared is only the room
	// holding the check. Another world's !collect sends every check it still
	// has, so the two disagree and the run list has to show what you did.
	Played bool `json:"played"`
}

/*
	Unlock is one thing the run has handed this slot, named for a person.

The bridge serves the unlock set by kind and key, the way the plugin wants it:
"class" holds "scout", "weapon_buff" holds "weapon-001-damage", and a buff
held twice is its key twice. A player reading the tab wants the Scout, the
Scattergun's damage bonus and a level, so the keys are turned into names here,
once, from the same tables the bridge named them from.
*/
type Unlock struct {
	Kind  string
	Name  string
	Level int
}

// Snapshot is one reading of the run.
type Snapshot struct {
	Health   Health
	Missions []Mission
	Unlocks  []Unlock
}

// Fetch reads the run off the bridge at baseURL.
func Fetch(ctx context.Context, baseURL string) (Snapshot, error) {
	var snapshot Snapshot
	if err := get(ctx, baseURL+"/healthz", &snapshot.Health); err != nil {
		return Snapshot{}, err
	}
	var missions struct {
		Missions []Mission `json:"missions"`
	}
	if err := get(ctx, baseURL+"/missions", &missions); err != nil {
		return Snapshot{}, err
	}
	snapshot.Missions = missions.Missions
	var unlocks struct {
		ByKind map[string][]string `json:"unlocks"`
	}
	if err := get(ctx, baseURL+"/unlocks", &unlocks); err != nil {
		return Snapshot{}, err
	}
	snapshot.Unlocks = Describe(unlocks.ByKind)
	return snapshot, nil
}

// kindOrder is the order the tab lists kinds in: what you can play, then what
// you can hold, then where you can go, then what your weapons gained.
var kindOrder = []string{"class", "weapon_slot", "mission_ticket", "weapon_buff"}

// kindLabels is what each kind reads as on the tab.
var kindLabels = map[string]string{
	"class": "Class", "weapon_slot": "Weapon slot", "mission_ticket": "Mission", "weapon_buff": "Weapon buff",
}

// Describe turns the bridge's unlock set into rows: one per distinct key, in a
// fixed order of kinds and then by name, with the level counting how many
// times the key was handed over.
func Describe(byKind map[string][]string) []Unlock {
	names := buffNames()
	var out []Unlock
	for _, kind := range kindOrder {
		counts := map[string]int{}
		var order []string
		for _, key := range byKind[kind] {
			if counts[key] == 0 {
				order = append(order, key)
			}
			counts[key]++
		}
		rows := make([]Unlock, 0, len(order))
		for _, key := range order {
			rows = append(rows, Unlock{Kind: kindLabels[kind], Name: nameOf(kind, key, names), Level: counts[key]})
		}
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
		out = append(out, rows...)
	}
	return out
}

func nameOf(kind, key string, buffs map[string]string) string {
	switch kind {
	case "class":
		for _, class := range gamedata.Classes {
			if class.Key == key {
				return class.Name
			}
		}
	case "weapon_slot":
		for _, slot := range gamedata.WeaponSlots {
			if slot.Key == key {
				return slot.Name
			}
		}
	case "mission_ticket":
		if mission, ok := gamedata.MissionByPopFile(key); ok {
			return mission.Name
		}
	case "weapon_buff":
		if name, ok := buffs[key]; ok {
			return name
		}
	}
	return key
}

// buffNames is every weapon buff by key, as "Weapon: what it does".
func buffNames() map[string]string {
	names := make(map[string]string, len(gamedata.WeaponBuffs))
	for _, buff := range gamedata.WeaponBuffs {
		names[buff.Key] = buff.Weapon + ": " + buff.Description
	}
	return names
}

func get(ctx context.Context, url string, into any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("the bridge is not answering: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the bridge answered %s for %s", response.Status, url)
	}
	return json.NewDecoder(response.Body).Decode(into)
}

// Summary is the one line the status bar shows about the multiworld.
func (h Health) Summary() string {
	switch {
	case h.Connected:
		return fmt.Sprintf("connected as %s, %d checks, %d items", h.Slot, h.Checks, h.Items)
	case h.LastError != "":
		return "not connected: " + h.LastError
	default:
		return "not connected yet"
	}
}
