package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func TestBedSettingsLeaveBotsOffUnlessAsked(t *testing.T) {
	if s := bedSettings(options{root: `C:\bed`}, settings.Defaults()); s.SrcdsBots || s.BotUpgradesChat {
		t.Fatalf("bots on by default: SrcdsBots=%v BotUpgradesChat=%v", s.SrcdsBots, s.BotUpgradesChat)
	}
	s := bedSettings(options{root: `C:\bed`, bots: true}, settings.Defaults())
	if !s.SrcdsBots || !s.BotUpgradesChat {
		t.Fatalf("-bots left them off: SrcdsBots=%v BotUpgradesChat=%v", s.SrcdsBots, s.BotUpgradesChat)
	}
}

func TestBedSettingsLoadSigmodUnlessToldNot(t *testing.T) {
	if s := bedSettings(options{root: `C:\bed`}, settings.Defaults()); len(s.SrcdsMods) != 1 || s.SrcdsMods[0] != sigmodKey {
		t.Fatalf("SigMod not loaded by default: %v", s.SrcdsMods)
	}
	s := bedSettings(options{root: `C:\bed`, noSigmod: true, bots: true}, settings.Defaults())
	if len(s.SrcdsMods) != 0 || !s.SrcdsBotHats || !s.SrcdsBotHatEffects {
		t.Fatalf("-no-sigmod: mods %v, hats %v, effects %v", s.SrcdsMods, s.SrcdsBotHats, s.SrcdsBotHatEffects)
	}
}

func TestBedSettingsKeepAPlayersBotsAndDropTheirRoom(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	body := `{"install_root": "C:\\Users\\someone\\tf2-archipelago", "ap_host": "archipelago.gg", "ap_port": 59776,
		"ap_slot_name": "tf2", "srcds_token": "secret", "srcds_reach": "steam", "srcds_mods": ["sigsegv-mvm"],
		"srcds_bots": true, "srcds_bot_team_size": 4, "srcds_bot_team_comp": ["soldier", "medic"],
		"srcds_bot_seat_loadouts": ["beggar", "custom:Banner"], "bot_upgrades_chat": false,
		"srcds_bot_custom_loadouts": {"Banner": {"Class": "soldier", "Primary": 228, "Second": 354, "Melee": 357, "PDA2": -1}}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	base, err := settings.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := bedSettings(options{root: `C:\bed`, noSigmod: true, botTeamSize: 6, config: path}, base)
	if !s.SrcdsBots || s.SrcdsBotTeamSize != 4 || s.BotUpgradesChat || len(s.SrcdsBotTeamComp) != 2 ||
		s.SrcdsBotSeatLoadouts[1] != "custom:Banner" || s.SrcdsBotCustomLoadouts["Banner"].Second != 354 {
		t.Fatalf("the file's bots were not kept: %+v", s)
	}
	d := settings.Defaults()
	if s.InstallRoot != `C:\bed` || s.APHost != d.APHost || s.APPort != d.APPort || s.APSlotName != d.APSlotName ||
		s.SrcdsToken != "" || s.SrcdsReach != settings.ReachLan || len(s.SrcdsMods) != 0 {
		t.Fatalf("the file's machine or room leaked into the bed: %+v", s)
	}
}
