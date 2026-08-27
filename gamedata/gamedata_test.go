package gamedata

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// freeze only adds ids; it never rewrites one already in the file.
var freeze = flag.Bool("freeze", false, "record new ids in "+frozenIDsFile)

const exportDir = "../apworld/tf2_mvm/data"

// frozenIDsFile pins every id ever exported; a changed id next to an existing key must be loud.
//
// Keyed on what the id derives from, never on the display name: the names are
// UNVERIFIED and expected to change, and a key that moves with a name reports
// nine tombstones for a rename that touches no id.
const frozenIDsFile = "testdata/ids-frozen.json"

// frozenKey identifies an entity the way its id does: by the tables behind it.
func frozenKey(kind, owner string, index int) string {
	return fmt.Sprintf("%s/%s/%d", kind, owner, index)
}

// currentIDs is every id this build produces, under its frozen key.
func currentIDs() map[string]int64 {
	ids := make(map[string]int64, len(Locations)+len(Items))
	for _, l := range Locations {
		mission, ok := MissionByID(l.Mission)
		if !ok {
			continue
		}
		ids[frozenKey(l.Kind.Key(), mission.PopFile, int(l.Wave))] = l.ID
	}
	for _, it := range Items {
		switch it.Kind {
		case ItemMissionTicket:
			if mission, ok := MissionByID(it.Mission); ok {
				ids[frozenKey(it.Kind.Key(), mission.PopFile, 0)] = it.ID
			}
		case ItemClass:
			if class, ok := ClassByID(it.Class); ok {
				ids[frozenKey(it.Kind.Key(), class.Key, 0)] = it.ID
			}
		case ItemWeaponSlot, ItemCredits:
			ids[frozenKey(it.Kind.Key(), "", 0)] = it.ID
		case ItemWeaponBuff:
			buff, ok := WeaponBuffByID(it.WeaponBuff)
			if ok {
				ids[frozenKey(it.Kind.Key(), buff.Key, 0)] = it.ID
			}
		}
	}
	return ids
}

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
	current := currentIDs()
	if *freeze {
		if err := recordNewIDs(frozen, current); err != nil {
			t.Fatal(err)
		}
	}
	for key, id := range frozen {
		switch got, ok := current[key]; {
		case !ok:
			t.Errorf("%s is gone; a deleted entity keeps its id as a tombstone", key)
		case got != id:
			t.Errorf("%s moved from %d to %d; every seed holding %d now means something else", key, id, got, id)
		}
	}
}

// TestFrozenKeysHoldOnlyStableIdentifiers is the reason the frozen file is
// keyed the way it is. missions.go says the display names are a guess to be
// corrected before the first seed is played. A key built from a name would
// report that correction as nine deleted entities.
func TestFrozenKeysHoldOnlyStableIdentifiers(t *testing.T) {
	stable := map[string]bool{"": true}
	for _, m := range Missions {
		stable[m.PopFile] = true
	}
	for _, c := range Classes {
		stable[c.Key] = true
	}
	for _, buff := range WeaponBuffs {
		stable[buff.Key] = true
	}

	for key := range currentIDs() {
		parts := strings.Split(key, "/")
		if len(parts) != 3 {
			t.Errorf("%q is not kind/owner/index", key)
			continue
		}
		if !stable[parts[1]] {
			t.Errorf("%q is keyed on unstable owner %q", key, parts[1])
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
	waves, clears, tanks, giants := 0, 0, 0, 0
	for _, l := range Locations {
		switch l.Kind {
		case ObjectiveWaveCleared:
			waves++
		case ObjectiveMissionCleared:
			clears++
		case ObjectiveTankDestroyed:
			tanks++
		case ObjectiveGiantKilled:
			giants++
		}
	}
	want, wantTanks, wantGiants := 0, 0, 0
	for _, m := range Missions {
		want += int(m.Waves)
		if m.HasTank {
			wantTanks++
		}
		if m.HasGiant {
			wantGiants++
		}
	}
	if waves != want {
		t.Errorf("%d wave locations, want %d", waves, want)
	}
	if clears != len(Missions) {
		t.Errorf("%d mission clear locations, want %d", clears, len(Missions))
	}
	// A tank check on a mission with no tank is a location nobody can reach,
	// and a run nobody can finish.
	if tanks != wantTanks {
		t.Errorf("%d tank locations, want %d", tanks, wantTanks)
	}
	if giants != wantGiants {
		t.Errorf("%d giant locations, want %d", giants, wantGiants)
	}
}

// The wiki's mission list gives every one of the 29 a giant. The field exists
// so that a mission without one is a decision somebody made, not an oversight
// that hands a seed a location nobody can reach.
func TestEveryMissionHasAGiant(t *testing.T) {
	for _, m := range Missions {
		if !m.HasGiant {
			t.Errorf("%s has no giant, which no source says", m.Name)
		}
	}
}

// The three are Mannhattan's, whose missions run on gates rather than tanks.
// Named here so a careless edit to the table has to argue with a test.
func TestOnlyMannhattanHasNoTank(t *testing.T) {
	var without []string
	for _, m := range Missions {
		if !m.HasTank {
			without = append(without, m.Name)
		}
	}
	want := []string{"Big Apple Barricade", "Empire Escalation", "Metro Malice"}
	if !slices.Equal(without, want) {
		t.Errorf("missions with no tank: %v, want %v", without, want)
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
			// Filler, counted by the pool builder rather than here.
		case ItemWeaponBuff:
			// Useful rewards, sampled and sometimes stacked by the pool builder.
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
