package webapi

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// fakeGame stands in for srcds and SteamCMD, and writes down what was done to
// it in order.
type fakeGame struct {
	running bool
	failure error
	steps   []string
	during  func()
}

func (g *fakeGame) Running() bool { return g.running }

func (g *fakeGame) Stop() {
	g.steps = append(g.steps, "stop")
	g.running = false
}

func (g *fakeGame) Update(_ context.Context, logf func(string, ...any)) error {
	g.steps = append(g.steps, "update")
	logf("downloading TF2: %d%%", 40)
	if g.during != nil {
		g.during()
	}
	return g.failure
}

func (g *fakeGame) Start() {
	g.steps = append(g.steps, "start")
	g.running = true
}

func newUpdateApp(t *testing.T) *App {
	t.Helper()
	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	return New(s, nil)
}

func TestUpdatingARunningGameStopsUpdatesAndStartsIt(t *testing.T) {
	app := newUpdateApp(t)
	app.gameUpdateAvailable = true
	game := &fakeGame{running: true}
	game.during = func() {
		if snapshot := app.Snapshot(); !snapshot.Busy || snapshot.Activity != "downloading TF2: 40%" {
			t.Errorf("snapshot during the update = busy %v, activity %q", snapshot.Busy, snapshot.Activity)
		}
	}
	run, err := app.beginGameUpdate(game)
	if err != nil {
		t.Fatal(err)
	}
	run()
	if want := []string{"stop", "update", "start"}; !slices.Equal(game.steps, want) {
		t.Fatalf("steps = %v, want %v", game.steps, want)
	}
	snapshot := app.Snapshot()
	if snapshot.Busy || snapshot.GameUpdateAvailable || snapshot.GameUpdateError != "" {
		t.Fatalf("after a good update: busy %v, available %v, error %q",
			snapshot.Busy, snapshot.GameUpdateAvailable, snapshot.GameUpdateError)
	}
}

func TestUpdatingAStoppedGameLeavesItStopped(t *testing.T) {
	app := newUpdateApp(t)
	game := &fakeGame{}
	run, err := app.beginGameUpdate(game)
	if err != nil {
		t.Fatal(err)
	}
	run()
	if want := []string{"update"}; !slices.Equal(game.steps, want) {
		t.Fatalf("steps = %v, want %v", game.steps, want)
	}
}

func TestASecondUpdateIsRefusedWhileOneRuns(t *testing.T) {
	app := newUpdateApp(t)
	game := &fakeGame{running: true}
	run, err := app.beginGameUpdate(game)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.beginGameUpdate(game); !errors.Is(err, errActivityRunning) {
		t.Fatalf("second update = %v, want %v", err, errActivityRunning)
	}
	run()
	if want := []string{"stop", "update", "start"}; !slices.Equal(game.steps, want) {
		t.Fatalf("steps = %v, want one update: %v", game.steps, want)
	}
	if _, err := app.beginGameUpdate(game); err != nil {
		t.Fatalf("an update after the first finished = %v", err)
	}
}

func TestAnUpdateIsRefusedWhileAnInstallRuns(t *testing.T) {
	app := newUpdateApp(t)
	_, done, ok := app.beginSettingsActivity("Preparing selected server mods…")
	if !ok {
		t.Fatal("the install was refused")
	}
	defer done()
	if _, err := app.beginGameUpdate(&fakeGame{}); !errors.Is(err, errActivityRunning) {
		t.Fatalf("update during an install = %v, want %v", err, errActivityRunning)
	}
}

func TestAFailedUpdateIsShownAndTheServerComesBack(t *testing.T) {
	app := newUpdateApp(t)
	app.gameUpdateAvailable = true
	game := &fakeGame{running: true, failure: errors.New("SteamCMD left app 232250 at state 0x6")}
	run, err := app.beginGameUpdate(game)
	if err != nil {
		t.Fatal(err)
	}
	run()
	if want := []string{"stop", "update", "start"}; !slices.Equal(game.steps, want) {
		t.Fatalf("steps = %v, want %v", game.steps, want)
	}
	snapshot := app.Snapshot()
	if !strings.Contains(snapshot.GameUpdateError, "state 0x6") || snapshot.Proto().GetGameUpdateError() != snapshot.GameUpdateError {
		t.Fatalf("error = %q", snapshot.GameUpdateError)
	}
	if !snapshot.GameUpdateAvailable || snapshot.Busy {
		t.Fatalf("after a failed update: available %v, busy %v", snapshot.GameUpdateAvailable, snapshot.Busy)
	}

	game.failure = nil
	run, err = app.beginGameUpdate(game)
	if err != nil {
		t.Fatal(err)
	}
	run()
	if snapshot := app.Snapshot(); snapshot.GameUpdateError != "" {
		t.Fatalf("a good update kept the last error: %q", snapshot.GameUpdateError)
	}
}

func TestAStoppedUpdateDoesNotStartTheServer(t *testing.T) {
	app := newUpdateApp(t)
	game := &fakeGame{running: true}
	game.during = app.Stop
	game.failure = context.Canceled
	run, err := app.beginGameUpdate(game)
	if err != nil {
		t.Fatal(err)
	}
	run()
	if want := []string{"stop", "update"}; !slices.Equal(game.steps, want) {
		t.Fatalf("steps = %v, want %v", game.steps, want)
	}
	if snapshot := app.Snapshot(); snapshot.GameUpdateError != "" {
		t.Fatalf("pressing Stop reported an error: %q", snapshot.GameUpdateError)
	}
}

func TestSteamSayingTheGameIsBehindRaisesUpdateAvailable(t *testing.T) {
	app := newUpdateApp(t)
	app.append(apruntime.Line{At: time.Now(), Source: "srcds", Text: "Your server needs to be restarted in order to receive the latest update."})
	snapshot := app.Snapshot()
	if !snapshot.GameUpdateAvailable || !snapshot.Proto().GetGameUpdateAvailable() {
		t.Fatal("the update line from srcds did not raise update available")
	}
	other := newUpdateApp(t)
	other.append(apruntime.Line{At: time.Now(), Source: "rcon", Text: "Your server needs to be restarted in order to receive the latest update."})
	if other.Snapshot().GameUpdateAvailable {
		t.Fatal("a line that did not come from srcds raised update available")
	}
}

func TestAttachedUpdateNamesCompose(t *testing.T) {
	app := NewAttached(settings.Defaults(), nil, "")
	if err := app.UpdateGame(); err == nil || !strings.Contains(err.Error(), "docker compose") {
		t.Fatalf("attached update = %v, want Compose advice", err)
	}
}
