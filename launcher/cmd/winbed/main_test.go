package main

import "testing"

func TestBedSettingsLeaveBotsOffUnlessAsked(t *testing.T) {
	if s := bedSettings(options{root: `C:\bed`}); s.SrcdsBots || s.BotUpgradesChat {
		t.Fatalf("bots on by default: SrcdsBots=%v BotUpgradesChat=%v", s.SrcdsBots, s.BotUpgradesChat)
	}
	s := bedSettings(options{root: `C:\bed`, bots: true})
	if !s.SrcdsBots || !s.BotUpgradesChat {
		t.Fatalf("-bots left them off: SrcdsBots=%v BotUpgradesChat=%v", s.SrcdsBots, s.BotUpgradesChat)
	}
}

func TestBedSettingsLoadSigmodUnlessToldNot(t *testing.T) {
	if s := bedSettings(options{root: `C:\bed`}); len(s.SrcdsMods) != 1 || s.SrcdsMods[0] != sigmodKey {
		t.Fatalf("SigMod not loaded by default: %v", s.SrcdsMods)
	}
	s := bedSettings(options{root: `C:\bed`, noSigmod: true, bots: true})
	if len(s.SrcdsMods) != 0 || !s.SrcdsBotHats || !s.SrcdsBotHatEffects {
		t.Fatalf("-no-sigmod: mods %v, hats %v, effects %v", s.SrcdsMods, s.SrcdsBotHats, s.SrcdsBotHatEffects)
	}
}
