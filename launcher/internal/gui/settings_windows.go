//go:build windows

package gui

import (
	"fmt"
	"strings"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"

	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// runSettingsDialog asks for the values worth changing between evenings. It
// returns the edited settings and whether the player accepted them.
func runSettingsDialog(owner walk.Form, s settings.Settings) (settings.Settings, bool, error) {
	var (
		dialog   *walk.Dialog
		accept   *walk.PushButton
		cancel   *walk.PushButton
		roomEdit *walk.LineEdit
		roomWarn *walk.Label
		slotEdit *walk.LineEdit
		nameEdit *walk.LineEdit
		mapBox   *walk.ComboBox
		botsBox  *walk.CheckBox
		botsSize *walk.NumberEdit
		tierBox  *walk.ComboBox
		missions *walk.NumberEdit
		goalBox  *walk.ComboBox
	)

	tiers := runshape.Tiers()
	tierLabels := make([]string, 0, len(tiers))
	for _, tier := range tiers {
		tierLabels = append(tierLabels, tier.Label())
	}
	goals := runshape.Goals()
	goalLabels := make([]string, 0, len(goals))
	for _, goal := range goals {
		goalLabels = append(goalLabels, goal.Label())
	}

	current := settings.Room{Host: s.APHost, Port: s.APPort}
	edited := s

	err := declarative.Dialog{
		AssignTo:     &dialog,
		Title:        "Settings",
		CancelButton: &cancel,
		Size:         declarative.Size{Width: 560, Height: 520},
		MinSize:      declarative.Size{Width: 520, Height: 470},
		Layout:       declarative.VBox{},
		Children: []declarative.Widget{
			declarative.GroupBox{
				Title:  "Archipelago room",
				Layout: declarative.Grid{Columns: 2},
				Children: []declarative.Widget{
					declarative.Label{Text: "Room address", MinSize: declarative.Size{Width: 130}},
					declarative.LineEdit{AssignTo: &roomEdit, Text: current.String(), CueBanner: "archipelago.gg:12345"},
					declarative.Label{Text: ""},
					declarative.Label{AssignTo: &roomWarn, Text: ""},
					declarative.Label{Text: "Slot name", MinSize: declarative.Size{Width: 130}},
					declarative.LineEdit{AssignTo: &slotEdit, Text: s.APSlotName},
				},
			},
			declarative.GroupBox{
				Title:  "Server",
				Layout: declarative.Grid{Columns: 2},
				Children: []declarative.Widget{
					declarative.Label{Text: "Server name", MinSize: declarative.Size{Width: 130}},
					declarative.LineEdit{AssignTo: &nameEdit, Text: s.SrcdsHostname},
					declarative.Label{Text: "Start map", MinSize: declarative.Size{Width: 130}},
					declarative.ComboBox{AssignTo: &mapBox, Model: mapNames(), Value: s.SrcdsStartMap},
					declarative.Label{Text: "Defender bots", MinSize: declarative.Size{Width: 130}},
					declarative.CheckBox{AssignTo: &botsBox, Text: "fill the RED team", Checked: s.SrcdsBots},
					declarative.Label{Text: "Fill RED to", MinSize: declarative.Size{Width: 130}},
					declarative.NumberEdit{AssignTo: &botsSize, Value: float64(s.SrcdsBotTeamSize), MinValue: 1, MaxValue: 6, Decimals: 0},
				},
			},
			declarative.GroupBox{
				Title:  "Run shape (used when you generate the seed)",
				Layout: declarative.Grid{Columns: 2},
				Children: []declarative.Widget{
					declarative.Label{Text: "Easiest tier", MinSize: declarative.Size{Width: 130}},
					declarative.ComboBox{AssignTo: &tierBox, Model: tierLabels, Value: tierLabel(tiers, s.MvmDifficulty)},
					declarative.Label{Text: "Missions used", MinSize: declarative.Size{Width: 130}},
					declarative.NumberEdit{
						AssignTo: &missions,
						Value:    float64(s.MvmMissionCount),
						MinValue: 1,
						MaxValue: float64(runshape.MissionsInPool(s.MvmDifficulty)),
						Decimals: 0,
					},
					declarative.Label{Text: "Goal", MinSize: declarative.Size{Width: 130}},
					declarative.ComboBox{AssignTo: &goalBox, Model: goalLabels, Value: goalLabel(goals, s.MvmGoal)},
				},
			},
			declarative.Composite{
				Layout: declarative.HBox{},
				Children: []declarative.Widget{
					declarative.PushButton{Text: "Clear cache", OnClicked: func() { clearCache(dialog, s.InstallRoot) }},
					declarative.HSpacer{},
					declarative.PushButton{AssignTo: &accept, Text: "Save", OnClicked: func() {
						// Read every field here, not after Run returns: closing
						// the dialog destroys its children, and a destroyed
						// LineEdit reads back empty.
						//
						// The address is checked here too, rather than as the
						// player types. A disabled button with no explanation
						// is a dead end, and a paste with the mouse sends no
						// keystroke to check on.
						room, err := settings.ParseRoom(roomEdit.Text())
						if err != nil {
							roomWarn.SetText(err.Error())
							return
						}
						edited = s
						edited.APHost, edited.APPort, edited.APTls = room.Host, room.Port, room.TLS
						edited.APSlotName = strings.TrimSpace(slotEdit.Text())
						edited.SrcdsHostname = strings.TrimSpace(nameEdit.Text())
						edited.SrcdsStartMap = mapBox.Text()
						edited.SrcdsBots = botsBox.Checked()
						edited.SrcdsBotTeamSize = int(botsSize.Value())
						edited.MvmDifficulty = tiers[max(tierBox.CurrentIndex(), 0)].Key
						edited.MvmMissionCount = int(missions.Value())
						edited.MvmGoal = goals[max(goalBox.CurrentIndex(), 0)].Key
						dialog.Accept()
					}},
					declarative.PushButton{AssignTo: &cancel, Text: "Cancel", OnClicked: func() { dialog.Cancel() }},
				},
			},
		},
	}.Create(owner)
	if err != nil {
		return s, false, err
	}

	// A harder floor leaves fewer missions to draw from, so the count that a
	// run can ask for follows the tier.
	tierBox.CurrentIndexChanged().Attach(func() {
		pool := tiers[max(tierBox.CurrentIndex(), 0)].Missions
		_ = missions.SetRange(1, float64(pool))
		if missions.Value() > float64(pool) {
			_ = missions.SetValue(float64(pool))
		}
	})

	// Clear the complaint as soon as the address looks right again.
	roomEdit.TextChanged().Attach(func() {
		if _, err := settings.ParseRoom(roomEdit.Text()); err == nil {
			roomWarn.SetText("")
		}
	})
	if s.APPort == 0 {
		roomWarn.SetText("paste the address from your Archipelago room page")
	}

	if dialog.Run() != walk.DlgCmdOK {
		return s, false, nil
	}
	return edited, true, nil
}

// clearCache throws away SteamCMD, the mods and Steam's download record, for a
// player whose install will not go through. The next Start puts them back.
func clearCache(owner walk.Form, installRoot string) {
	answer := walk.MsgBox(owner, "Clear cache",
		"This removes SteamCMD, the mods and Steam's record of the download, "+
			"then the next Start fetches them again.\n\n"+
			"It keeps the game files and the run: no 14 GB download, no lost checks.\n\n"+
			"Stop the server first if it is running.",
		walk.MsgBoxOKCancel|walk.MsgBoxIconQuestion)
	if answer != walk.DlgCmdOK {
		return
	}
	removed, err := installer.Clean(installRoot)
	if err != nil {
		walk.MsgBox(owner, "Clear cache", err.Error(), walk.MsgBoxIconError)
		return
	}
	if len(removed) == 0 {
		walk.MsgBox(owner, "Clear cache", "Nothing to remove.", walk.MsgBoxIconInformation)
		return
	}
	walk.MsgBox(owner, "Clear cache",
		"Removed:\n"+strings.Join(removed, "\n")+"\n\nPress Start when you are ready.",
		walk.MsgBoxIconInformation)
}

func tierLabel(tiers []runshape.Tier, key string) string {
	for _, tier := range tiers {
		if tier.Key == key {
			return tier.Label()
		}
	}
	if len(tiers) == 0 {
		return ""
	}
	return tiers[0].Label()
}

func goalLabel(goals []runshape.Goal, key string) string {
	for _, goal := range goals {
		if goal.Key == key {
			return goal.Label()
		}
	}
	return fmt.Sprintf("%v", goals[0].Label())
}
