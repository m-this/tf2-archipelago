package srcdsconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// gameDir returns the path Install writes to, given an install root.
func gameDir(installRoot string) string {
	return filepath.Join(installRoot, "tf-dedicated", "tf")
}

func TestInstallServerCfg(t *testing.T) {
	installRoot := t.TempDir()
	s := settings.Settings{
		InstallRoot:   installRoot,
		SrcdsHostname: "Test Server",
		SrcdsRconPw:   "secret-rcon",
		SrcdsPw:       "join-me",
		SrcdsReach:    settings.ReachLan,
	}

	if err := Install(s); err != nil {
		t.Fatalf("Install: %v", err)
	}

	cfg, err := os.ReadFile(filepath.Join(gameDir(installRoot), "cfg", "server.cfg"))
	if err != nil {
		t.Fatalf("cannot read server.cfg: %v", err)
	}
	body := string(cfg)
	if !strings.Contains(body, `hostname "Test Server"`) {
		t.Errorf("server.cfg does not contain the hostname:\n%s", body)
	}
	if !strings.Contains(body, `rcon_password "secret-rcon"`) {
		t.Errorf("server.cfg does not contain the rcon password:\n%s", body)
	}
	if !strings.Contains(body, `sv_lan 1`) {
		t.Errorf("server.cfg does not contain sv_lan 1:\n%s", body)
	}
	if !strings.Contains(body, `tf_mvm_min_players_to_start 1`) {
		t.Errorf("server.cfg missing the MvM ready-up line:\n%s", body)
	}
	for name, want := range map[string]string{
		"sv_allowdownload": "1",
		"sv_allowupload":   "1",
		"net_maxfilesize":  "64",
	} {
		if got := directive(body, name); got != want {
			t.Errorf("%s = %q, want %q:\n%s", name, got, want, body)
		}
	}
	// Without this the game takes an idle player's seat on RED and the bots
	// fill it, which is what a play-test hit.
	if !strings.Contains(body, `mp_idlemaxtime 0`) {
		t.Errorf("server.cfg leaves the idle check on:\n%s", body)
	}
}

// server.cfg runs after the command line, so its sv_lan is the one that
// sticks. A reach that gets out of the network and a server.cfg that says
// sv_lan 1 is a server nobody outside can join, with nothing in the log to
// say why. A reach with no token behind it is the other way round: sv_lan 0
// there is a server that refuses everybody, local players included.
func TestServerCfgFollowsTheReach(t *testing.T) {
	const token = "C7A1B2E3D4F5A6B7C8D9E0F1A2B3C4D5"
	for _, c := range []struct {
		name  string
		reach settings.Reach
		token string
		want  string
	}{
		{"lan", settings.ReachLan, "0", "1"},
		{"steam", settings.ReachSteam, token, "0"},
		{"port", settings.ReachPort, token, "0"},
		{"steam with no token", settings.ReachSteam, "0", "1"},
		{"port with no token", settings.ReachPort, "0", "1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			installRoot := t.TempDir()
			s := settings.Settings{InstallRoot: installRoot, SrcdsReach: c.reach, SrcdsToken: c.token}
			if err := Install(s); err != nil {
				t.Fatalf("Install: %v", err)
			}
			cfg, err := os.ReadFile(filepath.Join(gameDir(installRoot), "cfg", "server.cfg"))
			if err != nil {
				t.Fatalf("cannot read server.cfg: %v", err)
			}
			// The line, not the word: the comment above it says "sv_lan 0" too.
			if got := directive(string(cfg), "sv_lan"); got != c.want {
				t.Errorf("sv_lan = %q, want %q:\n%s", got, c.want, cfg)
			}
		})
	}
}

// directive returns what a server.cfg line sets, or "" when no line sets it.
// Comment lines start with // and are skipped.
func directive(cfg, name string) string {
	for line := range strings.Lines(cfg) {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == name {
			return fields[1]
		}
	}
	return ""
}

func TestSteamIDConversion(t *testing.T) {
	// 76561197960265728 is the base, so the first account is STEAM_0:0:0.
	got := steamIDForSourcemod("76561197960265728")
	if got != "STEAM_0:0:0" {
		t.Errorf("base SteamID64: got %q, want STEAM_0:0:0", got)
	}
	// 76561198014216803 -> account 53951075 -> STEAM_0:1:26975537
	got = steamIDForSourcemod("76561198014216803")
	if got != "STEAM_0:1:26975537" {
		t.Errorf("known SteamID64: got %q, want STEAM_0:1:26975537", got)
	}
	// A STEAM_0:id passes through.
	got = steamIDForSourcemod("STEAM_0:1:26975537")
	if got != "STEAM_0:1:26975537" {
		t.Errorf("STEAM_0 form: got %q, want STEAM_0:1:26975537", got)
	}
	// A short numeric string passes through.
	got = steamIDForSourcemod("12345")
	if got != "12345" {
		t.Errorf("short numeric: got %q, want 12345", got)
	}
}

// TestInstallAdminsSkipsWithoutSourcemod checks the admin list is skipped
// silently when SourceMod is not installed.
func TestInstallAdminsSkipsWithoutSourcemod(t *testing.T) {
	installRoot := t.TempDir()
	s := settings.Settings{
		InstallRoot:        installRoot,
		SrcdsAdminSteamIDs: "76561198014216803",
	}
	if err := Install(s); err != nil {
		t.Fatalf("Install: %v", err)
	}
	_, err := os.Stat(filepath.Join(gameDir(installRoot), "addons", "sourcemod", "configs", "admins_simple.ini"))
	if err == nil {
		t.Error("admins_simple.ini was written without a SourceMod install")
	}
}

// TestInstallPluginCfg checks the plugin config is copied once, then left to
// whoever runs the server.
func TestInstallPluginCfg(t *testing.T) {
	installRoot := t.TempDir()
	s := settings.Settings{InstallRoot: installRoot}
	if err := Install(s); err != nil {
		t.Fatalf("Install: %v", err)
	}
	path := filepath.Join(gameDir(installRoot), "cfg", "sourcemod", "tf2_archipelago.cfg")
	cfg, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read tf2_archipelago.cfg: %v", err)
	}
	if !strings.Contains(string(cfg), "tf2ap_bridge_url") {
		t.Errorf("tf2_archipelago.cfg does not contain the bridge url setting:\n%s", cfg)
	}

	edited := "tf2ap_debug \"1\"\n"
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install(s); err != nil {
		t.Fatalf("second Install: %v", err)
	}
	cfg, _ = os.ReadFile(path)
	if string(cfg) != edited {
		t.Errorf("the operator's edit was overwritten:\n%s", cfg)
	}
}

// The plugin and the mod read the run's shape from server.cfg: the mission
// the evening starts on, the classes the bots may not play, and the chat
// toggle for their purchases.
func TestInstallServerCfgCarriesTheRunAndTheBots(t *testing.T) {
	s := settings.Settings{
		InstallRoot:            t.TempDir(),
		SrcdsStartMission:      "mvm_coaltown_intermediate",
		SrcdsBotClassBlacklist: []string{"spy", "sniper"},
		SrcdsBotTeamComp:       []string{"engineer", "medic", "nobody", "heavyweapons"},
		SrcdsBotLoadouts:       map[string]string{"scout": "milk"},
		BotUpgradesChat:        true,
		SrcdsBotHats:           true,
	}
	if err := Install(s); err != nil {
		t.Fatalf("Install: %v", err)
	}
	cfg, err := os.ReadFile(filepath.Join(gameDir(s.InstallRoot), "cfg", "server.cfg"))
	if err != nil {
		t.Fatalf("cannot read server.cfg: %v", err)
	}
	for _, want := range []string{
		`tf2ap_start_mission "mvm_coaltown_intermediate"`,
		`sm_redbots_manager_class_blacklist "sniper,spy"`,
		// In the order given, and the class the mod does not have leaves a hole
		// rather than moving the seat after it up one.
		//
		// Padded to the full team because the blacklist above is not empty. A
		// named lineup outranks the blacklist, so any seat this does not name
		// is one the mod draws for itself without consulting it, which is how
		// an unticked Spy reached RED.
		`sm_redbots_manager_team_composition "engineer,medic,,heavyweapons,,"`,
		"sm_redbots_manager_use_custom_loadouts 1",
		"tf2ap_bot_upgrades_chat 1",
		// The looks, which are three separate ticks and not one
		"sm_redbots_manager_bot_hats 1",
		"sm_redbots_manager_bot_hat_effects 0",
	} {
		if !strings.Contains(string(cfg), want) {
			t.Errorf("missing %q in:\n%s", want, cfg)
		}
	}
}

// The loadout file exists exactly when a class has a preset, and only once
// the mod's config directory does: writing it into an empty tree would make
// a directory the installer then finds already there.
func TestInstallBotLoadoutFollowsThePresets(t *testing.T) {
	s := settings.Settings{InstallRoot: t.TempDir(), SrcdsBotLoadouts: map[string]string{"scout": "milk"}}
	target := filepath.Join(gameDir(s.InstallRoot), "addons", "sourcemod", "configs", "defenderbots", "loadout.cfg")

	if err := Install(s); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Error("loadout.cfg was written before the mod was installed")
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(s); err != nil {
		t.Fatalf("Install: %v", err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("loadout.cfg was not written: %v", err)
	}
	if !strings.Contains(string(body), `"scout"`) {
		t.Errorf("loadout.cfg does not hold the scout preset:\n%s", body)
	}

	s.SrcdsBotLoadouts = nil
	if err := Install(s); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Error("loadout.cfg survived every class going back to stock")
	}
}

func TestInstallServerCfgBots(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    settings.Settings
		want []string
	}{
		{
			name: "on",
			s:    settings.Settings{SrcdsBots: true, SrcdsBotTeamSize: 6},
			want: []string{"sm_redbots_manager_mode 2", "sm_redbots_manager_defender_team_size 6"},
		},
		{
			name: "off",
			s:    settings.Settings{SrcdsBots: false, SrcdsBotTeamSize: 6},
			want: []string{"sm_redbots_manager_mode 0"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.s.InstallRoot = t.TempDir()
			if err := Install(tc.s); err != nil {
				t.Fatalf("Install: %v", err)
			}
			cfg, err := os.ReadFile(filepath.Join(gameDir(tc.s.InstallRoot), "cfg", "server.cfg"))
			if err != nil {
				t.Fatalf("cannot read server.cfg: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(string(cfg), want) {
					t.Errorf("missing %q in:\n%s", want, cfg)
				}
			}
			// The mod's own gate counts RED before the wave, where a solo
			// player has no bots yet, so it must always be off.
			if !strings.Contains(string(cfg), "sm_redbots_manager_min_players -1") {
				t.Error("the ready-up gate was left on")
			}
			// The mod kicks every bot inside mvm_wave_complete by default.
			// With that, the game server froze at wave clear; with this, it
			// does not.
			if !strings.Contains(string(cfg), "sm_redbots_manager_kick_bots 0") {
				t.Error("the bots are still kicked at wave end")
			}
		})
	}
}

// tf2ap_debug used to be a boolean that wrote to the chat, and off was the
// sensible default. It is a level now and 1 writes to the log, where a debug
// bundle can carry it. The config is written once, so nobody who installed
// before that change has ever had it.
func TestTheDebugDefaultIsRaisedOnce(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "cfg", "sourcemod", "tf2_archipelago.cfg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("cannot create the directory: %v", err)
	}
	old := "// a comment somebody wrote\ntf2ap_debug \"0\"\ntf2ap_announce \"1\"\n"
	if err := os.WriteFile(target, []byte(old), 0o644); err != nil {
		t.Fatalf("cannot write the config: %v", err)
	}

	if err := installPluginCfg(dir); err != nil {
		t.Fatalf("installPluginCfg: %v", err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("cannot read it back: %v", err)
	}
	if !strings.Contains(string(body), `tf2ap_debug "1"`) {
		t.Errorf("the debug default was not raised:\n%s", body)
	}
	// Everything else in the file belongs to whoever runs the server.
	for _, keep := range []string{"// a comment somebody wrote", `tf2ap_announce "1"`} {
		if !strings.Contains(string(body), keep) {
			t.Errorf("the migration lost %q:\n%s", keep, body)
		}
	}
}

// A 2 is somebody asking for the chat, and it stays.
func TestTheDebugLevelSomebodyChoseIsLeftAlone(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "cfg", "sourcemod", "tf2_archipelago.cfg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("cannot create the directory: %v", err)
	}
	if err := os.WriteFile(target, []byte("tf2ap_debug \"2\"\n"), 0o644); err != nil {
		t.Fatalf("cannot write the config: %v", err)
	}
	if err := installPluginCfg(dir); err != nil {
		t.Fatalf("installPluginCfg: %v", err)
	}
	body, _ := os.ReadFile(target)
	if !strings.Contains(string(body), `tf2ap_debug "2"`) {
		t.Errorf("a chosen level was overwritten:\n%s", body)
	}
}

// sv_downloadurl names the launcher's own file server on the address friends
// reach this machine at. The operator's own host wins when they named one.
// A relayed server hands out no address of its own, so nothing is written and
// the game server's transfer stays the way in.
func TestDownloadURL(t *testing.T) {
	const token = "C7A1B2E3D4F5A6B7C8D9E0F1A2B3C4D5"
	for _, c := range []struct {
		name string
		s    settings.Settings
		host string
		want string
	}{
		{"lan", settings.Settings{SrcdsReach: settings.ReachLan, FastDLPort: 27080}, "192.168.1.10", "http://192.168.1.10:27080/tf"},
		{"port", settings.Settings{SrcdsReach: settings.ReachPort, SrcdsToken: token, FastDLPort: 27080}, "192.168.1.10", "http://192.168.1.10:27080/tf"},
		{"steam", settings.Settings{SrcdsReach: settings.ReachSteam, SrcdsToken: token, FastDLPort: 27080}, "192.168.1.10", ""},
		{"no route", settings.Settings{SrcdsReach: settings.ReachLan, FastDLPort: 27080}, "", ""},
		{"off", settings.Settings{SrcdsReach: settings.ReachLan, FastDLPort: 0}, "192.168.1.10", ""},
		{"own host", settings.Settings{SrcdsReach: settings.ReachSteam, SrcdsToken: token, SrcdsDownloadURL: "https://example.test/tf"}, "", "https://example.test/tf"},
	} {
		if got := DownloadURL(c.s, c.host); got != c.want {
			t.Errorf("%s: DownloadURL = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestServerCfgCarriesTheDownloadURL(t *testing.T) {
	s := settings.Settings{
		InstallRoot:      t.TempDir(),
		SrcdsReach:       settings.ReachLan,
		SrcdsDownloadURL: "https://example.test/tf",
	}
	body, err := RenderServerCfg(s)
	if err != nil {
		t.Fatal(err)
	}
	if got := directive(body, "sv_downloadurl"); got != `"https://example.test/tf"` {
		t.Errorf("sv_downloadurl = %q:\n%s", got, body)
	}
	s.SrcdsDownloadURL, s.FastDLPort = "", 0
	body, err = RenderServerCfg(s)
	if err != nil {
		t.Fatal(err)
	}
	if got := directive(body, "sv_downloadurl"); got != "" {
		t.Errorf("sv_downloadurl = %q with nothing to point it at:\n%s", got, body)
	}
}
