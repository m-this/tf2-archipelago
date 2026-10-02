package installer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/stockpop"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// gameUpdateTimeout bounds the update on Start, recovery included. A routine
// update is a few hundred MB; this leaves room for a full download on a slow
// link, and no more.
var gameUpdateTimeout = 2 * time.Hour

// errUpdateStuck is SteamCMD's "state is 0x6 after update job". The manifest
// keeps that state, and every later app_update stops on it, validate included.
var errUpdateStuck = errors.New("SteamCMD left app " + AppID + " at state 0x6")

var stuckUpdatePattern = regexp.MustCompile(`state is 0x6\b`)

// ErrGameNotUpdated is an update that did not happen while the installed build
// still works: Steam out of reach, a 0x602, the time limit. A start goes on
// with the installed build; an update the player asked for reports it.
var ErrGameNotUpdated = errors.New("the TF2 update did not finish")

func notUpdated(cause error) error { return fmt.Errorf("%w (%w)", ErrGameNotUpdated, cause) }

// StartAnyway is what a start does with UpdateGame's answer. ErrGameNotUpdated
// is logged and dropped, so the installed build starts; any other error stays.
func StartAnyway(err error, logf func(string, ...any)) error {
	if !errors.Is(err, ErrGameNotUpdated) {
		return err
	}
	logf("%v. Starting the installed build: players on a newer TF2 cannot join until it updates", err)
	return nil
}

func stuckUpdate(line string) bool { return stuckUpdatePattern.MatchString(line) }

func steamcmdPath(installRoot string) string { return filepath.Join(installRoot, "steamcmd") }

func gamePath(installRoot string) string { return filepath.Join(installRoot, "tf-dedicated") }

// UpdateGame brings the installed TF2 server to Steam's current build. Run it
// after Ensure and before every start: a server one build behind refuses every
// client that has updated. It then remakes Caliginous Caper's mission from the
// build that will run, updated or not.
func UpdateGame(ctx context.Context, installRoot string, logf func(string, ...any)) error {
	gameDir := gamePath(installRoot)
	if !gameInstalled(gameDir) {
		return fmt.Errorf("the TF2 dedicated server is not installed in %s", gameDir)
	}
	err := updateGame(ctx, steamcmdPath(installRoot), gameDir, logf)
	if ctx.Err() == nil {
		installStockMission(filepath.Join(gameDir, "tf"), logf)
	}
	return err
}

// installStockMission writes the file the plugin plays Caliginous Caper from.
// Without it the plugin refuses that one mission and the rest of the run still
// plays, so a failure is a warning and never stops a start.
func installStockMission(modDir string, logf func(string, ...any)) {
	changed, err := stockpop.Install(modDir, stockpop.Destination(modDir))
	switch {
	case err != nil:
		logf("Caliginous Caper cannot be played on this server: its mission could not be made from the game's files (%v)", err)
	case changed:
		logf("made Caliginous Caper's mission from the game's files")
	}
}

// updateGame runs app_update on an installed server, without validate, which
// would reread 14 GB on every start.
//
// A failed update is retried once, after the same warm-up an install gets.
// When SteamCMD reported state 0x6, the app manifest is moved aside first,
// which is what got two stuck servers updating again by hand. A second 0x6 is
// an error. Any other failure, Steam out of reach say, is ErrGameNotUpdated:
// the installed build still starts, which beats no server at all.
func updateGame(ctx context.Context, steamcmdDir, gameDir string, logf func(string, ...any)) error {
	exe := firstExisting(steamcmdDir, steamcmdNames())
	if exe == "" {
		return fmt.Errorf("SteamCMD is not in %s", steamcmdDir)
	}
	updateCtx, cancel := context.WithTimeout(ctx, gameUpdateTimeout)
	defer cancel()
	args := appUpdateArgs(winproc.ShortPath(gameDir), false)

	logf("updating the TF2 dedicated server, if Steam has a newer build (after a TF2 update this takes a while)")
	err := runSteamcmd(updateCtx, exe, steamcmdDir, logf, args...)
	if err == nil || ctx.Err() != nil {
		return err
	}
	if updateCtx.Err() != nil {
		return notUpdated(fmt.Errorf("it took longer than %v", gameUpdateTimeout))
	}

	stuck := errors.Is(err, errUpdateStuck)
	manifest := manifestPath(gameDir)
	aside := manifest + ".0x6-" + time.Now().Format("2006-01-02T150405")
	if stuck {
		switch err := os.Rename(manifest, aside); {
		case errors.Is(err, fs.ErrNotExist):
			return notUpdated(fmt.Errorf("%w, and %s is not there to set aside", errUpdateStuck, manifest))
		case err != nil:
			return fmt.Errorf("SteamCMD reported state 0x6 and %s could not be set aside: %w", manifest, err)
		}
		logf("SteamCMD left the update stuck at state 0x6, moved its manifest aside to %s", aside)
	}
	logf("the TF2 update failed (%v), trying once more", err)
	warmUpSteamcmd(updateCtx, exe, steamcmdDir, logf)
	err = runSteamcmd(updateCtx, exe, steamcmdDir, logf, args...)
	switch {
	case err == nil || ctx.Err() != nil:
		return err
	case stuck:
		return fmt.Errorf("the TF2 update is stuck: SteamCMD reported state 0x6, "+
			"so %s was moved to %s, SteamCMD was warmed up and app_update ran again, which failed: %w. %s",
			manifest, aside, err, RepairAdvice)
	}
	return notUpdated(err)
}
