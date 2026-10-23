// Package srcdsconfig writes the game server's configuration files from the
// launcher's settings: server.cfg through provision, admins_simple.ini, the
// bots' files, and the plugin's own tf2_archipelago.cfg.
package srcdsconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/botfiles"
	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/botnames"
	"github.com/m-this/tf2-archipelago/launcher/internal/lanaddr"
	"github.com/m-this/tf2-archipelago/launcher/internal/provision"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Install writes server.cfg, admins_simple.ini, the bots' loadout file and
// tf2_archipelago.cfg into the game tree, under <installRoot>/tf-dedicated/tf.
// It rewrites only when the content differs, so it is safe to call on every
// start.
func Install(s settings.Settings) error {
	gameDir := filepath.Join(s.InstallRoot, "tf-dedicated", "tf")
	server, err := Server(s)
	if err != nil {
		return err
	}
	if _, err := server.Install(gameDir); err != nil {
		return err
	}
	if err := installAdmins(gameDir, s.SrcdsAdminSteamIDs); err != nil {
		return err
	}
	if err := installBotLoadout(gameDir, botlive.LibraryOf(s), s.SrcdsBotLoadouts,
		botlive.SeatsOf(s)); err != nil {
		return err
	}
	if err := installBotNames(gameDir, s); err != nil {
		return err
	}
	return installPluginCfg(gameDir)
}

// Host is the machine the launcher runs on, starting exe.
func Host(exe string) provision.Host {
	return provision.Host{GOOS: runtime.GOOS, Exe: exe, LanHost: lanaddr.Preferred()}
}

// Server is what the game server starts with on this machine.
func Server(s settings.Settings) (provision.Server, error) {
	exe := "srcds_run"
	if runtime.GOOS == "windows" {
		exe = "srcds.exe"
	}
	return provision.For(s, Host(exe))
}

// RenderServerCfg returns what server.cfg holds for these settings, without
// writing it. The debug bundle uses it to say whether the file the server is
// reading still matches the settings the launcher holds.
func RenderServerCfg(s settings.Settings) (string, error) {
	server, err := Server(s)
	return server.ServerCfg, err
}

// installAdmins writes admins_simple.ini from a comma/space/newline-separated
// list of Steam ids. The 17-digit SteamID64 form is converted to SourceMod's
// STEAM_0:X:Y, matching the shell entrypoint. An empty list writes nothing.
func installAdmins(gameDir, list string) error {
	target := filepath.Join(gameDir, "addons", "sourcemod", "configs", "admins_simple.ini")
	if list == "" {
		return nil
	}
	dir := filepath.Dir(target)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	var b strings.Builder
	b.WriteString("// Managed by tf2ap, from SRCDS_ADMIN_STEAMIDS.\n")
	b.WriteString("// Edits here are replaced the next time the launcher starts.\n")
	count := 0
	for _, raw := range splitAdmins(list) {
		admin := steamIDForSourcemod(raw)
		if admin == "" {
			continue
		}
		fmt.Fprintf(&b, "\"%s\" \"99:z\"\n", admin)
		count++
	}
	fmt.Fprintf(&b, "// %d admin(s)\n", count)
	return writeIfChanged(target, []byte(b.String()))
}

// installBotLoadout writes the bots' loadout file when a class has a preset,
// and removes it otherwise: the mod reads the file's presence as "the server
// decides", and stock everywhere is the mod's own default.
func installBotLoadout(gameDir string, library botloadout.Library, picks map[string]string, seats []botloadout.Seat) error {
	target := filepath.Join(gameDir, "addons", "sourcemod", "configs", "defenderbots", "loadout.cfg")
	if !library.Anything(picks, seats) && !botloadout.Named(seats) {
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cannot remove %s: %w", target, err)
		}
		return nil
	}
	if _, err := os.Stat(filepath.Dir(target)); os.IsNotExist(err) {
		return nil
	}
	return writeIfChanged(target, []byte(library.Render(picks, seats)))
}

/*
installBotNames writes the pool the bots draw their names from.

Unlike the loadout file this one is always written, because the mod reads it
whatever the settings say and a missing one leaves every bot called "You forgot
to give me a name!". The mod's package ships the same list, so a server the
launcher has never configured still has one.
*/
func installBotNames(gameDir string, s settings.Settings) error {
	target := botfiles.NamesPath(gameDir)
	if _, err := os.Stat(filepath.Dir(target)); os.IsNotExist(err) {
		return nil
	}
	return writeIfChanged(target, []byte(botnames.Render(s.SrcdsBotNamesExcluded, s.SrcdsBotNamesAdded)))
}

// installPluginCfg drops the plugin's config once. After that the file belongs
// to whoever runs the server: a debug flag they turned on stays on.
func installPluginCfg(gameDir string) error {
	target := filepath.Join(gameDir, "cfg", "sourcemod", "tf2_archipelago.cfg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("cannot create the sourcemod cfg directory: %w", err)
	}
	if _, err := os.Stat(target); err == nil {
		return raiseDebugDefault(target)
	}
	return os.WriteFile(target, assets.PluginConfig(), 0o644)
}

// oldDebugOff is the line every install written before tf2ap_debug became a
// level carries, and newDebugLog is what it means now.
const (
	oldDebugOff = `tf2ap_debug "0"`
	newDebugLog = `tf2ap_debug "1"`
)

/* raiseDebugDefault turns the old off into the new log-only.
 *
 * The file is written once and then belongs to whoever runs the server, which
 * is right for a setting they chose and wrong for one they never saw. tf2ap_debug
 * used to be a boolean that wrote to the chat, and off was the sensible default;
 * it is a level now, and 1 writes to the console and the log where a debug
 * bundle can carry it. Nobody who installed before that has ever had it.
 *
 * A play-test cost two bundles to this: both were collected with the plugin
 * saying nothing, because the file on disk still said 0 and the file is what
 * AutoExecConfig reads.
 *
 * Only that exact line, and only from 0. A 2 is somebody asking for chat and a
 * 0 written by hand today reads the same as the old default, which is the one
 * case this gets wrong; it costs them a log line and no chat.
 */
func raiseDebugDefault(target string) error {
	body, err := os.ReadFile(target)
	if err != nil {
		return nil //nolint:nilerr // an unreadable config is the server's business, not the installer's
	}
	if !bytes.Contains(body, []byte(oldDebugOff)) {
		return nil
	}
	updated := bytes.Replace(body, []byte(oldDebugOff), []byte(newDebugLog), 1)
	return os.WriteFile(target, updated, 0o644)
}

// steamIDForSourcemod converts a SteamID64 (17 digits) to STEAM_0:X:Y, and
// passes anything else through. Mirrors the shell entrypoint.
func steamIDForSourcemod(value string) string {
	if len(value) < 17 {
		return value
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return value
		}
	}
	const steamID64Base = 76561197960265728
	var account int64
	for _, r := range value {
		account = account*10 + int64(r-'0')
	}
	account -= steamID64Base
	return fmt.Sprintf("STEAM_0:%d:%d", account%2, account/2)
}

func splitAdmins(list string) []string {
	out := strings.FieldsFunc(list, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\t' || r == ' '
	})
	return out
}

func writeIfChanged(path string, content []byte) error {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, content) {
		return nil
	}
	return os.WriteFile(path, content, 0o644)
}
