package form

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/botcards"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/botnames"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Seats is how many seats the team composition names, which is the largest team
// the mod will field.
const Seats = 6

/*
	The bots, and the loadouts they carry

Two pages. Bots is what the team is: how many, which classes sit in which seat,
what each seat carries and what a class falls back on. Loadouts is the editor
for one loadout at a time, and it writes to the draft rather than the settings
until the row that names it is pressed.

The draft is why State exists. Neither the team name nor the loadout being built
is a setting: nothing in the config file holds them and leaving the page throws
them away. Both interfaces used to keep them in their own screen struct, which
is how the loadout builder ended up the one part of the settings the window and
the terminal did not agree about even in shape.
*/
func botSpecs(s State, env Env) []Spec {
	const tab = "Bots"

	specs := inGroup("Team",
		toggle("bots.fill", tab, "Fill RED with bots",
			"Valve balances every wave for six players. Off leaves the seats empty until an admin runs sm_addbots.",
			"bots on",
			func(s State) bool { return s.Settings.SrcdsBots },
			func(s State, v bool) State { s.Settings.SrcdsBots = v; return s }),

		number("bots.team_size", tab, "Fill RED to",
			"How many players the server fills RED to, humans included. Lower is harder.",
			1, Seats,
			func(s State) int { return s.Settings.SrcdsBotTeamSize },
			func(s State, v int) State { s.Settings.SrcdsBotTeamSize = v; return s }),

		toggle("bots.upgrades_chat", tab, "Say what they buy",
			"Write every bot purchase at the upgrade station to the chat. It is a lot of chat.",
			"in the chat",
			func(s State) bool { return s.Settings.BotUpgradesChat },
			func(s State, v bool) State { s.Settings.BotUpgradesChat = v; return s }),

		// A team is worth naming once. The window has a menu and two buttons
		// for this; the terminal has a list to load from and a name to save
		// under. Both are these four rows.
		openChoice("bots.team_preset", tab, "Saved team",
			"A team somebody named and kept: the seats, their loadouts, and the classes the mod may draw from.",
			teamPresetOptions,
			func(s State) string { return "" },
			loadTeamPreset),

		text("bots.team_name", tab, "Team name",
			"The name Save this team as keeps it under.",
			"two engineers, one medic",
			func(s State) string { return s.Draft.TeamName },
			func(s State, v string) State { s.Draft.TeamName = v; return s }),

		press("bots.save_team", tab, "Save this team as",
			"Keeps the seats and their loadouts under the name in the box. Saving over a name replaces it."),
		press("bots.remove_team", tab, "Remove this team",
			"Forget the saved team named above. The seats on screen are left alone."),
		cardPrioritySpec(),
	)
	for _, card := range botcards.Cards {
		specs = append(specs, inGroup("Team", cardFormSpec(card))...)
	}

	// The seats, in the order they fill, each with what it plays and what it
	// carries. Two engineers are only worth naming separately if they can hold
	// different weapons.
	for seat := range Seats {
		specs = append(specs, inGroup("Team", seatCardSpec(seat, env), seatClassSpec(seat), seatLoadoutSpec(seat), seatNameSpec(seat))...)
	}

	// What a class falls back on when the mod draws it rather than a seat
	// naming it, and whether it may be drawn at all.
	for _, class := range botloadout.Classes {
		specs = append(specs, inGroup("Classes", classAllowedSpec(class), classLoadoutSpec(class))...)
	}

	specs = append(specs, nameSpecs(s)...)

	// Last, because none of it changes a wave.
	specs = append(specs, inGroup("Looks",
		toggle("bots.hats", tab, "Cosmetic items",
			"A random cosmetic item on every bot, hat or not, drawn from the ones its class can wear. It changes nothing about how they play.",
			"one each",
			func(s State) bool { return s.Settings.SrcdsBotHats },
			func(s State, v bool) State { s.Settings.SrcdsBotHats = v; return s }),

		toggle("bots.hat_effects", tab, "Unusual effects",
			"A random unusual effect on that cosmetic item. Six particle effects on screen for the whole wave.",
			"and an effect on it",
			func(s State) bool { return s.Settings.SrcdsBotHatEffects },
			func(s State, v bool) State { s.Settings.SrcdsBotHatEffects = v; return s }),
	)...)

	return append(specs, loadoutSpecs(s)...)
}

/*
	nameSpecs is the pool the bots draw their names from

A tick per name this launcher ships, then a row per name somebody added, then
the box that adds one. Stored as the difference from the shipped list rather
than as a copy of it, the same way the mission pool is stored: a name added to
the shipped file in a later release then reaches a settings file written before
it existed.

The mod reads the pool at map start, so a change here reaches the bots on the
next mission rather than the next wave.
*/
func nameSpecs(s State) []Spec {
	const tab = "Bots"
	specs := []Spec{
		text("bots.name_new", tab, "Add a name",
			fmt.Sprintf("A name to put in the pool, %d characters at most. The bots draw from it as they take their seats, and each one keeps what it drew until it leaves.", botnames.NameMax),
			"Gravel Pit Gary",
			func(s State) string { return s.Draft.BotName },
			func(s State, v string) State { s.Draft.BotName = v; return s }),

		press("bots.name_add", tab, "Add this name",
			"Puts the name in the box into the pool."),
	}

	for _, name := range s.Settings.SrcdsBotNamesAdded {
		specs = append(specs, addedNameSpec(name))
	}
	for _, name := range botnames.Shipped() {
		specs = append(specs, shippedNameSpec(name))
	}
	return inGroup("Names", specs...)
}

// addedNameSpec is one name somebody typed. Off removes it outright rather
// than holding it aside: it was never shipped, so there is nothing to go back
// to, and a list of names somebody turned off is a list nobody reads.
func addedNameSpec(name string) Spec {
	return twoWayToggle("bots.name.added."+name, "Bots", name,
		"A name you added. Off takes it out of the pool and forgets it.",
		"in the pool", "removed",
		func(s State) bool { return slices.Contains(s.Settings.SrcdsBotNamesAdded, name) },
		func(s State, in bool) State {
			if in {
				return s
			}
			s.Settings.SrcdsBotNamesAdded = slices.DeleteFunc(
				slices.Clone(s.Settings.SrcdsBotNamesAdded),
				func(one string) bool { return one == name })
			return s
		})
}

// shippedNameSpec is one of the names this launcher ships.
func shippedNameSpec(name string) Spec {
	return twoWayToggle("bots.name.shipped."+name, "Bots", name,
		"A name this launcher ships. Off leaves it out of the pool.",
		"in the pool", "left out",
		func(s State) bool { return !slices.Contains(s.Settings.SrcdsBotNamesExcluded, name) },
		func(s State, in bool) State {
			excluded := slices.DeleteFunc(slices.Clone(s.Settings.SrcdsBotNamesExcluded),
				func(one string) bool { return one == name })
			if !in {
				excluded = append(excluded, name)
			}
			s.Settings.SrcdsBotNamesExcluded = excluded
			return s
		})
}

// seatClassSpec is which class sits in one seat. The draw is what the mod does
// when the seat names nobody, which is the default: a team named here beats the
// blacklist, and an unnamed seat leaves the mod to pick.
func seatClassSpec(seat int) Spec {
	values := []string{""}
	labels := []string{"let the mod draw"}
	for _, class := range botloadout.Classes {
		values = append(values, class.Key)
		labels = append(labels, class.Name)
	}

	return choice(fmt.Sprintf("bots.seat.%d.class", seat), "Bots",
		fmt.Sprintf("Seat %d", seat+1),
		"Which class fills this seat. Left to the mod, it draws from the classes ticked below.",
		options(values, labels),
		func(s State) string { return at(s.Settings.SrcdsBotTeamComp, seat) },
		func(s State, v string) State {
			s.Settings.SrcdsBotTeamComp = withAt(s.Settings.SrcdsBotTeamComp, seat, v)
			return s
		})
}

// seatLoadoutSpec is what one seat carries. It is per seat rather than per class
// so that one engineer can hold the gunslinger and the other the stock wrench.
func seatLoadoutSpec(seat int) Spec {
	return openChoice(fmt.Sprintf("bots.seat.%d.loadout", seat), "Bots",
		fmt.Sprintf("  Seat %d carries", seat+1),
		"The loadout this seat plays. Stock leaves the game's own weapons alone.",
		func(s State, _ Env) []Option { return loadoutOptions(s, at(s.Settings.SrcdsBotTeamComp, seat)) },
		func(s State) string { return at(s.Settings.SrcdsBotSeatLoadouts, seat) },
		func(s State, v string) State {
			s.Settings.SrcdsBotSeatLoadouts = withAt(s.Settings.SrcdsBotSeatLoadouts, seat, v)
			return s
		})
}

/*
	seatNameSpec is the name one seat is given

A choice rather than a box, out of the pool the Names section holds, because a
name the pool does not carry is one the bots could never have drawn: two places
to type a name is two places for it to be spelled differently. The empty option
leaves the seat drawing, which is what every seat does until somebody names one.
*/
func seatNameSpec(seat int) Spec {
	return openChoice(fmt.Sprintf("bots.seat.%d.name", seat), "Bots",
		fmt.Sprintf("  Seat %d is called", seat+1),
		"The name the bot in this seat is given, out of the pool on the Names section. Left to the pool, it draws one as it sits down and keeps it.",
		func(s State, _ Env) []Option {
			out := []Option{{Value: "", Label: "draw from the pool"}}
			for _, name := range botnames.Pool(s.Settings.SrcdsBotNamesExcluded, s.Settings.SrcdsBotNamesAdded) {
				out = append(out, Option{Value: name, Label: name})
			}
			return out
		},
		func(s State) string { return at(s.Settings.SrcdsBotSeatNames, seat) },
		func(s State, v string) State {
			s.Settings.SrcdsBotSeatNames = withAt(s.Settings.SrcdsBotSeatNames, seat, v)
			return s
		})
}

// classAllowedSpec is whether the mod may draw a class at all. The setting is
// the blacklist, so a class the game adds later is drawable by default rather
// than silently missing from every settings file written before it existed.
func classAllowedSpec(class botloadout.Class) Spec {
	return toggle("bots.class."+class.Key+".allowed", "Bots",
		class.Name,
		"Whether the mod may draw this class for a seat that names nobody.",
		"may be drawn",
		func(s State) bool { return !slices.Contains(s.Settings.SrcdsBotClassBlacklist, class.Key) },
		func(s State, allowed bool) State {
			list := slices.DeleteFunc(slices.Clone(s.Settings.SrcdsBotClassBlacklist),
				func(k string) bool { return k == class.Key })
			if !allowed {
				list = append(list, class.Key)
			}
			s.Settings.SrcdsBotClassBlacklist = list
			return s
		})
}

// classLoadoutSpec is what a class carries when a seat did not say.
func classLoadoutSpec(class botloadout.Class) Spec {
	return openChoice("bots.class."+class.Key+".loadout", "Bots",
		"  "+class.Name+" carries",
		"The loadout this class plays when a seat does not name one. Stock leaves the game's own weapons alone.",
		func(s State, _ Env) []Option { return loadoutOptions(s, class.Key) },
		func(s State) string { return s.Settings.SrcdsBotLoadouts[class.Key] },
		func(s State, v string) State {
			loadouts := maps.Clone(s.Settings.SrcdsBotLoadouts)
			if loadouts == nil {
				loadouts = map[string]string{}
			}
			if v == "" {
				delete(loadouts, class.Key)
			} else {
				loadouts[class.Key] = v
			}
			// nil rather than an empty map, so clearing the last entry leaves
			// the settings equal to what they were before the first was added.
			if len(loadouts) == 0 {
				loadouts = nil
			}
			s.Settings.SrcdsBotLoadouts = loadouts
			return s
		})
}

/*
	loadoutSpecs is the editor: one loadout, built a slot at a time

The rows change with the class, because the weapons a slot can hold do and
because the Spy has no primary. That is the second reason Specs is a function of
the state rather than a package variable.
*/
func loadoutSpecs(s State) []Spec {
	const tab = "Loadouts"
	class := s.Draft.Loadout.Class
	if class == "" {
		class = botloadout.Classes[0].Key
	}

	values := make([]string, 0, len(botloadout.Classes))
	labels := make([]string, 0, len(botloadout.Classes))
	for _, c := range botloadout.Classes {
		values = append(values, c.Key)
		labels = append(labels, c.Name)
	}

	specs := []Spec{
		choice("loadout.class", tab, "Class",
			"Who holds this loadout. A loadout belongs to one class, so only that class can pick it.",
			options(values, labels),
			func(s State) string { return s.Draft.Loadout.Class },
			func(s State, v string) State {
				if v == s.Draft.Loadout.Class {
					return s
				}
				// Every slot goes back to stock: a weapon of the class this
				// loadout no longer belongs to is not a choice anybody made.
				s.Draft.Loadout = StockLoadout(v)
				return s
			}),
	}
	if class == "spy" {
		// Spy has no ordinary primary picker, but a prototype may still try a
		// definition in the mod's primary key and observe what TF2 equips.
		specs = append(specs, slotAnyItemSpec("primary"))
	}

	for _, slot := range LoadoutSlots(class) {
		specs = append(specs, slotSpec(class, slot), slotAnyItemSpec(slot))
	}

	return append(specs, loadoutLibrarySpecs(tab)...)
}

// loadoutLibrarySpecs are the rows that name a loadout rather than build one:
// what it is called, and the three things done with a saved one.
func loadoutLibrarySpecs(tab string) []Spec {
	return []Spec{
		text("loadout.name", tab, "Name",
			"What this loadout is called in the menus on the Bots page.",
			"gas runner",
			func(s State) string { return s.Draft.LoadoutName },
			func(s State, v string) State { s.Draft.LoadoutName = v; return s }),

		press("loadout.save", tab, "Save this loadout as",
			"Keeps the weapons above under the name in the box. Saving over a name replaces it."),

		/* Load and remove are choices rather than buttons: both act on one
		   saved loadout by name, and picking it from a list beats typing it
		   into the name box and hoping it matches. Picking is the whole action,
		   so loading writes straight into the draft. */
		openChoice("loadout.load", tab, "Load a loadout",
			"Brings a saved loadout back into the menus above, to change it or to save it under another name.",
			savedLoadoutOptions("keep the loadout above"),
			func(State) string { return "" },
			func(s State, name string) State {
				built, ok := s.Settings.SrcdsBotCustomLoadouts[name]
				if !ok {
					return s
				}
				s.Draft.LoadoutName, s.Draft.Loadout = name, built
				return s
			}),

		openChoice("loadout.remove", tab, "Remove a loadout",
			"Throws one away. A seat still naming it plays stock rather than refusing to load.",
			savedLoadoutOptions("keep them all"),
			func(State) string { return "" },
			func(s State, name string) State {
				if _, ok := s.Settings.SrcdsBotCustomLoadouts[name]; !ok {
					return s
				}
				kept := make(map[string]botloadout.Built, len(s.Settings.SrcdsBotCustomLoadouts)-1)
				for existing, built := range s.Settings.SrcdsBotCustomLoadouts {
					if existing != name {
						kept[existing] = built
					}
				}
				if len(kept) == 0 {
					kept = nil
				}
				s.Settings.SrcdsBotCustomLoadouts = kept
				return s
			}),
	}
}

// slotSpec is the weapon in one slot of the loadout being built. The weapon's
// definition index is what is stored, because that is what the mod's file
// carries and a name is not unique across classes.
func slotSpec(class, slot string) Spec {
	weapons := gamedata.WeaponsFor(class, slot)
	values := []string{""}
	labels := []string{"stock"}
	for _, weapon := range weapons {
		values = append(values, fmt.Sprint(weapon.DefIndex))
		labels = append(labels, weapon.Name)
	}

	return openChoice("loadout.slot."+slot, "Loadouts", "  "+SlotName(slot),
		"The weapon in this slot. Stock leaves the game's own alone.",
		func(s State, _ Env) []Option {
			out := options(values, labels)
			held := s.Draft.Slot(slot)
			if held != botloadout.Stock && !slices.Contains(values, fmt.Sprint(held)) {
				out = append(out, Option{Value: fmt.Sprint(held), Label: botloadout.WeaponName(held) + " (manual)"})
			}
			return out
		},
		func(s State) string {
			if held := s.Draft.Slot(slot); held != botloadout.Stock {
				return fmt.Sprint(held)
			}
			return ""
		},
		func(s State, v string) State {
			if v == "" {
				s.Draft = s.Draft.WithSlot(slot, botloadout.Stock)
				return s
			}
			var defIndex int
			_, _ = fmt.Sscan(v, &defIndex)
			s.Draft = s.Draft.WithSlot(slot, defIndex)
			return s
		})
}

// A raw definition index lets a test team experiment with combinations the
// class-filtered picker does not offer. The defender mod accepts the number;
// whether TF2 equips a cross-class/cross-slot item is an engine limitation to
// verify on the server, not something this editor can guarantee.
func slotAnyItemSpec(slot string) Spec {
	return Spec{
		ID: "loadout.any_item." + slot, Tab: "Loadouts", Kind: Text,
		Label:       "  " + SlotName(slot) + " item definition",
		Help:        "Experimental: enter any positive TF2 item definition index for this slot. Clear for stock. Some incompatible items may be rejected by TF2.",
		Placeholder: "e.g. 997",
		Get: func(s State) string {
			if held := s.Draft.Slot(slot); held != botloadout.Stock {
				return fmt.Sprint(held)
			}
			return ""
		},
		Set: func(s State, raw string) (State, error) {
			if raw == "" {
				s.Draft = s.Draft.WithSlot(slot, botloadout.Stock)
				return s, nil
			}
			index, err := strconv.Atoi(raw)
			if err != nil || index <= 0 {
				return s, fmt.Errorf("item definition must be a positive integer")
			}
			s.Draft = s.Draft.WithSlot(slot, index)
			return s, nil
		},
	}
}

// SlotName is what the pages call a slot.
func SlotName(slot string) string {
	if slot == "pda2" {
		return "Watch"
	}
	return strings.ToUpper(slot[:1]) + slot[1:]
}

// loadoutOptions are the loadouts a class may be handed: stock, the presets the
// mod ships for that class, and whatever has been saved for it here.
func loadoutOptions(s State, class string) []Option {
	out := []Option{{Value: "", Label: "stock"}}
	if class == "" {
		return out
	}
	if known, ok := botloadout.ClassByKey(class); ok {
		for _, loadout := range known.Loadouts {
			out = append(out, Option{Value: loadout.Key, Label: loadout.Label()})
		}
	}
	// The value is the key the library answers to, prefix and all. It was the
	// bare name once, and a seat handed one played stock: the file, the tab
	// and the mod all pass keys around, and a name is not a key.
	for _, name := range slices.Sorted(customLoadoutNames(s, class)) {
		out = append(out, Option{Value: botloadout.CustomKey(name), Label: name + " (saved)"})
	}
	return out
}

// customLoadoutNames are the saved loadouts that belong to one class. A loadout
// built for a Medic is not offered to an Engineer: the weapons would not exist.
func customLoadoutNames(s State, class string) func(func(string) bool) {
	return func(yield func(string) bool) {
		for name, built := range s.Settings.SrcdsBotCustomLoadouts {
			if built.Class == class && !yield(name) {
				return
			}
		}
	}
}

// savedLoadoutOptions is every saved loadout, whatever class it belongs to,
// with a first entry that does nothing. Load and remove both act across the
// classes, unlike the menus that hand one to a seat.
func savedLoadoutOptions(nothing string) func(State, Env) []Option {
	return func(s State, _ Env) []Option {
		out := []Option{{Value: "", Label: nothing}}
		for _, name := range slices.Sorted(mapKeys(s.Settings.SrcdsBotCustomLoadouts)) {
			label := name
			if class, ok := botloadout.ClassByKey(s.Settings.SrcdsBotCustomLoadouts[name].Class); ok {
				label = name + " (" + class.Name + ")"
			}
			out = append(out, Option{Value: name, Label: label})
		}
		return out
	}
}

func teamPresetOptions(s State, _ Env) []Option {
	out := []Option{{Value: "", Label: "the seats below"}}
	for _, name := range slices.Sorted(mapKeys(s.Settings.SrcdsBotTeamPresets)) {
		out = append(out, Option{Value: name, Label: name})
	}
	return out
}

// loadTeamPreset puts a saved team into the seats. Choosing the empty option
// leaves the seats alone, which is what "the seats below" means.
func loadTeamPreset(s State, name string) State {
	if name == "" {
		return s
	}
	team, ok := s.Settings.SrcdsBotTeamPresets[name]
	if !ok {
		return s
	}
	s.Settings = settings.WithBotTeam(s.Settings, team)
	s.Draft.TeamName = name
	return s
}

// at reads one seat out of a list that may be shorter than the seats, which is
// what an unnamed seat looks like in the settings file.
func at(list []string, i int) string {
	if i < 0 || i >= len(list) {
		return ""
	}
	return list[i]
}

/*
	withAt writes one seat, growing the list to reach it and trimming it after

The trailing empties are dropped so that a team with three named seats is three
entries in the settings file rather than six, three of which mean nothing. It
matters because the mod reads the length: a comp of six with three blanks is not
the same instruction as a comp of three.
*/
func withAt(list []string, i int, value string) []string {
	if i < 0 || i >= Seats {
		return list
	}
	next := slices.Clone(list)
	for len(next) <= i {
		next = append(next, "")
	}
	next[i] = value
	for len(next) > 0 && next[len(next)-1] == "" {
		next = next[:len(next)-1]
	}
	// nil rather than an empty slice, so writing a seat back to what it already
	// held leaves the settings equal to what they were rather than merely
	// holding the same values.
	if len(next) == 0 {
		return nil
	}
	return next
}

func mapKeys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}
