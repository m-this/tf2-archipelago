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
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Install writes server.cfg, admins_simple.ini and tf2_archipelago.cfg into the
// game tree, under <installRoot>/tf-dedicated/tf. It rewrites only when the
// content differs, so it is safe to call on every start.
func Install(s settings.Settings) error {
	gameDir := filepath.Join(s.InstallRoot, "tf-dedicated", "tf")
	if err := installServerCfg(gameDir, s); err != nil {
		return err
	}
	if err := installAdmins(gameDir, s.SrcdsAdminSteamIDs); err != nil {
		return err
	}
	return installPluginCfg(gameDir)
}

func installServerCfg(gameDir string, s settings.Settings) error {
	target := filepath.Join(gameDir, "cfg", "server.cfg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("cannot create the cfg directory: %w", err)
	}
	tmpl, err := template.New("server.cfg").Parse(assets.ServerCfgTemplate())
	if err != nil {
		return fmt.Errorf("cannot parse the server.cfg template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"Hostname":       s.SrcdsHostname,
		"RconPassword":   s.SrcdsRconPw,
		"PlayerPassword": s.SrcdsPw,
		"Lan":            boolToInt(s.SrcdsLan),
		"BotsMode":       botsMode(s.SrcdsBots),
		"BotTeamSize":    s.SrcdsBotTeamSize,
	}); err != nil {
		return fmt.Errorf("cannot render server.cfg: %w", err)
	}
	return writeIfChanged(target, buf.Bytes())
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

func installPluginCfg(gameDir string) error {
	target := filepath.Join(gameDir, "cfg", "sourcemod", "tf2_archipelago.cfg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("cannot create the sourcemod cfg directory: %w", err)
	}
	return writeIfChanged(target, assets.PluginConfig())
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
