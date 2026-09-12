package form

import (
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

/*
	Every setting is on a page, and this is what replaced uiparity.

That package compared the window's source with the terminal's by regular
expression. It could only ever ask whether the two files wrote the same struct
fields, never whether they told the player the same thing, and they did not:
nine rows had different help text and nobody had chosen one of the differences.
It also could not answer the question that actually matters, which is not
"do the two agree" but "is anything missing".

There is one list now, so agreement is not a thing that can fail. What can fail
is a setting nobody put on a page: added to settings.Settings, saved and loaded
correctly, and unreachable from any interface. This walks the struct and asks.

notOnAPage is the decided exceptions, each with the reason. It is short and it
is meant to stay short: an entry is a decision, not a way to quieten this.
*/
var notOnAPage = map[string]string{
	"SrcdsRconPw":         "generated per run and never shown; a player who types one has no use for it",
	"SrcdsMaxPlayers":     "derived from the bot team size and the reach, never asked",
	"CommunityContentDir": "the browser imports into the launcher's internal asset cache; old files may still override its location",
	"SrcdsStartMission": "written by the start mission row on the Missions page, which owns both this " +
		"and MvmStartMission so the seed and the server cannot disagree",
	"SrcdsStartMap":       "read from files written before 1.3 and never written again",
	"SrcdsLanLegacy":      "the srcds_lan boolean SrcdsReach replaced, read once and never written",
	"SrcdsBotTeamPresets": "written by the save and remove team buttons, not by a row",
	"APHost":              "written by the room address row, which parses one line into three fields",
	"APPort":              "written by the room address row",
	"APTls":               "written by the room address row",
	"MetricsPort": "the launcher's own metrics listener, on a fixed port. No interface has ever " +
		"offered it and nothing has asked for one; it is here to be changed in the file.",
}

func TestEverySettingIsOnAPageOrSaysWhyNot(t *testing.T) {
	written := writtenFields(t)
	typ := reflect.TypeFor[settings.Settings]()

	for field := range typ.Fields() {
		name := field.Name
		if slices.Contains(written, name) {
			if _, excused := notOnAPage[name]; excused {
				t.Errorf("%s is on a page and also listed as not on one; drop the exception", name)
			}
			continue
		}
		if reason := notOnAPage[name]; reason == "" {
			t.Errorf("%s is a setting no row writes, and notOnAPage does not say why", name)
		}
	}

	for name := range notOnAPage {
		if _, ok := typ.FieldByName(name); !ok {
			t.Errorf("notOnAPage names %s, which is not a setting any more", name)
		}
	}
}

/*
	writtenFields is which settings fields the specs actually write

Discovered rather than declared. A Spec could carry the name of the struct field
it writes, but then the name would be a second copy of what Set already does and
it would go stale the first time a Set was edited and the label was not. So each
spec is handed a value it does not already hold and the state is compared before
and after: whatever moved is what that spec writes.
*/
func writtenFields(t *testing.T) []string {
	t.Helper()
	base := populated()
	env := Env{}
	typ := reflect.TypeOf(base.Settings)

	var names []string
	for _, spec := range Specs(base, env) {
		if spec.Set == nil {
			continue
		}
		other, ok := differentValue(spec, base, env)
		if !ok {
			continue
		}
		next, err := spec.Set(base, other)
		if err != nil {
			t.Errorf("%s refused %q, a value it offers: %v", spec.ID, other, err)
			continue
		}
		before, after := reflect.ValueOf(base.Settings), reflect.ValueOf(next.Settings)
		for i := range typ.NumField() {
			if reflect.DeepEqual(before.Field(i).Interface(), after.Field(i).Interface()) {
				continue
			}
			if name := typ.Field(i).Name; !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	return names
}

/*
	populated is a state where every row has somewhere to move to

The defaults leave several rows with one option and nothing else: no seat names
a class, so no seat has a loadout to pick beyond stock, and no team has been
saved. A row that cannot be changed cannot be discovered, and the discovery
below would report the setting as unreachable when it is only unreachable from
an empty team.

So the seats are given classes and a loadout is saved, which is the state a
player is in by the time those rows matter.
*/
func populated() State {
	s := NewState(settings.Defaults())
	s.Settings.SrcdsBotTeamComp = []string{"engineer", "medic"}
	s.Settings.SrcdsBotCustomLoadouts = map[string]botloadout.Built{
		"gas runner": StockLoadout("engineer"),
	}
	s.Settings.SrcdsBotTeamPresets = map[string]settings.BotTeam{
		"two engineers": {},
	}
	return s
}

// differentValue is a value the spec takes and does not already hold, so that
// setting it moves something.
func differentValue(spec Spec, s State, env Env) (string, bool) {
	current := spec.Get(s)
	switch spec.Kind {
	case Toggle:
		return strconv.FormatBool(current != "true"), true
	case Choice:
		for _, o := range spec.Options(s, env) {
			if o.Value != current {
				return o.Value, true
			}
		}
		return "", false
	case Number:
		low, high := spec.Bounds(s, env)
		if strconv.Itoa(low) != current {
			return strconv.Itoa(low), true
		}
		if strconv.Itoa(high) != current {
			return strconv.Itoa(high), true
		}
		return "", false
	default:
		return current + "-moved", true
	}
}
