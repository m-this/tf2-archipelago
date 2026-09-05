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
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// AppID is the Steam AppID for the TF2 dedicated server.
const AppID = "232250"

// gameBytesNeeded is the room the game files take, plus the room SteamCMD
// needs while it unpacks them. Checked before the download rather than after,
// where SteamCMD reports it as "state is 0x202" and nothing else.
const (
	gigabyte                  = 1_000_000_000
	gameBytesNeeded           = 20 * gigabyte
	communityProgressInterval = 250_000_000
	communityDownloadTimeout  = 2 * time.Hour
)

// communityArchiveURLs are the full Potato asset packs. The "no maps"
// variants are intentionally not accepted: the launcher's community catalog
// needs the BSP and NAV files from these archives as well as their shared
// assets.
var communityArchiveURLs = map[string]string{
	"archive-assets.zip":   "https://dlarchive.potato.tf/archive-assets.zip",
	"mlarchive-assets.zip": "https://dlml.potato.tf/mlarchive-assets.zip",
}

var communityHTTPClient = &http.Client{Timeout: communityDownloadTimeout}

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
func Ensure(ctx context.Context, installRoot string, communityArchives []string, logf func(format string, args ...any)) (Result, error) {
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
		if free, ok := winproc.FreeBytes(installRoot); ok && free < gameBytesNeeded {
			return result, fmt.Errorf(
				"the game server needs about %d GB and %s has %d GB free",
				gameBytesNeeded/gigabyte, installRoot, free/gigabyte)
		}
		logf("installing the TF2 dedicated server (~14 GB, this is the long part)")
		if err := installGame(ctx, result.SteamcmdDir, result.GameDir, logf); err != nil {
			return result, err
		}
		result.Done.GameInstalled = true
		result.Done.Message = "TF2 dedicated server installed"
	} else {
		result.Done.GameInstalled = true
	}

	// After the game, because it links what SteamCMD downloaded into where the
	// game server looks. Every start, because it is a link and cheap, and a
	// game directory moved between disks leaves a stale one.
	if err := linkSteamClient(installRoot, result.SteamcmdDir); err != nil {
		return result, err
	}

	if err := installMods(ctx, filepath.Join(result.GameDir, "tf"), logf); err != nil {
		return result, err
	}
	if err := installCommunityArchives(communityArchives, filepath.Join(result.GameDir, "tf"), logf); err != nil {
		return result, err
	}
	result.Done.SourcemodInstalled = true

	return result, nil
}

func installCommunityArchives(archives []string, modDir string, logf func(string, ...any)) error {
	if err := ValidateCommunityArchives(archives, logf); err != nil {
		return err
	}
	removed, err := removeUnsupportedCommunityPopfiles(modDir)
	if err != nil {
		return err
	}
	if removed > 0 {
		logf("removed %d unsupported community mission popfile(s)", removed)
	}
	stampDir := filepath.Join(modDir, ".tf2ap-community")
	for _, path := range archives {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("cannot use community pack %s: %w", path, err)
		}
		identity := fmt.Sprintf("%s\n%d\n%d\n", path, info.Size(), info.ModTime().UnixNano())
		sum := sha256.Sum256([]byte(path))
		stamp := filepath.Join(stampDir, fmt.Sprintf("%x.stamp", sum[:8]))
		if body, err := os.ReadFile(stamp); err == nil && string(body) == identity {
			logf("community pack already installed: %s", filepath.Base(path))
			continue
		}
		logf("installing community pack %s (this can take a minute)", filepath.Base(path))
		if err := installCommunityZip(path, modDir); err != nil {
			return fmt.Errorf("cannot install community pack %s: %w", path, err)
		}
		if err := os.MkdirAll(stampDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(stamp, []byte(identity), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// DownloadCommunityArchives is the only path that fetches community content.
// It downloads recognized full-with-maps Potato packs that are not already
// cached, then validates every selected ZIP. Start deliberately does not call
// this function: network downloads require an explicit UI action.
func DownloadCommunityArchives(ctx context.Context, archives []string, logf func(string, ...any)) error {
	for _, path := range archives {
		_, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			if err := downloadCommunityArchive(ctx, path, logf); err != nil {
				return fmt.Errorf("cannot download community pack %s: %w", filepath.Base(path), err)
			}
		} else if err != nil {
			return fmt.Errorf("cannot use community pack %s: %w", path, err)
		}
	}
	return ValidateCommunityArchives(archives, logf)
}

// ValidateCommunityArchives accepts only existing, readable ZIPs and performs
// no network access. Ensure uses this before installing selected local packs.
func ValidateCommunityArchives(archives []string, logf func(string, ...any)) error {
	for _, path := range archives {
		if err := validateCommunityArchive(path); err != nil {
			return err
		}
		logf("community pack ready: %s", filepath.Base(path))
	}
	return nil
}

// AvailableCommunityArchives returns the valid files from archives. It is the
// launchers' source of truth for which community mission rows may be shown.
func AvailableCommunityArchives(archives []string) []string {
	available := make([]string, 0, len(archives))
	for _, path := range archives {
		if validateCommunityArchive(path) == nil {
			available = append(available, path)
		}
	}
	return available
}

func validateCommunityArchive(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("community pack %s is not a valid ZIP: %w", path, err)
	}
	if err := reader.Close(); err != nil {
		return fmt.Errorf("cannot close community pack %s: %w", path, err)
	}
	return nil
}

// downloadCommunityArchive caches a recognized full asset pack next to the
// locally supplied packs. A partial response is written under a temporary
// name and validated before rename, so cancelling the explicit download can
// never leave a corrupt ZIP that the next attempt mistakes for a completed
// download.
func downloadCommunityArchive(ctx context.Context, path string, logf func(string, ...any)) error {
	name := filepath.Base(path)
	url, known := communityArchiveURLs[name]
	if !known {
		return fmt.Errorf("%s is not a recognized Potato asset pack", name)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	logf("downloading %s from %s (the full pack includes maps)", name, url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "tf2-archipelago-launcher")
	resp, err := communityHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+name+"-*.partial")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()
	progress := &communityDownloadWriter{
		Writer: tmp,
		name:   name,
		total:  resp.ContentLength,
		next:   communityProgressInterval,
		logf:   logf,
	}
	written, copyErr := io.Copy(progress, resp.Body)
	if closeErr := tmp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return copyErr
	}
	if resp.ContentLength >= 0 && written != resp.ContentLength {
		return fmt.Errorf("downloaded %d bytes, expected %d", written, resp.ContentLength)
	}
	reader, err := zip.OpenReader(tmpPath)
	if err != nil {
		return fmt.Errorf("download is not a valid ZIP: %w", err)
	}
	if err := reader.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	keep = true
	logf("downloaded %s (%.1f GB) into %s", name, float64(written)/float64(gigabyte), filepath.Dir(path))
	return nil
}

type communityDownloadWriter struct {
	io.Writer
	name    string
	total   int64
	written int64
	next    int64
	logf    func(string, ...any)
}

func (w *communityDownloadWriter) Write(body []byte) (int, error) {
	n, err := w.Writer.Write(body)
	w.written += int64(n)
	if w.written >= w.next {
		if w.total > 0 {
			w.logf("downloading %s: %.0f%% (%.1f GB)", w.name, 100*float64(w.written)/float64(w.total), float64(w.written)/float64(gigabyte))
		} else {
			w.logf("downloading %s: %.1f GB", w.name, float64(w.written)/float64(gigabyte))
		}
		for w.next <= w.written {
			w.next += communityProgressInterval
		}
	}
	return n, err
}

// installCommunityZip streams a Potato-style tf/download tree into SRCDS's
// tf directory. The archives are several gigabytes, so they are never read
// into memory as the small embedded mod archives are.
func installCommunityZip(path, modDir string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return err
	}
	for _, entry := range reader.File {
		name := filepath.ToSlash(entry.Name)
		var relative string
		switch {
		case strings.HasPrefix(name, "tf/download/"):
			relative = strings.TrimPrefix(name, "tf/download/")
		case strings.HasPrefix(name, "tf/"):
			relative = strings.TrimPrefix(name, "tf/")
		default:
			continue
		}
		if relative == "" {
			continue
		}
		// Full Potato packs contain missions for server mods we do not ship.
		// Keep their maps and shared assets, but do not let stock TF2 discover
		// and select an incompatible mission as that map's default.
		if unsupportedCommunityPopfile(relative) {
			continue
		}
		target, err := safeJoin(modDir, relative)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := extractFile(entry, target); err != nil {
			return err
		}
	}
	return nil
}

var supportedCommunityPopfiles, communityMapNames = communityPopfilePolicy()

func communityPopfilePolicy() (map[string]struct{}, []string) {
	supported := make(map[string]struct{})
	for _, mission := range gamedata.PlayableMissions() {
		if gamedata.IsCommunityMission(mission.ID) {
			supported[strings.ToLower(mission.PopFile)] = struct{}{}
		}
	}
	maps := make([]string, 0)
	for _, played := range gamedata.Maps {
		if gamedata.IsCommunityMap(played.ID) {
			maps = append(maps, strings.ToLower(played.Name))
		}
	}
	return supported, maps
}

// unsupportedCommunityPopfile recognizes only mission files belonging to
// community maps in our catalog. It deliberately leaves robot templates and
// population files for unrelated user-installed maps alone.
func unsupportedCommunityPopfile(relative string) bool {
	clean := strings.ToLower(filepath.ToSlash(relative))
	if filepath.ToSlash(filepath.Dir(clean)) != "scripts/population" || filepath.Ext(clean) != ".pop" {
		return false
	}
	name := strings.TrimSuffix(filepath.Base(clean), ".pop")
	for _, mapName := range communityMapNames {
		if name != mapName && !strings.HasPrefix(name, mapName+"_") {
			continue
		}
		_, supported := supportedCommunityPopfiles[name]
		return !supported
	}
	return false
}

// removeUnsupportedCommunityPopfiles repairs installations made by older
// launchers even when the ZIP stamp says the pack is already installed.
func removeUnsupportedCommunityPopfiles(modDir string) (int, error) {
	populationDir := filepath.Join(modDir, "scripts", "population")
	entries, err := os.ReadDir(populationDir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("cannot inspect installed community missions: %w", err)
	}
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || !unsupportedCommunityPopfile(filepath.Join("scripts", "population", entry.Name())) {
			continue
		}
		if err := os.Remove(filepath.Join(populationDir, entry.Name())); err != nil {
			return removed, fmt.Errorf("cannot remove unsupported community mission %s: %w", entry.Name(), err)
		}
		removed++
	}
	return removed, nil
}

// installMods puts everything that loads inside the game server into tf/:
// Metamod, SourceMod, ripext, this project's plugin and the defender bots.
// The first two are skipped when they are already there; the rest are written
// every time, because they are ours and a stale copy is our bug to have.
func installMods(ctx context.Context, modDir string, logf func(string, ...any)) error {
	if missing := firstMissing(modDir, metamodFiles(runtime.GOOS)); missing != "" {
		logf("installing Metamod:Source %s: %s is missing", assets.MetamodVersion, missing)
		if err := installMetamod(ctx, modDir); err != nil {
			return err
		}
	}
	if err := dropMismatchedLoader(modDir); err != nil {
		return err
	}

	if missing := firstMissing(modDir, sourcemodFiles(runtime.GOOS)); missing != "" {
		logf("installing SourceMod %s: %s is missing", assets.SourcemodVersion, missing)
		if err := installSourcemod(ctx, modDir, logf); err != nil {
			return err
		}
	}

	logf("installing ripext %s, the plugin and the defender bots", assets.RipextVersion)
	if err := installRipextAndPlugin(modDir); err != nil {
		return err
	}
	if err := unzipTo(assets.DefenderBotsZip(), modDir); err != nil {
		return fmt.Errorf("cannot install the defender bots: %w", err)
	}
	return nil
}

/*
The files the engine reads to load Metamod, and Metamod reads to load
SourceMod, relative to the mod directory.

A directory that exists is not a mod that loads. Cowser's server had an
addons/ tree and started as stock Mann vs Machine, every tf2ap_ convar unknown,
and the launcher looked at the directory, found it, and installed nothing. It
looks for these now, and reinstalls when one is gone.
*/
func metamodFiles(goos string) []string {
	if goos == "windows" {
		return []string{"addons/metamod.vdf", "addons/metamod/bin/server.dll", "addons/metamod/bin/metamod.2.tf2.dll"}
	}
	return []string{"addons/metamod.vdf", "addons/metamod/bin/server.so", "addons/metamod/bin/metamod.2.tf2.so"}
}

func sourcemodFiles(goos string) []string {
	if goos == "windows" {
		return []string{"addons/sourcemod/bin/sourcemod_mm.dll", "addons/sourcemod/bin/sourcemod.2.tf2.dll", "addons/sourcemod/bin/sourcemod.logic.dll"}
	}
	return []string{"addons/sourcemod/bin/sourcemod_mm_i486.so", "addons/sourcemod/bin/sourcemod.2.tf2.so", "addons/sourcemod/bin/sourcemod.logic.so"}
}

// firstMissing is the first of the files that is not a regular file under
// modDir, or empty when every one is there.
func firstMissing(modDir string, files []string) string {
	for _, relative := range files {
		info, err := os.Stat(filepath.Join(modDir, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() {
			return relative
		}
	}
	return ""
}

// installSteamcmd downloads and unpacks Valve's SteamCMD bootstrap: a zip on
// Windows, a tarball everywhere else.
func installSteamcmd(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := fetch(ctx, steamcmdURL())
	if err != nil {
		return fmt.Errorf("cannot download SteamCMD: %w", err)
	}
	return unpackTo(data, dir)
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
	exe := firstExisting(steamcmdDir, steamcmdNames())
	if exe == "" {
		return fmt.Errorf("SteamCMD is not in %s", steamcmdDir)
	}

	// Two runs before the real one, both of which SteamCMD needs and neither
	// of which is allowed to fail the install.
	//
	// The first is the bootstrap: a freshly unpacked steamcmd.exe downloads
	// the rest of itself and exits non-zero (7 is the usual one) to say it
	// restarted.
	//
	// The second is a login and nothing else. A SteamCMD that has never logged
	// in fails its first app_update with "Failed to install app '232250'
	// (Missing configuration)", every time, and the same command works on the
	// next run. Spending one session on the login is what makes the install
	// work on the first press rather than the second.
	logf("preparing SteamCMD")
	if err := runSteamcmd(ctx, exe, steamcmdDir, logf, "+quit"); err != nil {
		logf("SteamCMD updated itself (%v), carrying on", err)
	}
	if err := runSteamcmd(ctx, exe, steamcmdDir, logf, "+login", "anonymous", "+quit"); err != nil {
		logf("the warm-up login did not finish (%v), carrying on", err)
	}

	// SteamCMD tokenises its own command line, so a path holding a space or an
	// accent reaches it broken. The short form has neither, and it needs the
	// directory to exist before Windows will name it.
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", gameDir, err)
	}
	installDir := winproc.ShortPath(gameDir)
	if installDir != gameDir {
		logf("installing into %s (the short name for %s)", installDir, gameDir)
	}

	args := []string{
		// No console is attached, so a prompt would hang with nobody to answer
		// it, and a failed command has to stop the script rather than carry on
		// to +quit and report success.
		"+@NoPromptForPassword", "1",
		"+@ShutdownOnFailedCommand", "1",
		"+force_install_dir", installDir,
		"+login", "anonymous",
		// Without fresh app info, app_update fails with "Missing
		// configuration" however good the login was.
		"+app_info_update", "1",
		"+app_update", AppID, "validate",
		"+quit",
	}
	logf("steamcmd %s", strings.Join(args, " "))
	err := runSteamcmd(ctx, exe, steamcmdDir, logf, args...)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return err
	}

	// Plain retry, keeping the app cache. Deleting it puts SteamCMD back in
	// the state that fails, so the second attempt fails the same way.
	logf("SteamCMD failed (%v), trying once more", err)
	if err := runSteamcmd(ctx, exe, steamcmdDir, logf, args...); err != nil {
		// Repair throws SteamCMD away and fetches it again, which is no help
		// at all when the C library it needs was the thing missing.
		if strings.Contains(err.Error(), Missing32BitAdvice) {
			return fmt.Errorf("SteamCMD could not install app %s: %w", AppID, err)
		}
		return fmt.Errorf("SteamCMD could not install app %s: %w. %s", AppID, err, RepairAdvice)
	}
	return nil
}

// steamcmdStateAdvice turns the state SteamCMD reports into something a player
// can act on. It prints the number and stops.
func steamcmdStateAdvice(line string) string {
	switch {
	case strings.Contains(line, "state is 0x202"):
		return "SteamCMD could not write the game files. The disk is full, or the folder is not writable."
	case strings.Contains(line, "state is 0x602"):
		return "SteamCMD lost the download. Check the connection, then press Start again."
	case missing32Bit(line):
		return Missing32BitAdvice
	}
	return ""
}

/*
missing32Bit reports whether this line is a 32-bit binary that will not start.

SteamCMD and the game server are both 32-bit, on every platform. A 64-bit
Linux without the 32-bit C library runs neither, and the shell says so in a
way that names no library: "cannot execute: required file not found" is the
loader itself missing, not the file the line names, and the exit status is
127 whatever went wrong.

Left alone, the installer answered that with "press Repair", which throws
SteamCMD away and downloads it again to fail in exactly the same way.
*/
func missing32Bit(line string) bool {
	return strings.Contains(line, "cannot execute: required file not found") ||
		strings.Contains(line, "No such file or directory") &&
			strings.Contains(line, "ld-linux.so.2")
}

// Missing32BitAdvice names the package, because the error the shell prints
// names nothing a package manager knows.
const Missing32BitAdvice = "The game server is 32-bit and this system has no 32-bit C library. " +
	"Install it: glibc.i686 on Fedora, lib32-glibc on Arch, " +
	"or `sudo dpkg --add-architecture i386 && sudo apt update && sudo apt install lib32gcc-s1` on Debian."

// runSteamcmd runs one SteamCMD command with its own directory as the working
// directory, which is where it keeps the files it downloads for itself.
func runSteamcmd(ctx context.Context, exe, dir string, logf func(string, ...any), args ...string) error {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = dir
	// One splitter for both streams: the shell writes the 32-bit failure to
	// stderr and everything else comes down stdout, and the caller needs the
	// answer whichever it arrived on.
	output := lineWriter(logf)
	cmd.Stdout = output
	cmd.Stderr = output
	winproc.HideConsole(cmd)

	err := cmd.Run()
	if err != nil && output.saw32BitFailure {
		// Wrapped rather than logged again: this is the whole reason the
		// install stopped, and it has to reach the error the window shows.
		return fmt.Errorf("%w. %s", err, Missing32BitAdvice)
	}
	return err
}

// installMetamod unpacks Metamod:Source, which loads SourceMod. Without it
// SourceMod is inert and every plugin in this project is missing.
func installMetamod(ctx context.Context, modDir string) error {
	data, err := fetch(ctx, metamodURL())
	if err != nil {
		return fmt.Errorf("cannot download Metamod:Source: %w", err)
	}
	return unpackTo(data, modDir)
}

// installSourcemod fetches the Windows SourceMod build and unpacks it into the
// mod directory. These archives root at addons/ and cfg/, which belong under
// tf/, not next to srcds.exe.
func installSourcemod(ctx context.Context, modDir string, logf func(string, ...any)) error {
	url := sourcemodURL()
	logf("downloading SourceMod from %s", url)
	data, err := fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("cannot download SourceMod: %w", err)
	}
	return unpackTo(data, modDir)
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
	if err := os.WriteFile(pluginPath, assets.Plugin(), 0o644); err != nil {
		return err
	}
	gamedataDir := filepath.Join(modDir, "addons", "sourcemod", "gamedata")
	if err := os.MkdirAll(gamedataDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(gamedataDir, "tf2_archipelago.txt"),
		assets.PluginGameData(), 0o644)
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
	dst, err := openForWrite(target, file.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()
	_, err = io.Copy(dst, src)
	return err
}

// openForWrite opens the target, waiting for a game server that has not let go
// of it yet.
//
// Windows refuses to open a file another process has mapped, and srcds holds
// rip.ext.dll for a moment after it exits. Every start reinstalls the mods, so
// a Restart pressed straight after a Stop failed the whole install on a file
// that was about to be free.
func openForWrite(target string, mode os.FileMode) (*os.File, error) {
	delay := 200 * time.Millisecond
	var err error
	for attempt := range openAttemptsMax {
		var file *os.File
		file, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err == nil {
			return file, nil
		}
		if attempt == openAttemptsMax-1 || !errors.Is(err, fs.ErrPermission) {
			break
		}
		time.Sleep(delay)
		delay *= 2
	}
	return nil, err
}

// gameInstalled reports whether the TF2 server is installed, by looking for
// srcds.exe (Windows) or srcds_run (Linux) in the game dir.
func gameInstalled(gameDir string) bool {
	for _, name := range srcdsNames() {
		if exists(filepath.Join(gameDir, name)) {
			return true
		}
	}
	return false
}

// firstExisting returns the first of names that is in dir, or "" for none.
func firstExisting(dir string, names []string) string {
	for _, name := range names {
		if candidate := filepath.Join(dir, name); exists(candidate) {
			return candidate
		}
	}
	return ""
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// lineWriter passes each complete line to logf, so subprocess output shows up
// in the launcher's log without drowning it in partial lines.
func lineWriter(logf func(string, ...any)) *lineSplitter {
	return &lineSplitter{logf: logf}
}

type lineSplitter struct {
	buf  []byte
	logf func(string, ...any)

	// saw32BitFailure is the one line worth remembering rather than only
	// printing: it decides what the caller tells the operator to do.
	saw32BitFailure bool
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
			if missing32Bit(line) {
				l.saw32BitFailure = true
			}
			if advice := steamcmdStateAdvice(line); advice != "" {
				l.logf("%s", advice)
			}
		}
	}
	return len(p), nil
}

// RepairAdvice is what to tell a player whose install will not go through. It
// is one sentence because it lands in a log line and in an error.
const RepairAdvice = "If it keeps failing, open Settings and press Repair, " +
	"which throws away SteamCMD and the mods and fetches them again"

// Clean removes what an install can leave broken: SteamCMD itself, the mods
// this project installs, and Steam's record of what it downloaded. Ensure puts
// all three back on the next start.
//
// It keeps the two things a player cannot get back cheaply: the 14 GB of game
// files, and bridge-state, which holds the checks and the unlocks of the run.
// Removing the app manifest costs a verify pass over those game files, not
// another download.
//
// Stop the server and any install in flight first. Windows will not unlink a
// file another process has open.
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
		if err := removeWithRetry(target); err != nil {
			return removed, err
		}
		removed = append(removed, target)
	}
	return removed, nil
}

// removeWithRetry deletes a directory, waiting for whatever still holds it.
//
// Windows refuses to unlink a file another process has open, and SteamCMD can
// take a moment to go after it is asked to stop. Without this, a repair is a
// button the player has to press three times.
func removeWithRetry(target string) error {
	if !exists(target) {
		return nil
	}
	delay := 200 * time.Millisecond
	var err error
	for attempt := range removeAttemptsMax {
		if err = os.RemoveAll(target); err == nil {
			return nil
		}
		if attempt == removeAttemptsMax-1 {
			break
		}
		time.Sleep(delay)
		delay *= 2
	}
	return fmt.Errorf("cannot remove %s, something still has it open: %w", target, err)
}

// removeAttemptsMax spans about six seconds of doubling waits, which covers a
// process on its way out. Longer than that is a file lock a wait will not fix.
const removeAttemptsMax = 6

// openAttemptsMax is the same wait for a file the game server has not released
// yet, on the way in rather than on the way out.
const openAttemptsMax = 6
