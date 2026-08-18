//go:build windows

package gui

import (
	"fmt"
	"strings"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"

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

	err := declarative.Dialog{
		AssignTo:      &dialog,
		Title:         "Settings",
		DefaultButton: &accept,
		CancelButton:  &cancel,
		MinSize:       declarative.Size{Width: 460, Height: 380},
		Layout:        declarative.VBox{},
		Children: []declarative.Widget{
			declarative.GroupBox{
				Title:  "Archipelago room",
				Layout: declarative.Grid{Columns: 2},
				Children: []declarative.Widget{
					declarative.Label{Text: "Room address"},
					declarative.LineEdit{AssignTo: &roomEdit, Text: current.String(), CueBanner: "archipelago.gg:12345"},
					declarative.Label{Text: ""},
					declarative.Label{AssignTo: &roomWarn, Text: ""},
					declarative.Label{Text: "Slot name"},
					declarative.LineEdit{AssignTo: &slotEdit, Text: s.APSlotName},
				},
			},
			declarative.GroupBox{
				Title:  "Server",
				Layout: declarative.Grid{Columns: 2},
				Children: []declarative.Widget{
					declarative.Label{Text: "Server name"},
					declarative.LineEdit{AssignTo: &nameEdit, Text: s.SrcdsHostname},
					declarative.Label{Text: "Start map"},
					declarative.ComboBox{AssignTo: &mapBox, Model: mapNames(), Value: s.SrcdsStartMap},
					declarative.Label{Text: "Defender bots"},
					declarative.CheckBox{AssignTo: &botsBox, Text: "fill the RED team", Checked: s.SrcdsBots},
					declarative.Label{Text: "Fill RED to"},
					declarative.NumberEdit{AssignTo: &botsSize, Value: float64(s.SrcdsBotTeamSize), MinValue: 1, MaxValue: 6, Decimals: 0},
				},
			},
			declarative.GroupBox{
				Title:  "Run shape (used when you generate the seed)",
				Layout: declarative.Grid{Columns: 2},
				Children: []declarative.Widget{
					declarative.Label{Text: "Difficulty pool"},
					declarative.ComboBox{AssignTo: &tierBox, Model: tierLabels, Value: tierLabel(tiers, s.MvmDifficulty)},
					declarative.Label{Text: "Missions"},
					declarative.NumberEdit{AssignTo: &missions, Value: float64(s.MvmMissionCount), MinValue: 1, MaxValue: 29, Decimals: 0},
					declarative.Label{Text: "Goal"},
					declarative.ComboBox{AssignTo: &goalBox, Model: goalLabels, Value: goalLabel(goals, s.MvmGoal)},
				},
			},
			declarative.Composite{
				Layout: declarative.HBox{},
				Children: []declarative.Widget{
					declarative.HSpacer{},
					declarative.PushButton{AssignTo: &accept, Text: "Save", OnClicked: func() { dialog.Accept() }},
					declarative.PushButton{AssignTo: &cancel, Text: "Cancel", OnClicked: func() { dialog.Cancel() }},
				},
			},
		},
	}.Create(owner)
	if err != nil {
		return s, false, err
	}

	// The address is the one field with a wrong answer, so it is checked as it
	// is typed and Save stays disabled until it parses.
	validate := func() {
		if _, err := settings.ParseRoom(roomEdit.Text()); err != nil {
			roomWarn.SetText(err.Error())
			accept.SetEnabled(false)
			return
		}
		roomWarn.SetText("")
		accept.SetEnabled(true)
	}
	roomEdit.TextChanged().Attach(validate)
	validate()

	if dialog.Run() != walk.DlgCmdOK {
		return s, false, nil
	}

	room, err := settings.ParseRoom(roomEdit.Text())
	if err != nil {
		return s, false, err
	}
	s.APHost, s.APPort, s.APTls = room.Host, room.Port, room.TLS
	s.APSlotName = strings.TrimSpace(slotEdit.Text())
	s.SrcdsHostname = strings.TrimSpace(nameEdit.Text())
	s.SrcdsStartMap = mapBox.Text()
	s.SrcdsBots = botsBox.Checked()
	s.SrcdsBotTeamSize = int(botsSize.Value())
	s.MvmDifficulty = tiers[max(tierBox.CurrentIndex(), 0)].Key
	s.MvmMissionCount = int(missions.Value())
	s.MvmGoal = goals[max(goalBox.CurrentIndex(), 0)].Key
	return s, true, nil
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
