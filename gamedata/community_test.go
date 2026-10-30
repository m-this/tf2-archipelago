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
			"requires": "no_nav",
			"loadout": "medieval"
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
	if got := content.Loadouts[101]; got != "medieval" {
		t.Fatalf("loadout = %q", got)
	}
}

func TestCommunityManifestRejectsAnUnknownVersion(t *testing.T) {
	if _, err := loadCommunity([]byte(`{"format_version": 99}`)); err == nil {
		t.Fatal("a newer manifest version loaded")
	}
}

func TestCommunityPopulationMedicOnly(t *testing.T) {
	const eightExcluded = `ClassLimit // players
{
 Scout 0
 Soldier 0
 Pyro 0
 Demoman 0
 Heavyweapons 0
 Engineer 0
 Sniper 0
 Spy 0
}`
	for _, test := range []struct {
		name, body string
		want       bool
	}{
		{"all other classes excluded", eightExcluded, true},
		{"Medic explicitly allowed", strings.Replace(eightExcluded, "Spy 0", "Spy 0\n Medic 1", 1), true},
		{"Medic also excluded", strings.Replace(eightExcluded, "Spy 0", "Spy 0\n Medic 0", 1), false},
		{"Soldier allowed", strings.Replace(eightExcluded, "Soldier 0", "Soldier 1", 1), false},
		{"incomplete limits", strings.Replace(eightExcluded, "Sniper 0", "", 1), false},
		{"Villa Medic relay", `InitWaveOutput { Target medonly_hill_init_relay Action Trigger }`, true},
		{"Villa relay only in comment", `// Target medonly_hill_init_relay`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := CommunityPopulationMedicOnly([]byte(test.body)); got != test.want {
				t.Fatalf("Medic-only restriction = %t, want %t", got, test.want)
			}
		})
	}
}

func TestVillaMissionsAreMedicOnly(t *testing.T) {
	for _, popFile := range []string{
		"mvm_villa_b13f_adv_forgotten",
		"mvm_villa_b13f_adv_recalled_to_life",
	} {
		if !CommunityMissionMedicOnlyForCatalog(popFile, nil) {
			t.Errorf("%s should be marked Medic only", popFile)
		}
	}
	if CommunityMissionMedicOnlyForCatalog("mvm_villa_b13f_adv_other", nil) {
		t.Error("unlisted Villa mission should not inherit the label")
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
		"unknown loadout":     `{"format_version":1,"missions":[{"id":100,"pop_file":"mvm_example_test","name":"Test","map_id":1,"difficulty":"normal","waves":1,"has_tank":false,"has_giant":true,"loadout":"magic"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := loadCommunity([]byte(body)); err == nil {
				t.Fatal("invalid manifest loaded")
			}
		})
	}
}

func TestCommunityMissionsNameTheirSpecialRestrictions(t *testing.T) {
	want := map[string]string{
		"mvm_frostwynd_rc1_int_wicked_wizardry":  "medieval",
		"mvm_frostwynd_rc1_adv_fiefdom_fiasco":   "medieval",
		"mvm_frostwynd_rc1_adv_medieval_madness": "medieval",
		"mvm_chateau_rc3_adv_remedic":            "medic_only",
		"mvm_villa_b13f_adv_forgotten":           "medic_only",
		"mvm_villa_b13f_adv_recalled_to_life":    "medic_only",
	}
	for _, mission := range communityMissions {
		got := MissionLoadout(mission.ID)
		if restriction, expected := want[mission.PopFile]; expected {
			if got != restriction {
				t.Errorf("%s loadout = %q, want %q", mission.PopFile, got, restriction)
			}
			delete(want, mission.PopFile)
		} else if got != "" {
			t.Errorf("%s unexpectedly has loadout %q", mission.PopFile, got)
		}
	}
	if len(want) != 0 {
		t.Fatalf("catalog is missing restricted missions: %v", want)
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

func TestCommunityCatalogueCounts(t *testing.T) {
	if got, want := len(communityMaps), 46; got != want {
		t.Errorf("community maps = %d, want %d", got, want)
	}
	counts := map[string]int{}
	for _, mission := range communityMissions {
		requirement := MissionRequirement(mission.ID)
		if requirement == "" {
			requirement = "ready"
		}
		counts[requirement]++
	}
	if got, want := len(communityMissions), 201; got != want {
		t.Errorf("community missions = %d, want %d", got, want)
	}
	for requirement, want := range map[string]int{"ready": 87, "sigsegv-mvm": 114, "no_nav": 0} {
		if got := counts[requirement]; got != want {
			t.Errorf("%s missions = %d, want %d", requirement, got, want)
		}
	}
}

func TestPortableCommunityMissionCountsByMap(t *testing.T) {
	want := map[string]int{
		"mvm_area_52_rc3":        9,
		"mvm_autumnull_rc2":      2,
		"mvm_bogland_rc12":       2,
		"mvm_cyberia_rc6a":       6,
		"mvm_condemned_b3":       2,
		"mvm_downpour_rc3a":      4,
		"mvm_frostwynd_rc1":      2,
		"mvm_heatrock_rc6a":      2,
		"mvm_hideout_b3":         6,
		"mvm_kelly_rc1b":         1,
		"mvm_lotus_b6":           2,
		"mvm_memorial_b1":        1,
		"mvm_nightsky_rc4d":      1,
		"mvm_null_b9c":           1,
		"mvm_oilrig_rc5d":        5,
		"mvm_oxidize_rc3":        3,
		"mvm_oxidize_rr18":       4,
		"mvm_radar_b10":          3,
		"mvm_redstone_ridge_rc5": 2,
		"mvm_robotfactory_b30":   1,
		"mvm_skeleclipse_b7a":    1,
		"mvm_snowpine_rc4_fix1":  4,
		"mvm_teien_rc6":          3,
		"mvm_transmission_rc7a":  2,
		"mvm_yiresa_rc5a":        1,

		// The Valve maps. A community mission that plays on one needs its pack
		// for the population file and nothing else: the .bsp and the .nav ship
		// with the game.
		"mvm_decoy":      3,
		"mvm_coaltown":   2,
		"mvm_mannworks":  4,
		"mvm_mannhattan": 2,
		"mvm_rottenburg": 5,
		"mvm_ghost_town": 1,
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
	for _, name := range []string{"mvm_underground_rc3", "mvm_mannworks_int_xlr-8"} {
		if !safeConsoleName(name) {
			t.Errorf("safeConsoleName(%q) = false, and it is a real popfile", name)
		}
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

func TestCommunityPopulationRequiresSigMod(t *testing.T) {
	for name, test := range map[string]struct {
		body string
		want bool
	}{
		"stock":               {`WaveSchedule { Wave { } }`, false},
		"commented extension": {`// ItemAttributes { } [$SIGSEGV]`, false},
		"delivery hint":       {`PrecacheModel "example.mdl" [$SIGSEGV]`, false},
		"active annotation":   {`ItemAttributes { } [$SIGSEGV]`, true},
		"unguarded template":  {`SpawnTemplate Example`, true},
		"LuaScriptFile":       {`LuaScriptFile "scripts/example.lua"`, true},
		"unknown populator":   {`ItemBlacklist { }`, true},
		"squad extension":     {`NoWaitForFormation 1`, true},
		"random choice":       {`Shuffle 1`, true},
	} {
		t.Run(name, func(t *testing.T) {
			if got := CommunityPopulationRequiresSigMod([]byte(test.body)); got != test.want {
				t.Errorf("CommunityPopulationRequiresSigMod() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestCommunityPopulationRequiresSigModWithIncludes(t *testing.T) {
	files := map[string][][]byte{
		"extensions.pop":  {[]byte("WaveSchedule\n{\nLuaScriptFile example.lua\n}")},
		"nested.pop":      {[]byte("#base extensions.pop\nWaveSchedule { }")},
		"optional.pop":    {[]byte("WaveSchedule [$SIGSEGV]\n{\nLuaScriptFile example.lua\n}")},
		"later_guard.pop": {[]byte("#base robot_standard.pop\nWaveSchedule [$SIGSEGV]\n{\nLuaScriptFile example.lua\n}")},
		"cycle.pop":       {[]byte("#base cycle.pop\nWaveSchedule { }")},
	}
	for name, test := range map[string]struct {
		body string
		want bool
	}{
		"direct":             {"LuaScriptFile example.lua", true},
		"included":           {"#base extensions.pop\nWaveSchedule { }", true},
		"nested":             {"#base nested.pop\nWaveSchedule { }", true},
		"optional extension": {"#base optional.pop\nWaveSchedule { }", false},
		"guard after base":   {"#base later_guard.pop\nWaveSchedule { }", false},
		"missing include":    {"#base absent.pop\nWaveSchedule { }", false},
		"cycle":              {"#base cycle.pop\nWaveSchedule { }", false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := CommunityPopulationRequiresSigModWithIncludes([]byte(test.body), files); got != test.want {
				t.Errorf("CommunityPopulationRequiresSigModWithIncludes() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestSourceRequiresSigModUsesTheMissionArchive(t *testing.T) {
	bodies := []sourcedPopulation{
		{body: []byte("#base shared.pop\nWaveSchedule { }"), source: "potato", pack: "archive-assets.zip"},
		{body: []byte("#base shared.pop\nWaveSchedule { }"), source: "moonlight", pack: "mlarchive-assets.zip"},
	}
	includes := map[string]map[string][][]byte{
		"potato":    {"shared.pop": {[]byte("LuaScriptFile example.lua")}},
		"moonlight": {"shared.pop": {[]byte("WaveSchedule { }")}},
	}
	if sourceRequiresSigMod(bodies, "mlarchive-assets.zip", includes) {
		t.Fatal("Moonlight mission inherited a Potato-only extension")
	}
	if !sourceRequiresSigMod(bodies, "archive-assets.zip", includes) {
		t.Fatal("Potato mission lost its own extension")
	}
}

// The launcher's Missions page and the bridge's refusal name a pack by its
// label, and a pack added to the catalog without one would be named by its
// file, which nobody ticks.
func TestEveryCommunityPackHasALabel(t *testing.T) {
	for _, mission := range Missions {
		pack := MissionPack(mission.ID)
		if pack != "" && PackLabel(pack) == "" {
			t.Errorf("%s comes from %s, which has no label", mission.PopFile, pack)
		}
	}
}
