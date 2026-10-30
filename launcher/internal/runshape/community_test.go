package runshape

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/gamedata/availability"
)

func startChoice(choices []MissionChoice, popFile string) (MissionChoice, bool) {
	for _, choice := range choices {
		if choice.PopFile == popFile {
			return choice, true
		}
	}
	return MissionChoice{}, false
}

func TestMedievalLoadoutAppearsInStartMissionChoices(t *testing.T) {
	choices := StartMissionChoices(availability.Planned([]string{"archive-assets.zip"}, nil))
	choice, ok := startChoice(choices, "mvm_frostwynd_rc1_int_wicked_wizardry")
	if !ok {
		t.Fatal("Wicked Wizardry is missing from the start mission choices")
	}
	if !strings.Contains(choice.Label, "Medieval") {
		t.Fatalf("Wicked Wizardry label = %q", choice.Label)
	}
}

func TestMedicOnlyRestrictionAppearsInStartMissionChoices(t *testing.T) {
	choices := StartMissionChoices(availability.Planned([]string{"archive-assets.zip"}, []string{"sigsegv-mvm"}))
	for _, popFile := range []string{
		"mvm_chateau_rc3_adv_remedic",
		"mvm_villa_b13f_adv_forgotten",
		"mvm_villa_b13f_adv_recalled_to_life",
	} {
		choice, ok := startChoice(choices, popFile)
		if !ok {
			t.Errorf("%s is missing from the start mission choices", popFile)
			continue
		}
		if !strings.Contains(choice.Label, "Medic only") {
			t.Errorf("%s label = %q", popFile, choice.Label)
		}
	}
}

// Picking a start mission ticks its pack, so a pack that is on disk and not
// ticked keeps its missions on the menu. Nothing else that stands in a
// mission's way does.
func TestStartMissionChoicesFollowTheVerdict(t *testing.T) {
	server := availability.Server{
		Platform:     "linux",
		PacksPresent: []string{"archive-assets.zip"},
		ModsSelected: []string{"sigsegv-mvm"},
	}
	choices := StartMissionChoices(server)
	tests := []struct {
		popFile string
		offered bool
	}{
		{"mvm_decoy", true},
		{"mvm_kelly_rc1b_adv_homestead_happenings", true},
		{"mvm_bogland_rc12_adv_swamp_fever", true},
		{"mvm_bronx_rc2_adv_point_of_impact", false},
		{"mvm_area_52_rc3_int_anomalous_materials", false},
	}
	for _, test := range tests {
		if _, offered := startChoice(choices, test.popFile); offered != test.offered {
			t.Errorf("%s offered = %v, want %v (%v)", test.popFile, offered, test.offered,
				server.Check(mustMission(t, test.popFile)).Reason)
		}
	}
	if choices[0].PopFile != "" || choices[0].Label != AnyLabel {
		t.Errorf("the draw is not first: %+v", choices[0])
	}
}

func mustMission(t *testing.T, popFile string) gamedata.Mission {
	t.Helper()
	mission, ok := gamedata.MissionByPopFile(popFile)
	if !ok {
		t.Fatalf("%s is not in the catalog", popFile)
	}
	return mission
}
