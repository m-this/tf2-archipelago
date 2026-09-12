package webapi

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func TestMissionPoolRowsCarryTableMetadata(t *testing.T) {
	rows := missionPoolRows(form.NewState(settings.Defaults()), nil, nil, nil)
	if len(rows) == 0 {
		t.Fatal("the mission pool table is empty")
	}
	row := rows[0]
	if row.Field == "" || row.Source == "" || row.Map == "" || row.Name == "" || row.Waves == "" ||
		row.Compatibility == "" || row.Mods == "" {
		t.Fatalf("mission pool row has an empty column: %+v", row)
	}
	if !strings.HasPrefix(row.Field, "missions.pool.") {
		t.Errorf("mission pool field = %q", row.Field)
	}
}

func TestMissionPoolRowsNameRequiredServerMods(t *testing.T) {
	state := form.NewState(settings.Defaults())
	state.Settings.SrcdsMods = []string{"sigsegv-mvm"}
	rows := missionPoolRows(state, []string{
		settings.CommunityPackPotato,
		settings.CommunityPackMoonlight,
	}, nil, []string{"sigsegv-mvm"})
	for _, row := range rows {
		if row.Field != "missions.pool.mvm_bronx_rc2_adv_point_of_impact" {
			continue
		}
		if row.Mods != "SigMod" || row.Compatibility != "Ready" {
			t.Fatalf("SigMod mission row = %+v", row)
		}
		return
	}
	t.Fatal("expanded Potato mission is missing from the mission table")
}

func TestMissionPoolRowsNameTheExactMissingNavigationMesh(t *testing.T) {
	rows := missionPoolRows(form.NewState(settings.Defaults()),
		[]string{settings.CommunityPackPotato}, nil, nil)
	for _, row := range rows {
		if row.Field != "missions.pool.mvm_bogland_rc12_adv_swamp_fever" {
			continue
		}
		if !row.Disabled || row.Map != "mvm_bogland_rc12" || row.Compatibility != "Missing maps/mvm_bogland_rc12.nav" {
			t.Fatalf("missing-NAV mission row = %+v", row)
		}
		return
	}
	t.Fatal("missing-NAV mission is missing from the mission table")
}

func TestMissionPoolRowsExplainDifficultyFloor(t *testing.T) {
	state := form.NewState(settings.Defaults())
	state.Settings.MvmDifficulty = "advanced"
	rows := missionPoolRows(state, nil, nil, nil)
	if !slices.ContainsFunc(rows, func(row MissionPoolRow) bool {
		return strings.HasPrefix(row.Compatibility, "Below Advanced")
	}) {
		t.Fatal("the table does not explain why lower-tier missions are ineligible")
	}
}

func TestPoolNoneClearsTheNamedStartMission(t *testing.T) {
	app := New(settings.Defaults(), nil)
	app.OpenSettings("Missions")
	app.draft.Settings.MvmStartMission = "mvm_decoy"
	app.setPool(false)
	if got := app.draft.Settings.MvmStartMission; got != "" {
		t.Errorf("pool none kept start mission %q", got)
	}
}

func TestSettingsAreTheFormModelAndChangesGoThroughApply(t *testing.T) {
	app := New(settings.Defaults(), nil)
	app.OpenSettings("Rewards")
	if err := app.Change(form.Change{Field: "rewards.traps", Value: "42"}); err != nil {
		t.Fatal(err)
	}

	snapshot := app.Snapshot()
	if snapshot.Form == nil {
		t.Fatal("opening settings produced no form model")
	}
	field, ok := snapshot.Form.Field("rewards.traps")
	if !ok {
		t.Fatal("the form omitted rewards.traps")
	}
	if field.Value != "42" {
		t.Errorf("the trap share is %q, want 42", field.Value)
	}
	if app.settings.MvmTrapPct == 42 {
		t.Error("editing the draft changed the running settings before Save")
	}
}

func TestSnapshotDoesNotExposePasswords(t *testing.T) {
	s := settings.Defaults()
	s.APPassword = "archipelago-secret"
	s.SrcdsRconPw = "rcon-secret"
	s.SrcdsPw = "game-secret"
	app := New(s, nil)
	app.OpenSettings("")

	data, err := json.Marshal(app.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, secret := range []string{"archipelago-secret", "rcon-secret"} {
		if strings.Contains(text, secret) {
			t.Errorf("the browser snapshot exposes %q", secret)
		}
	}
	// The join line intentionally contains the ordinary game password: it is
	// what the player copies to friends and what the old window showed.
	if !strings.Contains(text, "game-secret") {
		t.Error("the snapshot lost the join password")
	}
}

func TestEveryFormButtonIsWiredAndNothingStaleIsWired(t *testing.T) {
	state := form.NewState(settings.Defaults())
	declared := map[string]bool{}
	for _, spec := range form.Specs(state, form.Env{}) {
		if spec.Kind == form.Action || spec.Kind == form.Confirm {
			declared[spec.ID] = true
			if !slices.Contains(wiredActions, spec.ID) {
				t.Errorf("form declares %q and the browser does not wire it", spec.ID)
			}
		}
	}
	for _, id := range wiredActions {
		if !declared[id] {
			t.Errorf("the browser wires %q and form does not declare it", id)
		}
	}
}

func TestRestartOnSaveAppearsOnlyForAChangeThatNeedsOne(t *testing.T) {
	before := settings.Defaults()

	port := form.NewState(before)
	port.Settings.SrcdsPort++
	if !restartNeeded(true, before, &port) {
		t.Error("a running server did not offer Restart on save for a port change")
	}
	if restartNeeded(false, before, &port) {
		t.Error("a stopped server offered Restart on save")
	}

	team := form.NewState(before)
	team.Settings.SrcdsBotTeamSize++
	if restartNeeded(true, before, &team) {
		t.Error("a bot lineup change offered Restart on save")
	}

	runShape := form.NewState(before)
	runShape.Settings.MvmMissionCount++
	if restartNeeded(true, before, &runShape) {
		t.Error("a run-shape change offered Restart on save")
	}
}

func TestSnapshotProvidesSteamURLToTheBrowser(t *testing.T) {
	snapshot := New(settings.Defaults(), nil).Snapshot()
	if !strings.HasPrefix(snapshot.JoinURL, "steam://run/440//+connect%20") {
		t.Errorf("join URL = %q", snapshot.JoinURL)
	}
}

func TestSteamJoinWaitsForThePublishedAddress(t *testing.T) {
	s := settings.Defaults()
	s.SrcdsToken = "real-token"
	s.SrcdsReach = settings.ReachSteam
	app := New(s, nil)
	if got := app.Snapshot().JoinURL; got != "" {
		t.Fatalf("Join used a local address while Steam was still starting: %q", got)
	}
	app.append(apruntime.Line{Source: "srcds", Text: "FakeIP allocation succeeded: 169.254.13.42:20232, 20233"})
	if got := app.Snapshot().JoinURL; !strings.Contains(got, "169.254.13.42:20232") {
		t.Fatalf("Join did not use Steam's published address: %q", got)
	}
}
