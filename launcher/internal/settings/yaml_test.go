package settings

import (
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
