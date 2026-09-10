package settings

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

func TestPlayerYAMLHoldsTheRunShape(t *testing.T) {
	s := Defaults()
	s.APSlotName = "mathis"
	s.MvmMissionCount = 12
	s.MvmGoal = "missionsanity"
	s.MvmMissionsanityPct = 60
	s.MvmDeathLink = true

	got := PlayerYAML(s, "0.6.7")
	for _, want := range []string{
		`name: "mathis"`,
		`game: "Team Fortress 2 Mann vs Machine"`,
		"  version: 0.6.7",
		"  mission_count: 12",
		"  difficulty_pool: intermediate",
		"  goal: missionsanity",
		"  missionsanity_percentage: 60",
		"  death_link: true",
		"  mission_ticket_importance: progression",
		"  class_unlock_importance: progression",
		"  weapon_slot_importance: progression",
		"  weapon_buff_importance: useful",
		"  cash_rewards: false",
		"  weapon_buff_percentage: 75",
		"  weapon_buff_stack_chance: 25",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// The game name holds spaces, so both places it appears have to be quoted or
// the generator reads the mapping key as three tokens.
func TestPlayerYAMLQuotesTheGameKey(t *testing.T) {
	got := PlayerYAML(Defaults(), "")
	if !strings.Contains(got, "\n\"Team Fortress 2 Mann vs Machine\":\n") {
		t.Errorf("the options key is not a quoted scalar:\n%s", got)
	}
	if strings.Contains(got, "requires") {
		t.Error("an empty Archipelago version still wrote a requires block")
	}
}

func TestWritePlayerFile(t *testing.T) {
	s := Defaults()
	s.InstallRoot = filepath.Join(t.TempDir(), "not-created-yet")
	s.APSlotName = "mathis"

	path, err := WritePlayerFile(s, "0.6.7")
	if err != nil {
		t.Fatalf("WritePlayerFile: %v", err)
	}
	if want := filepath.Join(s.InstallRoot, PlayerFileName); path != want {
		t.Errorf("wrote %s, want %s", path, want)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read it back: %v", err)
	}
	if !strings.Contains(string(body), `name: "mathis"`) {
		t.Errorf("the file does not hold the slot name:\n%s", body)
	}

	// The run shape can change between evenings, so the file follows it.
	s.MvmGoal = "missionsanity"
	if _, err := WritePlayerFile(s, "0.6.7"); err != nil {
		t.Fatalf("second write: %v", err)
	}
	body, _ = os.ReadFile(path)
	if !strings.Contains(string(body), "goal: missionsanity") {
		t.Errorf("the file was not rewritten:\n%s", body)
	}
}

func TestWritePlayerFileRejectsAnEmptyMissionPool(t *testing.T) {
	s := Defaults()
	s.InstallRoot = t.TempDir()
	for _, mission := range gamedata.PlayableMissions() {
		s.MvmExcludedMissions = append(s.MvmExcludedMissions, mission.PopFile)
	}
	_, err := WritePlayerFile(s, "0.6.7")
	if err == nil || !strings.Contains(err.Error(), "no missions remain") {
		t.Fatalf("WritePlayerFile error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(s.InstallRoot, PlayerFileName)); !os.IsNotExist(statErr) {
		t.Fatalf("invalid player file was written: %v", statErr)
	}
}

func TestPlayerYAMLNamesTheStartMissionAndClass(t *testing.T) {
	s := Defaults()
	s.MvmStartMission = "mvm_ghost_town_666"
	s.MvmStartClass = "medic"

	got := PlayerYAML(s, "")
	if !strings.Contains(got, `  start_mission: "Caliginous Caper"`) {
		t.Errorf("the start mission is not named:\n%s", got)
	}
	// The mod spells classes in lower case; the apworld option takes the name.
	if !strings.Contains(got, `  start_class: "Medic"`) {
		t.Errorf("the start class is not named:\n%s", got)
	}
}

// A settings file from before these options, or one naming a mission this
// build's tables do not have, must still generate. Random is what the seed did
// anyway, so it is the safe reading of both.
func TestPlayerYAMLFallsBackToRandom(t *testing.T) {
	for _, c := range []struct{ mission, class string }{
		{"", ""},
		{"mvm_nowhere", "Wizard"},
	} {
		s := Defaults()
		s.MvmStartMission, s.MvmStartClass = c.mission, c.class
		got := PlayerYAML(s, "")
		if !strings.Contains(got, `  start_mission: "random"`) {
			t.Errorf("start_mission %q did not fall back:\n%s", c.mission, got)
		}
		if !strings.Contains(got, `  start_class: "random"`) {
			t.Errorf("start_class %q did not fall back:\n%s", c.class, got)
		}
	}
}

func TestPlayerYAMLNamesTheExcludedMissions(t *testing.T) {
	s := Defaults()
	s.MvmExcludedMissions = []string{"mvm_ghost_town_666", "mvm_nowhere"}

	got := PlayerYAML(s, "")
	if !strings.Contains(got, "  excluded_missions:\n    - \"Caliginous Caper\"\n") {
		t.Errorf("the excluded mission is not named:\n%s", got)
	}
	if strings.Contains(got, "mvm_nowhere") {
		t.Errorf("a popfile the tables do not know reached the file:\n%s", got)
	}
	empty := Defaults()
	empty.MvmExcludedMissions = []string{}
	if !strings.Contains(PlayerYAML(empty, ""), "  excluded_missions: []\n") {
		t.Error("an empty exclusion list is not written as an empty list")
	}
}

// The player file names the mods the server loads, in catalog order, and
// nothing the catalog does not know: the apworld refuses a key it has never
// heard of, and a refused file is a seed nobody gets.
func TestPlayerYAMLSaysWhetherCommunityMissionsPlay(t *testing.T) {
	s := Defaults()
	if !strings.Contains(PlayerYAML(s, ""), "  community_missions: true\n") {
		t.Error("the default did not write community_missions on")
	}
	s.MvmCommunityMissions = false
	if !strings.Contains(PlayerYAML(s, ""), "  community_missions: false\n") {
		t.Error("turning it off did not reach the file")
	}
}

func TestPlayerYAMLNamesTheServerMods(t *testing.T) {
	s := Defaults()
	s.SrcdsMods = []string{"rafmod", "sigsegv-mvm"}
	got := PlayerYAML(s, "")
	if !strings.Contains(got, "  server_mods:\n    - \"sigsegv-mvm\"\n") {
		t.Errorf("server_mods missing or wrong:\n%s", got)
	}
	if strings.Contains(got, "rafmod") {
		t.Errorf("an unknown mod reached the player file:\n%s", got)
	}
	s.SrcdsMods = nil
	if got := PlayerYAML(s, ""); !strings.Contains(got, "  server_mods: []\n") {
		t.Errorf("no mods should write an empty list:\n%s", got)
	}
}

/*
TestPlayerYAMLWritesEveryOptionTheApworldDeclares walks the apworld rather than
a list written here.

TF2MvMOptions is the contract: a field on it is an option the seed is generated
from, and a key the launcher does not write is an option nobody who uses the
launcher can reach. server_settings was exactly that for a release. The
Grappling Hook was in the pool code, in the item tables and in the plugin, and
the only way to turn it on was to hand-edit a file the launcher then rewrote.

Reading the Python is the same trade gamedata/manifest_test.go makes with the
plugin source: the two languages cannot share a declaration, so the test reads
one and checks the other against it.
*/
func TestPlayerYAMLWritesEveryOptionTheApworldDeclares(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "apworld", "tf2_mvm", "options.py"))
	if err != nil {
		t.Fatal(err)
	}
	body := regexp.MustCompile(`(?s)class TF2MvMOptions\(PerGameCommonOptions\):\n(.*?)\n\n`).FindSubmatch(source)
	if body == nil {
		t.Fatal("options.py declares no TF2MvMOptions dataclass")
	}
	fields := regexp.MustCompile(`(?m)^\s+(\w+):`).FindAllSubmatch(body[1], -1)
	if len(fields) < 15 {
		t.Fatalf("found %d options, which is too few to be the whole dataclass", len(fields))
	}

	yaml := PlayerYAML(Defaults(), "")
	for _, field := range fields {
		option := string(field[1])
		if !strings.Contains(yaml, "\n  "+option+":") {
			t.Errorf("the apworld declares %s and the player file never writes it", option)
		}
	}
}
