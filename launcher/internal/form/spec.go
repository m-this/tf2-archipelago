package form

import (
	"fmt"
	"strconv"
)

/*
Spec is one row, declared once.

It holds closures where Field holds resolved data, because a Spec lives in the
process that owns the state and a Field may be on its way to a browser. Get and
Set are the whole binding: nothing else in this package knows which struct field
a row writes, which is what keeps a new setting to one declaration rather than
one per interface.

Set returns an error rather than clamping. A number quietly corrected under the
cursor is one the player never sees, and the console prompts have refused out of
range rather than clamped since they were written.
*/
type Spec struct {
	ID   string
	Tab  string
	Kind Kind

	/*
		Group is the section of the page this row belongs to, for an interface
		with room to give each section a page of its own.

		The window makes them the sub-tabs the Bots page has always had: six
		seats, nine classes and two cosmetic ticks are one long list on a
		terminal and a wall in a window. The terminal ignores it and draws the
		rows in order, which is the same order either way. Empty means the row
		sits directly on its page, which is most of them.
	*/
	Group string

	/*
		Bar marks an Action about the whole window rather than about the page it
		was declared on, so an interface with a button bar puts it there.

		Debug logs, Repair and Reset settings are the three. None of them edits a
		setting and none belongs to the Game server page in particular; they were
		along the bottom of the window until the rows moved here, and a player
		looking for the debug bundle looked at the bottom of the window.
	*/
	Bar bool

	Label string
	Help  string

	Placeholder string
	Hint        string
	HintOff     string
	Warning     string

	// Browse marks a Text row that names a folder, so an interface with a file
	// picker offers one. The terminal has none and ignores it, which is the
	// right kind of difference: the row is the same row either way.
	Browse bool

	/*
		Deferred marks a row whose value is parsed rather than stored.

		The room address is the one. It is typed a character at a time and only
		the finished string is an address, so a row that refused everything that
		did not parse could never be typed into at all. A deferred row keeps
		what was typed and reports the parse error without dropping the
		character; Save is where an address that never became one is refused.
	*/
	Deferred bool

	// Bounds is a Number's floor and ceiling. It takes the state because some
	// ceilings depend on another answer.
	Bounds func(State, Env) (low, high int)

	// Options are a Choice's answers. It takes the state and the environment
	// for a list that is not fixed: the missions a tier leaves, the loadouts
	// that have been named, the community packs whose ZIPs are actually there.
	Options func(State, Env) []Option

	// Get reads the row's value as text. Nil for Action and Confirm.
	Get func(State) string

	// Set writes it back. Nil for Action and Confirm, whose Change is
	// dispatched by ID rather than applied to the state.
	Set func(State, string) (State, error)

	// Unavailable says why this row cannot be used now, or returns an empty
	// string when it can. A row is shown either way: hiding it leaves the
	// player hunting for a setting the documentation says exists.
	Unavailable func(State, Env) string
}

/*
Env is what the specs need that neither the settings nor the draft hold.

The launcher's answers depend on the machine as well as the file: which
community asset packs have a valid ZIP on disk, where the Archipelago app would
be found if the folder is left blank. A spec asks this rather than reading the
disk itself, so Build stays a pure function of its two arguments and a test
hands it whatever it likes.
*/
type Env struct {
	// CommunityAvailable names the packs with a valid local ZIP, never merely
	// the ones checked. A mission row for a pack that cannot be used is a row
	// that fails at run time instead of at the tick.
	CommunityAvailable []string

	// ServerModsReady names managed server mods whose pinned files and install
	// stamp were found. Platform is GOOS; tests may leave it blank for the
	// normal platform-neutral form.
	ServerModsReady []string
	Platform        string

	// AppDirDefault is where the Archipelago app is looked for when the setting
	// is blank. Shown as the placeholder, never written to the settings.
	AppDirDefault string
}

// Tabs are the pages, in the order they are offered. The order is here rather
// than derived from the specs so that moving a setting between pages does not
// silently move a page.
var Tabs = []string{
	"Player options",
	"Rewards",
	"Balancing",
	"Missions",
	"Archipelago room",
	"Game server",
	"Bots",
	"Loadouts",
	"Networking",
}

/*
	Nested says which page a page is a section of, for an interface with room to

nest one inside another.

Loadouts is a page of the Bots page: a loadout is built there and then handed to
a seat or a class, and the two were one tab with sub-tabs until the rows moved
here. The terminal has no room to nest and draws every page side by side, which
is what it has always done, so it ignores this.
*/
var Nested = map[string]string{
	"Loadouts": "Bots",
}

// Intros are the paragraphs above the rows, for the pages whose rows do not
// explain themselves on their own.
var Intros = map[string]string{
	"Balancing": "Valve tunes every wave for six defenders. This takes the robots down for a team that is short of them, and applies equally regardless of how many humans are playing.",
	"Player options": "These are the options the Archipelago website calls player options. They go in tf2.yaml, which the seed is generated from. " +
		"The Missions page picks which missions the run may draw.",
	"Loadouts": "A loadout is a class and the weapons it carries. Build one here, name it, then hand it to a seat on the Team page or to a class on the Classes page.",
}

// text declares a line of text.
func text(id, tab, label, help, placeholder string, get func(State) string, set func(State, string) State) Spec {
	return Spec{
		ID: id, Tab: tab, Kind: Text, Label: label, Help: help, Placeholder: placeholder,
		Get: get,
		Set: func(s State, v string) (State, error) { return set(s, v), nil },
	}
}

// secret declares a line of text that is never shown back, only replaced.
func secret(id, tab, label, help, placeholder string, get func(State) string, set func(State, string) State) Spec {
	spec := text(id, tab, label, help, placeholder, get, set)
	spec.Kind = Password
	return spec
}

// folder declares a line of text naming a directory, so an interface with a
// file picker offers one beside it.
func folder(id, tab, label, help, placeholder string, get func(State) string, set func(State, string) State) Spec {
	spec := text(id, tab, label, help, placeholder, get, set)
	spec.Browse = true
	return spec
}

// number declares a whole number with fixed bounds. A ceiling that depends on
// another answer is declared with Bounds instead.
func number(id, tab, label, help string, low, high int, get func(State) int, set func(State, int) State) Spec {
	spec := boundedNumber(id, tab, label, help, func(State, Env) (int, int) { return low, high }, get, set)
	return spec
}

/*
	boundedNumber is a number whose floor or ceiling depends on another answer

The mission count is the one that needs it: its ceiling is however many missions
the chosen tier leaves in the pool, so raising the tier lowers the ceiling. The
bounds are resolved when the screen is built and refused against when a change
arrives, which is why they are a function of the state rather than constants.
*/
func boundedNumber(id, tab, label, help string, bounds func(State, Env) (int, int), get func(State) int, set func(State, int) State) Spec {
	return Spec{
		ID: id, Tab: tab, Kind: Number, Label: label, Help: help,
		Bounds: bounds,
		Get:    func(s State) string { return strconv.Itoa(get(s)) },
		Set: func(s State, v string) (State, error) {
			n, err := strconv.Atoi(v)
			if err != nil {
				return s, fmt.Errorf("%s takes a whole number, not %q", label, v)
			}
			low, high := bounds(s, Env{})
			if n < low || n > high {
				return s, fmt.Errorf("%s is between %d and %d, not %d", label, low, high, n)
			}
			return set(s, n), nil
		},
	}
}

// toggle declares a yes or no. on is what the box says next to the tick, which
// is the label of the thing being switched on and not a repeat of the row.
func toggle(id, tab, label, help, on string, get func(State) bool, set func(State, bool) State) Spec {
	return Spec{
		ID: id, Tab: tab, Kind: Toggle, Label: label, Help: help, Hint: on,
		Get: func(s State) string { return strconv.FormatBool(get(s)) },
		Set: func(s State, v string) (State, error) {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return s, fmt.Errorf("%s is on or off, not %q", label, v)
			}
			return set(s, b), nil
		},
	}
}

// twoWayToggle is a toggle that says a different thing when it is off, for a
// row that is read down a long list rather than looked at on its own.
func twoWayToggle(id, tab, label, help, on, off string, get func(State) bool, set func(State, bool) State) Spec {
	spec := toggle(id, tab, label, help, on, get, set)
	spec.HintOff = off
	return spec
}

// choice declares one answer out of a fixed list.
func choice(id, tab, label, help string, options []Option, get func(State) string, set func(State, string) State) Spec {
	return openChoice(id, tab, label, help, func(State, Env) []Option { return options }, get, set)
}

// openChoice is a choice whose answers depend on the state: which missions the
// packs on disk make available, which loadouts have been named, which weapons
// the drafted class can hold.
func openChoice(id, tab, label, help string, options func(State, Env) []Option, get func(State) string, set func(State, string) State) Spec {
	return Spec{
		ID: id, Tab: tab, Kind: Choice, Label: label, Help: help,
		Options: options,
		Get:     get,
		Set: func(s State, v string) (State, error) {
			for _, o := range options(s, Env{}) {
				if o.Value == v {
					return set(s, v), nil
				}
			}
			return s, fmt.Errorf("%s has no option %q", label, v)
		},
	}
}

// press declares a button: what it is called, what it is for, and nothing about
// what it does. The work is the interface's, because it ends in that
// interface's own idea of a command.
func press(id, tab, label, help string) Spec {
	return Spec{ID: id, Tab: tab, Kind: Action, Label: label, Help: help}
}

// confirm declares a button that cannot be taken back. The window asks with a
// message box and the terminal by wanting a second Enter, so neither goes off
// under a finger that was scrolling.
func confirm(id, tab, label, help, warning string) Spec {
	spec := press(id, tab, label, help)
	spec.Kind = Confirm
	spec.Warning = warning
	return spec
}

// importance is the same question asked of four different unlocks, so it is
// written once. Progression gates the run; useful only widens it.
func importance(id, label, help string, get func(State) string, set func(State, string) State) Spec {
	return choice(id, "Rewards", label, help, []Option{
		{Value: "useful", Label: "Useful"},
		{Value: "progression", Label: "Required for progression"},
	}, get, set)
}

// options turns parallel value and label lists into a Choice's answers, which
// is the shape runshape and settings already hand out.
func options(values, labels []string) []Option {
	out := make([]Option, 0, len(values))
	for i, value := range values {
		label := value
		if i < len(labels) {
			label = labels[i]
		}
		out = append(out, Option{Value: value, Label: label})
	}
	return out
}

// inGroup stamps a section on a run of rows. Declared beside the rows rather
// than on each helper, because a group is a property of where a row sits and
// not of what it holds.
func inGroup(group string, specs ...Spec) []Spec {
	for i := range specs {
		specs[i].Group = group
	}
	return specs
}

// onBar marks a row as belonging to the window's button bar.
func onBar(spec Spec) Spec {
	spec.Bar = true
	return spec
}
