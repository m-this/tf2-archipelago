package installer

import (
	"context"
	"errors"
	"fmt"
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

// ensureGame installs the TF2 server when it is missing and updates it
// otherwise: a server one build behind refuses every client that has updated.
// It reports whether it installed.
func ensureGame(ctx context.Context, installRoot, steamcmdDir, gameDir string, logf func(string, ...any)) (bool, error) {
	if gameInstalled(gameDir) {
		return false, updateGame(ctx, steamcmdDir, gameDir, logf)
	}
	if free, ok := winproc.FreeBytes(installRoot); ok && free < gameBytesNeeded {
		return false, fmt.Errorf(
			"the game server needs about %d GB and %s has %d GB free",
			gameBytesNeeded/gigabyte, installRoot, free/gigabyte)
	}
	logf("installing the TF2 dedicated server (~14 GB, this is the long part)")
	if err := installGame(ctx, steamcmdDir, gameDir, logf); err != nil {
		return false, err
	}
	return true, nil
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
		if err := os.Rename(manifest, aside); err != nil {
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
	logf("the TF2 update did not finish (%v). Starting the installed build: "+
		"players on a newer TF2 cannot join until it updates", err)
	return nil
}
