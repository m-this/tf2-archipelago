//go:build windows

package gui

import (
	"fmt"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"

	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

/*
	botsTab is the Bot Switcher: the team RED is playing, and the button that

hands a new one to the server that is already running.

Not a second editor. The seats, the weapons and the ticked classes live on the
Bots page of the settings, and building the same menus twice is two places to
change every time a class gains a preset. This is the view of what they say and
the one press that applies it.
*/
type botsTab struct {
	table   *walk.TableView
	drawn   *walk.Label
	hint    *walk.Label
	model   *seatsModel
	running bool
}

func newBotsTab() *botsTab {
	return &botsTab{model: &seatsModel{}}
}

// page builds the tab. onEdit opens the settings, where saving a team hands it
// to the running server on its own.
func (t *botsTab) page(onEdit func()) declarative.TabPage {
	return declarative.TabPage{
		Title:  "Bot Switcher",
		Layout: declarative.VBox{},
		Children: []declarative.Widget{
			declarative.Label{
				Text:        "What each seat on RED plays, and what it carries. Seats fill from the top, so a human joining takes seat one.",
				TextColor:   colorMuted,
				ToolTipText: "The team the server is running with.",
			},
			declarative.TableView{
				AssignTo:         &t.table,
				Model:            t.model,
				AlternatingRowBG: true,
				StretchFactor:    1,
				ToolTipText:      "Change this on the Bots page of the settings, then press Apply.",
				Columns: []declarative.TableViewColumn{
					{Title: "Seat", Width: 50},
					{Title: "Class", Width: 130},
					{Title: "Weapons", Width: 320},
				},
			},
			declarative.Label{AssignTo: &t.drawn, Text: "", TextColor: colorMuted},
			declarative.Composite{
				Layout:  declarative.HBox{MarginsZero: true},
				MaxSize: declarative.Size{Height: 28},
				Children: []declarative.Widget{
					declarative.PushButton{
						Text:        "Change the team...",
						ToolTipText: "Opens the settings on the Bots page, where the seats and the weapons are.",
						MinSize:     declarative.Size{Width: 150},
						OnClicked:   onEdit,
					},
					declarative.HSpacer{},
				},
			},
			declarative.Label{AssignTo: &t.hint, Text: botsHintIdle, TextColor: colorMuted},
		},
	}
}

// botsHintIdle is where the same switch lives for somebody who is in the game
// rather than in front of this window.
// Saving on the Bots page hands the team over on its own, so nothing here asks
// for a second press.
const botsHintIdle = "Change the team, save, and the running server takes it. In the game, !ap bots opens the same team."

// say puts one line under the buttons, where the press was.
func (t *botsTab) say(text string) { t.hint.SetText(text) }

// show puts the team these settings describe on the tab.
func (t *botsTab) show(s settings.Settings) {
	t.model.set(botlive.Team(s))
	if drawn := botlive.Drawn(s); drawn != "" {
		t.drawn.SetText("The mod draws the seats it picks from: " + drawn)
	} else {
		t.drawn.SetText("The mod draws the seats it picks from any class.")
	}
}

// A team saved while the server is down is one the next start reads anyway, so
// the line says that rather than offering something to press.
func (t *botsTab) setRunning(running bool) {
	t.running = running
	if !running {
		t.say("The server is not running. Press Start, or change the team and it starts with it.")
		return
	}
	t.say(botsHintIdle)
}

// seatsModel is the table's data: one row per seat RED holds.
type seatsModel struct {
	walk.TableModelBase
	seats []botlive.Seat
}

func (m *seatsModel) set(seats []botlive.Seat) {
	m.seats = seats
	m.PublishRowsReset()
}

func (m *seatsModel) RowCount() int { return len(m.seats) }

func (m *seatsModel) Value(row, col int) any {
	seat := m.seats[row]
	switch col {
	case 0:
		return fmt.Sprintf("%d", seat.Number)
	case 1:
		return seat.Class
	default:
		return seat.Weapons
	}
}
