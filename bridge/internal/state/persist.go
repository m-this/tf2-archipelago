package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FormatVersion is the shape of the state file. A file from a version this
// binary does not know is an error rather than a guess: it holds the only
// record of what a run has already checked.
const FormatVersion = 1

// snapshot is what sits on disk: only facts the server or the plugin told us.
type snapshot struct {
	FormatVersion int `json:"format_version"`

	// Seed is the Archipelago room's seed name. A different one means the held
	// checks and items belong to a run that no longer exists.
	Seed string `json:"seed"`

	Checks []int64 `json:"checks"`

	// Items are received item ids in the order Archipelago sent them; the index
	// into this list is the index it deduplicates on.
	Items []int64 `json:"items"`

	GoalSent bool `json:"goal_sent"`
}

func readSnapshot(path string) (snapshot, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot{FormatVersion: FormatVersion}, nil
	}
	if err != nil {
		return snapshot{}, err
	}
	var loaded snapshot
	if err := json.Unmarshal(body, &loaded); err != nil {
		return snapshot{}, fmt.Errorf(
			"%s cannot be read, and it is the only record of this run's checks: %w. "+
				"Move the file aside to start the run again", path, err,
		)
	}
	if loaded.FormatVersion != FormatVersion {
		return snapshot{}, fmt.Errorf(
			"%s is format version %d, this bridge reads %d",
			path, loaded.FormatVersion, FormatVersion,
		)
	}
	return loaded, nil
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

// syncDir makes the rename durable: without it the file survives a crash but its name may not.
func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = handle.Close() }()
	return handle.Sync()
}
