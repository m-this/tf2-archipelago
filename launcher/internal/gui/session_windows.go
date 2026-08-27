//go:build windows

package gui

import (
	"strings"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"

	"github.com/m-this/tf2-archipelago/launcher/internal/session"
)

// sessionTab is the view of the run: the multiworld line, the run's counters,
// and the missions with what the run has done to them. It is fed by
// watchSession and touched only on the UI thread.
type sessionTab struct {
	multiworld *walk.Label
	run        *walk.Label
	table      *walk.TableView
	switchBt   *walk.PushButton
	hint       *walk.Label
	model      *missionsModel
	running    bool

	// The mission the last button press asked for, held until the server says
	// it loaded. The line under the button is the only feedback a player gets
	// without opening the log, and a map change takes long enough to doubt.
	//
	// The name and not the popfile: the server announces the mission it loaded
	// by name, and that announcement is what clears this.
	switching session.Mission
}

func newSessionTab() *sessionTab {
	return &sessionTab{model: &missionsModel{}}
}

// page builds the tab. onSwitch gets the popfile of the selected mission.
func (t *sessionTab) page(onSwitch func(popFile string)) declarative.TabPage {
	return declarative.TabPage{
		Title:  "Session",
		Layout: declarative.VBox{},
		Children: []declarative.Widget{
			declarative.Label{AssignTo: &t.multiworld, Text: "The server is not running.", ToolTipText: "The bridge's connection to the Archipelago room."},
			declarative.Label{AssignTo: &t.run, Text: "", TextColor: colorMuted},
			declarative.TableView{
				AssignTo:         &t.table,
				Model:            t.model,
				AlternatingRowBG: true,
				StretchFactor:    1,
				ToolTipText:      "The run's missions in the order the seed drew them. Locked ones wait for their ticket.",
				Columns: []declarative.TableViewColumn{
					{Title: "#", Width: 30},
					{Title: "Mission", Width: 200},
					{Title: "Map", Width: 130},
					{Title: "Source", Width: 110},
					{Title: "Waves", Width: 50},
					{Title: "State", Width: 120},
				},
				StyleCell: func(style *walk.CellStyle) {
					if style.Row() < 0 || style.Row() >= len(t.model.missions) {
						return
					}
					mission := t.model.missions[style.Row()]
					switch {
					case mission.Played:
						style.TextColor = colorRunning
					case !mission.Unlocked:
						style.TextColor = colorMuted
					}
				},
				OnCurrentIndexChanged: func() { t.refreshButtons() },
			},
			declarative.Composite{
				Layout:  declarative.HBox{MarginsZero: true},
				MaxSize: declarative.Size{Height: 30},
				Children: []declarative.Widget{
					declarative.PushButton{
						AssignTo:    &t.switchBt,
						Text:        "Play this mission",
						ToolTipText: "Load the selected mission now, through the plugin's own switcher. A locked mission is refused.",
						OnClicked: func() {
							mission, ok := t.selected()
							if !ok {
								return
							}
							t.switching = mission
							t.saySwitch("Loading " + mission.Name + " on " + mission.Map +
								". The map changes, which takes a few seconds.")
							onSwitch(mission.PopFile)
						},
					},
					declarative.Label{AssignTo: &t.hint, Text: hintIdle, TextColor: colorMuted},
					declarative.HSpacer{},
				},
			},
		},
	}
}

// hintIdle is what the line under the button says when nothing is happening:
// where the same two answers live for someone who is in the game rather than
// in front of this window.
const hintIdle = "In the game, !ap status prints the same picture and !mission the same list."

// saySwitch puts one line under the button, where the press was.
func (t *sessionTab) saySwitch(text string) {
	t.hint.SetText(text)
}

func (t *sessionTab) selected() (session.Mission, bool) {
	index := t.table.CurrentIndex()
	if index < 0 || index >= len(t.model.missions) {
		return session.Mission{}, false
	}
	return t.model.missions[index], true
}

func (t *sessionTab) setRunning(running bool) {
	t.running = running
	if !running {
		t.multiworld.SetText("The server is not running.")
		t.run.SetText("")
		t.model.set(nil)
		t.switching = session.Mission{}
		t.hint.SetText(hintIdle)
	}
	t.refreshButtons()
}

/* The button says what pressing it will do, and refuses what the run refuses.
 *
 * A locked mission used to offer the same button as an unlocked one, and the
 * refusal arrived as a line of chat on a server the player may not be looking
 * at. The ticket is somewhere in the multiworld, which is a rule of the game
 * rather than a failure, so it belongs on the button rather than in a log.
 */
func (t *sessionTab) refreshButtons() {
	mission, ok := t.selected()
	if !ok {
		t.switchBt.SetEnabled(false)
		t.switchBt.SetText("Play this mission")
		return
	}
	playable := t.running && mission.Unlocked
	t.switchBt.SetEnabled(playable)
	switch {
	case !mission.Unlocked:
		t.switchBt.SetText("Locked: " + mission.Name)
	case t.running:
		t.switchBt.SetText("Play " + mission.Name)
	default:
		t.switchBt.SetText("Play " + mission.Name)
	}
}

// update takes a reading of the run, or the reason there is none.
func (t *sessionTab) update(snapshot session.Snapshot, err error) {
	if !t.running {
		return
	}
	if err != nil {
		t.multiworld.SetText("Bridge: " + err.Error())
		return
	}
	health := snapshot.Health
	t.multiworld.SetText("Multiworld: " + health.Summary())

	var run []string
	if health.Seed != "" {
		run = append(run, "seed "+health.Seed)
	}
	if health.LastCheck != "" {
		run = append(run, "last check: "+health.LastCheck)
	}
	if health.DeathLink {
		run = append(run, "Death Link on")
	}
	if health.GoalSent {
		run = append(run, "goal reached")
	}
	t.run.SetText(strings.Join(run, "   "))
	t.model.set(snapshot.Missions)
	t.refreshButtons()
}

// noteSwitchLanded clears the waiting line once the run is on the mission the
// button asked for. Nothing else clears it: a player who pressed the button
// and got no answer is exactly the person who needs the line to stay.
func (t *sessionTab) noteSwitchLanded(name string) {
	if t.switching.Name == "" || name == "" || !strings.EqualFold(name, t.switching.Name) {
		return
	}
	t.hint.SetText("Playing " + t.switching.Name + ".   " + hintIdle)
	t.switching = session.Mission{}
}

// missionsModel is the table's data: the run's missions, in seed order.
type missionsModel struct {
	walk.TableModelBase
	missions []session.Mission
}

// set replaces the rows. A same-sized list is a change of rows rather than a
// reset, which keeps the player's selection through the five-second refresh.
func (m *missionsModel) set(missions []session.Mission) {
	same := len(missions) == len(m.missions)
	m.missions = missions
	if same && len(missions) > 0 {
		m.PublishRowsChanged(0, len(missions)-1)
		return
	}
	m.PublishRowsReset()
}

func (m *missionsModel) RowCount() int { return len(m.missions) }

func (m *missionsModel) Value(row, col int) any {
	mission := m.missions[row]
	switch col {
	case 0:
		return row + 1
	case 1:
		return mission.Name
	case 2:
		return mission.Map
	case 3:
		return mission.Source
	case 4:
		return mission.Waves
	default:
		return missionState(mission)
	}
}

/* missionState is what this player did, not what the room holds.
 *
 * A mission another world's !collect touched has its check on the disk without
 * anybody here playing it. Drawn as "cleared" it made the run list stop saying
 * what this player had done, which is what Peppy reported. The two are shown
 * apart: cleared is yours, collected is the room's.
 */
func missionState(mission session.Mission) string {
	switch {
	case mission.Played:
		return "cleared"
	case mission.Cleared:
		return "collected"
	case mission.Unlocked:
		return "unlocked"
	default:
		return "locked"
	}
}
