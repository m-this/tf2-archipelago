package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// screen is a model with a size and a supervisor, which is what View needs
// before it can draw anything.
func screen(t *testing.T) *model {
	t.Helper()
	s := settings.Defaults()
	s.APHost, s.APPort = "archipelago.gg", 12345
	s.SrcdsRconPw = "secret"

	m := newModel(s)
	m.supervisor = apruntime.NewSupervisor(s, nil, m.take)
	m.width, m.height, m.ready = 100, 30, true
	return m
}

// The main screen says what the window says: what the server is doing, where
// the room is, what a friend types to join, and which keys do what.
func TestTheMainScreenSaysWhatIsGoingOn(t *testing.T) {
	m := screen(t)
	m.take(apruntime.Line{At: time.Now(), Source: "srcds", Text: "fake server up"})
	m.drain()

	// The screen opens on the run, so the log is the other tab.
	m.view = viewLog
	view := m.View()
	for _, want := range []string{
		"stopped",
		"room archipelago.gg:12345",
		"join",
		"Log",
		"Session",
		"fake server up",
		"rcon",
		"settings",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("the screen does not say %q:\n%s", want, view)
		}
	}
}

// A log longer than the screen shows its end, because the line that matters is
// the last one until somebody scrolls.
func TestTheLogShowsItsEnd(t *testing.T) {
	m := screen(t)
	for i := range 200 {
		m.take(apruntime.Line{At: time.Now(), Source: "srcds", Text: "line " + itoa(i)})
	}
	m.drain()
	m.view = viewLog

	view := m.View()
	if !strings.Contains(view, "line 199") {
		t.Errorf("the last line is not on screen:\n%s", view)
	}
	if strings.Contains(view, "line 0 ") {
		t.Errorf("the first line is still on screen:\n%s", view)
	}
}

// Every key the footer offers has to do something, or the footer is a lie.
func TestTheKeysDoWhatTheFooterSays(t *testing.T) {
	m := screen(t)

	// Round the three tabs and back, which is what the footer's one entry says
	// tab does. shift+tab is the same ring the other way.
	for _, want := range []view{viewBots, viewLog, viewSession} {
		if _, _ = m.Update(key("tab")); m.view != want {
			t.Errorf("tab reached view %d, want %d", m.view, want)
		}
	}
	if _, _ = m.Update(key("shift+tab")); m.view != viewLog {
		t.Errorf("shift+tab reached view %d, want %d", m.view, viewLog)
	}
	m.view = viewSession
	if _, _ = m.Update(key("i")); !m.typing {
		t.Error("i did not reach the rcon line")
	}
	if _, _ = m.Update(key("esc")); m.typing {
		t.Error("esc did not leave the rcon line")
	}
	if _, _ = m.Update(key(",")); m.form == nil {
		t.Error("the settings never opened")
	}
	if _, _ = m.Update(key("esc")); m.form != nil {
		t.Error("esc did not close the settings")
	}
}

// What is typed at the rcon line is the command, and nothing typed there is
// read as a key that starts or stops the server.
func TestTypingGoesToTheCommandLine(t *testing.T) {
	m := screen(t)
	m.Update(key("i"))
	for _, k := range []string{"s", "t", "a", "t", "u", "s"} {
		m.Update(key(k))
	}

	if m.command != "status" {
		t.Errorf("the command line holds %q", m.command)
	}
	if m.supervisor.Running() {
		t.Error("typing s started the server")
	}
}

// The settings are the window's tabs, and what they change is saved to the
// settings the launcher runs on.
func TestTheSettingsScreenEditsTheRun(t *testing.T) {
	m := screen(t)
	m.Update(key(","))

	if got := len(m.form.tabs); got != 9 {
		t.Errorf("the settings have %d tabs, want 9", got)
	}
	view := m.form.view(100, 30)
	for _, want := range []string{"Player options", "Balancing", "Bots", "Loadouts", "Networking", "Easiest tier"} {
		if !strings.Contains(view, want) {
			t.Errorf("the settings do not show %q:\n%s", want, view)
		}
	}

	// Death Link is the fifth row of the first tab: move to it and turn it on.
	before := m.form.edited.MvmDeathLink
	for range 4 {
		m.Update(key("down"))
	}
	m.Update(key(" "))
	if m.form.edited.MvmDeathLink == before {
		t.Error("space did not change the row it was on")
	}
}

// The tabs wrap, and every one of them draws: a tab whose fields panic is a
// tab nobody sees until a player opens it.
func TestEveryTabDraws(t *testing.T) {
	m := screen(t)
	m.Update(key(","))

	for i := range m.form.tabs {
		m.form.tab = i
		m.form.focused = 0
		if view := m.form.view(100, 30); !strings.Contains(view, m.form.tabs[i].title) {
			t.Errorf("tab %d does not draw its own title", i)
		}
		for _, row := range m.form.fields() {
			if row.Label() == "" {
				t.Errorf("tab %q has a row with no name", m.form.tabs[i].title)
			}
		}
	}
}

// All and None are the whole pool or none of it, and the rows they rebuild say
// the same thing they wrote.
func TestTheMissionPoolTakesAllAndNone(t *testing.T) {
	m := screen(t)
	m.Update(key(","))
	m.form.tab = tabNamed(t, m, "Missions")

	find := func(label string) int {
		for i, row := range m.form.fields() {
			if row.Label() == label {
				return i
			}
		}
		t.Fatalf("missing %q row", label)
		return -1
	}

	m.form.focused = find("None in the pool")
	m.Update(key("enter"))
	if got := len(m.form.edited.MvmExcludedMissions); got != len(gamedata.PlayableMissions()) {
		t.Errorf("None left %d missions out, want all %d", got, len(gamedata.PlayableMissions()))
	}
	if view := m.form.view(100, 30); !strings.Contains(view, "left out") {
		t.Errorf("the rows still say the missions are in the pool:\n%s", view)
	}

	m.form.focused = find("All in the pool")
	m.Update(key("enter"))
	for _, mission := range runshape.VisibleMissions(m.form.communityAvailable) {
		if gamedata.IsPlayableMission(mission.ID) && slices.Contains(m.form.edited.MvmExcludedMissions, mission.PopFile) {
			t.Errorf("All left visible mission %s out", mission.PopFile)
		}
	}
}

func TestCommunityMissionsStayHiddenUntilTheirAssetsAreAvailable(t *testing.T) {
	m := screen(t)
	m.Update(key(","))
	m.form.tab = tabNamed(t, m, "Missions")
	for _, row := range m.form.fields() {
		if strings.Contains(row.Label(), "Swamp Fever") {
			t.Fatal("Swamp Fever is visible before its community archive is available")
		}
	}

	m.form.communityAvailable = []string{settings.CommunityPackPotato}
	m.form.build()
	for _, row := range m.form.fields() {
		if !strings.Contains(row.Label(), "Swamp Fever") {
			continue
		}
		if !strings.Contains(row.Value(), "missing bot .nav") {
			t.Fatalf("unavailable row = %q", row.Value())
		}
		if row.Handle(key(" ")) {
			t.Fatal("the unavailable mission accepted a toggle")
		}
		return
	}
	t.Fatal("Swamp Fever is not visible in the mission list")
}

// Repair and Reset ask twice, because there is no taking either one back.
func TestTheUndoableActionsAskTwice(t *testing.T) {
	ran := 0
	row := &confirmField{
		label: "Repair", run: func() tea.Cmd { ran++; return nil },
		warning: "this stops the server",
	}

	row.Handle(key("enter"))
	if ran != 0 {
		t.Fatal("the first enter ran it")
	}
	if !strings.Contains(row.Value()+row.Help(), "this stops the server") {
		t.Error("the armed row does not say what it is about to do")
	}

	row.Handle(key("down"))
	row.Handle(key("enter"))
	if ran != 0 {
		t.Fatal("a key in between did not take the arming away")
	}

	row.Handle(key("enter"))
	if ran != 1 {
		t.Errorf("the second enter ran it %d times", ran)
	}
}

func key(name string) tea.KeyMsg {
	switch name {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// The run is the first thing on screen and the log is the second, the way the
// window has them.
func TestTheScreenOpensOnTheRun(t *testing.T) {
	m := screen(t)
	if m.view != viewSession {
		t.Errorf("the screen opens on view %d, want the session", m.view)
	}
}

// tabNamed finds a tab by its title, because an index moves whenever a tab is
// added and a test that navigates by number then edits the wrong page.
func tabNamed(t *testing.T, m *model, title string) int {
	t.Helper()
	for i, tab := range m.form.tabs {
		if tab.title == title {
			return i
		}
	}
	t.Fatalf("no tab called %q", title)
	return -1
}
