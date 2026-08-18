// Package installer fetches and installs everything the launcher does not
// embed: SteamCMD, the TF2 dedicated server (~14 GB via steamcmd), and
// SourceMod. The plugin and ripext come from the binary's embeds.
//
// Idempotent: each step checks what is already there and skips the work. Safe
// to re-run on every start, which is how the launcher uses it.
package installer

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// AppID is the Steam AppID for the TF2 dedicated server.
const AppID = "232250"

// Status reports what the installer did, for the UI to show.
type Status struct {
	SteamcmdInstalled  bool   `json:"steamcmd_installed"`
	GameInstalled      bool   `json:"game_installed"`
	SourcemodInstalled bool   `json:"sourcemod_installed"`
	GameSizeGB         string `json:"game_size_gb,omitempty"`
	Message            string `json:"message,omitempty"`
}

// Result is the outcome of an Ensure call: whether work was done, and where
// things landed.
type Result struct {
	SteamcmdDir string
	GameDir     string
	Done        Status
}

// Ensure installs whatever is missing. It prints progress to logf as it goes.
// Cancel the context to abort a download or a steamcmd run.
func Ensure(ctx context.Context, installRoot string, logf func(format string, args ...any)) (Result, error) {
	if err := assets.RequireVersions(); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		return Result{}, fmt.Errorf("cannot create the install root %s: %w", installRoot, err)
	}
	result := Result{
		SteamcmdDir: filepath.Join(installRoot, "steamcmd"),
		GameDir:     filepath.Join(installRoot, "tf-dedicated"),
	}

	if !exists(result.SteamcmdDir) {
		logf("installing SteamCMD")
		if err := installSteamcmd(ctx, result.SteamcmdDir); err != nil {
			return result, err
		}
		result.Done.SteamcmdInstalled = true
		result.Done.Message = "SteamCMD installed"
	} else {
		result.Done.SteamcmdInstalled = true
	}

	if !gameInstalled(result.GameDir) {
		logf("installing the TF2 dedicated server (~14 GB, this is the long part)")
		if err := installGame(ctx, result.SteamcmdDir, result.GameDir, logf); err != nil {
			return result, err
		}
		result.Done.GameInstalled = true
		result.Done.Message = "TF2 dedicated server installed"
	} else {
		result.Done.GameInstalled = true
	}

	modDir := filepath.Join(result.GameDir, "tf")
	if !exists(filepath.Join(modDir, "addons", "metamod")) {
		logf("installing Metamod:Source %s", assets.MetamodVersion)
		if err := installMetamod(ctx, modDir); err != nil {
			return result, err
		}
	}

	if !exists(filepath.Join(modDir, "addons", "sourcemod")) {
		logf("installing SourceMod %s", assets.SourcemodVersion)
		if err := installSourcemod(ctx, modDir, logf); err != nil {
			return result, err
		}
	}
	result.Done.SourcemodInstalled = true

	logf("installing ripext %s, the plugin and the defender bots", assets.RipextVersion)
	if err := installRipextAndPlugin(modDir); err != nil {
		return result, err
	}
	if err := unzipTo(assets.DefenderBotsZip(), modDir); err != nil {
		return result, fmt.Errorf("cannot install the defender bots: %w", err)
	}

	return result, nil
}

// installSteamcmd downloads and unpacks the SteamCMD zip from Valve.
func installSteamcmd(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	zipData, err := fetch(ctx, "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip")
	if err != nil {
		return fmt.Errorf("cannot download SteamCMD: %w", err)
	}
	return unzipTo(zipData, dir)
}

// installGame runs steamcmd to install the TF2 dedicated server.
//
// Two things about the command, both of which SteamCMD is picky over:
//
//   - force_install_dir comes before login. Valve's own example has that
//     order, and the other way round SteamCMD can refuse the app with
//     "Failed to install app '232250' (Missing configuration)".
//   - A freshly unpacked steamcmd.exe is a bootstrapper: its first run
//     downloads the rest of itself. Doing that in the same run as app_update
//     is what leaves it without the app configuration, so the bootstrap gets
//     a run of its own.
//
// A failed install is retried once. SteamCMD drops a download often enough
// that one retry is the difference between a working install and a player
// starting over.
func installGame(ctx context.Context, steamcmdDir, gameDir string, logf func(string, ...any)) error {
	exe := filepath.Join(steamcmdDir, "steamcmd.exe")
	if !exists(exe) {
		exe = filepath.Join(steamcmdDir, "steamcmd.sh")
	}

	// The bootstrap run is expected to fail. A freshly unpacked steamcmd.exe
	// downloads the rest of itself and then exits non-zero (7 is the usual
	// one) to say it restarted. Treating that as an error stops the install
	// before it starts.
	logf("preparing SteamCMD")
	if err := runSteamcmd(ctx, exe, steamcmdDir, logf, "+quit"); err != nil {
		logf("SteamCMD updated itself (%v), carrying on", err)
	}

	args := []string{
		"+force_install_dir", gameDir,
		"+login", "anonymous",
		"+app_update", AppID, "validate",
		"+quit",
	}
	err := runSteamcmd(ctx, exe, steamcmdDir, logf, args...)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return err
	}
	logf("SteamCMD failed (%v), trying once more", err)
	if err := runSteamcmd(ctx, exe, steamcmdDir, logf, args...); err != nil {
		return fmt.Errorf("SteamCMD could not install app %s: %w. %s", AppID, err, RepairAdvice)
	}
	return nil
}

// RepairAdvice is what to tell a player whose install will not go through. It
// is one sentence because it lands in a log line and in an error.
const RepairAdvice = "If it keeps failing, open Settings and press Clear cache, " +
	"which throws away SteamCMD and the mods and fetches them again"

// Clean removes what an install can leave broken: SteamCMD itself, the mods
// this project installs, and Steam's record of what it downloaded. Ensure puts
// all three back on the next start.
//
// It keeps the two things a player cannot get back cheaply: the 14 GB of game
// files, and bridge-state, which holds the checks and the unlocks of the run.
// Removing the app manifest costs a verify pass over those game files, not
// another download.
func Clean(installRoot string) ([]string, error) {
	targets := []string{
		filepath.Join(installRoot, "steamcmd"),
		filepath.Join(installRoot, "tf-dedicated", "tf", "addons"),
		filepath.Join(installRoot, "tf-dedicated", "steamapps"),
	}
	var removed []string
	for _, target := range targets {
		if !exists(target) {
			continue
		}
		if err := os.RemoveAll(target); err != nil {
			return removed, fmt.Errorf("cannot remove %s: %w", target, err)
		}
		removed = append(removed, target)
	}
	return removed, nil
}

// runSteamcmd runs one SteamCMD command with its own directory as the working
// directory, which is where it keeps the files it downloads for itself.
func runSteamcmd(ctx context.Context, exe, dir string, logf func(string, ...any), args ...string) error {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = dir
	cmd.Stdout = lineWriter(logf)
	cmd.Stderr = lineWriter(logf)
	winproc.HideConsole(cmd)
	return cmd.Run()
}

// installMetamod unpacks Metamod:Source, which loads SourceMod. Without it
// SourceMod is inert and every plugin in this project is missing.
func installMetamod(ctx context.Context, modDir string) error {
	url := fmt.Sprintf("https://mms.alliedmods.net/mmsdrop/%s/mmsource-%s-windows.zip",
		assets.MetamodBranch, assets.MetamodVersion)
	data, err := fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("cannot download Metamod:Source: %w", err)
	}
	return unzipTo(data, modDir)
}

// installSourcemod fetches the Windows SourceMod build and unpacks it into the
// mod directory. These archives root at addons/ and cfg/, which belong under
// tf/, not next to srcds.exe.
func installSourcemod(ctx context.Context, modDir string, logf func(string, ...any)) error {
	url := fmt.Sprintf("https://sm.alliedmods.net/smdrop/%s/sourcemod-%s-windows.zip",
		assets.SourcemodBranch, assets.SourcemodVersion)
	logf("downloading SourceMod from %s", url)
	data, err := fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("cannot download SourceMod: %w", err)
	}
	return unzipTo(data, modDir)
}

// installRipextAndPlugin unpacks the embedded ripext zip and copies the plugin
// into the game's SourceMod tree.
func installRipextAndPlugin(modDir string) error {
	if err := unzipTo(assets.RipextZip(), modDir); err != nil {
		return fmt.Errorf("cannot install ripext: %w", err)
	}
	pluginDir := filepath.Join(modDir, "addons", "sourcemod", "plugins")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return err
	}
	pluginPath := filepath.Join(pluginDir, "tf2_archipelago.smx")
	return os.WriteFile(pluginPath, assets.Plugin(), 0o644)
}

// fetch downloads a URL into memory with a timeout.
func fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// unzipTo extracts a zip into dir, preserving paths. Strips no prefix: the zip
// files we use already ship addons/sourcemod/... at the root.
func unzipTo(zipData []byte, dir string) error {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("cannot read the zip: %w", err)
	}
	for _, file := range reader.File {
		target := filepath.Join(dir, filepath.FromSlash(file.Name))
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dir)) {
			return fmt.Errorf("zip entry %q escapes the install dir", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := extractFile(file, target); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(file *zip.File, target string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()
	_, err = io.Copy(dst, src)
	return err
}

// gameInstalled reports whether the TF2 server is installed, by looking for
// srcds.exe (Windows) or srcds_run (Linux) in the game dir.
func gameInstalled(gameDir string) bool {
	for _, name := range []string{"srcds.exe", "srcds_run"} {
		if exists(filepath.Join(gameDir, name)) {
			return true
		}
	}
	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// lineWriter passes each complete line to logf, so subprocess output shows up
// in the launcher's log without drowning it in partial lines.
func lineWriter(logf func(string, ...any)) io.Writer {
	return &lineSplitter{logf: logf}
}

type lineSplitter struct {
	buf  []byte
	logf func(string, ...any)
}

func (l *lineSplitter) Write(p []byte) (int, error) {
	l.buf = append(l.buf, p...)
	for {
		i := bytes.IndexByte(l.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(l.buf[:i]), "\r")
		l.buf = l.buf[i+1:]
		if line != "" {
			l.logf("  %s", line)
		}
	}
	return len(p), nil
}
