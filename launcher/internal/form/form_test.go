package form

import (
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func base() State { return NewState(settings.Defaults()) }

/*
	Every spec is complete, and this is what replaces uiparity

The two interfaces used to be compared by reading their source with a regular
expression, which could only ever ask whether they wrote the same struct fields.
It could not ask whether they said the same thing, and they did not: nine rows
had different help text and nobody had chosen a single one of the differences.

There is one list now, so there is nothing to compare. What is left to check is
that the list is filled in, which no regular expression over two files could
have told anybody either.
*/
func TestEverySpecIsComplete(t *testing.T) {
	seen := make(map[string]bool)
	for _, spec := range Specs(base(), Env{}) {
		if spec.ID == "" {
			t.Fatalf("a spec has no ID: %+v", spec.Label)
		}
		if seen[spec.ID] {
			t.Errorf("%s is declared twice, so one of them is unreachable", spec.ID)
		}
		seen[spec.ID] = true

		if spec.Label == "" {
			t.Errorf("%s has no label", spec.ID)
		}
		if spec.Help == "" {
			t.Errorf("%s has no help, and a name alone does not tell anybody what it is for", spec.ID)
		}
		if !strings.Contains(spec.ID, ".") {
			t.Errorf("%s is not scoped: an ID is <page>.<setting> so two pages may both have a share", spec.ID)
		}
		if got := spec.Tab; !contains(Tabs, got) {
			t.Errorf("%s is on the page %q, which is not one of the pages", spec.ID, got)
		}

		switch spec.Kind {
		case Action, Confirm:
			if spec.Get != nil || spec.Set != nil {
				t.Errorf("%s is a %s and has nothing to read or write", spec.ID, spec.Kind)
			}
			if spec.Hint != "" {
				t.Errorf("%s uses %q instead of its action label", spec.ID, spec.Hint)
			}
		default:
			if spec.Get == nil || spec.Set == nil {
				t.Errorf("%s is a %s and needs both a Get and a Set", spec.ID, spec.Kind)
			}
		}

		if spec.Kind == Number && spec.Bounds == nil {
			t.Errorf("%s is a number with no bounds, so nothing stops a seed asking for -1%%", spec.ID)
		}
		if spec.Kind == Choice && spec.Options == nil {
			t.Errorf("%s is a choice with no options", spec.ID)
		}
	}
}

/*
	What a row shows is what a row saves

Get and Set are declared separately, which is the one place a spec can be wrong
in a way that compiles: reading MvmWeaponBuffPct and writing
MvmWeaponBuffStackChance is two correct-looking lines. Writing back what was
read has to leave the settings alone, and it does not for a mismatched pair
unless the two fields happen to hold the same value, which the defaults make
sure they do not here.
*/
func TestGetAndSetAreTheSameField(t *testing.T) {
	for _, spec := range Specs(base(), Env{}) {
		if spec.Get == nil {
			continue
		}
		s := base()
		next, err := spec.Set(s, spec.Get(s))
		if err != nil {
			t.Errorf("%s refused the value it just reported: %v", spec.ID, err)
			continue
		}
		if !reflect.DeepEqual(next, s) {
			t.Errorf("%s wrote a different field than it read: settings changed on a no-op set", spec.ID)
		}
	}
}

// A number takes its own bounds and refuses a step past either. The floor and
// the ceiling are the two values most likely to be off by one, so both are
// tried rather than a value from the middle.
func TestNumbersTakeTheirBoundsAndRefusePastThem(t *testing.T) {
	s := base()
	for _, spec := range Specs(base(), Env{}) {
		if spec.Kind != Number {
			continue
		}
		low, high := spec.Bounds(s, Env{})
		if low >= high {
			t.Errorf("%s has a floor of %d and a ceiling of %d", spec.ID, low, high)
			continue
		}
		for _, ok := range []int{low, high} {
			if _, err := spec.Set(s, strconv.Itoa(ok)); err != nil {
				t.Errorf("%s refused %d, which is inside %d to %d: %v", spec.ID, ok, low, high, err)
			}
		}
		for _, bad := range []int{low - 1, high + 1} {
			if _, err := spec.Set(s, strconv.Itoa(bad)); err == nil {
				t.Errorf("%s took %d, which is outside %d to %d", spec.ID, bad, low, high)
			}
		}
		if _, err := spec.Set(s, "eight"); err == nil {
			t.Errorf("%s took a word for a number", spec.ID)
		}
	}
}

// A choice takes each of its options and nothing else. The refusal matters more
// than the acceptance: the web interface will send whatever a browser posts.
func TestChoicesTakeTheirOptionsAndNothingElse(t *testing.T) {
	s := base()
	for _, spec := range Specs(base(), Env{}) {
		if spec.Kind != Choice {
			continue
		}
		options := spec.Options(s, Env{})
		/* One option is a real state for a row whose answers depend on
		   another: a seat that names no class has only stock to carry, and a
		   run with no saved team has only the seats on screen. Nought is not:
		   a row nobody can answer is a row that should not be drawn. */
		if len(options) == 0 {
			t.Errorf("%s offers no options at all", spec.ID)
			continue
		}
		for _, o := range options {
			if o.Label == "" {
				t.Errorf("%s offers %q with no label", spec.ID, o.Value)
			}
			if _, err := spec.Set(s, o.Value); err != nil {
				t.Errorf("%s refused its own option %q: %v", spec.ID, o.Value, err)
			}
		}
		if _, err := spec.Set(s, "not-an-option"); err == nil {
			t.Errorf("%s took a value it never offered", spec.ID)
		}
	}
}

func TestRewardDisabledChoicesHaveTheRightMeaning(t *testing.T) {
	for _, row := range []struct {
		id, label string
	}{
		{"rewards.mission_tickets", "Disabled - All Unlocked"},
		{"rewards.class_unlocks", "Disabled - All Unlocked"},
		{"rewards.weapon_slots", "Disabled - All Unlocked"},
		{"rewards.weapon_buffs", "Disabled - No Buffs"},
	} {
		t.Run(row.id, func(t *testing.T) {
			var found bool
			for _, spec := range Specs(base(), Env{}) {
				if spec.ID != row.id {
					continue
				}
				found = true
				var label string
				for _, option := range spec.Options(base(), Env{}) {
					if option.Value == "disabled" {
						label = option.Label
					}
				}
				if label != row.label {
					t.Errorf("disabled is labeled %q, want %q", label, row.label)
				}
				next, err := Apply(base(), Env{}, Change{Field: row.id, Value: "disabled"})
				if err != nil {
					t.Fatal(err)
				}
				if got := spec.Get(next); got != "disabled" {
					t.Errorf("saved %q instead of disabled", got)
				}
			}
			if !found {
				t.Fatal("reward row is missing")
			}
		})
	}
}

// A toggle reads back as the bool it was set to, in the spelling ParseBool and
// FormatBool agree on, because the value crosses a socket as text.
func TestTogglesRoundTripAsText(t *testing.T) {
	for _, spec := range Specs(base(), Env{}) {
		if spec.Kind != Toggle {
			continue
		}
		for _, want := range []string{"true", "false"} {
			next, err := spec.Set(base(), want)
			if err != nil {
				t.Errorf("%s refused %q: %v", spec.ID, want, err)
				continue
			}
			if got := spec.Get(next); got != want {
				t.Errorf("%s was set to %q and reads back %q", spec.ID, want, got)
			}
		}
		if _, err := spec.Set(base(), "maybe"); err == nil {
			t.Errorf("%s took a value that is neither", spec.ID)
		}
	}
}

func TestApplyRefusesWhatItCannotWrite(t *testing.T) {
	if _, err := Apply(base(), Env{}, Change{Field: "rewards.no_such_thing", Value: "1"}); err == nil {
		t.Error("Apply took a change to a setting that does not exist")
	}
	if _, err := Apply(base(), Env{}, Change{Field: "rewards.traps", Value: "101"}); err == nil {
		t.Error("Apply took a trap share of 101%")
	}
	next, err := Apply(base(), Env{}, Change{Field: "rewards.traps", Value: "12"})
	if err != nil {
		t.Fatalf("Apply refused a trap share of 12%%: %v", err)
	}
	if next.Settings.MvmTrapPct != 12 {
		t.Errorf("the trap share is %d, wanted 12", next.Settings.MvmTrapPct)
	}
}

// A refused change leaves the caller's settings alone. Apply takes and returns
// by value so there is no half-written state to undo, and this is the assertion
// that says so rather than the comment.
func TestARefusedChangeChangesNothing(t *testing.T) {
	s := base()
	if _, err := Apply(s, Env{}, Change{Field: "rewards.traps", Value: "-5"}); err == nil {
		t.Fatal("Apply took a trap share of -5%")
	}
	if !reflect.DeepEqual(s, base()) {
		t.Error("a refused change wrote to the settings it was given")
	}
}

func TestPoolCannotLeaveAnExcludedStartMission(t *testing.T) {
	s := base()
	s = startMission(s, "mvm_decoy")
	next, err := Apply(s, Env{}, Change{Field: "missions.pool.mvm_decoy", Value: "false"})
	if err != nil {
		t.Fatal(err)
	}
	if next.Settings.MvmStartMission != "" {
		t.Errorf("excluded start mission remains %q", next.Settings.MvmStartMission)
	}
}

func TestDifficultyCannotLeaveAStartBelowItsFloor(t *testing.T) {
	s := base()
	s = startMission(s, "mvm_decoy")
	next, err := Apply(s, Env{}, Change{Field: "run.tier", Value: "advanced"})
	if err != nil {
		t.Fatal(err)
	}
	if next.Settings.MvmStartMission != "" {
		t.Errorf("normal start mission remains under the advanced floor: %q", next.Settings.MvmStartMission)
	}
}

func TestOpeningSettingsClearsAnAlreadyExcludedStart(t *testing.T) {
	s := settings.Defaults()
	s.MvmStartMission = "mvm_decoy"
	s.MvmExcludedMissions = append(s.MvmExcludedMissions, "mvm_decoy")
	if got := NewState(s).Settings.MvmStartMission; got != "" {
		t.Errorf("opening settings kept excluded start mission %q", got)
	}
}

/*
	The Model survives a round trip through JSON

This is the whole reason a Field holds data and a Spec holds the closures. The
web interface gets the Model over a socket, so anything a closure would have
answered has to already be in it. A Model that does not come back the same is
one the browser draws differently from the window.
*/
func TestTheModelSurvivesJSON(t *testing.T) {
	want := Build(base(), Env{})
	body, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("the model does not marshal: %v", err)
	}
	var got Model
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("the model does not unmarshal: %v", err)
	}

	if len(got.Tabs) != len(want.Tabs) {
		t.Fatalf("%d pages went out and %d came back", len(want.Tabs), len(got.Tabs))
	}
	for i, tab := range want.Tabs {
		if got.Tabs[i].Title != tab.Title || got.Tabs[i].Intro != tab.Intro {
			t.Errorf("page %d came back as %q", i, got.Tabs[i].Title)
		}
		if len(got.Tabs[i].Fields) != len(tab.Fields) {
			t.Errorf("page %q sent %d rows and got %d back", tab.Title, len(tab.Fields), len(got.Tabs[i].Fields))
			continue
		}
		for j, field := range tab.Fields {
			if got.Tabs[i].Fields[j].ID != field.ID {
				t.Errorf("page %q row %d came back as %q", tab.Title, j, got.Tabs[i].Fields[j].ID)
			}
		}
	}
}

// Build shows the value the settings hold, not the default, and it moves when
// the settings do.
func TestBuildShowsTheSettingsItWasGiven(t *testing.T) {
	s := base()
	s.Settings.MvmTrapPct = 37
	s.Settings.MvmCashRewards = false

	model := Build(s, Env{})
	traps, ok := model.Field("rewards.traps")
	if !ok {
		t.Fatal("the trap share is not on the screen")
	}
	if traps.Value != "37" {
		t.Errorf("the trap share shows %q, wanted 37", traps.Value)
	}
	if traps.Low != 0 || traps.High != 100 {
		t.Errorf("the trap share is bounded %d to %d, wanted 0 to 100", traps.Low, traps.High)
	}

	cash, _ := model.Field("rewards.cash")
	if cash.Value != "false" {
		t.Errorf("cash rewards show %q, wanted false", cash.Value)
	}
	if cash.Label != "Cash rewards" || cash.Hint != "enabled" || cash.HintOff != "disabled" {
		t.Errorf("cash rewards are labelled %+v, wanted an enabled/disabled Cash rewards toggle", cash)
	}

	hook, ok := model.Field("rewards.server_settings")
	if !ok {
		t.Fatal("the Grappling Hook setting is not on the screen")
	}
	if hook.Label != "Grappling Hook" || hook.Hint != "enabled" || hook.HintOff != "disabled" {
		t.Errorf("the Grappling Hook is labelled %+v, wanted an enabled/disabled Grappling Hook toggle", hook)
	}
}

// An empty page is dropped rather than drawn, and the pages that are drawn keep
// the order Tabs declares.
func TestBuildDropsEmptyPagesAndKeepsTheOrder(t *testing.T) {
	model := Build(base(), Env{})
	if len(model.Tabs) == 0 {
		t.Fatal("the screen has no pages")
	}

	var last int
	for _, tab := range model.Tabs {
		if len(tab.Fields) == 0 {
			t.Errorf("the page %q is drawn with no rows on it", tab.Title)
		}
		at := indexOf(Tabs, tab.Title)
		if at < last {
			t.Errorf("the page %q comes after a page Tabs puts later", tab.Title)
		}
		last = at
	}
}

// The paragraph above the rows is on the page it explains. Balancing is the one
// that needs it: "Robot health" means nothing until the player is told Valve
// tunes every wave for six defenders.
func TestTheBalancingPageKeepsItsIntro(t *testing.T) {
	for _, tab := range Build(base(), Env{}).Tabs {
		if tab.Title != "Balancing" {
			continue
		}
		if !strings.Contains(tab.Intro, "six defenders") {
			t.Errorf("the Balancing page's intro is %q", tab.Intro)
		}
		return
	}
	t.Fatal("there is no Balancing page")
}

func contains(list []string, want string) bool { return indexOf(list, want) >= 0 }

func indexOf(list []string, want string) int {
	for i, got := range list {
		if got == want {
			return i
		}
	}
	return -1
}

// The menu's value is what a seat is handed and what the library is asked for,
// so it has to be the key. It was the name once, and the library answered stock.
func TestASavedLoadoutIsOfferedByItsKey(t *testing.T) {
	s := NewState(settings.Defaults())
	s.Settings.SrcdsBotCustomLoadouts = map[string]botloadout.Built{
		"gas runner": {Class: "pyro", Primary: 40, Second: botloadout.Stock, Melee: botloadout.Stock, PDA2: botloadout.Stock},
	}
	options := loadoutOptions(s, "pyro")
	want := botloadout.CustomKey("gas runner")
	if !slices.ContainsFunc(options, func(o Option) bool { return o.Value == want }) {
		t.Fatalf("the Pyro's menu offers %v, want a value of %q", options, want)
	}
}

func TestTrapRowsPickTheTraps(t *testing.T) {
	s := base()
	if !slices.Equal(s.Settings.MvmTraps, []string{"team_jarate"}) {
		t.Fatalf("the default traps are %v, want team_jarate alone", s.Settings.MvmTraps)
	}
	steps := []struct {
		field, value string
		want         []string
	}{
		{"rewards.trap.team_mad_milk", "true", []string{"team_jarate", "team_mad_milk"}},
		{"rewards.trap.team_jarate", "false", []string{"team_mad_milk"}},
		{"rewards.trap.team_mad_milk", "false", []string{"team_jarate"}},
	}
	for _, step := range steps {
		next, err := Apply(s, Env{}, Change{Field: step.field, Value: step.value})
		if err != nil {
			t.Fatalf("%s=%s: %v", step.field, step.value, err)
		}
		s = next
		if got := settings.TrapKeys(s.Settings); !slices.Equal(got, step.want) {
			t.Errorf("after %s=%s the seed draws %v, want %v", step.field, step.value, got, step.want)
		}
	}
}
