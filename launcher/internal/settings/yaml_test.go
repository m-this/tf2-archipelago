package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
