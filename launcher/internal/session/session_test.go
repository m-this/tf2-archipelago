package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchReadsTheBridge(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"api_version":3,"connected":true,"slot":"tf2","checks":4,"items":2}`))
	})
	mux.HandleFunc("GET /missions", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"missions":[{"popfile":"mvm_decoy","name":"Doe's Drill","map":"mvm_decoy","waves":8,"unlocked":true,"cleared":false}]}`))
	})
	mux.HandleFunc("GET /unlocks", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"resume_from":0,"unlocks":{"class":["scout"],"weapon_slot":["primary"],"mission_ticket":["mvm_decoy"],"weapon_buff":["weapon-001-damage","weapon-001-damage"]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	snapshot, err := Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !snapshot.Health.Connected || snapshot.Health.Checks != 4 {
		t.Errorf("health = %+v", snapshot.Health)
	}
	if len(snapshot.Missions) != 1 || snapshot.Missions[0].Name != "Doe's Drill" || !snapshot.Missions[0].Unlocked {
		t.Errorf("missions = %+v", snapshot.Missions)
	}
	if got := snapshot.Health.Summary(); got != "connected as tf2, 4 checks, 2 items" {
		t.Errorf("summary = %q", got)
	}
	if len(snapshot.Unlocks) != 4 {
		t.Fatalf("unlocks = %+v", snapshot.Unlocks)
	}
	buff := snapshot.Unlocks[3]
	if buff.Kind != "Weapon buff" || buff.Level != 2 || buff.Name == "weapon-001-damage" {
		t.Errorf("a buff held twice reads as %+v", buff)
	}
}

// The tab reads names, not keys: the Scout rather than "scout", the mission's
// name rather than its popfile, and a buff held twice as level 2. A key the
// tables do not know is shown as it came, which is a seed from a newer build.
func TestDescribeNamesTheUnlocks(t *testing.T) {
	rows := Describe(map[string][]string{
		"class":          {"scout"},
		"weapon_slot":    {"primary"},
		"mission_ticket": {"mvm_decoy", "mvm_potato"},
		"weapon_buff":    {"weapon-001-damage", "weapon-001-damage", "weapon-001-clip-size"},
	})
	want := []string{"Class Scout 1", "Weapon slot Primary 1", "Mission Doe's Drill 1", "Mission mvm_potato 1"}
	for i, expect := range want {
		if got := rows[i].Kind + " " + rows[i].Name + " " + itoa(rows[i].Level); got != expect {
			t.Errorf("row %d = %q, want %q", i, got, expect)
		}
	}
	levels := map[string]int{}
	for _, row := range rows[4:] {
		levels[row.Name] = row.Level
	}
	if len(levels) != 2 {
		t.Errorf("buffs = %v", levels)
	}
	for name, level := range levels {
		if (level == 2) != (name != "" && name[len(name)-len("damage"):] == "damage") {
			t.Errorf("%s has level %d", name, level)
		}
	}
}

func itoa(n int) string { return string(rune('0' + n)) }

func TestFetchSaysWhenTheBridgeIsGone(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	if _, err := Fetch(context.Background(), server.URL); err == nil {
		t.Fatal("a closed bridge did not fail")
	}
}
