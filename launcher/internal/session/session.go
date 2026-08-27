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
	"time"
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

// Snapshot is one reading of the run.
type Snapshot struct {
	Health   Health
	Missions []Mission
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
	return snapshot, nil
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
