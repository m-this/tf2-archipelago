package tui

import (
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// The page builds a loadout and saves it under the name typed, and the saved
// one is then something a seat can pick.
func TestBuildingALoadoutAndPickingIt(t *testing.T) {
	m := screen(t)
	m.Update(key(","))
	form := m.form

	form.draft = stockDraft("pyro")
	form.draft.name = "Gas runner"
	form.draft.built.Primary = 594
	form.draft.built.Second = 1180

	// Save is an action field, so it answers with work rather than a value.
	saved := false
	for _, row := range form.loadoutFields() {
		action, ok := row.(*actionField)
		if !ok || !strings.HasPrefix(action.Label(), "Save this loadout") {
			continue
		}
		action.run()
		saved = true
	}
	if !saved {
		t.Fatal("the page has no save action")
	}

	built, found := form.edited.SrcdsBotCustomLoadouts["Gas runner"]
	if !found {
		t.Fatalf("nothing saved: %v", form.edited.SrcdsBotCustomLoadouts)
	}
	if built.Class != "pyro" || built.Primary != 594 || built.Second != 1180 {
		t.Errorf("saved %+v", built)
	}

	// And the Pyro's menus now offer it, while another class's do not.
	pyro, _ := botloadout.ClassByKey("pyro")
	if !hasChoice(form.library().Choices(pyro), "Gas runner") {
		t.Error("the pyro cannot pick the loadout it was built for")
	}
	medic, _ := botloadout.ClassByKey("medic")
	if hasChoice(form.library().Choices(medic), "Gas runner") {
		t.Error("the medic was offered a pyro's loadout")
	}
}

// Changing the class clears the weapons: a Scattergun is not a choice anybody
// made for a Heavy.
func TestChangingTheClassClearsTheSlots(t *testing.T) {
	m := screen(t)
	m.Update(key(","))
	form := m.form

	form.draft = stockDraft("scout")
	form.draft.built.Primary = 448

	for _, row := range form.loadoutFields() {
		choice, ok := row.(*choiceField)
		if !ok || choice.Label() != "Class" {
			continue
		}
		// The Soldier is second in the class list.
		choice.apply(1)
	}
	if form.draft.built.Primary != botloadout.Stock {
		t.Errorf("the primary survived a class change: %+v", form.draft.built)
	}
	if form.draft.class != "soldier" {
		t.Errorf("class = %q", form.draft.class)
	}
}

// The Spy has no primary, and does have a watch. The sapper is not offered.
func TestTheSpyHasTheSlotsTheSpyHas(t *testing.T) {
	got := loadoutSlots("spy")
	for _, want := range []string{"secondary", "melee", "pda2"} {
		if !slices.Contains(got, want) {
			t.Errorf("the spy has no %s: %v", want, got)
		}
	}
	if slices.Contains(got, "primary") {
		t.Errorf("the spy was given a primary: %v", got)
	}
	// The sapper is in the mod's pools and has no key in the loadout file, so
	// offering it would be a menu entry the mod never reads.
	if slices.Contains(got, "building") {
		t.Errorf("the sapper was offered: %v", got)
	}
	if slices.Contains(loadoutSlots("scout"), "pda2") {
		t.Error("the scout was given a watch")
	}
}

func hasChoice(choices []botloadout.Loadout, name string) bool {
	for _, choice := range choices {
		if choice.Name == name {
			return true
		}
	}
	return false
}

// The Bot Switcher's key says it changes the team on the Bots page, so it opens
// there rather than on whatever tab happens to be first.
func TestShowTabOpensOnTheBotsPage(t *testing.T) {
	form := newSettingsForm(settings.Settings{}, settingsDeps{})
	form.showTab("Bots")
	if got := form.tabs[form.tab].title; got != "Bots" {
		t.Fatalf("opened on %q, want Bots", got)
	}
}

// A title no tab carries leaves the form where it was, rather than pushing it
// to some other page.
func TestShowTabIgnoresAnUnknownTitle(t *testing.T) {
	form := newSettingsForm(settings.Settings{}, settingsDeps{})
	form.showTab("Bots")
	form.showTab("no such tab")
	if got := form.tabs[form.tab].title; got != "Bots" {
		t.Fatalf("moved to %q, want it to stay on Bots", got)
	}
}
