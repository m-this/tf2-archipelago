package gamedata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommunityManifestLoadsMapsAndMissions(t *testing.T) {
	content, err := loadCommunity([]byte(`{
		"format_version": 1,
		"maps": [{"id": 101, "name": "mvm_example_rc1"}],
		"missions": [{
			"id": 101,
			"pop_file": "mvm_example_rc1_advanced",
			"name": "Example Exercise",
			"map_id": 101,
			"difficulty": "advanced",
			"waves": 6,
			"has_tank": true,
			"has_giant": true,
			"requires": "no_nav"
		}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(content.Maps) != 1 || content.Maps[0].Name != "mvm_example_rc1" {
		t.Fatalf("maps = %+v", content.Maps)
	}
	if len(content.Missions) != 1 || content.Missions[0].Difficulty != DifficultyAdvanced {
		t.Fatalf("missions = %+v", content.Missions)
	}
	if got := content.Requirements[101]; got != "no_nav" {
		t.Fatalf("requirement = %q", got)
	}
	if got := content.Packs[101]; got != "archive-assets.zip" {
		t.Fatalf("pack = %q", got)
	}
}

func TestCommunityManifestRejectsAnUnknownVersion(t *testing.T) {
	if _, err := loadCommunity([]byte(`{"format_version": 99}`)); err == nil {
		t.Fatal("a newer manifest version loaded")
	}
}

func TestCommunityManifestRejectsTyposAndReservedIDs(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":       `{"format_version":1,"mapps":[]}`,
		"reserved map id":     `{"format_version":1,"maps":[{"id":8,"name":"mvm_example"}]}`,
		"reserved mission id": `{"format_version":1,"missions":[{"id":30,"pop_file":"mvm_example_test","name":"Test","map_id":1,"difficulty":"normal","waves":1,"has_tank":false,"has_giant":true}]}`,
		"unknown difficulty":  `{"format_version":1,"missions":[{"id":100,"pop_file":"mvm_example_test","name":"Test","map_id":1,"difficulty":"impossible","waves":1,"has_tank":false,"has_giant":true}]}`,
		"unknown requirement": `{"format_version":1,"missions":[{"id":100,"pop_file":"mvm_example_test","name":"Test","map_id":1,"difficulty":"normal","waves":1,"has_tank":false,"has_giant":true,"requires":"magic"}]}`,
		"unknown pack":        `{"format_version":1,"missions":[{"id":100,"pop_file":"mvm_example_test","name":"Test","map_id":1,"difficulty":"normal","waves":1,"has_tank":false,"has_giant":true,"pack":"mystery.zip"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := loadCommunity([]byte(body)); err == nil {
				t.Fatal("invalid manifest loaded")
			}
		})
	}
}

func TestUnsupportedCommunityMissionsAreNotPlayableChoices(t *testing.T) {
	unsupported := 0
	for _, mission := range communityMissions {
		if MissionRequirement(mission.ID) != "" {
			unsupported++
			if IsPlayableMission(mission.ID) {
				t.Errorf("unsupported mission %s is playable", mission.PopFile)
			}
		}
	}
	if unsupported == 0 {
		t.Fatal("the catalog does not exercise unavailable community content")
	}
	for _, mission := range PlayableMissions() {
		if !IsPlayableMission(mission.ID) {
			t.Fatalf("PlayableMissions contains %s", mission.PopFile)
		}
	}
}

func TestPortableCommunityMissionCountsByMap(t *testing.T) {
	want := map[string]int{
		"mvm_area_52_rc3":        8,
		"mvm_autumnull_rc2":      2,
		"mvm_condemned_b3":       2,
		"mvm_downpour_rc3a":      3,
		"mvm_frostwynd_rc1":      2,
		"mvm_heatrock_rc6a":      1,
		"mvm_hideout_b3":         6,
		"mvm_kelly_rc1b":         1,
		"mvm_lotus_b6":           2,
		"mvm_null_b9c":           1,
		"mvm_oilrig_rc5d":        5,
		"mvm_oxidize_rc3":        3,
		"mvm_oxidize_rr18":       3,
		"mvm_radar_b10":          3,
		"mvm_redstone_ridge_rc5": 1,
		"mvm_snowpine_rc4_fix1":  4,
		"mvm_teien_rc6":          3,
		"mvm_transmission_rc7a":  2,
		"mvm_yiresa_rc5a":        1,
	}

	got := make(map[string]int, len(want))
	for _, mission := range PlayableMissions() {
		if !IsCommunityMission(mission.ID) {
			continue
		}
		played, ok := MapByID(mission.Map)
		if !ok {
			t.Fatalf("mission %s refers to unknown map %d", mission.PopFile, mission.Map)
		}
		got[played.Name]++
	}
	if len(got) != len(want) {
		t.Fatalf("portable community maps = %v, want %v", got, want)
	}
	for name, count := range want {
		if got[name] != count {
			t.Errorf("%s has %d portable missions, want %d", name, got[name], count)
		}
	}
}

func TestValidateCommunityFilesNamesTheMissingFile(t *testing.T) {
	if len(communityMaps) != 0 || len(communityMissions) != 0 {
		t.Skip("the committed manifest contains a real content pack")
	}
	// Exercise the file checker directly; the committed empty manifest should
	// also be valid against an empty content tree.
	root := t.TempDir()
	if err := ValidateCommunityFiles(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "maps", "mvm_example.bsp")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := requireCommunityFile(root, "maps/missing.bsp", "map", "mvm_missing"); err == nil || !strings.Contains(err.Error(), "maps/missing.bsp") {
		t.Fatalf("missing file error = %v", err)
	}
}

func TestValidateRejectsUnsafeConsoleNames(t *testing.T) {
	for _, name := range []string{"mvm_example;quit", "mvm example", "MVM_Example", "../mvm_example"} {
		if safeConsoleName(name) {
			t.Errorf("safeConsoleName(%q) = true", name)
		}
	}
	if !safeConsoleName("mvm_underground_rc3") {
		t.Error("a normal community map name was rejected")
	}
}

func TestInspectPopulationFindsOnlyReachableObjectiveKinds(t *testing.T) {
	body := []byte(`
// Wave { Tank { Template T_TFBot_Giant_Commented }
WaveSchedule
{
    Wave
    {
        WaveSpawn
        {
            TFBot { Template T_TFBot_Giant_Soldier }
        }
    }
    Wave { WaveSpawn { Tank { Health 10000 } } }
}`)
	want := populationFacts{Waves: 2, HasTank: true, HasGiant: true}
	if got := inspectPopulation(body); got != want {
		t.Fatalf("inspectPopulation() = %v, want %v", got, want)
	}

	withoutObjectives := []byte(`WaveSchedule {
        Wave { WaveSpawn { Where SpawnBot_Giant TFBot { Template T_TFBot_SentryBuster } } }
    }`)
	want = populationFacts{Waves: 1}
	if got := inspectPopulation(withoutObjectives); got != want {
		t.Fatalf("inspectPopulation() = %v, want %v", got, want)
	}
}
