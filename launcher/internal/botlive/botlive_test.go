package botlive

import (
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// The loadout file is read when the reseat runs, so the reseat is last or the
// team comes back holding what it held before.
/* The whole-team recycle only goes when the weapons moved.

A lineup change is handled by the mod's own hook on the composition convar,
which keeps the bots whose class the new list still wants. Sending the recycle
as well threw those bots' upgrades away for nothing.
*/
func TestOnlyAWeaponChangeRecyclesTheTeam(t *testing.T) {
	seats := settings.Settings{
		SrcdsBotTeamSize:     6,
		SrcdsBotTeamComp:     []string{"engineer", "medic"},
		SrcdsBotSeatLoadouts: []string{"ranger", "kritz"},
	}

	// Seat two changes class and both seats keep the preset they were on, so
	// nothing about the file changes but the class each seat is filed under.
	classSwap := clone(seats)
	classSwap.SrcdsBotTeamComp = []string{"engineer", "engineer"}
	classSwap.SrcdsBotSeatLoadouts = []string{"ranger", "ranger"}
	if !contains(Commands(seats, classSwap), "sm_redbots_reseat") {
		t.Error("seat two's weapons changed and the team was not recycled")
	}

	// The lineup alone, with the file untouched.
	sizeOnly := clone(seats)
	sizeOnly.SrcdsBotTeamSize = 4
	if contains(Commands(seats, sizeOnly), "sm_redbots_reseat") {
		t.Error("a team-size change recycled the whole team")
	}

	weaponSwap := clone(seats)
	weaponSwap.SrcdsBotSeatLoadouts = []string{"widowmaker", "kritz"}
	got := Commands(seats, weaponSwap)
	if last := got[len(got)-1]; last != "sm_redbots_reseat" {
		// Last, because the mod reads the loadout file when the recycle runs.
		t.Errorf("last command = %q, want the recycle", last)
	}
}

// A team with nothing custom writes no loadout file, so the convar that makes
// the mod look for one has to be off with it.
func TestTheLoadoutConvarAgreesWithTheFile(t *testing.T) {
	for _, test := range []struct {
		name string
		s    settings.Settings
		want string
	}{
		{"stock everywhere", settings.Settings{}, "sm_redbots_manager_use_custom_loadouts 0"},
		{
			"a class preset",
			settings.Settings{SrcdsBotLoadouts: map[string]string{"spy": "kunai"}},
			"sm_redbots_manager_use_custom_loadouts 1",
		},
		{
			"a seat preset and no class preset",
			settings.Settings{
				SrcdsBotTeamComp:     []string{"engineer"},
				SrcdsBotSeatLoadouts: []string{"ranger"},
			},
			"sm_redbots_manager_use_custom_loadouts 1",
		},
		{
			// A key no class has is stock, so there is nothing to write and
			// nothing for the mod to read.
			"a seat naming a loadout that does not exist",
			settings.Settings{
				SrcdsBotTeamComp:     []string{"engineer"},
				SrcdsBotSeatLoadouts: []string{"gunslinger"},
			},
			"sm_redbots_manager_use_custom_loadouts 0",
		},
		{
			"a seat naming a loadout the player built",
			settings.Settings{
				SrcdsBotTeamComp:     []string{"engineer"},
				SrcdsBotSeatLoadouts: []string{botloadout.CustomKey("Nest")},
				SrcdsBotCustomLoadouts: map[string]botloadout.Built{
					"Nest": {Class: "engineer", Primary: 997, Second: botloadout.Stock, Melee: botloadout.Stock, PDA2: botloadout.Stock},
				},
			},
			"sm_redbots_manager_use_custom_loadouts 1",
		},
	} {
		if !contains(Commands(test.s, test.s), test.want) {
			t.Errorf("%s: missing %q in %v", test.name, test.want, Commands(test.s, test.s))
		}
	}
}

// An unticked class reaches every seat the composition does not name, which is
// the whole reason the blacklist is sent alongside it.
func TestTheBlacklistAndTheCompositionGoTogether(t *testing.T) {
	after := settings.Settings{
		SrcdsBotTeamComp:       []string{"scout"},
		SrcdsBotClassBlacklist: []string{"spy"},
	}
	got := Commands(after, after)
	if !contains(got, `sm_redbots_manager_class_blacklist "spy"`) {
		t.Errorf("no blacklist in %v", got)
	}
	if !contains(got, `sm_redbots_manager_team_composition "scout,,,,,"`) {
		t.Errorf("the short lineup did not cover every seat: %v", got)
	}
}

// A composition is one argument however many classes it names, or the console
// reads the commas as the end of it.
func TestTheCompositionIsQuoted(t *testing.T) {
	team := settings.Settings{SrcdsBotTeamComp: []string{"engineer", "medic"}}
	for _, line := range Commands(team, team) {
		if strings.HasPrefix(line, "sm_redbots_manager_team_composition") {
			if line != `sm_redbots_manager_team_composition "engineer,medic"` {
				t.Errorf("composition command = %q", line)
			}
			return
		}
	}
	t.Error("no composition command")
}

func contains(lines []string, want string) bool {
	return slices.Contains(lines, want)
}

/*
	A bot team is the one thing a running server takes without a restart, and

anything else on the same save is still a restart.

The test is written against real fields rather than a list of names, so a
setting added to the struct later fails it here rather than silently going live.
*/
func TestOnlyTheBotTeamGoesLive(t *testing.T) {
	base := settings.Settings{
		SrcdsBotTeamSize: 6,
		SrcdsBotTeamComp: []string{"engineer", "medic"},
		SrcdsHostname:    "a server",
		SrcdsPort:        27015,
	}
	for _, test := range []struct {
		name  string
		after func(settings.Settings) settings.Settings
		want  bool
	}{
		{"nothing moved", func(s settings.Settings) settings.Settings { return s }, false},
		{
			"a seat changed class",
			func(s settings.Settings) settings.Settings {
				s.SrcdsBotTeamComp = []string{"engineer", "heavyweapons"}
				return s
			},
			true,
		},
		{
			"the team got bigger",
			func(s settings.Settings) settings.Settings { s.SrcdsBotTeamSize = 4; return s },
			true,
		},
		{
			"a class was unticked",
			func(s settings.Settings) settings.Settings {
				s.SrcdsBotClassBlacklist = []string{"spy"}
				return s
			},
			true,
		},
		{
			"a seat changed weapons",
			func(s settings.Settings) settings.Settings {
				s.SrcdsBotSeatLoadouts = []string{"gunslinger", ""}
				return s
			},
			true,
		},
		{
			"a team was saved under a name",
			func(s settings.Settings) settings.Settings {
				s.SrcdsBotTeamPresets = map[string]settings.BotTeam{"tanks": {}}
				return s
			},
			true,
		},
		{
			"the port moved, which the server reads once",
			func(s settings.Settings) settings.Settings { s.SrcdsPort = 27016; return s },
			false,
		},
		{
			"a seat changed and so did the port",
			func(s settings.Settings) settings.Settings {
				s.SrcdsBotTeamComp = []string{"scout"}
				s.SrcdsPort = 27016
				return s
			},
			false,
		},
	} {
		if got := LiveOnly(base, test.after(clone(base))); got != test.want {
			t.Errorf("%s: LiveOnly = %v, want %v", test.name, got, test.want)
		}
	}
}

// clone keeps one case from writing into the next one's base.
func clone(s settings.Settings) settings.Settings {
	out := s
	out.SrcdsBotTeamComp = append([]string(nil), s.SrcdsBotTeamComp...)
	out.SrcdsBotSeatLoadouts = append([]string(nil), s.SrcdsBotSeatLoadouts...)
	out.SrcdsBotClassBlacklist = append([]string(nil), s.SrcdsBotClassBlacklist...)
	return out
}

// The tab is the only place a player can check what the bots will carry, so a
// loadout they built has to read as itself rather than as stock.
func TestTeamNamesACustomLoadout(t *testing.T) {
	class := botloadout.Classes[0]
	weapons := gamedata.WeaponsFor(class.Key, "melee")
	if len(weapons) == 0 {
		t.Skip("this class ships no melee weapons")
	}

	s := settings.Settings{
		SrcdsBotTeamSize:     1,
		SrcdsBotTeamComp:     []string{class.Key},
		SrcdsBotSeatLoadouts: []string{botloadout.CustomKey("gas runner")},
		SrcdsBotCustomLoadouts: map[string]botloadout.Built{
			"gas runner": {Class: class.Key, Melee: weapons[0].DefIndex},
		},
	}

	seats := Team(s)
	if len(seats) != 1 {
		t.Fatalf("got %d seats, want 1", len(seats))
	}
	if !strings.Contains(seats[0].Weapons, "gas runner") {
		t.Fatalf("the seat reads %q, want it to name gas runner", seats[0].Weapons)
	}
	if !strings.Contains(seats[0].Weapons, weapons[0].Name) {
		t.Fatalf("the seat reads %q, want it to name %q", seats[0].Weapons, weapons[0].Name)
	}
}

// A class-wide custom loadout reaches the seats that do not name one of their
// own, by the same fallback the mod's loadout file makes.
func TestTeamNamesACustomLoadoutGivenToTheClass(t *testing.T) {
	class := botloadout.Classes[0]
	s := settings.Settings{
		SrcdsBotTeamSize: 1,
		SrcdsBotTeamComp: []string{class.Key},
		SrcdsBotLoadouts: map[string]string{class.Key: botloadout.CustomKey("gas runner")},
		SrcdsBotCustomLoadouts: map[string]botloadout.Built{
			"gas runner": {Class: class.Key, Melee: botloadout.Stock},
		},
	}

	if got := Team(s)[0].Weapons; !strings.Contains(got, "gas runner") {
		t.Fatalf("the seat reads %q, want it to name gas runner", got)
	}
}
