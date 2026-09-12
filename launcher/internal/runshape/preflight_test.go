package runshape

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// everything is the widest pool the checks below start from: community
// missions on and every cataloged mod loaded, so what a test excludes is the
// only thing keeping a mission out.
func everything(excluded ...string) Pool {
	return Pool{Mods: gamedata.ServerModKeys(), Community: true, Excluded: excluded}
}

func excludingAllBut(popFiles ...string) []string {
	keep := make(map[string]bool, len(popFiles))
	for _, popFile := range popFiles {
		keep[popFile] = true
	}
	var excluded []string
	for _, mission := range gamedata.MissionsPlayableWith(gamedata.ServerModKeys()) {
		if !keep[mission.PopFile] {
			excluded = append(excluded, mission.PopFile)
		}
	}
	return excluded
}

func TestCheckSelectionRejectsAnEmptyPool(t *testing.T) {
	_, err := CheckSelection(Selection{
		Pool:         everything(excludingAllBut()...),
		Difficulty:   "normal",
		MissionCount: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "no missions remain") {
		t.Fatalf("empty pool error = %v", err)
	}
}

func TestCheckSelectionRejectsAPoolWithTooFewLocations(t *testing.T) {
	// Thriller Terror has five waves, no tank, one giant and one clear: seven
	// checks. An intermediate start still owes seven classes and two weapon
	// slots, so a run containing only it cannot place all nine unlocks.
	popFile := "mvm_condemned_b3_int_thriller_terror"
	report, err := CheckSelection(Selection{
		Pool:         everything(excludingAllBut(popFile)...),
		Difficulty:   "intermediate",
		MissionCount: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "needs room for 9 unlocks") {
		t.Fatalf("short pool error = %v", err)
	}
	if report.AvailableChecks != 7 || report.RequiredUnlocks != 9 {
		t.Fatalf("report = %+v", report)
	}
}

func TestCheckSelectionAllowsTheGeneratorToWidenTheRun(t *testing.T) {
	report, err := CheckSelection(Selection{
		Pool: everything(excludingAllBut(
			"mvm_condemned_b3_int_thriller_terror",
			"mvm_kelly_rc1b_adv_homestead_happenings",
		)...),
		Difficulty:   "intermediate",
		MissionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Requested != 1 || report.Eligible != 2 {
		t.Fatalf("report = %+v, want one requested with two eligible", report)
	}
}

func TestCheckSelectionRejectsAnUnavailableNamedStart(t *testing.T) {
	_, err := CheckSelection(Selection{
		Pool:         everything(),
		Difficulty:   "advanced",
		MissionCount: 2,
		StartMission: "mvm_decoy",
	})
	if err == nil || !strings.Contains(err.Error(), "outside the advanced-or-harder pool") {
		t.Fatalf("start mission error = %v", err)
	}
}

func TestHauntedOnlyMatchesTheCurrentApworldCapacityMath(t *testing.T) {
	report, err := CheckSelection(Selection{
		Pool:         everything(excludingAllBut("mvm_ghost_town_666")...),
		Difficulty:   "haunted",
		MissionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.AvailableChecks != 4 || report.RequiredUnlocks != 4 {
		t.Fatalf("haunted report = %+v, want four checks for four unlocks", report)
	}
}
