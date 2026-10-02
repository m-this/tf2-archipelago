package webapi

import (
	"context"
	"errors"
	"io/fs"

	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// gameHost is what an update does around SteamCMD: the server it stops and
// starts again. The launcher's own is the supervisor; a test stands in for it.
type gameHost interface {
	Running() bool
	Stop()
	Update(ctx context.Context, logf func(string, ...any)) error
	Start()
}

type supervisedGame struct {
	app      *App
	settings settings.Settings
}

func (g supervisedGame) Running() bool { return g.app.supervisor.Running() }

func (g supervisedGame) Stop() {
	g.app.supervisor.Stop()
	g.app.publishState()
}

func (g supervisedGame) Update(ctx context.Context, logf func(string, ...any)) error {
	return installer.UpdateGame(ctx, g.settings.InstallRoot, logf)
}

func (g supervisedGame) Start() { g.app.boot(g.settings) }

// UpdateGame brings TF2 to Steam's current build on request. A running server
// is stopped first and started again after, the way Start starts it; a stopped
// one stays stopped. The work outlives the request that asked for it.
func (a *App) UpdateGame() error {
	a.mu.Lock()
	s, attached := a.settings, a.attached
	a.mu.Unlock()
	if attached {
		return errors.New("the game server belongs to Docker Compose, which updates TF2 every time srcds starts. Run: docker compose restart srcds")
	}
	run, err := a.beginGameUpdate(supervisedGame{app: a, settings: s})
	if err != nil {
		return err
	}
	go apruntime.Guard("the TF2 update", a.sayLine, run)
	return nil
}

// beginGameUpdate claims the one activity slot, so an install or a second
// update is refused, and answers with the update to run. The Stop button
// cancels it the way it cancels an install, and then nothing is started.
func (a *App) beginGameUpdate(game gameHost) (func(), error) {
	ctx, done, err := a.claimActivity("Updating TF2…")
	if err != nil {
		return nil, err
	}
	return func() {
		defer done()
		a.mu.Lock()
		a.gameUpdateError = ""
		a.mu.Unlock()
		wasRunning := game.Running()
		if wasRunning {
			a.reportSettingsActivity("Stopping the server to update TF2…")
			game.Stop()
		}
		err := game.Update(ctx, a.reportSettingsActivity)
		if ctx.Err() != nil {
			a.Say("the TF2 update was stopped")
			return
		}
		a.gameUpdateFinished(err)
		if err != nil {
			a.Say("TF2 update failed: %v", err)
		} else {
			a.Say("TF2 is up to date")
		}
		if wasRunning {
			a.reportSettingsActivity("Starting the server again…")
			game.Start()
		}
	}, nil
}

// gameUpdateFinished records how an update ended, from Start or on request. A
// good one answers Steam's "needs to be restarted", and the build it left is
// read again.
func (a *App) gameUpdateFinished(err error) {
	a.mu.Lock()
	root := a.settings.InstallRoot
	a.mu.Unlock()
	build := a.readGameBuild(root)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.gameBuild = build
	if err != nil {
		a.gameUpdateError = err.Error()
	} else {
		a.gameUpdateAvailable, a.gameUpdateError = false, ""
	}
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
}

// readGameBuild is the TF2 build to show, or empty. No manifest is a server
// not installed yet; any other failure goes in the log and shows as unknown.
func (a *App) readGameBuild(installRoot string) string {
	build, err := installer.InstalledBuild(installRoot)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		a.Say("cannot read the installed TF2 build: %v", err)
	}
	return build
}
