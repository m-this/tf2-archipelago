// Package stockpop makes Caliginous Caper's mission from the game's own files.
//
// Valve ships the wave as mvm_ghost_town_666.pop inside tf2_misc_dir.vpk. It
// names a tank path Ghost Town does not have and lacks the setting that spawns
// the Halloween robots, so the plugin plays a repaired copy under its own name
// rather than shipping Valve's file.
package stockpop

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// RuntimeName is the file the plugin checks for and loads.
const RuntimeName = "mvm_ghost_town_ap_caliginous_caper.pop"

// Destination is where the plugin looks for the mission in a tf directory.
func Destination(tfDir string) string {
	return filepath.Join(tfDir, "scripts", "population", RuntimeName)
}

// Install reads the stock mission out of tfDir's VPK, repairs it, and writes
// it to destination. A destination that already holds those bytes is left
// alone and reported unchanged. On any error the destination is untouched.
func Install(tfDir, destination string) (changed bool, err error) {
	body, err := extract(tfDir)
	if err != nil {
		return false, err
	}
	body, err = repair(body)
	if err != nil {
		return false, err
	}
	if old, err := os.ReadFile(destination); err == nil && bytes.Equal(old, body) {
		return false, nil
	}
	if err := writeAtomic(destination, body); err != nil {
		return false, fmt.Errorf("writing %s: %w", destination, err)
	}
	return true, nil
}

func writeAtomic(destination string, body []byte) error {
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".caliginous-*.pop")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // gone already after a successful rename
	if _, err := tmp.Write(body); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Chmod(0o644); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), destination)
}

var (
	tankPathStock    = []byte("boss_path_a1")
	tankPathMap      = []byte("boss_path_1")
	eventSetting     = []byte("EventPopfile Halloween")
	eventAfterHeader = []byte("CanBotsAttackWhileInSpawnRoom no")
)

func repair(body []byte) ([]byte, error) {
	// Ghost Town's BSP has boss_path_1 and none of the _666 file's four
	// boss_path_a1 nodes. A tank that cannot find its first path node cannot
	// start the later groups that wait for wave07 to die.
	if !bytes.Contains(body, tankPathStock) && !bytes.Contains(body, tankPathMap) {
		return nil, errors.New("the _666 mission has no recognized tank path")
	}
	body = bytes.ReplaceAll(body, tankPathStock, tankPathMap)

	// Without the event setting, TF2 spawns the regular robot models.
	if bytes.Contains(body, eventSetting) {
		return body, nil
	}
	at := bytes.Index(body, eventAfterHeader)
	if at < 0 {
		return nil, errors.New("the _666 mission has no expected WaveSchedule header")
	}
	lineEnd := bytes.IndexByte(body[at+len(eventAfterHeader):], '\n')
	if lineEnd < 0 {
		return nil, errors.New("the _666 mission has no header line ending")
	}
	insertAt := at + len(eventAfterHeader) + lineEnd + 1
	newline := "\n"
	if body[insertAt-2] == '\r' {
		newline = "\r\n"
	}
	line := []byte("\t" + string(eventSetting) + newline)
	return slices.Insert(body, insertAt, line...), nil
}
