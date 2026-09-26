package botloadout

import (
	"fmt"
	"slices"
	"strings"

	"github.com/m-this/tf2-archipelago/gamedata"
)

/*
CustomPrefix marks a loadout the player built rather than one this package
ships. The key carries it, because everything downstream passes loadout keys
around as strings: a saved team, a seat, the config file and both interfaces.
A separate field would mean changing all of them.
*/
const CustomPrefix = "custom:"

// CustomKey is the loadout key for a name the player typed.
func CustomKey(name string) string { return CustomPrefix + name }

// CustomName is the name behind a custom key, and empty for a built-in one.
func CustomName(key string) string {
	if !strings.HasPrefix(key, CustomPrefix) {
		return ""
	}
	return strings.TrimPrefix(key, CustomPrefix)
}

// Built is one loadout the player put together: a class, and an item
// definition index per slot. Stock in a slot leaves that weapon alone.
type Built struct {
	Class   string
	Primary int
	Second  int
	Melee   int
	PDA2    int
}

/*
Library is what a launcher can offer: the presets this package ships, plus the
loadouts the player has built and named.

The zero value is the built-in presets alone, so every caller with no custom
loadouts uses it without saying so.
*/
type Library struct {
	// Built is keyed by the name the player gave, without the prefix.
	Built map[string]Built
}

/*
Loadout is the weapons behind a key, for a class.

Unknown is stock, and that is a rule rather than an accident: a saved team
naming a loadout the player has since deleted still loads, and the seat plays
stock instead of the team refusing to open. A custom loadout built for another
class is unknown here for the same reason, since a Medic cannot hold a
Gunslinger.
*/
func (l Library) Loadout(class Class, key string) Loadout {
	name := CustomName(key)
	if name == "" {
		if key == "" || class.hasPreset(key) {
			return class.LoadoutByKey(key)
		}
		// A bare name. Releases 1.12 and before wrote a saved loadout's name
		// into the settings where the key belongs, so a file from one of them
		// still names its loadout this way.
		name = key
	}
	built, found := l.Built[name]
	if !found || built.Class != class.Key {
		return stock
	}
	return Loadout{
		Key:     key,
		Name:    name,
		Weapons: l.weapons(built),
		Primary: built.Primary,
		Second:  built.Second,
		Melee:   built.Melee,
		PDA2:    built.PDA2,
	}
}

// weapons is the line a menu shows under the name: what the loadout actually
// holds, and "stock" for a slot it leaves alone.
func (l Library) weapons(built Built) string {
	var parts []string
	for _, index := range []int{built.Primary, built.Second, built.Melee, built.PDA2} {
		if index == Stock {
			continue
		}
		parts = append(parts, WeaponName(index))
	}
	if len(parts) == 0 {
		return "stock weapons"
	}
	return strings.Join(parts, ", ")
}

/*
Named reports whether any seat was given a name.

Apart from Anything because the two ask for different things. A name means the
file has to exist, since the mod reads a seat's name out of it whatever the
weapons say; it does not mean the mod should hand out weapons, which is what
sm_redbots_manager_use_custom_loadouts turns on.
*/
func Named(seats []Seat) bool {
	for _, seat := range seats {
		if seat.Name != "" && seat.Class != "" {
			return true
		}
	}
	return false
}

/*
Anything reports whether this team asks for weapons at all, which is what
decides whether the loadout file is written and the mod told to read it.

Both halves, because either one alone is enough: a class pick and a seat pick
land in the same file.
*/
func (l Library) Anything(picks map[string]string, seats []Seat) bool {
	for _, class := range Classes {
		if l.Loadout(class, picks[class.Key]).Key != StockKey {
			return true
		}
	}
	for _, seat := range seats {
		if seat.Card {
			return true
		}
		class, found := ClassByKey(seat.Class)
		if !found {
			continue
		}
		if l.Loadout(class, seat.Loadout).Key != StockKey {
			return true
		}
	}
	return false
}

/*
WeaponName is what to call an item definition index in a menu.

The catalogue in gamedata is the source, and an index it does not carry falls
back to the number. That happens for a loadout somebody typed into the settings
file by hand, which the mod accepts: it validates nothing, so a name this
package cannot supply is not a reason to refuse the pick.
*/
func WeaponName(defIndex int) string {
	if defIndex == Stock {
		return "stock"
	}
	if weapon, found := gamedata.WeaponByIndex(defIndex); found {
		return weapon.Name
	}
	return fmt.Sprintf("item %d", defIndex)
}

/*
Choices is what a menu offers for one class: the presets this package ships,
then the loadouts the player built for that class.

Presets first and in their own order, because that order is what the menus have
always shown and stock is the first of them. The built ones follow, sorted by
name, so the list does not reorder itself when one is saved.
*/
func (l Library) Choices(class Class) []Loadout {
	out := slices.Clone(class.Loadouts)

	names := make([]string, 0, len(l.Built))
	for name, built := range l.Built {
		if built.Class == class.Key {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	for _, name := range names {
		out = append(out, l.Loadout(class, CustomKey(name)))
	}
	return out
}
