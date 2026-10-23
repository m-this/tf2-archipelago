/*
Package provision decides what the game server starts with: server.cfg, the
srcds command line, and which server mods it loads.

The native launcher and the Docker image both start srcds, and they used to
decide all three each in their own language. The two drifted both ways: the
image never kept an empty server awake or scaled the robots' health, and the
launcher never took RED out of SigMod's robot limit. Both now ask For, and
differ only in the Host they describe.
*/
package provision

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"text/template"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/fastdl"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

//go:embed server.cfg.tmpl
var serverCfgSource string

var serverCfgTemplate = template.Must(template.New("server.cfg").Parse(serverCfgSource))

const sigmodKey = "sigsegv-mvm"

// Host is what differs between the machines a server starts on.
type Host struct {
	// GOOS decides which server mods have a build to load.
	GOOS string
	// Exe is the program started: srcds.exe on Windows, srcds_run elsewhere.
	Exe string
	// LanHost is the address friends on the network reach this machine at,
	// for the download URL. Empty when there is none to give.
	LanHost string
	// NextMissionDelaySeconds overrides the plugin's own pause between a
	// mission cleared and the next. Nil leaves the plugin's default.
	NextMissionDelaySeconds *int
}

// Server is everything a game server starts with.
type Server struct {
	ServerCfg string
	Args      []string
	// Mods are the selected server mods this run loads.
	Mods []string
}

// For decides what the game server starts with, for these settings on this
// host. It reads nothing and writes nothing.
func For(s settings.Settings, host Host) (Server, error) {
	mods := settings.ServerModsToLoad(s, host.GOOS)
	var cfg bytes.Buffer
	if err := serverCfgTemplate.Execute(&cfg, map[string]any{
		"Hostname":          s.SrcdsHostname,
		"RconPassword":      s.SrcdsRconPw,
		"PlayerPassword":    s.SrcdsPw,
		"Lan":               boolToInt(s.EffectiveReach().Lan()),
		"BotsMode":          botsMode(s.SrcdsBots),
		"BotTeamSize":       s.SrcdsBotTeamSize,
		"BotClassBlacklist": botloadout.Blacklist(s.SrcdsBotClassBlacklist),
		"BotTeamComp":       botloadout.Composition(s.SrcdsBotTeamComp, s.SrcdsBotClassBlacklist),
		"BotCustomLoadouts": boolToInt(botlive.LibraryOf(s).Anything(s.SrcdsBotLoadouts,
			botlive.SeatsOf(s))),
		"BotUpgradesChat":  boolToInt(s.BotUpgradesChat),
		"BluHealth":        scaleOf(s.SrcdsBluHealthPct),
		"BotHats":          boolToInt(s.SrcdsBotHats),
		"BotBuyAnywhere":   boolToInt(s.SrcdsBotBuyAnywhere),
		"BotHatEffects":    boolToInt(s.SrcdsBotHatEffects),
		"StartMission":     s.SrcdsStartMission,
		"NextMissionDelay": host.NextMissionDelaySeconds,
		"SigModLoaded":     slices.Contains(mods, sigmodKey),
		"DownloadURL":      DownloadURL(s, host.LanHost),
	}); err != nil {
		return Server{}, fmt.Errorf("cannot render server.cfg: %w", err)
	}
	return Server{ServerCfg: cfg.String(), Args: args(s, host.Exe), Mods: mods}, nil
}

/*
Install writes server.cfg into the game's tf directory, and turns SigMod's RED
robot limit off in the convar file SigMod ships when SigMod is loaded. It
rewrites only what differs and reports whether server.cfg changed.

The convar file runs after server.cfg, so the line in server.cfg alone loses to
the "1" SigMod ships. The file stays the operator's otherwise: the installer
ignores its hash for that reason.
*/
func (srv Server) Install(gameDir string) (bool, error) {
	target := filepath.Join(gameDir, "cfg", "server.cfg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, fmt.Errorf("cannot create the cfg directory: %w", err)
	}
	changed, err := writeIfChanged(target, []byte(srv.ServerCfg))
	if err != nil {
		return false, err
	}
	if slices.Contains(srv.Mods, sigmodKey) {
		if err := leaveREDOutOfTheRobotLimit(filepath.Join(gameDir, "cfg", "sigsegv_convars.cfg")); err != nil {
			return changed, err
		}
	}
	return changed, nil
}

var redRobotLimitOn = regexp.MustCompile(`(?m)^sig_mvm_robot_limit_fix_red[ \t]*"1"`)

func leaveREDOutOfTheRobotLimit(convars string) error {
	body, err := os.ReadFile(convars)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", convars, err)
	}
	if !redRobotLimitOn.Match(body) {
		return nil
	}
	body = redRobotLimitOn.ReplaceAll(body, []byte(`sig_mvm_robot_limit_fix_red "0"`))
	if err := os.WriteFile(convars, body, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", convars, err)
	}
	return nil
}

// DownloadURL is what sv_downloadurl holds: the operator's own host when they
// named one, otherwise the launcher's file server on the address the friends
// on the network reach this machine at. Empty leaves the game server's in-band
// transfer, which is all a relayed client can reach anyway: over Steam the
// players never see this machine's address, and with no route out there is no
// address to give.
func DownloadURL(s settings.Settings, lanHost string) string {
	if s.SrcdsDownloadURL != "" {
		return s.SrcdsDownloadURL
	}
	if s.FastDLPort <= 0 || lanHost == "" || s.EffectiveReach().SteamNetworking() {
		return ""
	}
	return fastdl.URL(lanHost, s.FastDLPort)
}

// StartMap is the map the start mission runs on. Unknown missions can remain
// in a saved file after a catalog update; they fall back rather than being
// passed to srcds as a map name that cannot exist.
func StartMap(s settings.Settings) string {
	if mission, ok := gamedata.MissionByPopFile(s.SrcdsStartMission); ok {
		if played, ok := gamedata.MapByID(mission.Map); ok {
			return played.Name
		}
	}
	mission, _ := gamedata.MissionByPopFile(settings.Defaults().SrcdsStartMission)
	played, _ := gamedata.MapByID(mission.Map)
	return played.Name
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

The mod accepts 0.1 through 10.0 and reads 1.0 as neutral, so a page nobody has
touched writes 1.0 and changes nothing. A zero is somebody who has not set it
rather than somebody asking for zero-health robots.
*/
func scaleOf(pct int) string {
	if pct <= 0 {
		pct = settings.RobotHealthPercentNeutral
	}
	pct = min(max(pct, settings.RobotHealthPercentMin), settings.RobotHealthPercentMax)
	return fmt.Sprintf("%.2f", float64(pct)/100.0)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolArg(on bool) string { return strconv.Itoa(boolToInt(on)) }

func writeIfChanged(path string, content []byte) (bool, error) {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, content) {
		return false, nil
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return false, fmt.Errorf("cannot write %s: %w", path, err)
	}
	return true, nil
}
