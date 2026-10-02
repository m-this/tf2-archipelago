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
