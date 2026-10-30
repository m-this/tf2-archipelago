package apclient

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// The old message named two numbers and no action, and two players a week
// apart asked the same question about it in Discord.
func TestAFormatMismatchSaysWhatToDoAboutIt(t *testing.T) {
	err := SlotData{FormatVersion: oldestFormatVersion - 1}.validate()
	if err == nil {
		t.Fatal("a seed from another apworld was accepted")
	}
	for _, want := range []string{"apworld", "Regenerate"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal never says %q: %v", want, err)
		}
	}
}

func TestPreviousFormatStillRuns(t *testing.T) {
	seed := SlotData{
		FormatVersion: gamedata.FormatVersion - 1,
		Missions:      []string{"mvm_decoy"},
		Goal:          "missionsanity", MissionsanityTarget: 1,
	}
	if err := seed.validate(); err != nil {
		t.Fatalf("existing room was rejected: %v", err)
	}
}

// Seeds made before the six traps after Team Jarate are format 7. Written as a
// number on purpose: FormatVersion - 1 moves with the next bump, and these runs
// do not.
func TestASeedFromBeforeTheNewTrapsStillRuns(t *testing.T) {
	seed := SlotData{
		FormatVersion: 7,
		Missions:      []string{"mvm_decoy"},
		Goal:          "missionsanity", MissionsanityTarget: 1,
	}
	if err := seed.validate(); err != nil {
		t.Fatalf("a format 7 room was rejected: %v", err)
	}
}
