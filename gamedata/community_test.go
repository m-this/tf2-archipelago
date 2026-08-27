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
