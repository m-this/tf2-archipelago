package gamedata

import (
	"maps"
	"strings"
	"testing"
)

const modManifest = `{
  "format_version": 1,
  "maps": [{"id": 200, "name": "mvm_modded_rc1"}],
  "missions": [
    {"id": 200, "pop_file": "mvm_modded_rc1_adv_one", "name": "One", "map_id": 200, "difficulty": "advanced", "waves": 3, "has_tank": false, "has_giant": true, "requires": "sigsegv-mvm"},
    {"id": 201, "pop_file": "mvm_modded_rc1_adv_two", "name": "Two", "map_id": 200, "difficulty": "advanced", "waves": 3, "has_tank": false, "has_giant": true, "requires": "no_nav"},
    {"id": 202, "pop_file": "mvm_modded_rc1_adv_three", "name": "Three", "map_id": 200, "difficulty": "advanced", "waves": 3, "has_tank": false, "has_giant": true}
  ]
}`

func TestManifestAcceptsACatalogedModAndRejectsAnUnknownOne(t *testing.T) {
	content, err := loadCommunity([]byte(modManifest))
	if err != nil {
		t.Fatalf("a cataloged mod is a valid requirement: %v", err)
	}
	if got := content.Requirements[200]; got != "sigsegv-mvm" {
		t.Errorf("requirement = %q, want sigsegv-mvm", got)
	}
	bad := strings.Replace(modManifest, `"sigsegv-mvm"`, `"rafmod"`, 1)
	if _, err := loadCommunity([]byte(bad)); err == nil || !strings.Contains(err.Error(), "rafmod") {
		t.Errorf("an unknown requirement loaded: %v", err)
	}
}

func TestRequirementLabels(t *testing.T) {
	for requirement, want := range map[string]string{
		"":            "Ready",
		"no_nav":      "Missing bot .nav",
		"sigsegv-mvm": "Needs SigMod",
	} {
		if got := RequirementLabel(requirement); got != want {
			t.Errorf("RequirementLabel(%q) = %q, want %q", requirement, got, want)
		}
	}
}

func TestMissingNavigationMeshNamesTheMapAsset(t *testing.T) {
	saved := communityContent
	defer func() { communityContent = saved }()
	mission, ok := MissionByPopFile("mvm_bogland_rc12_adv_swamp_fever")
	if !ok {
		t.Fatal("missing fixture mission")
	}
	communityContent.Requirements = maps.Clone(communityContent.Requirements)
	communityContent.Requirements[mission.ID] = "no_nav"
	if got, want := MissingNavigationMesh(mission.ID), "maps/mvm_bogland_rc12.nav"; got != want {
		t.Errorf("MissingNavigationMesh = %q, want %q", got, want)
	}
	if got := MissingNavigationMesh(1); got != "" {
		t.Errorf("Valve mission claims to be missing %q", got)
	}
}

func TestMissionServerModFollowsTheCatalog(t *testing.T) {
	saved := communityContent
	defer func() { communityContent = saved }()
	content, err := loadCommunity([]byte(modManifest))
	if err != nil {
		t.Fatal(err)
	}
	communityContent = content

	if got := MissionServerMod(200); got != "sigsegv-mvm" {
		t.Errorf("MissionServerMod = %q", got)
	}
	if got := MissionServerMod(201); got != "" {
		t.Errorf("no_nav is not a mod, got %q", got)
	}
}

func TestEveryServerModHasABuildSomewhere(t *testing.T) {
	for _, mod := range ServerMods {
		if !mod.Linux && !mod.Windows {
			t.Errorf("%s has no build on any platform", mod.Key)
		}
		if _, ok := ServerModByKey(mod.Key); !ok {
			t.Errorf("%s is not found by its own key", mod.Key)
		}
	}
}
