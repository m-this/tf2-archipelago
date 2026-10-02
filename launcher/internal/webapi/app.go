/*
Package webapi is the launcher, as the browser talks to it.

App owns everything one run has: the settings, the supervisor holding the game
server and the bridge, the log, the session read off the bridge, and the
settings draft while the screen is open. It knows nothing about HTTP. Every
interface asks it the same questions and gets the same Snapshot back, which is
the point: a second interface cannot answer differently from the first.

The Connect services beside it are the wire. They translate a request into one
of App's methods and a Snapshot into proto, and hold no state of their own.
*/
package webapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/rcon"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/session"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/tailscalefastdl"
)

const (
	// linesMax is how much log one run keeps. Twenty thousand lines is a long
	// evening of srcds, and the oldest go when it is reached.
	linesMax     = 20000
	sessionEvery = 5 * time.Second

	// listenerQueue is how far one interface may fall behind before its
	// events start being dropped rather than buffered.
	listenerQueue = 128
)

type Event struct {
	Name string
	Data any
}

// App owns UI state and translates HTTP commands into launcher operations.
// It deliberately has no browser concepts beyond publishing plain events.
type App struct {
	mu sync.Mutex

	settings            settings.Settings
	supervisor          *apruntime.Supervisor
	logs                []apruntime.Line
	busy                bool
	activity            string
	install             context.CancelFunc
	steamURL            string
	mission             string
	snapshot            session.Snapshot
	fetchErr            error
	notice              string
	noticeSeq           uint64
	draft               *form.State
	formPage            string
	community           []string
	imported            []string
	serverMods          []string
	smAsked             bool
	smHeld              bool
	itemServer          string
	gameBuild           string
	gameUpdateAvailable bool
	gameUpdateError     string
	logFile             *os.File
	attached            bool
	attachedUp          bool
	attachedEnvFile     string
	authorizeFunnel     func(context.Context) (tailscalefastdl.Authorization, error)
	funnelAdvice        func(error) string

	listeners map[*Listener]struct{}
	quit      chan struct{}
	quitOnce  sync.Once
}

func New(s settings.Settings, logger *slog.Logger) *App {
	community := availableCommunityPackNames(s.CommunityContentDir)
	a := &App{
		settings:   s,
		community:  community,
		imported:   importedCommunityPackNames(s.CommunityContentDir, community),
		serverMods: installer.ReadyServerMods(s.InstallRoot),
		listeners:  make(map[*Listener]struct{}),
		quit:       make(chan struct{}),
		authorizeFunnel: func(ctx context.Context) (tailscalefastdl.Authorization, error) {
			return authorizeFunnel(ctx)
		},
		funnelAdvice: funnelSetupAdvice,
	}
	a.supervisor = apruntime.NewSupervisor(s, logger, a.append)
	a.gameBuild = a.readGameBuild(s.InstallRoot)
	return a
}

// NewAttached observes a bridge and game server that another supervisor owns.
// Compose is that supervisor: mounting the Docker socket just to make the
// launcher's Start and Stop buttons work would give this page control of the
// whole host, so the attached UI deliberately limits itself to observation,
// mission changes and RCON.
func NewAttached(s settings.Settings, logger *slog.Logger, envFile string) *App {
	a := New(s, logger)
	a.attached = true
	a.attachedEnvFile = envFile
	return a
}

func (a *App) append(line apruntime.Line) {
	a.mu.Lock()
	a.logs = append(a.logs, line)
	if len(a.logs) > linesMax {
		a.logs = a.logs[len(a.logs)-linesMax:]
	}
	restart := false
	stateChanged := false
	if a.logFile != nil {
		_, _ = fmt.Fprintf(a.logFile, "%s  %-8s %s\n", line.At.Format("15:04:05"), line.Source, line.Text)
	}
	if line.Source == "srcds" {
		if address := apruntime.FakeIPAddress(strings.TrimSpace(line.Text)); address != "" {
			stateChanged = address != a.steamURL
			a.steamURL = address
		}
		if mission := apruntime.LoadedMission(line.Text); mission != "" {
			a.mission = mission
		}
		if note := apruntime.ItemServerLine(line.Text); note != "" {
			a.itemServer = note
		}
		if apruntime.GameUpdateRequired(line.Text) && !a.gameUpdateAvailable {
			a.gameUpdateAvailable = true
			stateChanged = true
		}
		if apruntime.SourceModWasUpdated(line.Text) && !a.smAsked {
			a.smAsked = true
			a.smHeld = a.draft != nil
			restart = !a.smHeld
		}
	}
	a.publishLocked(Event{Name: "log", Data: line})
	if stateChanged {
		a.publishLocked(Event{Name: "state", Data: struct{}{}})
	}
	a.mu.Unlock()
	if restart {
		a.Say("SourceMod updated its gamedata. Restarting the server to load it.")
		a.Restart()
	}
}

func (a *App) Say(format string, args ...any) {
	a.append(apruntime.Line{At: time.Now(), Source: "launcher", Text: fmt.Sprintf(format, args...)})
}

func (a *App) sayLine(text string) { a.Say("%s", text) }

func (a *App) Notify(text string) {
	a.mu.Lock()
	a.notice = text
	a.noticeSeq++
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	a.mu.Unlock()
	a.Say("%s", text)
}

func (a *App) publishState() {
	a.mu.Lock()
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	a.mu.Unlock()
}

// publishLocked hands one event to every interface listening. A queue that is
// full is not waited on: the launcher must not stall because a browser stopped
// reading. The Listener is marked behind instead, and reads the whole state
// again once it catches up, so dropping an event never leaves a stale screen.
func (a *App) publishLocked(message Event) {
	for listener := range a.listeners {
		select {
		case listener.events <- message:
		default:
			listener.behind.Store(true)
		}
	}
}

// Starting the competing server processes is application-lifetime work. It
// deliberately survives the HTTP request that triggered it.
//
//nolint:contextcheck // The server lifecycle is independent of browser requests.
func (a *App) Start() {
	a.mu.Lock()
	if a.attached {
		a.mu.Unlock()
		a.Notify("Docker Compose owns the server lifecycle. Run: docker compose up -d")
		return
	}
	if a.busy || a.supervisor.Running() {
		a.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.busy, a.install, a.steamURL, a.mission = true, cancel, "", ""
	s := a.settings
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	a.mu.Unlock()
	go apruntime.Guard("the browser interface", a.sayLine, func() {
		defer func() {
			cancel()
			a.mu.Lock()
			a.busy, a.install = false, nil
			a.publishLocked(Event{Name: "state", Data: struct{}{}})
			a.mu.Unlock()
		}()
		logf := func(f string, args ...any) {
			a.append(apruntime.Line{At: time.Now(), Source: "install", Text: fmt.Sprintf(f, args...)})
		}
		if _, err := installer.Ensure(ctx, s.InstallRoot, settings.CommunityArchives(s), settings.ServerModKeys(s), logf); err != nil {
			if ctx.Err() == nil {
				a.Say("install failed: %v", err)
				a.Say("%s.", installer.RepairAdvice)
			}
			return
		}
		err := installer.UpdateGame(ctx, s.InstallRoot, logf)
		if ctx.Err() != nil {
			return
		}
		a.gameUpdateFinished(err)
		if err != nil {
			a.Say("TF2 update failed: %v", err)
			return
		}
		a.boot(s)
	})
}

// boot starts srcds and the bridge on an installed, updated server. Start
// runs it after the install and the update, and so does UpdateGame.
func (a *App) boot(s settings.Settings) {
	// Which mods this run loads, as opposed to has installed. Off and
	// "only when a mission needs it" are answered here, by writing or
	// removing each one's autoload marker before srcds reads it.
	if err := installer.SetServerModLoading(s.InstallRoot, settings.ServerModsToLoad(s, runtime.GOOS)); err != nil {
		a.Say("%v", err)
		return
	}
	a.mu.Lock()
	a.serverMods = installer.ReadyServerMods(s.InstallRoot)
	readyMods := slices.Clone(a.serverMods)
	a.mu.Unlock()
	if err := settings.CheckServerModsReady(s, readyMods); err != nil {
		a.Say("server mod setup is incomplete: %v", err)
		return
	}
	for _, line := range apruntime.ConnectLines(s) {
		a.Say("%s", line)
	}
	if err := a.supervisor.Start(func(err error) {
		if err != nil {
			a.Say("%v", err)
		}
		a.publishState()
	}); err != nil {
		a.Say("%v", err)
		var funnel *apruntime.TailscaleFastDLStartError
		if errors.As(err, &funnel) && funnel.ApprovalURL != "" {
			a.Notify("Tailscale Funnel approval required: " + funnel.ApprovalURL)
		}
	}
}

func (a *App) Stop() {
	a.mu.Lock()
	if a.attached {
		a.mu.Unlock()
		a.Notify("Docker Compose owns the server lifecycle. Run: docker compose stop")
		return
	}
	cancel := a.install
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	a.supervisor.Stop()
	a.publishState()
}

func (a *App) Restart() {
	a.mu.Lock()
	attached := a.attached
	a.mu.Unlock()
	if attached {
		a.Notify("Docker Compose owns the server lifecycle. Run: docker compose up -d --force-recreate")
		return
	}
	go func() {
		a.Stop()
		a.Start()
	}()
}

func (a *App) Quit() { a.quitOnce.Do(func() { close(a.quit) }) }

func (a *App) SendRCON(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}
	a.append(apruntime.Line{At: time.Now(), Source: "rcon", Text: "> " + command})
	go apruntime.Guard("an RCON command", a.sayLine, func() {
		client, err := dialRCON(a.supervisor.Settings())
		if err != nil {
			a.Say("rcon: %v", err)
			return
		}
		defer func() { _ = client.Close() }()
		reply, err := client.Exec(command)
		if err != nil {
			a.Say("rcon: %v", err)
			return
		}
		for line := range strings.SplitSeq(reply, "\n") {
			if strings.TrimSpace(line) != "" {
				a.append(apruntime.Line{At: time.Now(), Source: "rcon", Text: line})
			}
		}
	})
}

func dialRCON(s settings.Settings) (*rcon.Client, error) {
	var last error
	for _, address := range apruntime.RconAddresses(s) {
		client, err := rcon.Dial(address, s.SrcdsRconPw)
		if err == nil {
			return client, nil
		}
		last = err
	}
	return nil, last
}

func srcdsAvailable(s settings.Settings) bool {
	dialer := net.Dialer{Timeout: time.Second}
	for _, address := range apruntime.RconAddresses(s) {
		connection, err := dialer.DialContext(context.Background(), "tcp", address)
		if err == nil {
			_ = connection.Close()
			return true
		}
	}
	return false
}

func (a *App) WatchSession() {
	ticker := time.NewTicker(sessionEvery)
	defer ticker.Stop()
	for {
		a.refreshSession()
		select {
		case <-a.quit:
			return
		case <-ticker.C:
		}
	}
}

func (a *App) refreshSession() {
	a.mu.Lock()
	attached := a.attached
	a.mu.Unlock()
	if !attached && !a.supervisor.Running() {
		return
	}
	snapshot, err := session.Fetch(context.Background(), session.BridgeURL)
	up := false
	if attached {
		// A TCP probe is enough to know SRCDS is up. Authenticating through RCON
		// here made the server log an empty command every five seconds; actual
		// browser commands still use authenticated RCON above.
		up = srcdsAvailable(a.supervisor.Settings())
	}
	a.mu.Lock()
	a.snapshot, a.fetchErr, a.attachedUp = snapshot, err, up
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	a.mu.Unlock()
}

// Quitting closes when the interface has been told to stop, by the Quit button
// or by a failure the launcher cannot serve through. A caller waits on it
// beside the operating system's signals.
func (a *App) Quitting() <-chan struct{} { return a.quit }

// LogTo copies every line to file as well as to the interfaces. The caller owns
// the file and closes it; a nil file turns the copy off.
func (a *App) LogTo(file *os.File) {
	a.mu.Lock()
	a.logFile = file
	a.mu.Unlock()
}

// Listener is one interface listening. behind says the launcher had to drop
// an event for it, which is a promise to read everything again rather than an
// event lost.
type Listener struct {
	events chan Event
	behind atomic.Bool
}

// Subscribe returns one listener and the func that ends it. Call the func once,
// when done.
func (a *App) Subscribe() (*Listener, func()) {
	listener := &Listener{events: make(chan Event, listenerQueue)}
	a.mu.Lock()
	a.listeners[listener] = struct{}{}
	a.mu.Unlock()
	return listener, func() {
		a.mu.Lock()
		delete(a.listeners, listener)
		a.mu.Unlock()
	}
}

// Events is what the listener has been handed and has not read yet.
func (s *Listener) Events() <-chan Event { return s.events }

// Behind reports and clears whether the launcher had to drop an event. A
// listener that sees true owes itself a whole snapshot.
func (s *Listener) Behind() bool { return s.behind.Swap(false) }
