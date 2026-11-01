package settings

import (
	"slices"
	"testing"
)

const (
	sigmodKey     = "sigsegv-mvm"
	sigmodMission = "mvm_bronx_rc2_adv_point_of_impact"
	stockMission  = "mvm_decoy"
)

func TestWithoutServerMods(t *testing.T) {
	base := Defaults()
	base.SrcdsMods = []string{sigmodKey}
	base.MvmCommunityMissions = true
	base.MvmExcludedMissions = nil

	tests := []struct {
		name         string
		blocked      []string
		start        string
		wantMods     []string
		wantExcluded bool
		wantStart    string
	}{
		{"nothing blocked", nil, sigmodMission, []string{sigmodKey}, false, sigmodMission},
		{"SigMod blocked, start on a SigMod mission", []string{sigmodKey}, sigmodMission, nil, true, ""},
		{"SigMod blocked, start on a stock mission", []string{sigmodKey}, stockMission, nil, true, stockMission},
		{"another mod blocked", []string{"rafmod"}, sigmodMission, []string{sigmodKey}, false, sigmodMission},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := base
			s.MvmStartMission = test.start
			got := WithoutServerMods(s, test.blocked)

			if !slices.Equal(got.SrcdsMods, test.wantMods) {
				t.Errorf("mods = %v, want %v", got.SrcdsMods, test.wantMods)
			}
			if slices.Contains(got.MvmExcludedMissions, sigmodMission) != test.wantExcluded {
				t.Errorf("%s excluded = %v, want %v", sigmodMission, !test.wantExcluded, test.wantExcluded)
			}
			if slices.Contains(got.MvmExcludedMissions, stockMission) {
				t.Errorf("a mission that needs no mod was excluded")
			}
			if got.MvmStartMission != test.wantStart {
				t.Errorf("start mission = %q, want %q", got.MvmStartMission, test.wantStart)
			}
			if err := CheckServerModsReady(got, nil); test.wantExcluded && err != nil {
				t.Errorf("a seed without the blocked mod still asks for it: %v", err)
			}
		})
	}

	t.Run("the caller's settings are not changed", func(t *testing.T) {
		s := base
		s.MvmStartMission = sigmodMission
		_ = WithoutServerMods(s, []string{sigmodKey})
		if !slices.Equal(s.SrcdsMods, []string{sigmodKey}) || len(s.MvmExcludedMissions) != 0 {
			t.Errorf("input mutated: mods %v, excluded %d", s.SrcdsMods, len(s.MvmExcludedMissions))
		}
	})
}
