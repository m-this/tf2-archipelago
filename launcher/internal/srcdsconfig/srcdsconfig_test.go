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

// TestInstallServerCfg checks the template renders the right values.
func TestInstallServerCfg(t *testing.T) {
	installRoot := t.TempDir()
	s := settings.Settings{
		InstallRoot:   installRoot,
		SrcdsHostname: "Test Server",
		SrcdsRconPw:   "secret-rcon",
		SrcdsPw:       "join-me",
		SrcdsLan:      true,
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
}

// TestSteamIDConversion checks the SteamID64 to STEAM_0:X:Y conversion.
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

// TestInstallPluginCfg checks the plugin config is copied.
func TestInstallPluginCfg(t *testing.T) {
	installRoot := t.TempDir()
	s := settings.Settings{InstallRoot: installRoot}
	if err := Install(s); err != nil {
		t.Fatalf("Install: %v", err)
	}
	cfg, err := os.ReadFile(filepath.Join(gameDir(installRoot), "cfg", "sourcemod", "tf2_archipelago.cfg"))
	if err != nil {
		t.Fatalf("cannot read tf2_archipelago.cfg: %v", err)
	}
	if !strings.Contains(string(cfg), "tf2ap_bridge_url") {
		t.Errorf("tf2_archipelago.cfg does not contain the bridge url setting:\n%s", cfg)
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
