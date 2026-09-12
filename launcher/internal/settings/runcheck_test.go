package settings

import (
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
)

func TestCheckServerModsReadyRequiresIntentAndVerifiedInstall(t *testing.T) {
	s := Defaults()
	popFile := "mvm_bronx_rc2_adv_point_of_impact"
	s.MvmExcludedMissions = slices.DeleteFunc(s.MvmExcludedMissions, func(one string) bool { return one == popFile })
	if err := CheckServerModsReady(s, nil); err == nil || !strings.Contains(err.Error(), "turn on SigMod") {
		t.Fatalf("missing mod selection error = %v", err)
	}
	s.SrcdsMods = []string{"sigsegv-mvm"}
	if err := CheckServerModsReady(s, nil); err == nil || !strings.Contains(err.Error(), "missing or incomplete") {
		t.Fatalf("missing install error = %v", err)
	}
	if err := CheckServerModsReady(s, []string{"sigsegv-mvm"}); err != nil {
		t.Fatalf("verified install refused: %v", err)
	}
}

/*
apw-2kw: the number the launcher offers and the number the generator draws from
have to be one number.

A fresh settings file leaves every community mission out, so the honest ceiling
is the Valve pool. It used to be the whole catalogue, and a player who took the
launcher at its word asked for 82 missions and generated a run of 29.
*/
func TestTheCeilingAndThePreflightCountTheSamePool(t *testing.T) {
	for _, tier := range runshape.Tiers(MissionPool(Defaults())) {
		s := Defaults()
		s.MvmDifficulty = tier.Key
		s.MvmMissionCount = 1

		report, err := CheckRunSelection(s)
		if err != nil {
			t.Fatalf("%s: %v", tier.Key, err)
		}
		ceiling := runshape.MissionsInPool(MissionPool(s), tier.Key)
		if ceiling != report.Eligible {
			t.Fatalf("%s: the ceiling offers %d missions and the preflight finds %d eligible",
				tier.Key, ceiling, report.Eligible)
		}
	}
}

// Turning the community missions back on is what widens the pool, so the
// ceiling has to move with the exclusion list rather than ignore it.
func TestKeepingACommunityMissionRaisesTheCeiling(t *testing.T) {
	s := Defaults()
	before := runshape.MissionsInPool(MissionPool(s), s.MvmDifficulty)

	kept := "mvm_kelly_rc1b_adv_homestead_happenings"
	var excluded []string
	for _, popFile := range s.MvmExcludedMissions {
		if popFile != kept {
			excluded = append(excluded, popFile)
		}
	}
	if len(excluded) == len(s.MvmExcludedMissions) {
		t.Fatalf("%s is not excluded by default; pick another community mission", kept)
	}
	s.MvmExcludedMissions = excluded

	if got := runshape.MissionsInPool(MissionPool(s), s.MvmDifficulty); got != before+1 {
		t.Fatalf("ceiling after keeping one community mission = %d, want %d", got, before+1)
	}
}
