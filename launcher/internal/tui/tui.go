/*
Package tui is the launcher's interface in a terminal: the log, the run, the
keys that start and stop the server, and a line that sends RCON commands to it.

It is Bubble Tea, which is pure Go, so this is the same interface on Linux and
on Windows and needs nothing installed. The Windows window in internal/gui is
the other one, and it stays: this is what Linux gets, and what anyone on
Windows gets with -tui or over SSH.

Lines arrive from goroutines that are not the one drawing, so the sink writes
them to a buffer and the model drains it on a tick. Sending each line into the
event loop instead makes an install that prints a hundred lines a second redraw
a hundred times a second.
*/
package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/rcon"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/session"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/srcdsconfig"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

const (
	// linesMax caps what is kept for scrolling back. A wave is a few hundred
	// lines and an evening tens of thousands.
	linesMax = 20000

	// drainEvery is how often the lines the server printed reach the screen,
	// and sessionEvery how often the bridge is asked what the run has done.
	drainEvery   = 120 * time.Millisecond
	sessionEvery = 5 * time.Second
)

// Run opens the interface and blocks until the player quits. The server is
// stopped on the way out, so quitting is a clean shutdown.
func Run(s settings.Settings, logger *slog.Logger) error {
	m := newModel(s)
	program := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	m.supervisor = apruntime.NewSupervisor(s, logger, m.take)
	defer m.supervisor.Stop()

	_, err := program.Run()
	return err
}

// view is which half of the screen the player is looking at.
type view int

/* The run first, then what you change about it, then the log.
 *
 * The log is what you want when something is wrong and the run is what you came
 * for, so the switcher goes between them: it is the one thing you reach for
 * during a mission that is going fine. */
const (
	viewSession view = iota
	viewBots
	viewLog
)

// viewCount is how many there are, so tab rings round them.
const viewCount = 3

type model struct {
	settings   settings.Settings
	supervisor *apruntime.Supervisor

	width, height int
	view          view
	ready         bool

	// mu guards what the sink writes and the model reads.
	mu      sync.Mutex
	pending []string

	lines   []string
	offset  int // how far up the log the player has scrolled, in lines
	follow  bool
	command string
	typing  bool

	status  string
	mission string
	notice  string
	// itemServer is the last thing the game server said about Steam's item
	// server, which is what hands out weapons. Kept on the model rather than
	// left in the log, because a player who is playing full stock needs to be
	// told why without reading a thousand lines.
	itemServer string
	steamURL   string
	form       *settingsForm
	snapshot   session.Snapshot
	fetchErr   error
	selected   int
}

func newModel(s settings.Settings) *model {
	return &model{settings: s, follow: true, status: "stopped"}
}

// take is the sink every line arrives on. It only buffers: the tick is what
// draws.
func (m *model) take(line apruntime.Line) {
	text := fmt.Sprintf("%s  %-8s %s", line.At.Format("15:04:05"), line.Source, line.Text)

	m.mu.Lock()
	m.pending = append(m.pending, text)
	m.mu.Unlock()
}

func (m *model) drain() {
	m.mu.Lock()
	pending := m.pending
	m.pending = nil
	m.mu.Unlock()

	for _, line := range pending {
		if address := apruntime.FakeIPAddress(strings.TrimSpace(line)); address != "" {
			m.steamURL = address
		}
		if note := apruntime.ItemServerLine(line); note != "" {
			m.itemServer = note
		}
		if mission := apruntime.LoadedMission(line); mission != "" {
			m.mission = mission
		}
	}
	m.lines = append(m.lines, pending...)
	if len(m.lines) > linesMax {
		m.lines = m.lines[len(m.lines)-linesMax:]
	}
}

// The messages the model waits on: a tick that draws what arrived, a tick that
// asks the bridge for the run, and the answer to that question.
type (
	drainMsg   time.Time
	sessionMsg struct {
		snapshot session.Snapshot
		err      error
	}
	fetchMsg time.Time
)

func (m *model) Init() tea.Cmd {
	return tea.Batch(drainTick(), fetchTick(), m.start())
}

func drainTick() tea.Cmd {
	return tea.Tick(drainEvery, func(t time.Time) tea.Msg { return drainMsg(t) })
}

func fetchTick() tea.Cmd {
	return tea.Tick(sessionEvery, func(t time.Time) tea.Msg { return fetchMsg(t) })
}

// start brings the server up, off the drawing goroutine: Start installs
// nothing but does write the server configs and dial the room.
func (m *model) start() tea.Cmd {
	return func() tea.Msg {
		if err := m.supervisor.Start(func(error) {}); err != nil {
			m.take(apruntime.Line{At: time.Now(), Source: "launcher", Text: err.Error()})
		}
		return nil
	}
}

func (m *model) stop() tea.Cmd {
	return func() tea.Msg {
		m.supervisor.Stop()
		return nil
	}
}

// openSettings opens the same tabs the window opens, over the top of
// everything. It shows the one named, so the key that says it changes the team
// lands on the page that holds it.
func (m *model) openSettings(tab string) {
	m.form = newSettingsForm(m.settings, settingsDeps{
		saved:  m.applySettings,
		repair: m.repair,
		reset:  m.resetSettings,
	})
	m.form.showTab(tab)
}

// repair is the window's Repair button: everything the launcher started has to
// be down before the files it holds can go, so the server stops first.
func (m *model) repair() ([]string, error) {
	m.supervisor.Stop()
	root := m.supervisor.Settings().InstallRoot
	if _, err := winproc.KillUnder(root); err != nil {
		return nil, fmt.Errorf("cannot check the running programs: %w", err)
	}
	return installer.Clean(root)
}

// resetSettings is the factory answers, saved, with a new RCON password
// because the old one goes with the rest.
//
// The install root is not one of them. It says where 14 GB of game files are,
// not how the run is played, and resetting it would start the download again
// somewhere else.
func (m *model) resetSettings() (settings.Settings, error) {
	fresh := settings.Defaults()
	fresh.InstallRoot = m.supervisor.Settings().InstallRoot
	if password, err := settings.NewRconPassword(); err == nil {
		fresh.SrcdsRconPw = password
	}
	if err := settings.Save(fresh); err != nil {
		return settings.Settings{}, fmt.Errorf("cannot save the settings: %w", err)
	}
	// The environment still wins over the file, the way it does at startup.
	applied := settings.ApplyEnv(fresh)
	m.settings = applied
	m.supervisor.SetSettings(applied)
	return applied, nil
}

/*
applySettings saves what the form ended with and puts it where it is read.

The server reads its half at startup, so a change while it is running is a
change the running server does not have: the window restarts it for that
reason, and so does this.
*/
func (m *model) applySettings(next settings.Settings) tea.Cmd {
	before := m.settings
	if next.SrcdsRconPw == "" {
		if password, err := settings.NewRconPassword(); err == nil {
			next.SrcdsRconPw = password
		}
	}
	if _, err := settings.CheckRunSelection(next); err != nil {
		return func() tea.Msg { return noticeMsg(err.Error()) }
	}
	if err := settings.Save(next); err != nil {
		return func() tea.Msg { return noticeMsg("cannot save the settings: " + err.Error()) }
	}
	m.settings = next
	m.supervisor.SetSettings(next)
	// The player file is what the seed is generated from, so it follows the run
	// shape without being asked, the way it does in the window.
	if _, err := settings.WritePlayerFile(next, assets.ArchipelagoVersion); err != nil {
		return func() tea.Msg { return noticeMsg(err.Error()) }
	}

	if !m.supervisor.Running() {
		return func() tea.Msg { return noticeMsg("settings saved") }
	}
	/* A bot team is the one change a running mission takes: the mod re-reads
	 * its lineup from a convar and its weapons from a file, so the wave carries
	 * on. Everything else is read once at startup and needs the restart. */
	if botlive.LiveOnly(before, next) {
		return m.applyTeam(before)
	}
	return tea.Sequence(
		func() tea.Msg { return noticeMsg("settings saved, restarting the server") },
		m.stop(), m.start())
}

/*
	applyTeam hands the team the settings now hold to the running server.

The files first, because the mod reads the loadout file when it is told to
reseat, and then the commands in the order botlive puts them in.
*/
func (m *model) applyTeam(before settings.Settings) tea.Cmd {
	if err := srcdsconfig.Install(m.settings); err != nil {
		return func() tea.Msg { return noticeMsg("cannot write the bot files: " + err.Error()) }
	}
	commands := botlive.Commands(before, m.settings)
	sends := make([]tea.Cmd, 0, len(commands)+1)
	sends = append(sends, func() tea.Msg { return noticeMsg("applying the bot team to the running server") })
	for _, command := range commands {
		sends = append(sends, m.send(command))
	}
	return tea.Sequence(sends...)
}

// join starts Team Fortress 2 and connects it, the way the window's Join
// button does: Steam owns the steam:// scheme and carries the password with it.
func (m *model) join() tea.Cmd {
	return func() tea.Msg {
		link := apruntime.SteamConnectURL(m.settings, m.steamURL)
		if err := winproc.OpenURL(link); err != nil {
			return noticeMsg("cannot ask Steam to join: " + link +
				". Start the game yourself and find the server under the LAN tab of the server browser.")
		}
		return noticeMsg("joining: " + link)
	}
}

// copyJoin puts the join line where a paste will find it. OSC 52 is the
// terminal's own clipboard, so this works over SSH as well as locally, and on
// a terminal that does not take it the line is still on screen to read out.
func (m *model) copyJoin() tea.Cmd {
	line := strings.Join(m.joinAddresses(), " ")
	return func() tea.Msg {
		termenv.Copy(line)
		return noticeMsg("copied " + line)
	}
}

func (m *model) fetch() tea.Cmd {
	return func() tea.Msg {
		if !m.supervisor.Running() {
			return sessionMsg{}
		}
		snapshot, err := session.Fetch(context.Background(), session.BridgeURL)
		return sessionMsg{snapshot: snapshot, err: err}
	}
}

// send runs one RCON command and puts both the command and the answer in the
// log, which is where the player is already looking.
func (m *model) send(command string) tea.Cmd {
	return func() tea.Msg {
		m.take(apruntime.Line{At: time.Now(), Source: "rcon", Text: "> " + command})

		var client *rcon.Client
		var err error
		for _, address := range apruntime.RconAddresses(m.settings) {
			client, err = rcon.Dial(address, m.settings.SrcdsRconPw)
			if err == nil {
				break
			}
		}
		if err != nil {
			m.take(apruntime.Line{At: time.Now(), Source: "rcon", Text: err.Error()})
			return nil
		}
		defer func() { _ = client.Close() }()

		reply, err := client.Exec(command)
		if err != nil {
			m.take(apruntime.Line{At: time.Now(), Source: "rcon", Text: err.Error()})
			return nil
		}
		for line := range strings.SplitSeq(reply, "\n") {
			if strings.TrimSpace(line) != "" {
				m.take(apruntime.Line{At: time.Now(), Source: "rcon", Text: line})
			}
		}
		return nil
	}
}
