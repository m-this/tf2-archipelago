//go:build windows

package gui

import (
	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"

	"github.com/m-this/tf2-archipelago/launcher/internal/session"
)

/*
	unlocksTab is what the run has handed this slot so far.

The bridge holds the unlock set on disk and serves it to the plugin on
loopback; this tab is the same reading for the person at the window. Asked for
on Discord: "a new tab in the client that tracks all your current weapon
upgrades/unlocks in general". In the game, !ap buffs opens the same for the
loadout being held.
*/
type unlocksTab struct {
	model   *unlocksModel
	table   *walk.TableView
	status  *walk.Label
	running bool
}

func newUnlocksTab() *unlocksTab {
	return &unlocksTab{model: &unlocksModel{}}
}

func (t *unlocksTab) page() declarative.TabPage {
	return declarative.TabPage{
		Title:  "Unlocks",
		Layout: declarative.VBox{},
		Children: []declarative.Widget{
			declarative.Label{AssignTo: &t.status, Text: "The server is not running.", TextColor: colorMuted},
			declarative.TableView{
				AssignTo:         &t.table,
				Model:            t.model,
				AlternatingRowBG: true,
				StretchFactor:    1,
				ToolTipText:      "Everything the multiworld has handed this run: classes, weapon slots, missions and weapon buffs, with how many times a buff was handed over.",
				Columns: []declarative.TableViewColumn{
					{Title: "Kind", Width: 100},
					{Title: "Unlock", Width: 360},
					{Title: "Level", Width: 50},
				},
			},
		},
	}
}

func (t *unlocksTab) setRunning(running bool) {
	t.running = running
	if !running {
		t.status.SetText("The server is not running.")
		t.model.set(nil)
	}
}

// update takes a reading of the run, or the reason there is none.
func (t *unlocksTab) update(snapshot session.Snapshot, err error) {
	if !t.running {
		return
	}
	if err != nil {
		t.status.SetText("Bridge: " + err.Error())
		return
	}
	t.status.SetText(unlocksSummary(snapshot.Unlocks))
	t.model.set(snapshot.Unlocks)
}

// unlocksSummary is the line above the table.
func unlocksSummary(unlocks []session.Unlock) string {
	if len(unlocks) == 0 {
		return "Nothing unlocked yet."
	}
	return "In the game, !ap buffs shows the buffs on the loadout you hold."
}

type unlocksModel struct {
	walk.TableModelBase
	rows []session.Unlock
}

// set replaces the rows the way missionsModel does: a same-sized list keeps
// the selection through the refresh.
func (m *unlocksModel) set(rows []session.Unlock) {
	same := len(rows) == len(m.rows)
	m.rows = rows
	if same && len(rows) > 0 {
		m.PublishRowsChanged(0, len(rows)-1)
		return
	}
	m.PublishRowsReset()
}

func (m *unlocksModel) RowCount() int { return len(m.rows) }

func (m *unlocksModel) Value(row, col int) any {
	unlock := m.rows[row]
	switch col {
	case 0:
		return unlock.Kind
	case 1:
		return unlock.Name
	default:
		return unlock.Level
	}
}
