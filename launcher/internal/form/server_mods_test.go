package form

import (
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

const sigmodMissionField = "missions.pool.mvm_bronx_rc2_adv_point_of_impact"

func TestMissingNavigationMeshMissionNamesItsMapAndStaysOff(t *testing.T) {
	state := NewState(settings.Defaults())
	field, ok := Build(state, Env{
		CommunityAvailable: []string{settings.CommunityPackPotato},
		Platform:           "linux",
	}).Field("missions.pool.mvm_bogland_rc12_adv_swamp_fever")
	if !ok {
		t.Fatal("missing-NAV mission is missing from the mission list")
	}
	if !field.Disabled || field.Value != "false" || !strings.Contains(field.Reason, "maps/mvm_bogland_rc12.nav") {
		t.Fatalf("missing-NAV mission field = %+v", field)
	}
	if !strings.Contains(field.Help, "maps/mvm_bogland_rc12.nav") {
		t.Fatalf("missing-NAV mission help = %q", field.Help)
	}
}

func TestSigmodMissionExplainsAndEnforcesSetup(t *testing.T) {
	state := NewState(settings.Defaults())
	env := Env{CommunityAvailable: []string{settings.CommunityPackPotato}, Platform: "linux"}
	field, ok := Build(state, env).Field(sigmodMissionField)
	if !ok {
		t.Fatal("SigMod mission is missing from the mission list")
	}
	if !field.Disabled || !strings.Contains(field.Reason, "turn on SigMod") {
		t.Fatalf("unconfigured SigMod mission = %+v", field)
	}
	if _, err := Apply(state, env, Change{Field: sigmodMissionField, Value: "true"}); err == nil || !strings.Contains(err.Error(), "turn on SigMod") {
		t.Fatalf("selecting an unavailable SigMod mission error = %v", err)
	}

	state.Settings.SrcdsMods = []string{"sigsegv-mvm"}
	field, _ = Build(state, env).Field(sigmodMissionField)
	if !field.Disabled || !strings.Contains(field.Reason, "missing or incomplete") {
		t.Fatalf("uninstalled SigMod mission = %+v", field)
	}

	env.ServerModsReady = []string{"sigsegv-mvm"}
	field, _ = Build(state, env).Field(sigmodMissionField)
	if field.Disabled {
		t.Fatalf("verified SigMod mission is disabled: %+v", field)
	}
	next, err := Apply(state, env, Change{Field: sigmodMissionField, Value: "true"})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(next.Settings.MvmExcludedMissions, "mvm_bronx_rc2_adv_point_of_impact") {
		t.Fatal("verified SigMod mission was not put in the pool")
	}
	start, _ := Build(state, env).Field("missions.start")
	if !slices.ContainsFunc(start.Options, func(option Option) bool {
		return option.Value == "mvm_bronx_rc2_adv_point_of_impact"
	}) {
		t.Fatal("verified SigMod mission is missing from the start choices")
	}
}

// The Windows build is this project's port rather than upstream's release, and
// a player who is about to download a binary should be told whose it is.
func TestSigmodOnWindowsOffersThePortAndSaysSo(t *testing.T) {
	state := NewState(settings.Defaults())
	field, ok := Build(state, Env{Platform: "windows"}).Field("missions.mod.sigsegv-mvm")
	if !ok || field.Disabled {
		t.Fatalf("Windows SigMod field = %+v, found=%t", field, ok)
	}
	if !strings.Contains(field.Help, "sigsegv-mvm-win") {
		t.Errorf("Windows SigMod help does not name the port it installs: %q", field.Help)
	}
}

func TestUnavailableSelectedMissionsCanBeRemovedFromPool(t *testing.T) {
	const ordinary = "mvm_kelly_rc1b_adv_homestead_happenings"
	for _, tc := range []struct {
		name string
		pop  string
		env  Env
		edit func(*State)
	}{
		{"SigMod off", "mvm_bronx_rc2_adv_point_of_impact", Env{Platform: "linux"}, nil},
		{"SigMod not installed", "mvm_bronx_rc2_adv_point_of_impact", Env{Platform: "linux"}, func(s *State) {
			s.Settings.SrcdsMods = []string{"sigsegv-mvm"}
		}},
		{"community off", ordinary, Env{Platform: "linux"}, func(s *State) {
			s.Settings.MvmCommunityMissions = false
		}},
		{"SigMod community off", "mvm_bronx_rc2_adv_point_of_impact", Env{Platform: "linux", ServerModsReady: []string{"sigsegv-mvm"}}, func(s *State) {
			s.Settings.SrcdsMods = []string{"sigsegv-mvm"}
			s.Settings.MvmCommunityMissions = false
		}},
		{"SigMod on Windows", "mvm_bronx_rc2_adv_point_of_impact", Env{Platform: "windows"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState(settings.Defaults())
			state.Settings.MvmExcludedMissions = slices.DeleteFunc(state.Settings.MvmExcludedMissions,
				func(pop string) bool { return pop == tc.pop })
			if tc.edit != nil {
				tc.edit(&state)
			}
			tc.env.CommunityAvailable = []string{settings.CommunityPackPotato}
			id := "missions.pool." + tc.pop
			field, ok := Build(state, tc.env).Field(id)
			if !ok || field.Disabled || field.Value != "true" {
				t.Fatalf("selected unavailable mission = %+v, found=%t", field, ok)
			}
			next, err := Apply(state, tc.env, Change{Field: id, Value: "false"})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(next.Settings.MvmExcludedMissions, tc.pop) {
				t.Fatal("mission remained in the pool")
			}
			field, _ = Build(next, tc.env).Field(id)
			if !field.Disabled || field.Value != "false" {
				t.Fatalf("excluded unavailable mission = %+v", field)
			}
			if _, err := Apply(next, tc.env, Change{Field: id, Value: "true"}); err == nil {
				t.Fatal("unavailable mission could be reselected")
			}
		})
	}
}

func TestTurningOffMissionRequirementsUnticksTheirMissions(t *testing.T) {
	const sigmod = "mvm_bronx_rc2_adv_point_of_impact"
	const ordinary = "mvm_kelly_rc1b_adv_homestead_happenings"
	for _, tc := range []struct {
		name    string
		field   string
		matches func(gamedata.Mission) bool
	}{
		{"SigMod", "missions.mod.sigsegv-mvm", func(m gamedata.Mission) bool { return gamedata.MissionServerMod(m.ID) == "sigsegv-mvm" }},
		{"community missions", "missions.community", func(m gamedata.Mission) bool { return gamedata.IsCommunityMission(m.ID) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState(settings.Defaults())
			state.Settings.SrcdsMods = []string{"sigsegv-mvm"}
			state.Settings.MvmStartMission = sigmod
			state.Settings.SrcdsStartMission = sigmod
			state.Settings.MvmExcludedMissions = slices.DeleteFunc(state.Settings.MvmExcludedMissions,
				func(pop string) bool { return pop == sigmod || pop == ordinary })
			env := Env{Platform: "linux", CommunityAvailable: []string{settings.CommunityPackPotato}}
			next, err := Apply(state, env, Change{Field: tc.field, Value: "false"})
			if err != nil {
				t.Fatal(err)
			}
			for _, mission := range gamedata.Missions {
				if tc.matches(mission) && !slices.Contains(next.Settings.MvmExcludedMissions, mission.PopFile) {
					t.Errorf("%s remained in the pool", mission.PopFile)
				}
			}
			if next.Settings.MvmStartMission != "" {
				t.Errorf("ineligible start mission was kept: %s", next.Settings.MvmStartMission)
			}
			if next.Settings.SrcdsStartMission != settings.Defaults().SrcdsStartMission {
				t.Errorf("server would still boot on %s", next.Settings.SrcdsStartMission)
			}
			if tc.field == "missions.mod.sigsegv-mvm" && slices.Contains(next.Settings.MvmExcludedMissions, ordinary) {
				t.Error("turning off SigMod excluded an ordinary community mission")
			}
			if slices.Contains(next.Settings.MvmExcludedMissions, "mvm_decoy") {
				t.Error("turning off a requirement excluded a Valve mission")
			}
			if err := settings.CheckServerModsReady(next.Settings, nil); err != nil {
				t.Errorf("Check Run Selection still requires SigMod: %v", err)
			}
			if _, err := settings.CheckRunSelection(next.Settings); err != nil {
				t.Errorf("Check Run Selection refused the remaining pool: %v", err)
			}
		})
	}
}
