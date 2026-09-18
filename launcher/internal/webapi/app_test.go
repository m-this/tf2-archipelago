package webapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func TestMissionPoolRowsCarryTableMetadata(t *testing.T) {
	rows := testMissionPoolRows(form.NewState(settings.Defaults()), nil, nil)
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
	rows := testMissionPoolRows(state, []string{
		settings.CommunityPackPotato,
		settings.CommunityPackMoonlight,
	}, []string{"sigsegv-mvm"})
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
	rows := testMissionPoolRows(form.NewState(settings.Defaults()),
		[]string{settings.CommunityPackPotato}, nil)
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
	rows := testMissionPoolRows(state, nil, nil)
	if !slices.ContainsFunc(rows, func(row MissionPoolRow) bool {
		return strings.HasPrefix(row.Compatibility, "Below Advanced")
	}) {
		t.Fatal("the table does not explain why lower-tier missions are ineligible")
	}
}

func TestSelectedSigModMissionCanBeUntickedInTable(t *testing.T) {
	const pop = "mvm_bronx_rc2_adv_point_of_impact"
	state := form.NewState(settings.Defaults())
	state.Settings.MvmExcludedMissions = slices.DeleteFunc(state.Settings.MvmExcludedMissions,
		func(one string) bool { return one == pop })
	rows := testMissionPoolRows(state, []string{settings.CommunityPackPotato}, nil)
	for _, row := range rows {
		if row.Field == "missions.pool."+pop {
			if row.Disabled || row.Compatibility != "Turn on SigMod above" {
				t.Fatalf("selected SigMod row = %+v", row)
			}
			return
		}
	}
	t.Fatal("SigMod mission is missing from the table")
}

func testMissionPoolRows(state form.State, available, ready []string) []MissionPoolRow {
	built := form.Build(state, form.Env{CommunityAvailable: available, ServerModsReady: ready, Platform: "linux"})
	return missionPoolRows(state, built, available, nil, ready)
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

func TestPoolAllLeavesCommunityMissionsOutWhileTheSwitchIsOff(t *testing.T) {
	const community = "mvm_kelly_rc1b_adv_homestead_happenings"
	app := New(settings.Defaults(), nil)
	app.OpenSettings("Missions")
	app.community = []string{settings.CommunityPackPotato}
	app.draft.Settings.MvmCommunityMissions = false
	app.setPool(true)
	if !slices.Contains(app.draft.Settings.MvmExcludedMissions, community) {
		t.Error("pool all ticked a community mission while community missions are off")
	}
	if slices.Contains(app.draft.Settings.MvmExcludedMissions, "mvm_decoy") {
		t.Error("pool all left a Valve mission out")
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

func TestAttachedSnapshotNamesDockerAsTheLifecycleOwner(t *testing.T) {
	s := settings.Defaults()
	s.SrcdsPort = 27115
	app := NewAttached(s, nil, "")
	snapshot := app.Snapshot()
	if !snapshot.ManagedExternally || !snapshot.Proto().GetManagedExternally() {
		t.Fatal("an attached admin UI offers launcher-owned lifecycle controls")
	}
	if snapshot.Join != "127.0.0.1:27115" || !strings.Contains(snapshot.JoinURL, "127.0.0.1:27115") {
		t.Fatalf("attached join address escaped through the container network: %q, %q", snapshot.Join, snapshot.JoinURL)
	}
	app.OpenSettings("")
	if err := app.SaveSettings(false); err == nil || !strings.Contains(err.Error(), ".env") {
		t.Fatalf("attached settings save = %v, want .env advice", err)
	}
}

func TestSettingsActivityAppearsOnTheSnapshotWhileItRuns(t *testing.T) {
	app := New(settings.Defaults(), nil)
	_, done, ok := app.beginSettingsActivity("Checking cached community assets…")
	if !ok {
		t.Fatal("the first settings activity was refused")
	}
	app.reportSettingsActivity("downloading %s: %d%%", "archive-assets.zip", 25)
	if snapshot := app.Snapshot(); !snapshot.Busy || snapshot.Activity != "downloading archive-assets.zip: 25%" || snapshot.Proto().GetActivity() != snapshot.Activity {
		t.Fatalf("running activity snapshot = %+v", snapshot)
	}
	done()
	if snapshot := app.Snapshot(); snapshot.Busy || snapshot.Activity != "" {
		t.Fatalf("finished activity snapshot = %+v", snapshot)
	}
}

func TestAttachedServerModSetupExplainsComposeRecreate(t *testing.T) {
	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	s.SrcdsMods = []string{"sigsegv-mvm"}
	app := NewAttached(s, nil, "")

	app.installSelectedMods(s)
	if notice := app.Snapshot().Notice; !strings.Contains(notice, "docker compose up -d --force-recreate") {
		t.Fatalf("attached server mod setup notice = %q", notice)
	}
}

func TestAttachedSnapshotUsesConfiguredJoinHost(t *testing.T) {
	s := settings.Defaults()
	s.SrcdsPort = 27115
	s.SrcdsJoinHost = "192.0.2.42"
	app := NewAttached(s, nil, "")
	snapshot := app.Snapshot()
	if snapshot.Join != "192.0.2.42:27115" || !strings.Contains(snapshot.JoinURL, "192.0.2.42:27115") {
		t.Fatalf("attached join address = %q, %q", snapshot.Join, snapshot.JoinURL)
	}
}

func TestAttachedSteamJoinWaitsForAndUsesTheRelay(t *testing.T) {
	s := settings.Defaults()
	s.SrcdsToken = "real-token"
	s.SrcdsReach = settings.ReachSteam
	s.SrcdsJoinHost = "198.51.100.7"
	app := NewAttached(s, nil, "")
	if snapshot := app.Snapshot(); snapshot.JoinURL != "" || !strings.Contains(snapshot.Join, "waiting") {
		t.Fatalf("attached relay joined before allocation: %q, %q", snapshot.Join, snapshot.JoinURL)
	}
	app.append(apruntime.Line{Source: "srcds", Text: "FakeIP allocation succeeded: 169.254.13.42:20232, 20233"})
	snapshot := app.Snapshot()
	if snapshot.Join != "169.254.13.42:20232" || !strings.Contains(snapshot.JoinURL, "169.254.13.42:20232") {
		t.Fatalf("attached relay kept the configured host: %q, %q", snapshot.Join, snapshot.JoinURL)
	}
}

// The room of a self-hosted stack is archipelago:38281 with no TLS. Saving
// anything used to re-read that bare address as a public one, write AP_TLS=1,
// and leave the bridge dialling wss:// at a plain ws:// room after the next
// recreate.
func TestAttachedSaveKeepsAPlainRoomPlain(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("AP_HOST=archipelago\nAP_PORT=38281\nAP_TLS=false\nSRCDS_HOSTNAME=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := settings.Defaults()
	s.APHost, s.APPort, s.APTls = "archipelago", 38281, false
	s.SrcdsHostname = "old"
	app := NewAttached(s, nil, path)
	app.OpenSettings("Game server")
	if err := app.Change(form.Change{Field: "server.hostname", Value: "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := app.SaveSettings(false); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "AP_TLS='1'") || app.supervisor.Settings().APTls {
		t.Fatalf("saving the hostname turned TLS on for a plain room:\n%s", body)
	}
}

func TestAttachedSettingsPersistToComposeEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("SRCDS_HOSTNAME=old\nTF2AP_JOIN_HOST=127.0.0.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := settings.Defaults()
	s.SrcdsHostname = "old"
	s.SrcdsJoinHost = "127.0.0.1"
	app := NewAttached(s, nil, path)
	app.OpenSettings("Game server")
	if err := app.Change(form.Change{Field: "server.hostname", Value: "public test"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Change(form.Change{Field: "server.join_host", Value: "tf2.example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := app.SaveSettings(false); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "SRCDS_HOSTNAME='public test'") || !strings.Contains(text, "TF2AP_JOIN_HOST='tf2.example.com'") {
		t.Fatalf("attached save did not update .env:\n%s", text)
	}
	if got := app.Snapshot().Join; got != "tf2.example.com:27015" {
		t.Fatalf("join address did not update immediately: %q", got)
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

// Every button form.Act answers has to reach it through Dispatch. The pair of
// lists this used to keep, one in wiredActions and one in Dispatch's switch,
// drifted apart: bots.name_add was in the first and not the second, so the
// test above passed and the browser answered "is not wired" to anybody who
// typed a bot name. Ask Dispatch, not a list.
func TestEveryStateOnlyActionReachesFormAct(t *testing.T) {
	app := New(settings.Defaults(), nil)
	app.OpenSettings("")
	state := form.NewState(settings.Defaults())
	for _, spec := range form.Specs(state, form.Env{}) {
		if spec.Kind != form.Action && spec.Kind != form.Confirm {
			continue
		}
		if _, _, ok := form.Act(state, spec.ID); !ok {
			continue
		}
		if err := app.Dispatch(spec.ID); err != nil && strings.Contains(err.Error(), "is not wired") {
			t.Errorf("form.Act answers %q and Dispatch does not route it: %v", spec.ID, err)
		}
	}
}
