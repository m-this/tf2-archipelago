package gui

import (
	"slices"
	"strings"

	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

/* botTeamPicks is what the Bots tab's widgets say, as numbers. The tab only
 * builds on Windows, so the rules below hold what it means.
 *
 * A seat menu opens on the draw, so zero is the draw and a class is its place
 * in botloadout.Classes plus one. The loadout indexes are places in that
 * class's presets. Ticked holds one entry per class, in the same order.
 */
type botTeamPicks struct {
	SeatClass    []int
	SeatLoadout  []int
	Ticked       []bool
	ClassLoadout []int
}

// botTeamFrom is the team those picks describe. A seat left on the draw is an
// empty entry, because the mod counts a seat by its place in the list. It drops
// the trailing draws, which name no seat.
func botTeamFrom(picks botTeamPicks, library botloadout.Library) settings.BotTeam {
	var out settings.BotTeam
	named := -1
	for seat, index := range picks.SeatClass {
		if index > 0 && index <= len(botloadout.Classes) {
			named = seat
		}
	}
	for seat := 0; seat <= named; seat++ {
		index := picks.SeatClass[seat]
		if index <= 0 || index > len(botloadout.Classes) {
			out.Comp = append(out.Comp, "")
			out.SeatLoadouts = append(out.SeatLoadouts, "")
			continue
		}
		class := botloadout.Classes[index-1]
		out.Comp = append(out.Comp, class.Key)
		out.SeatLoadouts = append(out.SeatLoadouts, loadoutKeyAt(library, class, at(picks.SeatLoadout, seat)))
	}
	out.ClassLoadouts = make(map[string]string)
	for index, class := range botloadout.Classes {
		if index < len(picks.Ticked) && !picks.Ticked[index] {
			out.Blacklist = append(out.Blacklist, class.Key)
		}
		if key := loadoutKeyAt(library, class, at(picks.ClassLoadout, index)); key != botloadout.StockKey {
			out.ClassLoadouts[class.Key] = key
		}
	}
	return out
}

/*
	loadoutKeyAt is the loadout at that place in the menu, and stock for anything

else: a menu with nothing chosen reports -1.
*
* The list has to be the one the menu was filled from, or a built loadout at the
* bottom is read back as the preset that sits at its index.
*/
func loadoutKeyAt(library botloadout.Library, class botloadout.Class, index int) string {
	choices := library.Choices(class)
	if index < 0 || index >= len(choices) {
		return botloadout.StockKey
	}
	return choices[index].Key
}

func at(values []int, index int) int {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

/*
reselectLoadout is where a menu should sit after its list was rebuilt, given
what it said before.

By name rather than by index, because saving a loadout inserts it among the
others and every index after it moves. By name rather than by the whole label,
because saving over a name changes the weapons the label spells out and the
menu should stay on the loadout the seat still names.

A loadout that is gone falls back to zero, which is the class's stock: a seat
cannot go on holding weapons nobody has any more.
*/
func reselectLoadout(was string, choices []botloadout.Loadout) int {
	if at := slices.IndexFunc(choices, func(l botloadout.Loadout) bool { return l.Label() == was }); at >= 0 {
		return at
	}
	name := was
	if cut := strings.Index(was, ": "); cut >= 0 {
		name = was[:cut]
	}
	if at := slices.IndexFunc(choices, func(l botloadout.Loadout) bool { return l.Name == name }); at >= 0 {
		return at
	}
	return 0
}
