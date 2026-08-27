package botlive

import (
	"strings"

	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Seat is one place on RED, as a tab shows it. Class and Weapons are already
// the words a player reads, so a tab lines three columns up and prints them.
type Seat struct {
	Number  int
	Class   string
	Weapons string
}

// DrawnClass is what a seat says when the lineup leaves it to the mod.
const DrawnClass = "the mod picks"

/*
	Team is the seats these settings describe, one per place RED holds.

Always the full team size, not the length of the lineup: the seats a lineup does
not name are the ones the mod draws, and a tab that stops short of them says RED
holds four when it holds six.
*/
func Team(s settings.Settings) []Seat {
	seats := make([]Seat, 0, s.SrcdsBotTeamSize)
	for i := range s.SrcdsBotTeamSize {
		seats = append(seats, seatAt(s, i))
	}
	return seats
}

func seatAt(s settings.Settings, index int) Seat {
	seat := Seat{Number: index + 1, Class: DrawnClass, Weapons: ""}

	key := ""
	if index < len(s.SrcdsBotTeamComp) {
		key = s.SrcdsBotTeamComp[index]
	}
	class, found := botloadout.ClassByKey(key)
	if !found {
		return seat
	}
	seat.Class = class.Name

	loadout := ""
	if index < len(s.SrcdsBotSeatLoadouts) {
		loadout = s.SrcdsBotSeatLoadouts[index]
	}
	// A seat with no weapons of its own wears whatever its class was given,
	// which is the same fallback the mod's loadout file makes.
	if loadout == "" || loadout == botloadout.StockKey {
		loadout = s.SrcdsBotLoadouts[class.Key]
	}
	// Through the library, not through the class: the class only knows the
	// presets this package ships, and a custom loadout is unknown to it, which
	// is stock. The table said stock for every loadout the player built.
	seat.Weapons = LibraryOf(s).Loadout(class, loadout).Label()
	return seat
}

/*
	Drawn is the classes the mod may draw the unnamed seats from, in the class

menu's order, or nothing when every class is allowed.

Phrased as what is allowed rather than as what is blacklisted, because that is
the question a seat reading "the mod picks" raises.
*/
func Drawn(s settings.Settings) string {
	blocked := make(map[string]bool, len(s.SrcdsBotClassBlacklist))
	for _, key := range s.SrcdsBotClassBlacklist {
		blocked[key] = true
	}
	if len(blocked) == 0 {
		return ""
	}
	allowed := make([]string, 0, len(botloadout.Classes))
	for _, class := range botloadout.Classes {
		if !blocked[class.Key] {
			allowed = append(allowed, class.Name)
		}
	}
	if len(allowed) == 0 {
		return "nothing: every class is unticked"
	}
	return strings.Join(allowed, ", ")
}
