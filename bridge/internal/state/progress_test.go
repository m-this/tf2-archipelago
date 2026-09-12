package state

import (
	"path/filepath"
	"testing"
)

/*
	A restart must not cost the mission, so where the team was survives the

process that was playing it.

The checks already survive. The mission did not, which is what made a crash
read as an evening's work lost.
*/
func TestProgressSurvivesAReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.NoteProgress("mvm_decoy_advanced", 4); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Progress(); got.PopFile != "mvm_decoy_advanced" || got.Wave != 4 {
		t.Errorf("after a reopen the record is %+v", got)
	}
}

/*
	Only forward, and only within one mission.

A record naming a wave nobody reached is a way to skip content, and a wave
number left over from the mission before is worse than no record at all.
*/
func TestProgressOnlyEverMovesForward(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, wave := range []int{1, 2, 5} {
		if err := store.NoteProgress("mvm_decoy", wave); err != nil {
			t.Fatal(err)
		}
	}
	// A wave already passed is a replay, not progress.
	if err := store.NoteProgress("mvm_decoy", 3); err != nil {
		t.Fatal(err)
	}
	if got := store.Progress(); got.Wave != 5 {
		t.Errorf("the record went backwards to %d", got.Wave)
	}
	// Another mission starts its own count, however far the last one got.
	if err := store.NoteProgress("mvm_coaltown", 1); err != nil {
		t.Fatal(err)
	}
	if got := store.Progress(); got.PopFile != "mvm_coaltown" || got.Wave != 1 {
		t.Errorf("a new mission inherited the old one's wave: %+v", got)
	}
}

// A finished mission clears the record, or the next start drops the team into
// the end of one they already beat.
func TestAFinishedMissionForgetsWhereItWas(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.NoteProgress("mvm_decoy", 6); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearProgress("mvm_decoy"); err != nil {
		t.Fatal(err)
	}
	if got := store.Progress(); got != (Resume{}) {
		t.Errorf("the record survived the mission: %+v", got)
	}
}

// Nothing to record is not an error: a tank or a giant carries no wave.
func TestNothingToRecordIsNotAnError(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		popFile string
		wave    int
	}{
		{"", 3}, {"mvm_decoy", 0}, {"mvm_decoy", -1},
	} {
		if err := store.NoteProgress(test.popFile, test.wave); err != nil {
			t.Errorf("NoteProgress(%q, %d): %v", test.popFile, test.wave, err)
		}
		if got := store.Progress(); got != (Resume{}) {
			t.Errorf("NoteProgress(%q, %d) recorded %+v", test.popFile, test.wave, got)
		}
	}
}

/*
	The best each mission has ever seen, kept per mission.

Resume is only ever the mission the team is on, so switching away threw the old
one's progress out: three waves into Coal Town, go and look at Decoy, and Coal
Town was back at wave one. This is the record the Resume button on the mission
list reads, and it outlives the switch.
*/
func TestTheHighestWaveIsKeptPerMission(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.NoteProgress("mvm_coaltown", 3); err != nil {
		t.Fatal(err)
	}
	if err := store.NoteProgress("mvm_decoy", 1); err != nil {
		t.Fatal(err)
	}
	if got := store.Reached()["mvm_coaltown"]; got != 3 {
		t.Errorf("after switching missions Coal Town is at wave %d, want 3", got)
	}

	// Going back and doing worse does not lower the mark: it is the best the
	// team has ever done, not the last thing they did.
	if err := store.NoteProgress("mvm_coaltown", 2); err != nil {
		t.Fatal(err)
	}
	if got := store.Reached()["mvm_coaltown"]; got != 3 {
		t.Errorf("a worse run lowered the mark to %d", got)
	}
}

// A mission the team has beaten has nothing to go back to, so the button stops
// being offered for it. The other missions keep theirs.
func TestClearingAMissionForgetsOnlyItsMark(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.NoteProgress("mvm_coaltown", 3); err != nil {
		t.Fatal(err)
	}
	if err := store.NoteProgress("mvm_decoy", 2); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearProgress("mvm_decoy"); err != nil {
		t.Fatal(err)
	}
	reached := store.Reached()
	if _, kept := reached["mvm_decoy"]; kept {
		t.Error("a beaten mission still offers somewhere to resume")
	}
	if reached["mvm_coaltown"] != 3 {
		t.Errorf("clearing one mission took another's mark: %v", reached)
	}
}

// The mark survives a restart, which is the whole point of writing it down.
func TestTheMarkSurvivesAReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.NoteProgress("mvm_mannworks", 5); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Reached()["mvm_mannworks"]; got != 5 {
		t.Errorf("after a reopen the mark is %d, want 5", got)
	}
}
