package webapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	launcherv1 "github.com/m-this/tf2-archipelago/launcher/internal/gen/tf2ap/launcher/v1"
	"github.com/m-this/tf2-archipelago/launcher/internal/gen/tf2ap/launcher/v1/launcherv1connect"
	"github.com/m-this/tf2-archipelago/launcher/internal/session"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/tailscalefastdl"
)

// served starts the real mux over a real App and answers with clients for it.
// Nothing is stubbed: the launcher starts no server until Start is pressed, and
// every procedure below is one the browser reaches on a machine with nothing
// installed.
func served(t *testing.T, s settings.Settings) (*httptest.Server, *http.Client) {
	t.Helper()
	app := New(s, nil)
	server := httptest.NewServer(nil)
	server.Config.Handler = app.Handler(strings.TrimPrefix(server.URL, "http://"))
	t.Cleanup(server.Close)
	return server, server.Client()
}

func launcherClient(t *testing.T, s settings.Settings) launcherv1connect.LauncherServiceClient {
	t.Helper()
	server, client := served(t, s)
	return launcherv1connect.NewLauncherServiceClient(client, server.URL)
}

func settingsClient(t *testing.T, s settings.Settings) launcherv1connect.SettingsServiceClient {
	t.Helper()
	server, client := served(t, s)
	return launcherv1connect.NewSettingsServiceClient(client, server.URL)
}

func TestSnapshotCrossesTheWire(t *testing.T) {
	client := launcherClient(t, settings.Defaults())
	response, err := client.GetSnapshot(context.Background(), connect.NewRequest(&launcherv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := response.Msg.GetSnapshot()
	if snapshot.GetStatus() != launcherv1.ServerStatus_SERVER_STATUS_STOPPED {
		t.Errorf("a launcher that started nothing reports %s", snapshot.GetStatus())
	}
	if snapshot.GetTitle() == "" {
		t.Error("the snapshot carries no title")
	}
	if snapshot.GetForm() != nil {
		t.Error("a closed settings screen sent a model")
	}
}

// The buttons that own the two processes. Nothing is started here, so each one
// is being asked to be safe to press when there is nothing to press it on.
func TestTheServerButtonsAnswer(t *testing.T) {
	client := launcherClient(t, settings.Defaults())
	ctx := context.Background()
	if _, err := client.Stop(ctx, connect.NewRequest(&launcherv1.StopRequest{})); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if _, err := client.Restart(ctx, connect.NewRequest(&launcherv1.RestartRequest{})); err != nil {
		t.Errorf("Restart: %v", err)
	}
	if _, err := client.SendRcon(ctx, connect.NewRequest(&launcherv1.SendRconRequest{Command: "sm_ap_status"})); err != nil {
		t.Errorf("SendRcon: %v", err)
	}
	if _, err := client.SetMission(ctx, connect.NewRequest(&launcherv1.SetMissionRequest{PopFile: "mvm_decoy"})); err != nil {
		t.Errorf("SetMission: %v", err)
	}
}

func TestQuitCloses(t *testing.T) {
	app := New(settings.Defaults(), nil)
	server := httptest.NewServer(app.Handler("127.0.0.1"))
	defer server.Close()
	if _, err := (LauncherRPC{App: app}).Quit(context.Background(), connect.NewRequest(&launcherv1.QuitRequest{})); err != nil {
		t.Fatal(err)
	}
	select {
	case <-app.Quitting():
	default:
		t.Error("Quit did not close the interface")
	}
}

// The settings round trip: opening gives the rows, an answer comes back applied,
// and the running settings are untouched until Save.
func TestSettingsRoundTrip(t *testing.T) {
	client := settingsClient(t, settings.Defaults())
	ctx := context.Background()

	opened, err := client.OpenSettings(ctx, connect.NewRequest(&launcherv1.OpenSettingsRequest{Page: "Rewards"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Msg.GetScreen().GetModel().GetTabs()) == 0 {
		t.Fatal("opening the settings gave no rows")
	}

	changed, err := client.ChangeSetting(ctx, connect.NewRequest(&launcherv1.ChangeSettingRequest{
		Change: &launcherv1.Change{Field: "rewards.traps", Value: "42"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := fieldValue(changed.Msg.GetScreen(), "rewards.traps"); got != "42" {
		t.Errorf("the trap share came back as %q, want 42", got)
	}

	if _, err := client.CancelSettings(ctx, connect.NewRequest(&launcherv1.CancelSettingsRequest{})); err != nil {
		t.Fatal(err)
	}
}

func fieldValue(screen *launcherv1.Screen, id string) string {
	for _, tab := range screen.GetModel().GetTabs() {
		for _, field := range tab.GetFields() {
			if field.GetId() == id {
				return field.GetValue()
			}
		}
	}
	return ""
}

// A value form cannot apply is the player's mistake, not a broken call, so the
// message they read is the one form wrote.
func TestARefusedAnswerCarriesFormsWords(t *testing.T) {
	client := settingsClient(t, settings.Defaults())
	ctx := context.Background()
	if _, err := client.OpenSettings(ctx, connect.NewRequest(&launcherv1.OpenSettingsRequest{Page: "Rewards"})); err != nil {
		t.Fatal(err)
	}
	_, err := client.ChangeSetting(ctx, connect.NewRequest(&launcherv1.ChangeSettingRequest{
		Change: &launcherv1.Change{Field: "rewards.traps", Value: "not a number"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("a bad number answered %v, want invalid_argument", connect.CodeOf(err))
	}
}

// protovalidate is the boundary, so a request that never should have been sent
// is refused before it reaches the launcher at all.
func TestValidationRefusesWhatTheContractForbids(t *testing.T) {
	launcher := launcherClient(t, settings.Defaults())
	settingsService := settingsClient(t, settings.Defaults())
	ctx := context.Background()

	for name, call := range map[string]func() error{
		"an empty rcon command": func() error {
			_, err := launcher.SendRcon(ctx, connect.NewRequest(&launcherv1.SendRconRequest{Command: ""}))
			return err
		},
		"a mission name with a path in it": func() error {
			_, err := launcher.SetMission(ctx, connect.NewRequest(&launcherv1.SetMissionRequest{PopFile: "../../etc/passwd"}))
			return err
		},
		"a change naming no field": func() error {
			_, err := settingsService.ChangeSetting(ctx, connect.NewRequest(&launcherv1.ChangeSettingRequest{
				Change: &launcherv1.Change{Field: "", Value: "1"},
			}))
			return err
		},
		"a change that is not there": func() error {
			_, err := settingsService.ChangeSetting(ctx, connect.NewRequest(&launcherv1.ChangeSettingRequest{}))
			return err
		},
	} {
		if code := connect.CodeOf(call()); code != connect.CodeInvalidArgument {
			t.Errorf("%s answered %v, want invalid_argument", name, code)
		}
	}
}

// The app owns every path the Connect services do not, or a refresh on
// /settings/missions loses the player's page.
func TestTheAppOwnsEveryOtherPath(t *testing.T) {
	server, client := served(t, settings.Defaults())
	for _, path := range []string{"/", "/session", "/settings/missions"} {
		response, err := client.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Errorf("%s answered %d, want 200", path, response.StatusCode)
		}
	}
}

func TestARebindingHostCannotReachTheLauncher(t *testing.T) {
	server, client := served(t, settings.Defaults())
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "evil.example"
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("a rebound Host answered %d, want 403", response.StatusCode)
	}
}

// Showing a file is Go-side work. The launcher must name what it opened even
// when the desktop has no handler for it, or the player is left guessing.
func TestShowFileNamesWhatItOpened(t *testing.T) {
	app := New(settings.Defaults(), nil)
	opened := ""
	service := FilesRPC{App: app, open: func(path string) error {
		opened = path
		return errors.New("no desktop here")
	}}
	response, err := service.ShowFile(context.Background(), connect.NewRequest(&launcherv1.ShowFileRequest{
		Target: launcherv1.FileTarget_FILE_TARGET_SETTINGS_FILE,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetPath() == "" {
		t.Fatal("ShowFile named nothing")
	}
	if opened != response.Msg.GetPath() {
		t.Errorf("opened %q and reported %q", opened, response.Msg.GetPath())
	}
}

// The player file and the seed are written from the settings on the screen, so
// asking for one with the screen closed is a refusal rather than a stale file.
func TestFilesThatNeedTheScreenSayWhenItIsClosed(t *testing.T) {
	app := New(settings.Defaults(), nil)
	service := FilesRPC{App: app, open: func(string) error { return nil }}
	_, err := service.ShowFile(context.Background(), connect.NewRequest(&launcherv1.ShowFileRequest{
		Target: launcherv1.FileTarget_FILE_TARGET_INSTALL_ROOT,
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("a closed screen answered %v, want failed_precondition", connect.CodeOf(err))
	}
}

// The renderer switches on Kind, so a row whose Kind did not survive the trip
// draws as nothing. This walks a real model rather than a fixture.
func TestEveryRowKeepsItsKindOnTheWire(t *testing.T) {
	app := New(settings.Defaults(), nil)
	app.OpenSettings("")
	screen := app.Screen().Proto()
	seen := map[form.Kind]bool{}
	for _, tab := range screen.GetModel().GetTabs() {
		for _, field := range tab.GetFields() {
			if field.GetKind() == launcherv1.Kind_KIND_UNSPECIFIED {
				t.Fatalf("row %q crossed with no kind", field.GetId())
			}
			seen[form.Kind(field.GetKind())] = true
		}
	}
	for kind := form.Text; kind <= form.Confirm; kind++ {
		if !seen[kind] {
			t.Errorf("no row of kind %s reached the browser", kind)
		}
	}
}

// The bridge does not carry a tier, so the launcher reads it out of gamedata.
// A browser guessing which missions are Advanced would be guessing.
func TestTheTierComesFromGamedata(t *testing.T) {
	for popFile, want := range map[string]string{
		"mvm_decoy_advanced":    "Advanced",
		"mvm_coaltown":          "Normal",
		"mvm_mannworks_expert1": "Expert",
		"not_a_mission_that_is": "",
	} {
		if got := tierOf(popFile); got != want {
			t.Errorf("tierOf(%q) = %q, want %q", popFile, got, want)
		}
	}
}

/*
The archive a mission came from is the launcher's answer, not the bridge's.

The bridge fills Source out of its own state file, which for a mission it has
never seen is empty. gamedata knows the pack for every pop file, so the launcher
overwrites it, and this asks that it does so for every mission rather than only
for one out of an imported pack (kelly-cs, #50).
*/
func TestTheArchiveComesFromTheLauncherNotTheBridge(t *testing.T) {
	app := New(settings.Defaults(), nil)
	app.mu.Lock()
	app.snapshot = session.Snapshot{
		Missions: []session.Mission{
			{PopFile: "mvm_decoy", Name: "Doe's Drill", Source: "whatever the bridge said"},
			{PopFile: "not_a_mission", Name: "Made up", Source: "left alone"},
		},
	}
	app.mu.Unlock()

	missions := app.Snapshot().Session.Missions
	if got := missions[0].Source; got != "Valve" {
		t.Errorf("a Valve mission reads as %q, want Valve", got)
	}
	// A pop file gamedata does not know is one the launcher cannot name, so the
	// bridge's own answer is the only one there is.
	if got := missions[1].Source; got != "left alone" {
		t.Errorf("an unknown mission reads as %q, want the bridge's own answer", got)
	}
}

/*
A Funnel failure has to say what to do about it.

Taken from kelly-cs's #45. The usual cause is not a broken tailnet: it is
tailscaled refusing because this OS user was never made an operator. A browser
cannot run the fix and the launcher will not ask for root, so the only useful
answer is the exact command, in the message.
*/
func TestFunnelSaysHowToFixTheUsualFailure(t *testing.T) {
	restore := authorizeFunnel
	t.Cleanup(func() { authorizeFunnel = restore })

	app := New(settings.Defaults(), nil)
	service := LauncherRPC{App: app}

	authorizeFunnel = func(context.Context) (tailscalefastdl.Authorization, error) {
		return tailscalefastdl.Authorization{}, &tailscalefastdl.OperatorRequiredError{}
	}
	_, err := service.ApproveFunnel(context.Background(), connect.NewRequest(&launcherv1.ApproveFunnelRequest{}))
	if err == nil {
		t.Fatal("an unauthorized operator answered with no error")
	}
	said := err.Error()
	for _, want := range []string{"sudo tailscale set --operator=$USER", "-setup-funnel"} {
		if !strings.Contains(said, want) {
			t.Errorf("the answer does not name %q: %s", want, said)
		}
	}

	// Anything else still carries Tailscale's own words, because they are the
	// only thing that says which of the other failures it was.
	authorizeFunnel = func(context.Context) (tailscalefastdl.Authorization, error) {
		return tailscalefastdl.Authorization{}, errors.New("tailscaled is not running")
	}
	_, err = service.ApproveFunnel(context.Background(), connect.NewRequest(&launcherv1.ApproveFunnelRequest{}))
	if err == nil || !strings.Contains(err.Error(), "tailscaled is not running") {
		t.Errorf("a different failure lost what Tailscale said: %v", err)
	}
}

// A refused update is the player's to read, so it comes back as the error with
// the launcher's words in it rather than as a silent no-op.
func TestUpdateGameRefusalReachesTheBrowser(t *testing.T) {
	app := newUpdateApp(t)
	_, done, ok := app.beginSettingsActivity("Preparing selected server mods…")
	if !ok {
		t.Fatal("the install was refused")
	}
	defer done()
	_, err := LauncherRPC{App: app}.UpdateGame(context.Background(), connect.NewRequest(&launcherv1.UpdateGameRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || !strings.Contains(err.Error(), errActivityRunning.Error()) {
		t.Fatalf("update during an install = %v", err)
	}
}
