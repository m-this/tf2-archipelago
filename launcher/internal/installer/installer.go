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

	sourcemodDir := filepath.Join(result.GameDir, "tf", "addons", "sourcemod")
	if !exists(sourcemodDir) {
		logf("installing SourceMod %s", assets.SourcemodVersion)
		if err := installSourcemod(ctx, result.GameDir, logf); err != nil {
			return result, err
		}
		result.Done.SourcemodInstalled = true
	} else {
		result.Done.SourcemodInstalled = true
	}

	logf("installing ripext %s and the plugin", assets.RipextVersion)
	if err := installRipextAndPlugin(result.GameDir); err != nil {
		return result, err
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
func installGame(ctx context.Context, steamcmdDir, gameDir string, logf func(string, ...any)) error {
	exe := filepath.Join(steamcmdDir, "steamcmd.exe")
	if !exists(exe) {
		exe = filepath.Join(steamcmdDir, "steamcmd.sh")
	}
	cmd := exec.CommandContext(ctx, exe,
		"+login", "anonymous",
		"+force_install_dir", gameDir,
		"+app_update", AppID, "validate",
		"+quit",
	)
	cmd.Stdout = lineWriter(logf)
	cmd.Stderr = lineWriter(logf)
	return cmd.Run()
}

// installSourcemod fetches the Windows SourceMod build and unpacks it into the
// game tree.
func installSourcemod(ctx context.Context, gameDir string, logf func(string, ...any)) error {
	url := fmt.Sprintf("https://sm.alliedmods.net/smdrop/%s/sourcemod-%s-windows.zip",
		assets.SourcemodBranch, assets.SourcemodVersion)
	logf("downloading SourceMod from %s", url)
	data, err := fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("cannot download SourceMod: %w", err)
	}
	return unzipTo(data, gameDir)
}

// installRipextAndPlugin unpacks the embedded ripext zip and copies the plugin
// into the game's SourceMod tree.
func installRipextAndPlugin(gameDir string) error {
	if err := unzipTo(assets.RipextZip(), gameDir); err != nil {
		return fmt.Errorf("cannot install ripext: %w", err)
	}
	pluginDir := filepath.Join(gameDir, "tf", "addons", "sourcemod", "plugins")
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
