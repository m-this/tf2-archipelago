package form

import (
	"slices"
	"strings"
	"testing"

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

func TestSigmodIsUnavailableOnWindows(t *testing.T) {
	state := NewState(settings.Defaults())
	field, ok := Build(state, Env{Platform: "windows"}).Field("missions.mod.sigsegv-mvm")
	if !ok || !field.Disabled || !strings.Contains(field.Reason, "no Windows server build") {
		t.Fatalf("Windows SigMod field = %+v, found=%t", field, ok)
	}
}
