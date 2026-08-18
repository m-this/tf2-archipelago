// Package settings is the launcher's persisted user configuration. One JSON
// file under the OS's user config dir (on Windows, %APPDATA%\tf2ap\config.json),
// written on every successful start, restored on the next launch. It is the
// "last used parameters" memory the prompt asked for.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Settings is everything the launcher asks for and remembers. Fields are the
// .env values from the compose stack, renamed to the launcher's vocabulary.
type Settings struct {
	// InstallRoot is where the 14 GB of game files, SourceMod and steamcmd
	// live. Default is <home>/tf2-archipelago.
	InstallRoot string `json:"install_root"`

	// TestMode plays without Archipelago: the launcher serves a multiworld of
	// one on loopback, makes up a seed, and hands out unlocks as waves are
	// cleared. For trying the server out, and for play-testing.
	TestMode bool `json:"test_mode"`

	// Archipelago room.
	APHost     string `json:"ap_host"`
	APPort     int    `json:"ap_port"`
	APTls      bool   `json:"ap_tls"`
	APSlotName string `json:"ap_slot_name"`
	APPassword string `json:"ap_password,omitempty"`

	// Game server.
	SrcdsHostname      string `json:"srcds_hostname"`
	SrcdsRconPw        string `json:"srcds_rcon_pw,omitempty"`
	SrcdsPw            string `json:"srcds_pw,omitempty"`
	SrcdsPort          int    `json:"srcds_port"`
	SrcdsMaxPlayers    int    `json:"srcds_max_players"`
	SrcdsStartMap      string `json:"srcds_start_map"`
	SrcdsToken         string `json:"srcds_token"`
	SrcdsLan           bool   `json:"srcds_lan"`
	SrcdsAdminSteamIDs string `json:"srcds_admin_steamids,omitempty"`

	// Defender bots. RED is filled to SrcdsBotTeamSize when a wave begins.
	SrcdsBots        bool `json:"srcds_bots"`
	SrcdsBotTeamSize int  `json:"srcds_bot_team_size"`

	// Run shape, for seed generation guidance (the launcher does not generate
	// seeds itself, but it can write a starter YAML for the Archipelago app).
	MvmMissionCount     int    `json:"mvm_mission_count"`
	MvmDifficulty       string `json:"mvm_difficulty"`
	MvmGoal             string `json:"mvm_goal"`
	MvmMissionsanityPct int    `json:"mvm_missionsanity_percentage"`
	MvmDeathLink        bool   `json:"mvm_death_link"`

	// Whether to enable the metrics listener and on what port.
	MetricsPort int `json:"metrics_port"`
}

// Defaults returns the factory settings, matching deploy/.env.example.
func Defaults() Settings {
	return Settings{
		InstallRoot:         defaultInstallRoot(),
		APHost:              "archipelago.gg",
		APPort:              0,
		APTls:               true,
		APSlotName:          "tf2",
		SrcdsHostname:       "Mann vs Archipelago",
		SrcdsPort:           27015,
		SrcdsMaxPlayers:     32,
		SrcdsStartMap:       "mvm_decoy",
		SrcdsToken:          "0",
		SrcdsLan:            true,
		SrcdsBots:           true,
		SrcdsBotTeamSize:    6,
		MvmMissionCount:     8,
		MvmDifficulty:       "intermediate",
		MvmGoal:             "final_boss",
		MvmMissionsanityPct: 80,
		MetricsPort:         24681,
	}
}

// Path returns the config file location. On Windows this is
// %APPDATA%\tf2ap\config.json; on Linux ~/.config/tf2ap/config.json.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot find the user config directory: %w", err)
	}
	return filepath.Join(dir, "tf2ap", "config.json"), nil
}

// Load reads the config file, applying defaults for anything unset. A missing
// file is not an error: it returns Defaults.
func Load() (Settings, error) {
	path, err := Path()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Defaults(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("cannot parse %s: %w", path, err)
	}
	s.applyDefaults()
	return s, nil
}

// Render returns the settings as the JSON that Save writes. The debug bundle
// uses it to ship a copy with the passwords taken out.
func Render(s Settings) (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// Save writes the config file, creating the directory.
func Save(s Settings) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cannot create %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return nil
}

// applyDefaults fills zero values with the defaults, so an old config file that
// predates a field still works.
func (s *Settings) applyDefaults() {
	d := Defaults()
	if s.InstallRoot == "" {
		s.InstallRoot = d.InstallRoot
	}
	if s.APSlotName == "" {
		s.APSlotName = d.APSlotName
	}
	if s.SrcdsHostname == "" {
		s.SrcdsHostname = d.SrcdsHostname
	}
	if s.SrcdsPort == 0 {
		s.SrcdsPort = d.SrcdsPort
	}
	if s.SrcdsMaxPlayers == 0 {
		s.SrcdsMaxPlayers = d.SrcdsMaxPlayers
	}
	if s.SrcdsBotTeamSize == 0 {
		s.SrcdsBotTeamSize = d.SrcdsBotTeamSize
	}
	if s.SrcdsStartMap == "" {
		s.SrcdsStartMap = d.SrcdsStartMap
	}
	if s.SrcdsToken == "" {
		s.SrcdsToken = d.SrcdsToken
	}
	if s.MvmMissionCount == 0 {
		s.MvmMissionCount = d.MvmMissionCount
	}
	if s.MvmDifficulty == "" {
		s.MvmDifficulty = d.MvmDifficulty
	}
	if s.MvmGoal == "" {
		s.MvmGoal = d.MvmGoal
	}
	if s.MvmMissionsanityPct == 0 {
		s.MvmMissionsanityPct = d.MvmMissionsanityPct
	}
	if s.MetricsPort == 0 {
		s.MetricsPort = d.MetricsPort
	}
}

func defaultInstallRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "tf2-archipelago"
	}
	return filepath.Join(home, "tf2-archipelago")
}
