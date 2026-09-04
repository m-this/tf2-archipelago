package gamedata

import (
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
		"sigsegv-mvm": "Needs SigMod (Linux server only)",
	} {
		if got := RequirementLabel(requirement); got != want {
			t.Errorf("RequirementLabel(%q) = %q, want %q", requirement, got, want)
		}
	}
}

func TestPlayableWithModsFollowsTheCatalog(t *testing.T) {
	saved := communityContent
	defer func() { communityContent = saved }()
	content, err := loadCommunity([]byte(modManifest))
	if err != nil {
		t.Fatal(err)
	}
	communityContent = content

	for _, c := range []struct {
		id   MissionID
		mods []string
		want bool
	}{
		{200, nil, false},
		{200, []string{"sigsegv-mvm"}, true},
		{201, []string{"sigsegv-mvm"}, false},
		{202, nil, true},
		{1, nil, true},
	} {
		if got := IsMissionPlayableWith(c.id, c.mods); got != c.want {
			t.Errorf("IsMissionPlayableWith(%d, %v) = %t, want %t", c.id, c.mods, got, c.want)
		}
	}
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
