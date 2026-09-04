package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const (
	// FormatVersion is the shape of the state file this binary writes.
	// 4 says Played is written. A version 3 file can be from either side of
	// the day Played arrived, and the goal cannot tell a run that played
	// nothing from one written before the list existed.
	FormatVersion = 4

	// FormatVersionMin is the oldest shape it can still read. A file older than
	// that is an error rather than a guess: it holds the only record of what a
	// run has already checked. Version 1 had no acked_seq, and zero is the
	// right answer for it: nothing had been acknowledged.
	FormatVersionMin = 1

	// archivesMax bounds the copies kept of one seed. Past it the oldest name
	// is reused, because failing to bind a seed would leave the bridge retrying
	// forever.
	archivesMax = 10
)

// snapshot is what sits on disk: only facts the server or the plugin told us.
type snapshot struct {
	FormatVersion int `json:"format_version"`

	// Seed is the Archipelago room's seed name. A different one means the held
	// checks and items belong to a run that no longer exists.
	Seed string `json:"seed"`

	Checks []int64 `json:"checks"`

	/* Played is the subset of Checks this server did itself.
	 *
	 * Checks is what the multiworld says this slot has checked, and another
	 * player running !collect checks every location holding their items,
	 * including a mission's clear. Adopting that is right for the run's
	 * bookkeeping and wrong for the win: a play-tester was told their run was
	 * complete having beaten three of five missions.
	 *
	 * So the goal reads this list, which only the plugin writes.
	 */
	Played []int64 `json:"played"`

	// Items are received item ids in the order Archipelago sent them; the index
	// into this list is the index it deduplicates on.
	Items []int64 `json:"items"`

	// AckedSeq is how far through Items the plugin has confirmed applying
	// one-shot grants. An effect at or below it is never sent again. Without
	// it, a plugin reload asks from sequence zero and every cash bundle in the
	// run is paid a second time.
	AckedSeq int `json:"acked_seq"`

	GoalSent bool `json:"goal_sent"`

	/* Resume is where the team had got to, so a restart does not cost them the
	 * mission.
	 *
	 * A crash takes the game server's own state with it, and the mission it
	 * comes back on is whatever the settings name, at wave one. The checks
	 * survive because they are here; the mission does not, because nothing
	 * wrote it down. Reported from play as having the whole mission to do
	 * again.
	 */
	Resume Resume `json:"resume,omitzero"`
}

// Resume is the mission the team was playing and the last wave they cleared in
// it. Wave zero means they had cleared none, so there is nothing to skip.
type Resume struct {
	PopFile string `json:"popfile,omitempty"`
	Wave    int    `json:"wave,omitempty"`
}

// readSnapshot loads the state file and reports the format version it was
// written in, so the caller can keep a copy before promoting it.
func readSnapshot(path string) (snapshot, int, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot{FormatVersion: FormatVersion}, FormatVersion, nil
	}
	if err != nil {
		return snapshot{}, 0, err
	}
	var loaded snapshot
	if err := json.Unmarshal(body, &loaded); err != nil {
		return snapshot{}, 0, fmt.Errorf(
			"%s cannot be read, and it is the only record of this run's checks: %w. "+
				"Move the file aside to start the run again", path, err,
		)
	}
	if loaded.FormatVersion < FormatVersionMin || loaded.FormatVersion > FormatVersion {
		return snapshot{}, 0, fmt.Errorf(
			"%s is format version %d, this bridge reads %d to %d",
			path, loaded.FormatVersion, FormatVersionMin, FormatVersion,
		)
	}
	// Read as an old shape, written back as the current one. Every field added
	// since is zero, which is what an older file means by leaving it out.
	wasVersion := loaded.FormatVersion
	loaded.FormatVersion = FormatVersion
	/* A file written before Played existed cannot say which of its checks the
	   server made, and the run in it was played by somebody. Taking them all is
	   the reading that does not ask a team to replay an evening; the distinction
	   starts holding from here on.

	   Version 3 files exist on both sides of that day, because the list arrived
	   without a bump. One from before has no played key at all; one from after
	   always writes it, empty or not. kelly-cs's run crossed the upgrade with
	   every mission cleared reading as collected and the goal counting none of
	   them (gh-16). */
	if wasVersion < 4 && loaded.Played == nil {
		loaded.Played = slices.Clone(loaded.Checks)
	}
	return loaded, wasVersion, nil
}

// archiveFormat sets the file aside under the version it was written in, before
// anything rewrites it in a shape the binary that made it cannot read.
func archiveFormat(path string, version int) error {
	if free(path) {
		return nil
	}
	extension := filepath.Ext(path)
	target := fmt.Sprintf("%s.v%d%s", strings.TrimSuffix(path, extension), version, extension)
	if !free(target) {
		// The copy at that version is already there, and it is older than what
		// is about to be written. Keeping the first one is the safe direction.
		return nil
	}
	if err := copyFile(path, target); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func copyFile(source, target string) error {
	body, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.WriteFile(target, body, 0o644); err != nil {
		return err
	}
	handle, err := os.Open(target)
	if err != nil {
		return err
	}
	defer func() { _ = handle.Close() }()
	return handle.Sync()
}

// writeSnapshot replaces the state file atomically: a torn write would lose
// checks already acknowledged to the plugin.
func writeSnapshot(path string, data snapshot) error {
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".bridge-*.json")
	if err != nil {
		return err
	}
	// On the happy path the rename has already taken the file.
	defer func() { _ = os.Remove(temp.Name()) }()

	if _, err := temp.Write(append(body, '\n')); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return err
	}
	return syncDir(dir)
}

// archiveSnapshot moves the state file aside rather than letting a new seed
// overwrite it. Binding a seed wipes the run, and the operator who pointed the
// bridge at the wrong Archipelago server needs the old file to still exist.
func archiveSnapshot(path, seed string) (string, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	target := archivePath(path, seed)
	if err := os.Rename(path, target); err != nil {
		return "", err
	}
	return target, syncDir(filepath.Dir(path))
}

// archivePath is the state file's own name with the seed in it, plus a counter
// when that seed has been archived before.
//
// Only "the file is not there" frees a name. Anything else, a permissions
// problem or a failing disk, leaves the name taken: reading it as free is how
// an archive overwrites a run it cannot see.
func archivePath(path, seed string) string {
	extension := filepath.Ext(path)
	stem := strings.TrimSuffix(path, extension)
	base := fmt.Sprintf("%s.%s%s", stem, safeSeedName(seed), extension)
	if free(base) {
		return base
	}
	for attempt := 1; attempt < archivesMax; attempt++ {
		candidate := fmt.Sprintf("%s.%s.%d%s", stem, safeSeedName(seed), attempt, extension)
		if free(candidate) {
			return candidate
		}
	}
	return base
}

func free(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, os.ErrNotExist)
}

// safeSeedName keeps a seed name usable as a file name. Archipelago's are
// alphanumeric today, and a bridge that crashed on a path separator in one
// would be a poor trade.
func safeSeedName(seed string) string {
	const lengthMax = 64
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, seed)
	if safe == "" {
		return "unnamed"
	}
	if len(safe) > lengthMax {
		return safe[:lengthMax]
	}
	return safe
}

// syncDir flushes a directory, so a rename inside it survives a crash rather than losing its name.
//
// That is a Unix idea. Windows has no directory handle to fsync: opening a
// directory for it fails with "Access denied", and every persist after that
// reported an error it did not have. NTFS makes the rename durable on its own,
// so the flush is skipped there rather than failed.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = handle.Close() }()
	return handle.Sync()
}
