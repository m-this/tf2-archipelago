package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
)

// Update is every key, tick and answer the interface reacts to.
//
// The keys are single letters while the RCON line is not focused, and go into
// that line while it is. A terminal has no buttons to click, so the letters
// are the buttons, and i is what takes them away.
func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		return m, nil

	case drainMsg:
		m.drain()
		m.status = m.serverStatus()
		return m, drainTick()

	case fetchMsg:
		return m, tea.Batch(fetchTick(), m.fetch())

	case sessionMsg:
		m.snapshot, m.fetchErr = msg.snapshot, msg.err
		if m.selected >= len(m.snapshot.Missions) {
			m.selected = max(len(m.snapshot.Missions)-1, 0)
		}
		return m, nil

	case noticeMsg:
		m.notice = string(msg)
		m.take(apruntime.Line{At: time.Now(), Source: "launcher", Text: string(msg)})
		return m, nil

	case communityAssetsMsg:
		if m.form != nil {
			m.form.applyCommunityAssets(msg)
		}
		m.notice = msg.notice
		m.take(apruntime.Line{At: time.Now(), Source: "launcher", Text: msg.notice})
		return m, nil

	case tea.KeyMsg:
		// The settings are a screen over this one, the way the window's dialog
		// is a window over its own: while it is up, it has the keyboard.
		if m.form != nil {
			return m.formKey(msg)
		}
		return m.key(msg)
	}
	return m, nil
}

// formKey is the settings screen: the tabs, the rows, and the two keys that
// end it. Every other key belongs to whichever row is focused.
func (m *model) formKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	form := m.form

	switch msg.String() {
	case "esc":
		m.form = nil
		return m, nil
	case "ctrl+s":
		cmd := form.save()
		if form.closed {
			m.form = nil
		}
		return m, cmd
	case "ctrl+c":
		return m, tea.Sequence(m.stop(), tea.Quit)
	case "tab":
		form.leave()
		form.tab = (form.tab + 1) % len(form.tabs)
		form.focused, form.offset = 0, 0
		return m, nil
	case "shift+tab":
		form.leave()
		form.tab = (form.tab - 1 + len(form.tabs)) % len(form.tabs)
		form.focused, form.offset = 0, 0
		return m, nil
	case "up":
		form.leave()
		form.focused = max(form.focused-1, 0)
		return m, nil
	case "down":
		form.leave()
		form.focused = min(form.focused+1, len(form.fields())-1)
		return m, nil
	}

	fields := form.fields()
	if form.focused >= len(fields) {
		return m, nil
	}
	// Hold the row rather than its index: All and None on the Missions tab
	// rebuild every row of the tab, so the index no longer names the field the
	// key went to.
	row := fields[form.focused]
	if !row.Handle(msg) {
		return m, nil
	}
	if action, ok := row.(interface{ take() tea.Cmd }); ok {
		return m, action.take()
	}
	return m, nil
}

// leave takes the focus off the row it is on, so a field holding a state that
// only makes sense under the cursor does not keep it.
func (f *settingsForm) leave() {
	fields := f.fields()
	if f.focused >= len(fields) {
		return
	}
	if row, ok := fields[f.focused].(interface{ disarm() }); ok {
		row.disarm()
	}
}

// fields is the rows of the tab on screen.
func (f *settingsForm) fields() []field { return f.tabs[f.tab].fields }

func (m *model) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.typing {
		return m.typingKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Sequence(m.stop(), tea.Quit)
	case "s":
		if m.supervisor.Running() {
			return m, m.stop()
		}
		return m, m.start()
	case "r":
		return m, tea.Sequence(m.stop(), m.start())
	case "tab":
		m.view = (m.view + 1) % viewCount
		return m, nil
	case "shift+tab":
		m.view = (m.view - 1 + viewCount) % viewCount
		return m, nil
	case "a":
		return m, m.applyBots()
	case "i", ":":
		m.typing = true
		return m, nil
	case ",":
		// From the Bot Switcher, straight onto the page it names.
		tab := ""
		if m.view == viewBots {
			tab = "Bots"
		}
		m.openSettings(tab)
		return m, nil
	case "j":
		return m, m.join()
	case "c":
		return m, m.copyJoin()
	case "p":
		return m, m.play()
	case "g":
		m.follow, m.offset = true, 0
		return m, nil
	}
	m.scroll(msg)
	return m, nil
}

// typingKey is the RCON line: everything is a character except the few keys
// that send it, empty it, or give it back.
//
//nolint:exhaustive // the keys not named here are the ones a command line ignores
func (m *model) typingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.typing, m.command = false, ""
		return m, nil
	case tea.KeyEnter:
		command := strings.TrimSpace(m.command)
		m.command = ""
		if command == "" {
			return m, nil
		}
		m.follow, m.offset = true, 0
		return m, m.send(command)
	case tea.KeyBackspace:
		if m.command != "" {
			runes := []rune(m.command)
			m.command = string(runes[:len(runes)-1])
		}
		return m, nil
	case tea.KeyCtrlC:
		return m, tea.Sequence(m.stop(), tea.Quit)
	case tea.KeySpace:
		m.command += " "
		return m, nil
	case tea.KeyRunes:
		m.command += string(msg.Runes)
		return m, nil
	}
	return m, nil
}

// scroll moves the log, or the selection in the run's missions, depending on
// which half is on screen.
func (m *model) scroll(msg tea.KeyMsg) {
	step := 1
	switch msg.String() {
	case "pgup", "pgdown":
		step = max(m.bodyHeight()-1, 1)
	}

	switch msg.String() {
	case "up", "k", "pgup":
		if m.view == viewSession {
			m.selected = max(m.selected-step, 0)
			return
		}
		m.offset = min(m.offset+step, max(len(m.lines)-m.bodyHeight(), 0))
		m.follow = m.offset == 0
	case "down", "j", "pgdown":
		if m.view == viewSession {
			m.selected = min(m.selected+step, max(len(m.snapshot.Missions)-1, 0))
			return
		}
		m.offset = max(m.offset-step, 0)
		m.follow = m.offset == 0
	}
}

// applyBots hands the team the settings hold to the running server, without the
// restart every other setting costs. The team is edited on the settings screen,
// where the seats and the weapons already are; the tab shows what the server is
// about to be told, and a is what tells it.
func (m *model) applyBots() tea.Cmd {
	if m.view != viewBots {
		return nil
	}
	if !m.supervisor.Running() {
		return func() tea.Msg { return noticeMsg("the server is not running") }
	}
	/* Nothing moved since the last save, so the before and the after are the
	 * same settings: the tab is for handing the team over again, not for a
	 * change it does not hold. */
	return m.applyTeam(m.settings)
}

// play loads the selected mission through the plugin's own switcher, which is
// what the window's "Play this mission" button does.
func (m *model) play() tea.Cmd {
	if m.view != viewSession || m.selected >= len(m.snapshot.Missions) {
		return nil
	}
	if !m.supervisor.Running() {
		return nil
	}
	return m.send("sm_ap_mission " + m.snapshot.Missions[m.selected].PopFile)
}

func (m *model) serverStatus() string {
	if m.supervisor.Running() {
		return "running"
	}
	return "stopped"
}
