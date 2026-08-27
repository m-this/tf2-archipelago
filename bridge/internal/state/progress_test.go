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
	if err := store.ClearProgress(); err != nil {
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
