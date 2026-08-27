// Package generate makes a seed with the official Archipelago app.
//
// The generator is Python and stays with that app; the launcher does not bundle
// it. What the launcher does is the part a player gets wrong: find the app,
// put the apworld where the app looks for worlds, write the player file into a
// folder of its own, run the generator, and say where the archive landed.
package generate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// ErrNotInstalled says no Archipelago app was found.
var ErrNotInstalled = errors.New("the Archipelago app is not installed, or not where the launcher looked")

// Options is what one generation needs.
type Options struct {
	Settings settings.Settings
	// AppDir is where the Archipelago app is, or empty to look in the places
	// its installer uses.
	AppDir string
	// Apworld is the world file to install into the app, or empty to leave
	// whatever the app has.
	Apworld []byte
	// ArchipelagoVersion goes into the player file's requires block.
	ArchipelagoVersion string
	// Log takes one line at a time.
	Log func(string)
	// Timeout bounds the generator. A seed takes seconds; a hang should not
	// take the launcher with it.
	Timeout time.Duration
}

// Result is what a generation left behind.
type Result struct {
	// Archive is the AP_*.zip the room is created from.
	Archive string
	// AppDir is where the app was found.
	AppDir string
}

// Run generates a seed and returns where the archive is.
func Run(ctx context.Context, options Options) (Result, error) {
	if _, err := settings.CheckRunSelection(options.Settings); err != nil {
		return Result{}, err
	}
	logf := options.Log
	if logf == nil {
		logf = func(string) {}
	}
	if options.Timeout <= 0 {
		options.Timeout = 3 * time.Minute
	}

	appDir, err := FindApp(options.AppDir)
	if err != nil {
		return Result{}, err
	}
	logf("Archipelago app: " + appDir)

	if err := installApworld(appDir, options.Apworld, logf); err != nil {
		return Result{}, err
	}
	players, output, err := prepareFolders(options)
	if err != nil {
		return Result{}, err
	}

	exe := generatorPath(appDir)
	if !exists(exe) {
		return Result{}, fmt.Errorf("%s has no generator (%s)", appDir, filepath.Base(exe))
	}
	logf("running " + filepath.Base(exe))

	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	program, args := generatorCommand(exe)
	args = append(args,
		"--player_files_path", winproc.ShortPath(players),
		"--outputpath", winproc.ShortPath(output),
	)
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = appDir
	cmd.Stdout = lineWriter(logf)
	cmd.Stderr = lineWriter(logf)
	winproc.HideConsole(cmd)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return Result{}, fmt.Errorf("the generator did not finish in %s", options.Timeout)
		}
		return Result{}, fmt.Errorf("the generator failed: %w. Read the lines above for the reason", err)
	}

	archive, err := newestArchive(output)
	if err != nil {
		return Result{}, err
	}
	logf("seed written to " + archive)
	return Result{Archive: archive, AppDir: appDir}, nil
}

// FindApp returns the app's directory: the one given, if it holds a generator,
// else the first of the places the installer uses that does.
//
// The given path may name the app's own exe rather than its folder, because
// that is what a file picker hands back and what a player copies out of a
// shortcut.
func FindApp(given string) (string, error) {
	for _, dir := range SearchPath(given) {
		if exists(generatorPath(dir)) {
			return dir, nil
		}
	}
	return "", ErrNotInstalled
}

// SearchPath is every directory FindApp looks in, in order. The settings
// dialog prints it, because "not where the launcher looked" is only useful
// with the list beside it.
func SearchPath(given string) []string {
	dirs := candidateDirs()
	if given == "" {
		return dirs
	}
	return append([]string{appDirOf(given)}, dirs...)
}

// appDirOf takes the folder out of a path that names a file. A folder that is
// not there yet is passed through: the caller reports it as missing.
func appDirOf(given string) string {
	info, err := os.Stat(given)
	if err == nil && !info.IsDir() {
		return filepath.Dir(given)
	}
	return given
}

// installApworld puts the world file where the app looks for worlds. Nothing
// to install leaves whatever the app has.
func installApworld(appDir string, apworld []byte, logf func(string)) error {
	if len(apworld) == 0 {
		return nil
	}
	worldsDir := filepath.Join(appDir, "custom_worlds")
	if err := os.MkdirAll(worldsDir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", worldsDir, err)
	}
	target := filepath.Join(worldsDir, "tf2_mvm.apworld")
	if err := os.WriteFile(target, apworld, 0o644); err != nil {
		return fmt.Errorf("cannot install the apworld: %w", err)
	}
	logf("installed the apworld into " + target)
	return nil
}

// prepareFolders makes a fresh players folder holding the player file, and a
// fresh output folder, both under the install root. Folders of their own, so
// the archive that comes back is this run's and not the oldest one in the
// app's own output folder.
func prepareFolders(options Options) (players, output string, err error) {
	work := filepath.Join(options.Settings.InstallRoot, "generate")
	players = filepath.Join(work, "players")
	output = filepath.Join(work, "output")
	for _, dir := range []string{players, output} {
		if err := os.RemoveAll(dir); err != nil {
			return "", "", err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", err
		}
	}
	yaml := settings.PlayerYAML(options.Settings, options.ArchipelagoVersion)
	if err := os.WriteFile(filepath.Join(players, "tf2.yaml"), []byte(yaml), 0o644); err != nil {
		return "", "", err
	}
	return players, output, nil
}

func newestArchive(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "AP_") && strings.HasSuffix(name, ".zip") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "", errors.New("the generator produced no archive. Read the lines above for the reason")
	}
	sort.Strings(names)
	return filepath.Join(dir, names[len(names)-1]), nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// lineWriter passes each complete line to logf.
func lineWriter(logf func(string)) *lineSplitter { return &lineSplitter{logf: logf} }

type lineSplitter struct {
	buf  []byte
	logf func(string)
}

func (l *lineSplitter) Write(p []byte) (int, error) {
	l.buf = append(l.buf, p...)
	for {
		i := bytes.IndexByte(l.buf, '\n')
		if i < 0 {
			break
		}
		line := string(bytes.TrimRight(l.buf[:i], "\r"))
		l.buf = l.buf[i+1:]
		if line != "" {
			l.logf("  " + line)
		}
	}
	return len(p), nil
}
