package gui

import (
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

/* What the window's Bots tab means, without the window. From the 1.9.0
 * play-test: the seats named on the page were not the seats the mod filled.
 */
func TestBotTeamFromPicksKeepsTheSeatsAndTheTicks(t *testing.T) {
	engineer, heavy := classIndex(t, "engineer"), classIndex(t, "heavyweapons")
	sniper := classIndex(t, "sniper")

	picks := botTeamPicks{
		// Seat 1 draws, seat 2 is an Engineer, seat 3 draws, seat 4 a Heavy.
		SeatClass:    []int{0, engineer + 1, 0, heavy + 1},
		SeatLoadout:  make([]int, 4),
		Ticked:       ticks(true),
		ClassLoadout: make([]int, len(botloadout.Classes)),
	}
	picks.SeatLoadout[1] = 1 // the Engineer's first preset
	picks.Ticked[sniper] = false

	team := botTeamFrom(picks, botloadout.Library{})

	if got := strings.Join(team.Comp, ","); got != ",engineer,,heavyweapons" {
		t.Errorf("comp = %q", got)
	}
	// A draw seat carries no weapons; a named seat says stock, because that is a
	// choice somebody made.
	if got := strings.Join(team.SeatLoadouts, ","); got != ",ranger,,stock" {
		t.Errorf("seat loadouts = %q", got)
	}
	if got := strings.Join(team.Blacklist, ","); got != "sniper" {
		t.Errorf("blacklist = %q", got)
	}
	// The unticked class reaches the mod, and the seats keep their numbers.
	//
	// Padded to the full team because a class is unticked. A named lineup
	// outranks the blacklist, so any seat left unwritten is one the mod draws
	// without consulting it, which is how an unticked class reached RED.
	if got := botloadout.Composition(team.Comp, team.Blacklist); got != ",engineer,,heavyweapons,," {
		t.Errorf("composition = %q", got)
	}
}

// Every seat on the draw still writes its seats when a class is unticked, or
// the mod plays the map's own lineup instead.
func TestBotTeamFromPicksAllDraw(t *testing.T) {
	picks := botTeamPicks{
		SeatClass:    make([]int, 6),
		SeatLoadout:  make([]int, 6),
		Ticked:       ticks(true),
		ClassLoadout: make([]int, len(botloadout.Classes)),
	}
	picks.Ticked[classIndex(t, "spy")] = false

	team := botTeamFrom(picks, botloadout.Library{})

	if len(team.Comp) != 0 {
		t.Errorf("comp = %q, want nothing named", team.Comp)
	}
	if got := botloadout.Composition(team.Comp, team.Blacklist); got != ",,,,," {
		t.Errorf("composition = %q", got)
	}
}

func ticks(value bool) []bool {
	out := make([]bool, len(botloadout.Classes))
	for i := range out {
		out[i] = value
	}
	return out
}

func classIndex(t *testing.T, key string) int {
	t.Helper()
	for index, class := range botloadout.Classes {
		if class.Key == key {
			return index
		}
	}
	t.Fatalf("no class %q", key)
	return -1
}

/*
	A loadout the player built is read back out of the menu as itself.

The menus are read as indexes, so the list they were filled from and the list
they are read against have to be the same one. Fill from the presets and read
against presets-plus-built, or the other way round, and a built loadout at the
bottom comes back as whichever preset sits at its index.
*/
func TestABuiltLoadoutSurvivesTheMenuIndexes(t *testing.T) {
	library := botloadout.Library{Built: map[string]botloadout.Built{
		"Gas runner": {Class: "pyro", Primary: 594, Second: 1180, Melee: botloadout.Stock, PDA2: botloadout.Stock},
	}}
	pyro, _ := botloadout.ClassByKey("pyro")
	at := len(library.Choices(pyro)) - 1

	picks := botTeamPicks{
		SeatClass:    make([]int, 2),
		SeatLoadout:  make([]int, 2),
		Ticked:       make([]bool, len(botloadout.Classes)),
		ClassLoadout: make([]int, len(botloadout.Classes)),
	}
	pyroAt := slices.IndexFunc(botloadout.Classes, func(c botloadout.Class) bool { return c.Key == "pyro" })
	picks.SeatClass[0] = pyroAt + 1
	picks.SeatLoadout[0] = at
	picks.ClassLoadout[pyroAt] = at
	for i := range picks.Ticked {
		picks.Ticked[i] = true
	}

	team := botTeamFrom(picks, library)
	want := botloadout.CustomKey("Gas runner")
	if team.SeatLoadouts[0] != want {
		t.Errorf("seat one holds %q, want %q", team.SeatLoadouts[0], want)
	}
	if team.ClassLoadouts["pyro"] != want {
		t.Errorf("the pyro holds %q, want %q", team.ClassLoadouts["pyro"], want)
	}

	// And with no built loadouts, the same index is a preset rather than a
	// key nothing can resolve.
	bare := botTeamFrom(picks, botloadout.Library{})
	if botloadout.CustomName(bare.SeatLoadouts[0]) != "" {
		t.Errorf("an empty library produced a custom key: %q", bare.SeatLoadouts[0])
	}
}

// A loadout saved while the dialog is open joins the menus, and the menu stays
// on what it already named even though the new one shifted every index.
func TestReselectLoadoutKeepsThePickWhenAnotherIsSaved(t *testing.T) {
	class := botloadout.Classes[0]
	mine := botloadout.Built{Class: class.Key, Melee: botloadout.Stock}

	before := botloadout.Library{Built: map[string]botloadout.Built{"zulu": mine}}.Choices(class)
	was := before[len(before)-1].Label()
	wasAt := len(before) - 1

	// A name sorting before it, so the one already picked moves down a place.
	after := botloadout.Library{Built: map[string]botloadout.Built{
		"alpha": mine, "zulu": mine,
	}}.Choices(class)
	if after[wasAt].Label() == was {
		t.Fatal("the new loadout did not shift the picked one, so this proves nothing")
	}

	at := reselectLoadout(was, after)
	if after[at].Label() != was {
		t.Fatalf("landed on %q, want %q", after[at].Label(), was)
	}
}

// Saving over a name changes the weapons the label spells out. The menu stays
// on the loadout, because the seat still names it.
func TestReselectLoadoutFollowsARenamedWeapon(t *testing.T) {
	class := botloadout.Classes[0]
	built := botloadout.Built{Class: class.Key, Melee: botloadout.Stock}
	before := botloadout.Library{Built: map[string]botloadout.Built{"gas runner": built}}.Choices(class)
	was := before[len(before)-1].Label()

	weapons := gamedata.WeaponsFor(class.Key, "melee")
	if len(weapons) == 0 {
		t.Skip("this class ships no melee weapons")
	}
	built.Melee = weapons[0].DefIndex
	after := botloadout.Library{Built: map[string]botloadout.Built{"gas runner": built}}.Choices(class)

	at := reselectLoadout(was, after)
	if after[at].Name != "gas runner" {
		t.Fatalf("landed on %q, want the loadout still named gas runner", after[at].Name)
	}
}

// A removed loadout falls back to stock rather than to whatever took its index.
func TestReselectLoadoutFallsBackToStockWhenRemoved(t *testing.T) {
	class := botloadout.Classes[0]
	before := botloadout.Library{Built: map[string]botloadout.Built{
		"gas runner": {Class: class.Key, Melee: botloadout.Stock},
	}}.Choices(class)
	was := before[len(before)-1].Label()

	after := botloadout.Library{}.Choices(class)
	at := reselectLoadout(was, after)
	if after[at].Key != botloadout.StockKey {
		t.Fatalf("landed on %q, want stock", after[at].Key)
	}
}

/*
Saving a bot team hands it to the running server on its own.

There was a button for it, and pressing Save and then the button was two steps
for one intention: nobody changes a lineup and then decides not to use it. The
button is gone, so this holds the path that replaced it.
*/
func TestABotTeamChangeIsAppliedWithoutRestarting(t *testing.T) {
	before := settings.Settings{SrcdsBotTeamComp: []string{"scout"}}
	after := settings.Settings{SrcdsBotTeamComp: []string{"engineer"}}

	if !botlive.LiveOnly(before, after) {
		t.Error("a lineup change is not being treated as live, so saving would restart the server")
	}

	// And a change to anything else still needs the restart, or saving a port
	// would quietly leave the server on the old one.
	ported := before
	ported.SrcdsPort = 27016
	if botlive.LiveOnly(before, ported) {
		t.Error("a port change is being treated as live, so saving would not apply it")
	}
}
