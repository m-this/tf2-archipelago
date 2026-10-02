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

/*
SigMod has no Windows build, and the row says so rather than disappearing.

v1.17.0 and v1.17.1 offered it there. The port crashed the game server with a
corrupted heap before it finished loading, on every start, so a player who
ticked it is owed the reason the tick is gone. apw-5g4.14.
*/
/*
The mod is three answers, not a tick, and Windows carries the warning it earned.

apw-5g4.14: the port crashed a real Windows server, and "only when a mission
needs it" is what makes offering it defensible. A player picking it is told so
before they pick.
*/
func TestServerModOffersThreeAnswers(t *testing.T) {
	state := NewState(settings.Defaults())
	for _, platform := range []string{"linux", "windows"} {
		field, ok := Build(state, Env{Platform: platform}).Field("missions.mod.sigsegv-mvm")
		if !ok || field.Disabled {
			t.Fatalf("%s SigMod field = %+v, found=%t", platform, field, ok)
		}
		if field.Kind != Choice {
			t.Errorf("%s SigMod is a %v, not a choice", platform, field.Kind)
		}
		var values []string
		for _, option := range field.Options {
			values = append(values, option.Value)
		}
		if want := []string{"off", "required", "always"}; !slices.Equal(values, want) {
			t.Errorf("%s SigMod offers %v, want %v", platform, values, want)
		}
		// Off until somebody asks: an engine-patching extension is not a
		// default.
		if field.Value != "off" {
			t.Errorf("%s SigMod defaults to %q", platform, field.Value)
		}
	}
}

func TestAttachedSigmodExplainsDockerAndHasNoInstallAction(t *testing.T) {
	state := NewState(settings.Defaults())
	state.Settings.SrcdsMods = []string{"sigsegv-mvm"}
	env := Env{Platform: "linux", ManagedExternally: true, CommunityAvailable: []string{settings.CommunityPackPotato}}
	built := Build(state, env)
	mod, ok := built.Field("missions.mod.sigsegv-mvm")
	if !ok || !strings.Contains(mod.Help, "Docker image already includes SigMod") {
		t.Fatalf("attached SigMod choice = %+v, found=%t", mod, ok)
	}
	if _, ok := built.Field("missions.install_mods"); ok {
		t.Fatal("Docker settings still offer a native server mod installer")
	}
	mission, ok := built.Field(sigmodMissionField)
	if !ok || !strings.Contains(mission.Reason, "recreate the container") {
		t.Fatalf("attached SigMod mission = %+v, found=%t", mission, ok)
	}
}

func TestSigmodWarnsOnWindowsOnly(t *testing.T) {
	state := NewState(settings.Defaults())
	windows, _ := Build(state, Env{Platform: "windows"}).Field("missions.mod.sigsegv-mvm")
	if !strings.Contains(windows.Warning, "crashed a server") {
		t.Errorf("Windows SigMod does not warn about the port: %q", windows.Warning)
	}
	linux, _ := Build(state, Env{Platform: "linux"}).Field("missions.mod.sigsegv-mvm")
	if linux.Warning != "" {
		t.Errorf("Linux SigMod carries a Windows warning: %q", linux.Warning)
	}
}

/*
The Windows row is labelled unstable, and no other row is.

Windows downloads this project's port; Linux and Docker download upstream's
release. The label is the one place a player sees that difference before they
pick, so a Linux label carrying it would be a lie about their build.
*/
func TestSigmodIsLabelledUnstableOnWindowsOnly(t *testing.T) {
	state := NewState(settings.Defaults())
	windows, _ := Build(state, Env{Platform: "windows"}).Field("missions.mod.sigsegv-mvm")
	if windows.Label != "SigMod (unstable)" {
		t.Errorf("Windows SigMod label = %q, want SigMod (unstable)", windows.Label)
	}
	linux, _ := Build(state, Env{Platform: "linux"}).Field("missions.mod.sigsegv-mvm")
	if linux.Label != "SigMod" {
		t.Errorf("Linux SigMod label = %q, want SigMod", linux.Label)
	}
}

// Picking a loading answer selects the mod and keeps the answer.
func TestServerModChoiceSetsTheLoadingPolicy(t *testing.T) {
	state := NewState(settings.Defaults())
	env := Env{Platform: "linux"}
	next, err := Apply(state, env, Change{Field: "missions.mod.sigsegv-mvm", Value: "always"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(next.Settings.SrcdsMods, "sigsegv-mvm") {
		t.Error("choosing a loading answer did not select the mod")
	}
	if next.Settings.SrcdsModLoading != settings.ModLoadingAlways {
		t.Errorf("loading = %q, want always", next.Settings.SrcdsModLoading)
	}
	field, _ := Build(next, env).Field("missions.mod.sigsegv-mvm")
	if field.Value != "always" {
		t.Errorf("the row reads back %q", field.Value)
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
	// The mod row is a choice with three answers and the community row is
	// still a tick, so "off" is spelled differently in each.
	for _, tc := range []struct {
		name    string
		field   string
		off     string
		matches func(gamedata.Mission) bool
	}{
		{"SigMod", "missions.mod.sigsegv-mvm", "off", func(m gamedata.Mission) bool { return gamedata.MissionServerMod(m.ID) == "sigsegv-mvm" }},
		{"community missions", "missions.community", "false", func(m gamedata.Mission) bool { return gamedata.IsCommunityMission(m.ID) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState(settings.Defaults())
			state.Settings.SrcdsMods = []string{"sigsegv-mvm"}
			state.Settings.MvmStartMission = sigmod
			state.Settings.SrcdsStartMission = sigmod
			state.Settings.MvmExcludedMissions = slices.DeleteFunc(state.Settings.MvmExcludedMissions,
				func(pop string) bool { return pop == sigmod || pop == ordinary })
			env := Env{Platform: "linux", CommunityAvailable: []string{settings.CommunityPackPotato}}
			next, err := Apply(state, env, Change{Field: tc.field, Value: tc.off})
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
