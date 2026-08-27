// Package srcdsconfig renders the game server's configuration files from the
// launcher's settings. It is the Go port of the install_server_cfg and
// install_admin functions in deploy/srcds-entrypoint.sh: server.cfg,
// admins_simple.ini, and the plugin's own tf2_archipelago.cfg.
package srcdsconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Install writes server.cfg, admins_simple.ini, the bots' loadout file and
// tf2_archipelago.cfg into the game tree, under <installRoot>/tf-dedicated/tf.
// It rewrites only when the content differs, so it is safe to call on every
// start.
func Install(s settings.Settings) error {
	gameDir := filepath.Join(s.InstallRoot, "tf-dedicated", "tf")
	if err := installServerCfg(gameDir, s); err != nil {
		return err
	}
	if err := installAdmins(gameDir, s.SrcdsAdminSteamIDs); err != nil {
		return err
	}
	if err := installBotLoadout(gameDir, botlive.LibraryOf(s), s.SrcdsBotLoadouts,
		botloadout.Seats(s.SrcdsBotTeamComp, s.SrcdsBotSeatLoadouts)); err != nil {
		return err
	}
	return installPluginCfg(gameDir)
}

func installServerCfg(gameDir string, s settings.Settings) error {
	target := filepath.Join(gameDir, "cfg", "server.cfg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("cannot create the cfg directory: %w", err)
	}
	rendered, err := RenderServerCfg(s)
	if err != nil {
		return err
	}
	return writeIfChanged(target, []byte(rendered))
}

// RenderServerCfg returns what server.cfg holds for these settings, without
// writing it. The debug bundle uses it to say whether the file the server is
// reading still matches the settings the launcher holds.
func RenderServerCfg(s settings.Settings) (string, error) {
	tmpl, err := template.New("server.cfg").Parse(assets.ServerCfgTemplate())
	if err != nil {
		return "", fmt.Errorf("cannot parse the server.cfg template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"Hostname":          s.SrcdsHostname,
		"RconPassword":      s.SrcdsRconPw,
		"PlayerPassword":    s.SrcdsPw,
		"Lan":               boolToInt(s.EffectiveReach().Lan()),
		"BotsMode":          botsMode(s.SrcdsBots),
		"BotTeamSize":       s.SrcdsBotTeamSize,
		"BotClassBlacklist": botloadout.Blacklist(s.SrcdsBotClassBlacklist),
		"BotTeamComp":       botloadout.Composition(s.SrcdsBotTeamComp, s.SrcdsBotClassBlacklist),
		"BotCustomLoadouts": boolToInt(botlive.LibraryOf(s).Anything(s.SrcdsBotLoadouts,
			botloadout.Seats(s.SrcdsBotTeamComp, s.SrcdsBotSeatLoadouts))),
		"BotUpgradesChat": boolToInt(s.BotUpgradesChat),
		"BluDamage":       scaleOf(s.SrcdsBluDamagePct),
		"BluHealth":       scaleOf(s.SrcdsBluHealthPct),
		"BluSpeed":        scaleOf(s.SrcdsBluSpeedPct),
		"BotHats":         boolToInt(s.SrcdsBotHats),
		"BotHatEffects":   boolToInt(s.SrcdsBotHatEffects),
		"StartMission":    s.SrcdsStartMission,
	}); err != nil {
		return "", fmt.Errorf("cannot render server.cfg: %w", err)
	}
	return buf.String(), nil
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
	if !library.Anything(picks, seats) {
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

// botsMode maps the on/off setting to the mod's convar: 2 is AUTO_BOTS, 0
// leaves the bots to an admin.
func botsMode(enabled bool) int {
	if enabled {
		return 2
	}
	return 0
}

/*
	scaleOf is the mod's float for a percentage the settings hold.

The mod refuses anything under 0.1 and reads 1.0 as off, so a page nobody has
touched writes 1.0 and changes nothing. A zero is somebody who has not set it
rather than somebody asking for harmless robots.
*/
func scaleOf(pct int) string {
	if pct <= 0 || pct > 100 {
		return "1.0"
	}
	if pct < 10 {
		pct = 10
	}
	return fmt.Sprintf("%.2f", float64(pct)/100.0)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func writeIfChanged(path string, content []byte) error {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, content) {
		return nil
	}
	return os.WriteFile(path, content, 0o644)
}
