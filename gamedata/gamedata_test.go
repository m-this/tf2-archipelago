package gamedata

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// freeze adds ids that are not in the frozen file yet. It never rewrites one
// that is already there, so the append-only rule holds even when the tool that
// maintains the file is the test itself.
var freeze = flag.Bool("freeze", false, "record new ids in "+frozenIDsFile)

// exportDir is the committed copy the apworld ships with.
const exportDir = "../apworld/tf2_mvm/data"

// frozenIDsFile pins every id that has ever been exported. Adding a name to it
// is routine; changing the id next to a name that is already there is not, and
// the diff is meant to be loud in review. See ADR 0001.
const frozenIDsFile = "testdata/ids-frozen.json"

func TestValidate(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWaveCountsFitTheIDScheme(t *testing.T) {
	for _, m := range Missions {
		if m.Waves < 1 || m.Waves > WavesMax {
			t.Errorf("%s: %d waves, outside 1..%d", m.PopFile, m.Waves, WavesMax)
		}
		if got := m.WaveLocationID(m.Waves); got >= m.ClearLocationID() {
			t.Errorf("%s: last wave id %d collides with the mission clear at %d", m.PopFile, got, m.ClearLocationID())
		}
	}
}

func TestIDsAreUniqueAcrossTheWholeSpace(t *testing.T) {
	seen := make(map[int64]string, len(Locations)+len(Items))
	for _, l := range Locations {
		if other, clash := seen[l.ID]; clash {
			t.Errorf("id %d: %q and %q", l.ID, other, l.Name)
		}
		seen[l.ID] = l.Name
	}
	for _, it := range Items {
		if other, clash := seen[it.ID]; clash {
			t.Errorf("id %d: %q and %q", it.ID, other, it.Name)
		}
		seen[it.ID] = it.Name
	}
	if len(seen) != len(Locations)+len(Items) {
		t.Fatalf("%d distinct ids for %d entities", len(seen), len(Locations)+len(Items))
	}
}

func TestIDsNeverMove(t *testing.T) {
	body, err := os.ReadFile(frozenIDsFile)
	if err != nil {
		t.Fatal(err)
	}
	var frozen map[string]int64
	if err := json.Unmarshal(body, &frozen); err != nil {
		t.Fatal(err)
	}
	current := make(map[string]int64, len(Locations)+len(Items))
	for _, l := range Locations {
		current[l.Name] = l.ID
	}
	for _, it := range Items {
		current[it.Name] = it.ID
	}
	if *freeze {
		if err := recordNewIDs(frozen, current); err != nil {
			t.Fatal(err)
		}
	}
	for name, id := range frozen {
		switch got, ok := current[name]; {
		case !ok:
			t.Errorf("%q is gone; a deleted entity keeps its id as a tombstone", name)
		case got != id:
			t.Errorf("%q moved from %d to %d; every seed holding %d now means something else", name, id, got, id)
		}
	}
}

func recordNewIDs(frozen, current map[string]int64) error {
	for name, id := range current {
		if _, already := frozen[name]; !already {
			frozen[name] = id
		}
	}
	body, err := json.MarshalIndent(frozen, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(frozenIDsFile, append(body, '\n'), 0o644)
}

func TestLocationsCoverEveryWaveAndMission(t *testing.T) {
	waves, clears := 0, 0
	for _, l := range Locations {
		switch l.Kind {
		case ObjectiveWaveCleared:
			waves++
		case ObjectiveMissionCleared:
			clears++
		}
	}
	want := 0
	for _, m := range Missions {
		want += int(m.Waves)
	}
	if waves != want {
		t.Errorf("%d wave locations, want %d", waves, want)
	}
	if clears != len(Missions) {
		t.Errorf("%d mission clear locations, want %d", clears, len(Missions))
	}
}

func TestLocationByObjective(t *testing.T) {
	tests := []struct {
		name    string
		kind    ObjectiveKind
		popFile string
		wave    uint8
		want    int64
	}{
		{"first wave", ObjectiveWaveCleared, "mvm_decoy", 1, BaseID + 101},
		{"last wave", ObjectiveWaveCleared, "mvm_decoy", 8, BaseID + 108},
		{"mission clear", ObjectiveMissionCleared, "mvm_decoy", 0, BaseID + 199},
		{"wave past the end", ObjectiveWaveCleared, "mvm_decoy", 9, 0},
		{"wave zero", ObjectiveWaveCleared, "mvm_decoy", 0, 0},
		{"unknown pop file", ObjectiveWaveCleared, "mvm_potato", 1, 0},
		{"unknown kind", 0, "mvm_decoy", 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := LocationByObjective(tt.kind, tt.popFile, tt.wave)
			if ok != (tt.want != 0) {
				t.Fatalf("ok = %v, want %v", ok, tt.want != 0)
			}
			if ok && got.ID != tt.want {
				t.Errorf("id = %d, want %d", got.ID, tt.want)
			}
		})
	}
}

func TestLookupsRoundTrip(t *testing.T) {
	for _, l := range Locations {
		got, ok := LocationByID(l.ID)
		if !ok || got != l {
			t.Fatalf("LocationByID(%d) = %+v, %v", l.ID, got, ok)
		}
		if _, ok := MissionByID(l.Mission); !ok {
			t.Fatalf("%q belongs to unknown mission %d", l.Name, l.Mission)
		}
	}
	for _, it := range Items {
		got, ok := ItemByID(it.ID)
		if !ok || got != it {
			t.Fatalf("ItemByID(%d) = %+v, %v", it.ID, got, ok)
		}
	}
	if _, ok := LocationByID(BaseID); ok {
		t.Error("the base id itself resolves to a location")
	}
	if _, ok := ItemByID(0); ok {
		t.Error("id 0 resolves to an item")
	}
}

func TestCommittedExportIsCurrent(t *testing.T) {
	fresh := t.TempDir()
	if err := Export(fresh); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{FileMeta, FileMissions, FileItems} {
		want, err := os.ReadFile(filepath.Join(fresh, name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(exportDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s is stale, run go generate ./gamedata", name)
		}
	}
}

func TestItemPoolCoversEveryGate(t *testing.T) {
	tickets, classes, slots := 0, 0, 0
	for _, it := range Items {
		switch it.Kind {
		case ItemMissionTicket:
			tickets++
		case ItemClass:
			classes++
		case ItemWeaponSlot:
			slots += int(it.Count)
		case ItemCredits:
			// Filler. Counted by the pool builder, not by this test.
		}
		if it.Classification == Progression && it.Count == 0 {
			t.Errorf("%q is progression with no copies in the pool", it.Name)
		}
	}
	if tickets != len(Missions) {
		t.Errorf("%d mission tickets, want %d", tickets, len(Missions))
	}
	if classes != len(Classes) {
		t.Errorf("%d class items, want %d", classes, len(Classes))
	}
	if slots != len(WeaponSlots) {
		t.Errorf("%d weapon slot copies, want %d", slots, len(WeaponSlots))
	}
}
