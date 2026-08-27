//go:build windows

package gui

import (
	"slices"
	"strings"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
)

/* botLoadoutEditor is the Loadouts page: one editor, not a list.
 *
 * Pick a class, pick a weapon per slot, name it and Save. The saved ones then
 * appear at the bottom of every loadout menu on the Team and Classes pages,
 * for that class only, because a Medic cannot hold a Gunslinger.
 *
 * Load and Save read and write every menu at once, so they need all of them in
 * one place. That is the same reason botTeamEditor exists.
 */
type botLoadoutEditor struct {
	classBox *walk.ComboBox
	// One field per slot the mod's loadout file carries, which is four:
	// writeSlots names primary, secondary, melee and pda2 and nothing else.
	primaryBox *walk.ComboBox
	secondBox  *walk.ComboBox
	meleeBox   *walk.ComboBox
	watchBox   *walk.ComboBox
	nameBox    *walk.LineEdit
	savedBox   *walk.ComboBox

	/* built is the loadouts as they stand, and keep puts them on disk.
	 *
	 * A loadout somebody named outlives a Cancel, for the same reason a saved
	 * team does: naming one and then closing the dialog because a port was a
	 * typo is not asking for the loadout to be forgotten.
	 */
	built map[string]botloadout.Built
	keep  func(map[string]botloadout.Built)

	// changed is the menus on the other pages, which list what this page
	// holds. Saving a loadout is asking to assign it, so they are refilled on
	// the spot rather than at the next open of the dialog.
	changed func()
}

func newBotLoadoutEditor(built map[string]botloadout.Built, keep func(map[string]botloadout.Built)) *botLoadoutEditor {
	editor := &botLoadoutEditor{keep: keep, built: map[string]botloadout.Built{}}
	for name, loadout := range built {
		editor.built[name] = loadout
	}
	return editor
}

// stockChoice is the first entry of every slot menu: leave the game's own
// weapon alone.
const stockChoice = "stock"

// loadoutRows is the page, in the three columns the other bot pages use.
func loadoutRows(label func(text, help string) declarative.Label, editor *botLoadoutEditor) []declarative.Widget {
	classes := make([]string, 0, len(botloadout.Classes))
	for _, class := range botloadout.Classes {
		classes = append(classes, class.Name)
	}
	first := botloadout.Classes[0].Key

	return []declarative.Widget{
		declarative.TextLabel{
			Text: "Build a loadout, name it, and it joins the menus on the Team and Classes pages. " +
				"A loadout belongs to one class, so only that class can pick it.",
			ColumnSpan: 3,
			MaxSize:    declarative.Size{Width: sentenceWidth},
		},
		label("Class", "Who holds this loadout. Changing it clears the weapons, because a weapon of another class is not a choice anybody made."),
		declarative.ComboBox{
			AssignTo:              &editor.classBox,
			Model:                 classes,
			Value:                 classes[0],
			ColumnSpan:            2,
			OnCurrentIndexChanged: editor.classChanged,
		},
		label("Primary", "The weapon in this slot. Stock leaves the game's own alone. The Spy has no primary."),
		declarative.ComboBox{AssignTo: &editor.primaryBox, Model: weaponChoices(first, "primary"), Value: stockChoice, ColumnSpan: 2},
		label("Secondary", "The weapon in this slot. Stock leaves the game's own alone."),
		declarative.ComboBox{AssignTo: &editor.secondBox, Model: weaponChoices(first, "secondary"), Value: stockChoice, ColumnSpan: 2},
		label("Melee", "The weapon in this slot. Stock leaves the game's own alone."),
		declarative.ComboBox{AssignTo: &editor.meleeBox, Model: weaponChoices(first, "melee"), Value: stockChoice, ColumnSpan: 2},
		label("Watch", "The Spy's watch. Every other class leaves this on stock, because it has none."),
		declarative.ComboBox{AssignTo: &editor.watchBox, Model: weaponChoices(first, "pda2"), Value: stockChoice, ColumnSpan: 2},
		label("Name", "What this loadout is called in the menus on the other pages."),
		declarative.Composite{
			ColumnSpan: 2,
			Layout:     declarative.HBox{MarginsZero: true},
			Children: []declarative.Widget{
				declarative.LineEdit{AssignTo: &editor.nameBox, CueBanner: "gas runner", StretchFactor: 1},
				declarative.PushButton{
					Text:        "Save",
					ToolTipText: "Keeps the weapons above under this name. Saving over a name replaces it.",
					MinSize:     declarative.Size{Width: 70},
					OnClicked:   editor.save,
				},
			},
		},
		label("Saved loadouts", "Pick one, then Load to bring it back or Remove to throw it away. A seat still naming a removed one plays stock."),
		declarative.Composite{
			ColumnSpan: 2,
			Layout:     declarative.HBox{MarginsZero: true},
			Children: []declarative.Widget{
				declarative.ComboBox{
					AssignTo:      &editor.savedBox,
					Model:         editor.names(),
					Value:         firstName(editor.names()),
					StretchFactor: 1,
				},
				declarative.PushButton{Text: "Load", MinSize: declarative.Size{Width: 70}, OnClicked: editor.load},
				declarative.PushButton{Text: "Remove", MinSize: declarative.Size{Width: 70}, OnClicked: editor.remove},
			},
		},
	}
}

// weaponChoices is what a class can hold in a slot, stock first.
func weaponChoices(class, slot string) []string {
	weapons := gamedata.WeaponsFor(class, slot)
	out := make([]string, 0, len(weapons)+1)
	out = append(out, stockChoice)
	for _, weapon := range weapons {
		out = append(out, weapon.Name)
	}
	return out
}

// boxes pairs each slot with the menu that holds it, so the four are walked
// rather than named four times over.
func (e *botLoadoutEditor) boxes() []struct {
	slot string
	box  *walk.ComboBox
} {
	return []struct {
		slot string
		box  *walk.ComboBox
	}{
		{"primary", e.primaryBox},
		{"secondary", e.secondBox},
		{"melee", e.meleeBox},
		{"pda2", e.watchBox},
	}
}

// classChanged refills every slot menu, because the old lists belong to a class
// this loadout no longer names.
func (e *botLoadoutEditor) classChanged() {
	class := e.class()
	for _, pair := range e.boxes() {
		if pair.box == nil {
			continue
		}
		_ = pair.box.SetModel(weaponChoices(class, pair.slot))
		// By index, not by text. Setting the text of a menu that has a model
		// selects nothing, and nothing shows as an empty box.
		_ = pair.box.SetCurrentIndex(0)
	}
}

// class is the mod's key for the class the menu names.
func (e *botLoadoutEditor) class() string {
	if e.classBox == nil {
		return botloadout.Classes[0].Key
	}
	index := e.classBox.CurrentIndex()
	if index < 0 || index >= len(botloadout.Classes) {
		return botloadout.Classes[0].Key
	}
	return botloadout.Classes[index].Key
}

// built is what the menus currently describe.
func (e *botLoadoutEditor) current() botloadout.Built {
	class := e.class()
	out := botloadout.Built{
		Class:   class,
		Primary: botloadout.Stock,
		Second:  botloadout.Stock,
		Melee:   botloadout.Stock,
		PDA2:    botloadout.Stock,
	}
	for _, pair := range e.boxes() {
		index := indexOf(pair.box)
		if index <= 0 {
			continue
		}
		weapons := gamedata.WeaponsFor(class, pair.slot)
		if index > len(weapons) {
			continue
		}
		switch pair.slot {
		case "primary":
			out.Primary = weapons[index-1].DefIndex
		case "secondary":
			out.Second = weapons[index-1].DefIndex
		case "melee":
			out.Melee = weapons[index-1].DefIndex
		default:
			out.PDA2 = weapons[index-1].DefIndex
		}
	}
	return out
}

func indexOf(box *walk.ComboBox) int {
	if box == nil {
		return 0
	}
	return box.CurrentIndex()
}

func (e *botLoadoutEditor) save() {
	if e.nameBox == nil {
		return
	}
	name := strings.TrimSpace(e.nameBox.Text())
	if name == "" {
		return
	}
	e.built[name] = e.current()
	e.keep(e.built)
	e.refreshNames(name)
	e.tell()
}

func (e *botLoadoutEditor) load() {
	if e.savedBox == nil {
		return
	}
	name := e.savedBox.Text()
	built, found := e.built[name]
	if !found {
		return
	}
	at := slices.IndexFunc(botloadout.Classes, func(c botloadout.Class) bool { return c.Key == built.Class })
	if at >= 0 && e.classBox != nil {
		_ = e.classBox.SetCurrentIndex(at)
	}
	// SetCurrentIndex above fires classChanged, which has already refilled and
	// reset every slot menu, so the weapons go in after it rather than before.
	e.show(built)
	if e.nameBox != nil {
		_ = e.nameBox.SetText(name)
	}
}

// show puts one loadout's weapons into the menus.
func (e *botLoadoutEditor) show(built botloadout.Built) {
	for _, pair := range e.boxes() {
		if pair.box == nil {
			continue
		}
		want := botloadout.Stock
		switch pair.slot {
		case "primary":
			want = built.Primary
		case "secondary":
			want = built.Second
		case "melee":
			want = built.Melee
		default:
			want = built.PDA2
		}
		weapons := gamedata.WeaponsFor(built.Class, pair.slot)
		at := slices.IndexFunc(weapons, func(w gamedata.Weapon) bool { return w.DefIndex == want })
		_ = pair.box.SetCurrentIndex(at + 1)
	}
}

func (e *botLoadoutEditor) remove() {
	if e.savedBox == nil {
		return
	}
	name := e.savedBox.Text()
	if _, found := e.built[name]; !found {
		return
	}
	delete(e.built, name)
	e.keep(e.built)
	e.refreshNames("")
	e.tell()
}

// refreshNames rebuilds the saved menu and leaves it on the one named.
func (e *botLoadoutEditor) refreshNames(selected string) {
	if e.savedBox == nil {
		return
	}
	names := e.names()
	_ = e.savedBox.SetModel(names)
	at := slices.Index(names, selected)
	if at < 0 {
		at = 0
	}
	if len(names) > 0 {
		_ = e.savedBox.SetCurrentIndex(at)
	}
}

// names is every saved loadout, in one order so the menu does not shuffle.
func (e *botLoadoutEditor) names() []string {
	names := make([]string, 0, len(e.built))
	for name := range e.built {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// tell is the other pages, once this one has changed what they can offer.
func (e *botLoadoutEditor) tell() {
	if e.changed != nil {
		e.changed()
	}
}
