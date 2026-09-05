package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/bridge/config"
	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/tailscalefastdl"
)

func TestTestModeRejectsAnInvalidRunBeforeOpeningARoom(t *testing.T) {
	s := settings.Defaults()
	s.TestMode = true
	for _, mission := range gamedata.PlayableMissions() {
		s.MvmExcludedMissions = append(s.MvmExcludedMissions, mission.PopFile)
	}
	room, err := StartTestRoom(context.Background(), s, &config.Config{}, nil)
	if err == nil {
		if room != nil {
			_ = room.Close(context.Background())
		}
		t.Fatal("Test mode accepted an empty mission pool")
	}
}

// fakeServer writes a script in place of srcds_run, so a Start really does
// launch a process the Stop has to take down.
func fakeServer(t *testing.T, root string) settings.Settings {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake server is a shell script")
	}
	gameDir := filepath.Join(root, "tf-dedicated")
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatalf("cannot create %s: %v", gameDir, err)
	}
	script := "#!/bin/sh\necho fake server up\nwhile true; do sleep 1; done\n"
	path := filepath.Join(gameDir, "srcds_run")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write %s: %v", path, err)
	}

	s := settings.Defaults()
	s.InstallRoot = root
	s.SrcdsRconPw = "secret"
	// A port nothing serves: the bridge retries the connection rather than
	// exiting, which is what a supervisor test needs it to do.
	s.APHost, s.APPort, s.APTls = "127.0.0.1", 1, false
	s.MetricsPort = 0
	return s
}

func TestSupervisorStartStop(t *testing.T) {
	var mu sync.Mutex
	var lines []Line
	spoke := make(chan struct{})
	var once sync.Once
	s := NewSupervisor(fakeServer(t, t.TempDir()), nil, func(line Line) {
		mu.Lock()
		lines = append(lines, line)
		mu.Unlock()
		if line.Source == "srcds" && line.Text == "fake server up" {
			once.Do(func() { close(spoke) })
		}
	})

	exited := make(chan error, 1)
	if err := s.Start(func(err error) { exited <- err }); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !s.Running() {
		t.Fatal("Running is false after Start")
	}
	if err := s.Start(nil); err == nil {
		t.Error("a second Start was accepted")
	}

	select {
	case <-spoke:
	case <-time.After(10 * time.Second):
		t.Fatal("the server's own output never reached the sink")
	}

	s.Stop()
	if s.Running() {
		t.Error("Running is true after Stop")
	}
	select {
	case err := <-exited:
		if err != nil {
			t.Errorf("a Stop we asked for reported %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("onExit never fired")
	}

	// Stopping twice, and starting again, are both things a button can do.
	s.Stop()
	if err := s.Start(func(error) {}); err != nil {
		t.Fatalf("Start after Stop: %v", err)
	}
	s.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(lines) == 0 {
		t.Fatal("no lines reached the sink")
	}
}

func TestSupervisorRefusesAnIncompleteConfiguration(t *testing.T) {
	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	if err := NewSupervisor(s, nil, nil).Start(nil); err == nil {
		t.Fatal("Start with no room address and no RCON password succeeded")
	}
}

func TestSupervisorDoesNotHoldItsMutexWhileTailscaleStarts(t *testing.T) {
	s := settings.Defaults()
	s.TailscaleFastDL = true
	sup := NewSupervisor(s, nil, nil)
	entered := make(chan struct{})
	release := make(chan struct{})
	sup.configureTailscale = func(context.Context, int) (tailscalefastdl.Result, error) {
		close(entered)
		<-release
		return tailscalefastdl.Result{}, errors.New("setup stopped for the test")
	}

	startErr := make(chan error, 1)
	go func() { startErr <- sup.Start(nil) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("Tailscale setup did not start")
	}

	status := make(chan bool, 1)
	go func() { status <- sup.Running() }()
	select {
	case running := <-status:
		if !running {
			t.Fatal("Running is false while setup is in progress")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Running blocked on Tailscale setup")
	}

	close(release)
	if err := <-startErr; err == nil {
		t.Fatal("failed Tailscale setup was accepted")
	}
}

func TestSupervisorStopCancelsAndWaitsForTailscaleSetup(t *testing.T) {
	s := settings.Defaults()
	s.TailscaleFastDL = true
	sup := NewSupervisor(s, nil, nil)
	entered := make(chan struct{})
	sup.configureTailscale = func(ctx context.Context, _ int) (tailscalefastdl.Result, error) {
		close(entered)
		<-ctx.Done()
		return tailscalefastdl.Result{}, ctx.Err()
	}

	startErr := make(chan error, 1)
	go func() { startErr <- sup.Start(nil) }()
	<-entered
	sup.Stop()
	if err := <-startErr; err != nil {
		t.Fatalf("operator-cancelled Start returned %v", err)
	}
	if sup.Running() {
		t.Fatal("Running is true after cancelling setup")
	}
}

func TestSupervisorStopRemovesTheFastDLFunnelRoute(t *testing.T) {
	s := fakeServer(t, t.TempDir())
	s.TailscaleFastDL = true
	sup := NewSupervisor(s, nil, nil)
	sup.configureTailscale = func(context.Context, int) (tailscalefastdl.Result, error) {
		return tailscalefastdl.Result{URL: "https://host.example.ts.net/tf"}, nil
	}
	disabled := make(chan struct{}, 1)
	sup.disableTailscale = func(context.Context) error {
		disabled <- struct{}{}
		return nil
	}

	if err := sup.Start(nil); err != nil {
		t.Fatal(err)
	}
	sup.Stop()
	select {
	case <-disabled:
	default:
		t.Fatal("Stop did not remove the FastDL Funnel route")
	}
}

func TestSupervisorStartupFailureRemovesAConfiguredFunnelRoute(t *testing.T) {
	s := settings.Defaults()
	s.TailscaleFastDL = true
	// No RCON password makes bridgeConfig fail after Funnel was configured.
	sup := NewSupervisor(s, nil, nil)
	sup.configureTailscale = func(context.Context, int) (tailscalefastdl.Result, error) {
		return tailscalefastdl.Result{URL: "https://host.example.ts.net/tf"}, nil
	}
	disabled := false
	sup.disableTailscale = func(context.Context) error {
		disabled = true
		return nil
	}

	if err := sup.Start(nil); err == nil {
		t.Fatal("incomplete configuration was accepted")
	}
	if !disabled {
		t.Fatal("startup failure left the configured Funnel route behind")
	}
}

// The window passes a sink and no logger. Anything that reports has to survive
// that: a method call on a nil *slog.Logger panics, and one took the launcher
// down at startup.
func TestReportSurvivesANilLogger(t *testing.T) {
	var got []Line
	report(nil, func(line Line) { got = append(got, line) }, "the console did not come up")
	if len(got) != 1 || got[0].Source != "launcher" {
		t.Fatalf("the sink got %+v", got)
	}
	report(nil, nil, "nobody is listening") // must not panic
}
