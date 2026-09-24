package botlive

import (
	"slices"
	"strconv"
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

func TestCardSeatCarriesRobotGiantAndDistinctStackedInnates(t *testing.T) {
	s := settings.Settings{
		SrcdsBotTeamComp:     []string{"medic"},
		SrcdsBotSeatNames:    []string{"Herr Doktor"},
		SrcdsBotSeatLoadouts: []string{"kritz"},
		SrcdsBotGiantCards:   []string{"herr-doktor"},
	}
	seats := SeatsOf(s)
	if !seats[0].Robot || !seats[0].Giant || seats[0].Tier != 3 || !seats[0].Unusual || seats[0].UnusualEffect != 13 || seats[0].Cosmetic != 315 || len(seats[0].Innates) != 3 {
		t.Fatalf("card data missing: %+v", seats[0])
	}
	for _, innate := range seats[0].Innates {
		if innate.Stacks != 3 {
			t.Errorf("innate stacks = %d, want 3", innate.Stacks)
		}
	}
	file := loadoutFile(s)
	for _, field := range []string{`"robot"`, `"giant"`, `"tier"`, "\"cosmetic\"\t\"315\"", "\"unusual\"\t\"1\"", "\"unusual_effect\"\t\"13\"", `"innate_1"`, `"innate_2"`, `"innate_3"`} {
		if !strings.Contains(file, field) {
			t.Errorf("card file missing %s", field)
		}
	}
	for _, innate := range seats[0].Innates {
		pair := `"` + strconv.Itoa(innate.Effect) + `,` + strconv.Itoa(innate.Stacks) + `"`
		if !strings.Contains(file, pair) {
			t.Errorf("card file missing all-weapon innate %s", pair)
		}
	}
	normal := s
	normal.SrcdsBotGiantCards = nil
	if got := Commands(normal, s); !contains(got, `sm_ap_botcards_evict "Herr Doktor"`) || contains(got, "sm_redbots_reseat") {
		t.Errorf("turning Giant on should replace only Herr Doktor: %v", got)
	}
}

func TestAPCardUsesReceivedTierAndNeedsAReceivedRoll(t *testing.T) {
	s := settings.Settings{
		MvmBotCards:          true,
		SrcdsBotTeamComp:     []string{"scout"},
		SrcdsBotSeatNames:    []string{"Chucklenuts"},
		SrcdsBotSeatLoadouts: []string{"stock"},
		SrcdsBotGiantCards:   []string{"stock-scout"},
	}
	if SeatsOf(s)[0].Card {
		t.Fatal("unreceived card got card buffs")
	}
	s.SrcdsBotCardRolls = map[string]string{"stock-scout": "Bot: Chucklenuts | Legendary | Giant"}
	seat := SeatsOf(s)[0]
	if !seat.Card || seat.Tier != 3 || !seat.Giant || !seat.Unusual || seat.UnusualEffect != 13 || len(seat.Innates) != 3 {
		t.Fatalf("received Legendary Giant = %+v", seat)
	}
}

func TestCardPriorityDragKeepsExistingBotsAndRebindsSeats(t *testing.T) {
	before := settings.Settings{
		SrcdsBotTeamComp:     []string{"medic", "heavyweapons"},
		SrcdsBotSeatNames:    []string{"Herr Doktor", "IvanTheSpaceBiker"},
		SrcdsBotSeatLoadouts: []string{"kritz", "brass"},
	}
	after := before
	after.SrcdsBotTeamComp = []string{"heavyweapons", "medic"}
	after.SrcdsBotSeatNames = []string{"IvanTheSpaceBiker", "Herr Doktor"}
	after.SrcdsBotSeatLoadouts = []string{"brass", "kritz"}
	got := Commands(before, after)
	if !contains(got, "sm_redbots_rebind_seats") || !contains(got, "sm_ap_botcards_reconcile") || contains(got, "sm_redbots_reseat") {
		t.Fatalf("card reorder did not preserve the team: %v", got)
	}
	if got[len(got)-1] != "say "+Announcement {
		t.Fatalf("card update announced before reconciliation: %v", got)
	}
	for _, command := range got {
		if strings.HasPrefix(command, "sm_ap_botcards_evict") {
			t.Fatalf("dragging priority evicted a bot: %v", got)
		}
	}
}

func TestAddingCardReplacesLowestUnselectedBot(t *testing.T) {
	before := settings.Settings{
		SrcdsBotTeamComp:     []string{"medic", "soldier"},
		SrcdsBotSeatNames:    []string{"Herr Doktor", ""},
		SrcdsBotSeatLoadouts: []string{"kritz", "banner"},
	}
	after := before
	after.SrcdsBotTeamComp = []string{"medic", "engineer"}
	after.SrcdsBotSeatNames = []string{"Herr Doktor", "Chell"}
	after.SrcdsBotSeatLoadouts = []string{"kritz", "ranger"}
	got := Commands(before, after)
	if !contains(got, "sm_ap_botcards_reconcile") || contains(got, "sm_redbots_reseat") {
		t.Errorf("new card should replace the unselected bot: %v", got)
	}
}

func TestHumanCardKeepsTierInnatesWithoutRobotModel(t *testing.T) {
	s := settings.Settings{
		SrcdsBotTeamComp:     []string{"scout"},
		SrcdsBotSeatNames:    []string{"CreditToTeam"},
		SrcdsBotSeatLoadouts: []string{"milk"},
		SrcdsBotHumanCards:   []string{"credit-to-team"},
	}
	seat := SeatsOf(s)[0]
	if !seat.Card || seat.Robot || seat.Giant || seat.Tier != 1 || seat.Cosmetic != 111 || seat.Unusual || seat.UnusualEffect != 0 || len(seat.Innates) != 1 {
		t.Fatalf("human card data = %+v", seat)
	}
	if file := loadoutFile(s); !strings.Contains(file, `"card"`) || strings.Contains(file, `"robot"`) {
		t.Fatalf("human card file has wrong form: %s", file)
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

// The save is applied to a mission already running, so the chat line goes out
// before the reseat rather than after the bots have come back.
func TestTheTeamChangeIsAnnouncedFirst(t *testing.T) {
	before := settings.Settings{
		SrcdsBotTeamSize:     6,
		SrcdsBotTeamComp:     []string{"engineer", "medic"},
		SrcdsBotSeatLoadouts: []string{"ranger", "kritz"},
	}
	want := "say " + Announcement

	lineup := clone(before)
	lineup.SrcdsBotTeamComp = []string{"engineer", "heavyweapons"}
	if got := Commands(before, lineup); len(got) == 0 || got[0] != want {
		t.Errorf("first command = %v, want %q", got, want)
	}

	// The weapons move and the lineup does not, which is still a new team.
	weapons := clone(before)
	weapons.SrcdsBotSeatLoadouts = []string{"widowmaker", "kritz"}
	if got := Commands(before, weapons); len(got) == 0 || got[0] != want {
		t.Errorf("first command = %v, want %q", got, want)
	}

	if got := Commands(before, clone(before)); contains(got, want) {
		t.Errorf("a save that changed no bot setting announced a new team: %v", got)
	}
}

/*
Renaming a seat costs the team nothing.

A reseat rebuilds every bot and loses the upgrades they bought, which is the
right price for a weapon that is handed out on the way in and the wrong one for
a string on a player. The mod grew sm_redbots_reload_names for this in v0.16.0.
*/
func TestANameChangeRenamesRatherThanRecycles(t *testing.T) {
	before := settings.Defaults()
	before.SrcdsBotTeamComp = []string{"pyro", "engineer"}

	after := before
	after.SrcdsBotSeatNames = []string{"Gravel Pit Gary", ""}

	got := Commands(before, after)
	if slices.Contains(got, "sm_redbots_reseat") {
		t.Errorf("a rename recycled the team: %v", got)
	}
	if !slices.Contains(got, "sm_redbots_reload_names") {
		t.Errorf("nothing told the mod to read the names: %v", got)
	}
}

// A weapon that moved still costs a reseat, and covers the name that moved with
// it: a rebuilt bot is named on the way in.
func TestAWeaponChangeStillRecyclesAndCarriesTheNames(t *testing.T) {
	before := settings.Defaults()
	before.SrcdsBotTeamComp = []string{"pyro", "engineer"}

	after := before
	after.SrcdsBotSeatLoadouts = []string{"phlog", ""}
	after.SrcdsBotSeatNames = []string{"Gravel Pit Gary", ""}

	got := Commands(before, after)
	if !slices.Contains(got, "sm_redbots_reseat") {
		t.Errorf("a weapon change did not recycle: %v", got)
	}
	if slices.Contains(got, "sm_redbots_reload_names") {
		t.Errorf("the reseat was followed by a rename it already did: %v", got)
	}
}

// A pool somebody edited reaches the bots too: the file they draw from changed.
func TestChangingThePoolAsksTheModToReadItAgain(t *testing.T) {
	before := settings.Defaults()
	after := before
	after.SrcdsBotNamesAdded = []string{"Gravel Pit Gary"}

	if got := Commands(before, after); !slices.Contains(got, "sm_redbots_reload_names") {
		t.Errorf("an edited pool sent %v", got)
	}
}
