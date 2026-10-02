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

	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// gameUpdateTimeout bounds the update on Start, recovery included. A routine
// update is a few hundred MB; this leaves room for a full download on a slow
// link, and no more.
const gameUpdateTimeout = 2 * time.Hour

// errUpdateStuck is SteamCMD's "state is 0x6 after update job". The manifest
// keeps that state, and every later app_update stops on it, validate included.
var errUpdateStuck = errors.New("SteamCMD left app " + AppID + " at state 0x6")

var stuckUpdatePattern = regexp.MustCompile(`state is 0x6\b`)

func stuckUpdate(line string) bool { return stuckUpdatePattern.MatchString(line) }

func steamcmdPath(installRoot string) string { return filepath.Join(installRoot, "steamcmd") }

func gamePath(installRoot string) string { return filepath.Join(installRoot, "tf-dedicated") }

// UpdateGame brings the installed TF2 server to Steam's current build. Run it
// after Ensure and before every start: a server one build behind refuses every
// client that has updated.
func UpdateGame(ctx context.Context, installRoot string, logf func(string, ...any)) error {
	gameDir := gamePath(installRoot)
	if !gameInstalled(gameDir) {
		return fmt.Errorf("the TF2 dedicated server is not installed in %s", gameDir)
	}
	return updateGame(ctx, steamcmdPath(installRoot), gameDir, logf)
}

// updateGame runs app_update on an installed server, without validate, which
// would reread 14 GB on every start.
//
// A failed update is retried once, after the same warm-up an install gets.
// When SteamCMD reported state 0x6, the app manifest is moved aside first,
// which is what got two stuck servers updating again by hand. A second 0x6 is
// an error. Any other failure, Steam out of reach say, starts the build that
// is installed rather than no server at all.
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
		logf("the TF2 update did not finish in %v, starting the installed build", gameUpdateTimeout)
		return nil
	}

	stuck := errors.Is(err, errUpdateStuck)
	manifest := filepath.Join(gameDir, "steamapps", "appmanifest_"+AppID+".acf")
	aside := manifest + ".0x6-" + time.Now().Format("2006-01-02T150405")
	if stuck {
		switch err := os.Rename(manifest, aside); {
		case errors.Is(err, fs.ErrNotExist):
			logUpdateFailed(logf, fmt.Errorf("%w, and %s is not there to set aside", errUpdateStuck, manifest))
			return nil
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
	logUpdateFailed(logf, err)
	return nil
}

func logUpdateFailed(logf func(string, ...any), err error) {
	logf("the TF2 update did not finish (%v). Starting the installed build: "+
		"players on a newer TF2 cannot join until it updates", err)
}
