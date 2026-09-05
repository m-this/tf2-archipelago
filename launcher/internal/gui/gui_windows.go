//go:build windows

// Package gui is the launcher's window: a log view, a view of the run, the
// buttons that start and stop the server, and a box that sends RCON commands
// to it. It is the whole interface on Windows, where a player double-clicks
// the exe and never opens a terminal.
package gui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

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

// linesMax caps the log view: a wave produces a few hundred lines and an
// evening tens of thousands, which a TextEdit redraws badly. linesTrim is
// where the trim restarts, so the rewrite that costs happens once every few
// hundred lines rather than on every one.
const (
	linesMax  = 20000
	linesTrim = 5000

	// flushEvery batches the lines a busy install or a wave produces. Handing
	// the window one line at a time floods its message queue and it stops
	// repainting, which reads as a frozen log.
	flushEvery = 120 * time.Millisecond

	// sessionEvery is how often the Session tab asks the bridge for the run.
	// The bridge is on loopback and the answer is two small documents.
	sessionEvery = 5 * time.Second
)

// The status light. Red, amber and green, the way every other status light
// reads.
var (
	colorStopped  = walk.RGB(200, 40, 40)
	colorStarting = walk.RGB(220, 150, 0)
	colorRunning  = walk.RGB(30, 160, 60)
	colorMuted    = walk.RGB(110, 110, 110)
)

type window struct {
	main       *walk.MainWindow
	light      *walk.Label
	status     *walk.Label
	room       *walk.Label
	join       *walk.Label
	log        *walk.TextEdit
	command    *walk.LineEdit
	startStop  *walk.PushButton
	restart    *walk.PushButton
	joinBt     *walk.PushButton
	settingsBt *walk.PushButton
	session    *sessionTab
	unlocks    *unlocksTab
	bots       *botsTab

	supervisor *apruntime.Supervisor
	logger     *slog.Logger

	mu            sync.Mutex
	lines         []string
	pending       []string
	flushQueued   bool
	busy          bool
	cancelInstall context.CancelFunc
	logFile       *os.File

	// steamAddress is the relayed address the game server printed, empty
	// until it does and after every stop: it is a new one every start.
	steamAddress string

	// sourcemodRestarted is the one automatic restart SourceMod's updater
	// gets. It never clears: a second request in the same run means the
	// first restart did not settle it.
	sourcemodRestarted bool

	// mission is the one the plugin last loaded, empty until it says. The
	// settings only name where the run starts, and it moves on from there.
	mission string
}

// Run opens the window and blocks until the player closes it. The server is
// stopped on the way out, so closing the window is a clean shutdown.
func Run(s settings.Settings, logger *slog.Logger) error {
	// Win32 delivers a window's messages to the thread that created it, and Go
	// moves a goroutine between threads at any blocking call. Without this the
	// message loop can end up on a different thread from the window, which
	// leaves it half laid out: the toolbar and the log keep the sizes they had
	// at creation while the rest of the window resizes around them.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	w := &window{logger: logger, session: newSessionTab(), unlocks: newUnlocksTab(), bots: newBotsTab()}
	w.supervisor = apruntime.NewSupervisor(s, nil, w.append)
	w.openLogFile(s.InstallRoot)
	defer func() {
		if w.logFile != nil {
			_ = w.logFile.Close()
		}
	}()

	if err := w.build(); err != nil {
		return err
	}
	// An edit control holds 32 KB by default and silently drops everything
	// after that, which is a log that stops halfway through the install. 0
	// asks for the largest limit the control allows.
	win.SendMessage(w.log.Handle(), win.EM_SETLIMITTEXT, 0, 0)
	w.refresh()
	if w.logFile != nil {
		w.say("logging to %s", w.logFile.Name())
	}
	w.writePlayerFile(s)
	go w.watchSession()

	// A first run has no room address, so the settings dialog is the first
	// thing on screen rather than an error in the log.
	if s.APPort == 0 && !s.TestMode {
		w.main.Synchronize(func() { w.editSettings() })
	} else {
		go w.start()
	}

	w.main.Run()
	w.supervisor.Stop()
	return nil
}

func (w *window) build() error {
	return declarative.MainWindow{
		AssignTo: &w.main,
		Title:    assets.Title("Mann vs Archipelago"),
		// Wide enough for the settings dialog it opens: a dialog is laid out
		// inside its owner, so the owner is the ceiling on how much the Bots
		// tab can show in a line.
		Size:   declarative.Size{Width: 1200, Height: 760},
		Layout: declarative.VBox{},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout:    declarative.HBox{MarginsZero: true},
				MaxSize:   declarative.Size{Height: 28},
				Alignment: declarative.AlignHNearVCenter,
				Children: []declarative.Widget{
					declarative.Label{AssignTo: &w.light, Text: "●", TextColor: colorStopped, MinSize: declarative.Size{Width: 16}, ToolTipText: "Red: stopped. Amber: starting. Green: running."},
					declarative.Label{AssignTo: &w.status, Text: "stopped", MinSize: declarative.Size{Width: 60}},
					declarative.Label{AssignTo: &w.room, Text: "", TextColor: colorMuted, MinSize: declarative.Size{Width: 240}},
					declarative.HSpacer{},
					declarative.PushButton{AssignTo: &w.startStop, Text: "Start", OnClicked: w.onStartStop, MinSize: declarative.Size{Width: 90}},
					declarative.PushButton{AssignTo: &w.restart, Text: "Restart", OnClicked: w.onRestart, MinSize: declarative.Size{Width: 90}},
					declarative.PushButton{
						AssignTo: &w.joinBt,
						Text:     "Join",
						// The steam:// handoff can do nothing and report nothing:
						// Steam takes the link, the game starts, and the connect
						// is lost on the way. The server is running either way,
						// so the tooltip says how to get to it without this
						// button rather than leaving the player to guess.
						ToolTipText: "Start Team Fortress 2 and join this server. Steam does the connect and the password. " +
							"If the game opens without joining, the server is still there: find it in the game's own " +
							"server browser, under the LAN tab, or type the connect line below in the developer console.",
						OnClicked: w.onJoin,
						MinSize:   declarative.Size{Width: 90},
					},
					declarative.PushButton{AssignTo: &w.settingsBt, Text: "Settings", OnClicked: w.editSettings, MinSize: declarative.Size{Width: 90}},
				},
			},
			// The addresses friends type after "connect". Every address of
			// this machine, because the launcher cannot know which network
			// the friends are on, and the relayed one once Steam hands it out.
			declarative.Composite{
				Layout:  declarative.HBox{MarginsZero: true},
				MaxSize: declarative.Size{Height: 24},
				Children: []declarative.Widget{
					declarative.Label{Text: "Join:", TextColor: colorMuted, MinSize: declarative.Size{Width: 32}},
					declarative.Label{AssignTo: &w.join, Text: "", ToolTipText: "What your friends type after connect in the developer console."},
					declarative.HSpacer{},
					declarative.PushButton{Text: "Copy", ToolTipText: "Copy the join line to the clipboard.", OnClicked: w.copyJoin, MinSize: declarative.Size{Width: 60}},
				},
			},
			// The run first, the log second. The log is what a player needs
			// when something is wrong; the run is what they came for, and a
			// window that opens on a wall of server output reads as a console
			// somebody left running rather than as the thing they double-clicked.
			declarative.TabWidget{
				StretchFactor: 1,
				Pages: []declarative.TabPage{
					w.session.page(w.switchMission),
					w.unlocks.page(),
					w.bots.page(func() { w.editSettingsOn("Bots") }),
					{
						Title:  "Log",
						Layout: declarative.VBox{MarginsZero: true},
						Children: []declarative.Widget{
							// No HScroll, and a MinSize rather than a preferred
							// size: a TextEdit sized by its content makes one
							// long log line stretch the whole window past the
							// screen.
							declarative.TextEdit{
								AssignTo:      &w.log,
								ReadOnly:      true,
								VScroll:       true,
								MinSize:       declarative.Size{Width: 400, Height: 200},
								StretchFactor: 1,
							},
							declarative.Composite{
								Layout:  declarative.HBox{MarginsZero: true},
								MaxSize: declarative.Size{Height: 28},
								Children: []declarative.Widget{
									declarative.Label{Text: "rcon", MinSize: declarative.Size{Width: 30}},
									declarative.LineEdit{AssignTo: &w.command, OnKeyDown: w.onCommandKey},
									declarative.PushButton{Text: "Send", OnClicked: w.onSend, MinSize: declarative.Size{Width: 90}},
								},
							},
						},
					},
				},
			},
		},
	}.Create()
}

// append is the sink every log line arrives on, from goroutines that are not
// the UI thread. Synchronize hands the update to the thread that owns the
// window, which is the only one allowed to touch it.
//
// One line at a time, appended. Rewriting the whole buffer on every line is
// quadratic, and it makes the TextEdit ask the layout for a new size hundreds
// of times a wave, which drags the toolbar around with it.
func (w *window) append(line apruntime.Line) {
	text := fmt.Sprintf("%s  %-8s %s", line.At.Format("15:04:05"), line.Source, line.Text)

	w.mu.Lock()
	w.lines = append(w.lines, text)
	w.pending = append(w.pending, text)
	if w.logFile != nil {
		_, _ = w.logFile.WriteString(text + "\r\n")
	}
	queue := !w.flushQueued && w.main != nil
	if queue {
		w.flushQueued = true
	}
	w.mu.Unlock()

	if queue {
		time.AfterFunc(flushEvery, func() { w.main.Synchronize(w.flush) })
	}
	if line.Source == "srcds" {
		if address := apruntime.FakeIPAddress(line.Text); address != "" {
			w.noteSteamAddress(address)
		}
		if note := apruntime.ItemServerLine(line.Text); note != "" {
			w.say("%s", note)
		}
		if apruntime.SourceModWasUpdated(line.Text) {
			w.restartForSourcemod()
		}
		if mission := apruntime.LoadedMission(line.Text); mission != "" {
			w.noteMission(mission)
		}
	}
}

// noteMission keeps the mission the plugin last loaded, for the line above the
// log. The run moves itself from one mission to the next, so the one the
// settings name is only ever the first.
func (w *window) noteMission(mission string) {
	w.mu.Lock()
	changed := w.mission != mission
	w.mission = mission
	w.mu.Unlock()
	if changed && w.main != nil {
		w.main.Synchronize(func() {
			w.refresh()
			// The server's own word that the mission loaded, which is what
			// the line under the Session button is waiting for.
			w.session.noteSwitchLanded(mission)
		})
	}
}

// restartForSourcemod brings the server round once SourceMod has fetched new
// gamedata. The updater writes the files and then asks whoever is watching to
// restart, which is a line in a log nobody reads and a server left running on
// the gamedata it started with.
//
// Once per run of the launcher. The updater only speaks when it found
// something new, so a second time means the restart did not take, and
// restarting again would be a loop with a 14 GB game server in it.
func (w *window) restartForSourcemod() {
	w.mu.Lock()
	if w.sourcemodRestarted {
		w.mu.Unlock()
		return
	}
	w.sourcemodRestarted = true
	w.mu.Unlock()

	if !w.supervisor.Running() {
		return
	}
	w.say("SourceMod updated its gamedata. Restarting the server to load it.")
	go apruntime.Guard("a window task", w.sayLine, func() {
		w.supervisor.Stop()
		w.start()
	})
}

// noteSteamAddress keeps the relayed address the game server printed and
// puts it in the join line.
func (w *window) noteSteamAddress(address string) {
	w.mu.Lock()
	changed := w.steamAddress != address
	w.steamAddress = address
	w.mu.Unlock()
	if changed && w.main != nil {
		w.say("Steam relays this server at %s: your friends type connect %s", address, address)
		w.main.Synchronize(w.refresh)
	}
}

// flush writes the lines that arrived since the last one, on the UI thread.
func (w *window) flush() {
	w.mu.Lock()
	w.flushQueued = false
	pending := w.pending
	w.pending = nil
	trimmed := ""
	if len(w.lines) > linesMax {
		w.lines = w.lines[linesTrim:]
		trimmed = strings.Join(w.lines, "\r\n") + "\r\n"
	}
	w.mu.Unlock()

	switch {
	case trimmed != "":
		w.log.SetText(trimmed)
	case len(pending) > 0:
		w.log.AppendText(strings.Join(pending, "\r\n") + "\r\n")
	default:
		return
	}
	// Scroll the log itself to the bottom. Moving the caret instead makes the
	// window scroll sideways to reveal it.
	win.SendMessage(w.log.Handle(), win.WM_VSCROLL, win.SB_BOTTOM, 0)
}

// openLogFile keeps this run's log next to the game files. The window shows
// the same lines, but a file is what a player can send to somebody who can
// read it. One run per file, and the run before it is kept too.
func (w *window) openLogFile(installRoot string) {
	file, err := apruntime.CreateLogFile(installRoot)
	if err != nil {
		return
	}
	w.mu.Lock()
	w.logFile = file
	w.mu.Unlock()
}

func (w *window) say(format string, args ...any) {
	w.append(apruntime.Line{At: time.Now(), Source: "launcher", Text: fmt.Sprintf(format, args...)})
}

func (w *window) onStartStop() {
	if w.supervisor.Running() {
		go apruntime.Guard("a window task", w.sayLine, func() {
			w.supervisor.Stop()
			w.main.Synchronize(w.refresh)
		})
		return
	}
	go w.start()
}

func (w *window) onRestart() {
	go apruntime.Guard("a window task", w.sayLine, func() {
		w.supervisor.Stop()
		w.start()
	})
}

// start installs whatever is missing, writes the server configs, then brings
// the pair up. It runs off the UI thread: the first install downloads 14 GB.
func (w *window) start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.mu.Lock()
	if w.busy {
		w.mu.Unlock()
		cancel()
		return
	}
	w.busy, w.cancelInstall = true, cancel
	w.steamAddress = ""
	w.mission = ""
	w.mu.Unlock()
	defer func() {
		cancel()
		w.mu.Lock()
		w.busy, w.cancelInstall = false, nil
		w.mu.Unlock()
		w.main.Synchronize(w.refresh)
	}()

	w.main.Synchronize(w.refresh)
	s := w.supervisor.Settings()

	if _, err := installer.Ensure(ctx, s.InstallRoot, settings.CommunityArchives(s), w.installLog); err != nil {
		if ctx.Err() != nil {
			return
		}
		w.say("install failed: %v", err)
		w.say("%s.", installer.RepairAdvice)
		return
	}
	// The game server inherits the hidden console allocated at startup, so it
	// opens no window of its own. Taking the focus back covers the case where
	// something did appear.
	defer w.main.Synchronize(func() { _ = w.main.SetFocus() })

	for _, line := range apruntime.ConnectLines(s) {
		w.say("%s", line)
	}

	if err := w.supervisor.Start(func(err error) {
		if err != nil {
			w.say("%v", err)
		}
		w.main.Synchronize(w.refresh)
	}); err != nil {
		w.showStartError(err)
	}
}

// showStartError makes an enabled-but-unavailable Funnel impossible to miss.
// The server is still stopped when this runs. First-time approval gets the
// browser; sign-in and service failures leave the operator with the exact fix.
func (w *window) showStartError(err error) {
	w.say("%v", err)
	var fastDL *apruntime.TailscaleFastDLStartError
	if !errors.As(err, &fastDL) {
		return
	}
	w.main.Synchronize(func() {
		message := err.Error()
		if fastDL.ApprovalURL != "" {
			message += "\n\nThe Tailscale approval page will open after you dismiss this message."
		}
		walk.MsgBox(w.main, "Tailscale FastDL needs attention", message, walk.MsgBoxIconWarning)
		if fastDL.ApprovalURL == "" {
			return
		}
		if openErr := winproc.OpenURL(fastDL.ApprovalURL); openErr != nil {
			walk.MsgBox(w.main, "Enable Tailscale Funnel", fastDL.ApprovalURL+"\n\n"+openErr.Error(), walk.MsgBoxIconWarning)
		}
	})
}

func (w *window) installLog(format string, args ...any) {
	w.append(apruntime.Line{At: time.Now(), Source: "install", Text: fmt.Sprintf(format, args...)})
}

func (w *window) refresh() {
	running := w.supervisor.Running()
	s := w.supervisor.Settings()

	w.mu.Lock()
	busy := w.busy
	steamAddress := w.steamAddress
	mission := w.mission
	w.mu.Unlock()

	// What is loaded, once the plugin has said so. Before that, and with the
	// server down, the settings only know where the run is meant to begin.
	playing := apruntime.StartMap(s)
	if running && mission != "" {
		playing = mission
	}

	switch {
	case busy && !running:
		w.light.SetTextColor(colorStarting)
		w.status.SetText("starting")
	case running:
		w.light.SetTextColor(colorRunning)
		w.status.SetText("running")
	default:
		w.light.SetTextColor(colorStopped)
		w.status.SetText("stopped")
	}
	room := settings.Room{Host: s.APHost, Port: s.APPort}
	switch {
	case s.TestMode:
		w.room.SetText(fmt.Sprintf("test mode   %s", playing))
	case room.Port == 0:
		w.room.SetText("no room set")
	default:
		w.room.SetText(fmt.Sprintf("room %s   %s", room, playing))
	}
	w.join.SetText(joinLine(s, steamAddress))
	if running {
		w.startStop.SetText("Stop")
	} else {
		w.startStop.SetText("Start")
	}
	w.restart.SetEnabled(running)
	w.joinBt.SetEnabled(running)
	w.command.SetEnabled(running)
	w.session.setRunning(running)
	w.unlocks.setRunning(running)
	w.bots.show(s)
	w.bots.setRunning(running)
}

// applyBotTeam hands the team the settings hold to the running server. The
// files first, because the mod reads the loadout file when it is told to
// reseat: the other order gives the team back its old weapons.
func (w *window) applyBotTeam(before settings.Settings) {
	if !w.supervisor.Running() {
		w.bots.say("The server is not running. Press Start first.")
		return
	}
	s := w.supervisor.Settings()
	if err := srcdsconfig.Install(s); err != nil {
		w.bots.say("Cannot write the bot files: " + err.Error())
		w.say("bots: %v", err)
		return
	}
	w.bots.say("Applying the team. The bots keep the money they have earned.")
	for _, command := range botlive.Commands(before, s) {
		w.runRcon(command)
	}
}

// joinLine is what the status bar shows under the buttons: every address of
// this machine on the game port, and the relayed one when Steam has handed it
// out. Over Steam, that address is the one to give out; the others still work
// for the people in the room.
func joinLine(s settings.Settings, steamAddress string) string {
	port := fmt.Sprintf("%d", s.SrcdsPort)
	var parts []string
	// First, and named, because over Steam this is the address: the ones below
	// it are this network's and mean nothing to the friend being sent them.
	// Valve hands it out a moment after the server starts and gives a new one
	// every time, so it cannot be written down anywhere but here.
	if s.SrcdsReach == settings.ReachSteam {
		if steamAddress == "" {
			parts = append(parts, "Steam public IP: waiting for Steam to assign one")
		} else {
			parts = append(parts, "Steam public IP: "+steamAddress)
		}
	}
	for _, address := range apruntime.LocalAddresses() {
		parts = append(parts, address+":"+port)
	}
	if len(parts) == 0 {
		parts = append(parts, "127.0.0.1:"+port)
	}
	line := strings.Join(parts, "   ")
	if s.SrcdsPw != "" {
		line += "   (password " + s.SrcdsPw + ")"
	}
	return line
}

// copyJoin puts the join line on the clipboard, so it can go into a chat.
func (w *window) copyJoin() {
	if err := walk.Clipboard().SetText(w.join.Text()); err != nil {
		w.say("cannot copy: %v", err)
	}
}

// onJoin starts the game and joins, the way a server browser's Join button
// does: Steam owns the steam:// scheme and passes the connect and the password
// on itself.
//
// The game takes a while to start, and nothing comes back to say it worked, so
// the log carries the link. A player whose Steam did not answer can paste it.
func (w *window) onJoin() {
	w.mu.Lock()
	steamAddress := w.steamAddress
	w.mu.Unlock()
	link := apruntime.SteamConnectURL(w.supervisor.Settings(), steamAddress)
	w.say("joining: %s", link)
	if err := winproc.OpenURL(link); err != nil {
		w.say("cannot ask Steam to join: %v", err)
		w.say("start Team Fortress 2 yourself, then find the server under the LAN tab of the server browser, " +
			"or type the connect line above in the developer console")
	}
}

// dialRcon reaches the game server on whichever address it bound its rcon port
// to. See runtime.RconAddresses: which one that is depends on the reach.
func dialRcon(s settings.Settings) (*rcon.Client, error) {
	var err error
	for _, address := range apruntime.RconAddresses(s) {
		var client *rcon.Client
		client, err = rcon.Dial(address, s.SrcdsRconPw)
		if err == nil {
			return client, nil
		}
	}
	return nil, err
}

// onCommandKey sends on Enter. OnEditingFinished would also fire when the box
// loses focus, which would send whatever was half-typed.
func (w *window) onCommandKey(key walk.Key) {
	if key == walk.KeyReturn {
		w.onSend()
	}
}

// onSend runs one RCON command against the running server and prints what came
// back. The connection is per command: the server is on loopback, and a held
// connection is one more thing to reconnect after a map change.
func (w *window) onSend() {
	command := strings.TrimSpace(w.command.Text())
	if command == "" {
		return
	}
	w.command.SetText("")
	w.runRcon(command)
}

// runRcon sends one command and logs the answer. The Session tab uses it for
// the mission switch, the rcon box for whatever was typed.
func (w *window) runRcon(command string) {
	w.say("> %s", command)
	s := w.supervisor.Settings()
	go apruntime.Guard("a window task", w.sayLine, func() {
		client, err := dialRcon(s)
		if err != nil {
			w.say("rcon: %v", err)
			return
		}
		defer func() { _ = client.Close() }()
		reply, err := client.Exec(command)
		if err != nil {
			w.say("rcon: %v", err)
			return
		}
		for _, line := range strings.Split(reply, "\n") {
			if strings.TrimSpace(line) != "" {
				w.append(apruntime.Line{At: time.Now(), Source: "rcon", Text: line})
			}
		}
	})
}

// switchMission is what the Session tab's button does: the plugin's own
// switcher, over rcon, which refuses a mission the run has not unlocked.
func (w *window) switchMission(popFile string) {
	if !w.supervisor.Running() {
		w.session.saySwitch("The server is not running. Press Start first.")
		w.say("the server is not running")
		return
	}
	// The tab has already said what it is loading: it knows the mission's name
	// and its map, and this only has the popfile.
	w.runRcon("sm_ap_mission " + popFile)
}

// watchSession asks the bridge for the run every few seconds while the server
// is up, and hands the answer to the Session tab. It never touches the window
// itself: the tab does that on the UI thread.
func (w *window) watchSession() {
	ticker := time.NewTicker(sessionEvery)
	defer ticker.Stop()
	for range ticker.C {
		if !w.supervisor.Running() {
			continue
		}
		snapshot, err := session.Fetch(context.Background(), session.BridgeURL)
		w.main.Synchronize(func() {
			w.session.update(snapshot, err)
			w.unlocks.update(snapshot, err)
		})
	}
}

// idle stops the server and any install in flight, and waits for both. Repair
// deletes files those two hold open, so it has to run first.
func (w *window) idle() {
	w.mu.Lock()
	cancel := w.cancelInstall
	w.mu.Unlock()
	if cancel != nil {
		w.say("stopping the install first")
		cancel()
	}
	w.supervisor.Stop()
	for range 100 {
		w.mu.Lock()
		busy := w.busy
		w.mu.Unlock()
		if !busy {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// editSettings opens the dialog with the handful of values a player changes.
// The rest stay in `tf2ap.exe -configure`.
func (w *window) editSettings() { w.editSettingsOn("") }

// editSettingsOn opens the settings showing one tab, for a button that names
// what it is about to change.
func (w *window) editSettingsOn(tab string) {
	s := w.supervisor.Settings()
	next, ok, err := runSettingsDialog(w.main, s, w.repair, w.resetSettings, w.say, tab)
	if err != nil {
		w.say("settings: %v", err)
		return
	}
	if !ok {
		return
	}
	if next.SrcdsRconPw == "" {
		if password, err := settings.NewRconPassword(); err == nil {
			next.SrcdsRconPw = password
		}
	}
	if err := settings.Save(next); err != nil {
		w.say("cannot save the settings: %v", err)
		return
	}
	w.supervisor.SetSettings(next)
	w.writePlayerFile(next)
	w.refresh()

	/* Stopped stays stopped. Saving a setting is not asking for a server, and a
	 * Save that started one took the 14 GB install with it the first time.
	 * Start is the button that starts the server, and it is the only one. */
	if !w.supervisor.Running() {
		w.say("settings saved. Press Start when you want the server.")
		return
	}

	/* A bot team is the one change the running mission takes: the mod re-reads
	 * its lineup from a convar and its weapons from a file. Restarting for it
	 * ended the mission somebody was four waves into, which is what made
	 * changing a lineup mid-run not worth doing. */
	if botlive.LiveOnly(s, next) {
		w.applyBotTeam(s)
		return
	}

	// Everything else reaches the game server through server.cfg and the
	// command line, both of which it reads once at startup. Saving one while
	// the server runs used to change nothing until the player pressed Restart
	// themselves, and the log line saying so was easy to miss.
	w.say("settings saved. Restarting the server to apply them.")
	go apruntime.Guard("a window task", w.sayLine, func() {
		w.supervisor.Stop()
		w.start()
	})
}

// repair is what the dialog's Repair button calls. Everything the launcher
// started has to be down before the files it holds can go.
func (w *window) repair() ([]string, error) {
	w.idle()
	root := w.supervisor.Settings().InstallRoot
	killed, err := winproc.KillUnder(root)
	for _, path := range killed {
		w.say("repair stopped %s", path)
	}
	if err != nil {
		w.say("repair could not check the running programs: %v", err)
	}
	removed, err := installer.Clean(root)
	for _, path := range removed {
		w.say("repair removed %s", path)
	}
	if err != nil {
		w.say("repair: %v", err)
	}
	w.main.Synchronize(w.refresh)
	return removed, err
}

// resetSettings is what the dialog's Reset settings button calls: the factory
// answers, saved, with a new rcon password because the old one is gone with
// the rest.
//
// The install root is not one of them. It says where 14 GB of game files are,
// not how the run is played, and resetting it would start the download again
// somewhere else.
func (w *window) resetSettings() error {
	fresh := settings.Defaults()
	fresh.InstallRoot = w.supervisor.Settings().InstallRoot
	if password, err := settings.NewRconPassword(); err == nil {
		fresh.SrcdsRconPw = password
	}
	if err := settings.Save(fresh); err != nil {
		return fmt.Errorf("cannot save the settings: %w", err)
	}
	// The environment still wins over the file, the way it does at startup.
	w.supervisor.SetSettings(settings.ApplyEnv(fresh))
	w.say("settings reset. Press Settings to go through them, then Start.")
	w.main.Synchronize(w.refresh)
	return nil
}

// writePlayerFile keeps the Archipelago player file in step with the run
// shape. The player copies it into the Archipelago app and generates there;
// this launcher does not generate seeds, because that is Python.
func (w *window) writePlayerFile(s settings.Settings) {
	path, err := settings.WritePlayerFile(s, assets.ArchipelagoVersion)
	if err != nil {
		w.say("%v", err)
		return
	}
	w.say("wrote %s: copy it into the Archipelago app's Players folder to generate the seed", path)
}

// Available reports whether this build has a window.
func Available() bool { return true }

// sayLine is say with no formatting, which is the shape Guard wants: it hands
// over one finished line rather than a format and its arguments.
func (w *window) sayLine(text string) { w.say("%s", text) }
