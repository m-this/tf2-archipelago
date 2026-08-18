//go:build windows

// Package gui is the launcher's window: a log view, the buttons that start and
// stop the server, and a box that sends RCON commands to it. It is the whole
// interface on Windows, where a player double-clicks the exe and never opens a
// terminal.
package gui

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/rcon"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/srcdsconfig"
)

// linesMax caps the log view: a wave produces a few hundred lines and an
// evening tens of thousands, which a TextEdit redraws badly. linesTrim is
// where the trim restarts, so the rewrite that costs happens once every few
// hundred lines rather than on every one.
const (
	linesMax  = 4000
	linesTrim = 1000
)

type window struct {
	main       *walk.MainWindow
	status     *walk.Label
	room       *walk.Label
	log        *walk.TextEdit
	command    *walk.LineEdit
	startStop  *walk.PushButton
	restart    *walk.PushButton
	settingsBt *walk.PushButton

	supervisor *apruntime.Supervisor
	logger     *slog.Logger

	mu    sync.Mutex
	lines []string
	busy  bool
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

	w := &window{logger: logger}
	w.supervisor = apruntime.NewSupervisor(s, nil, w.append)

	if err := w.build(); err != nil {
		return err
	}
	w.refresh()

	// A first run has no room address, so the settings dialog is the first
	// thing on screen rather than an error in the log.
	if s.APPort == 0 {
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
		Title:    "Mann vs Archipelago",
		Size:     declarative.Size{Width: 900, Height: 600},
		Layout:   declarative.VBox{},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout:    declarative.HBox{MarginsZero: true},
				MaxSize:   declarative.Size{Height: 28},
				Alignment: declarative.AlignHNearVCenter,
				Children: []declarative.Widget{
					declarative.Label{AssignTo: &w.status, Text: "stopped", MinSize: declarative.Size{Width: 60}},
					declarative.Label{AssignTo: &w.room, Text: "", MinSize: declarative.Size{Width: 240}},
					declarative.HSpacer{},
					declarative.PushButton{AssignTo: &w.startStop, Text: "Start", OnClicked: w.onStartStop, MinSize: declarative.Size{Width: 90}},
					declarative.PushButton{AssignTo: &w.restart, Text: "Restart", OnClicked: w.onRestart, MinSize: declarative.Size{Width: 90}},
					declarative.PushButton{AssignTo: &w.settingsBt, Text: "Settings", OnClicked: w.editSettings, MinSize: declarative.Size{Width: 90}},
				},
			},
			// No HScroll, and a MinSize rather than a preferred size: a
			// TextEdit sized by its content makes one long log line stretch
			// the whole window past the screen.
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
	trimmed := ""
	if len(w.lines) > linesMax {
		w.lines = w.lines[linesTrim:]
		trimmed = strings.Join(w.lines, "\r\n") + "\r\n"
	}
	w.mu.Unlock()

	if w.main == nil {
		return
	}
	w.main.Synchronize(func() {
		if trimmed != "" {
			w.log.SetText(trimmed)
		} else {
			w.log.AppendText(text + "\r\n")
		}
		// Scroll the log itself to the bottom. Moving the caret instead makes
		// the window scroll sideways to reveal it.
		win.SendMessage(w.log.Handle(), win.WM_VSCROLL, win.SB_BOTTOM, 0)
	})
}

func (w *window) say(format string, args ...any) {
	w.append(apruntime.Line{At: time.Now(), Source: "launcher", Text: fmt.Sprintf(format, args...)})
}

func (w *window) onStartStop() {
	if w.supervisor.Running() {
		go func() {
			w.supervisor.Stop()
			w.main.Synchronize(w.refresh)
		}()
		return
	}
	go w.start()
}

func (w *window) onRestart() {
	go func() {
		w.supervisor.Stop()
		w.start()
	}()
}

// start installs whatever is missing, writes the server configs, then brings
// the pair up. It runs off the UI thread: the first install downloads 14 GB.
func (w *window) start() {
	w.mu.Lock()
	if w.busy {
		w.mu.Unlock()
		return
	}
	w.busy = true
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		w.busy = false
		w.mu.Unlock()
		w.main.Synchronize(w.refresh)
	}()

	w.main.Synchronize(w.refresh)
	s := w.supervisor.Settings()

	if _, err := installer.Ensure(context.Background(), s.InstallRoot, w.installLog); err != nil {
		w.say("install failed: %v", err)
		return
	}
	if err := srcdsconfig.Install(s); err != nil {
		w.say("cannot write the server configs: %v", err)
		return
	}
	if err := w.supervisor.Start(func(err error) {
		if err != nil {
			w.say("%v", err)
		}
		w.main.Synchronize(w.refresh)
	}); err != nil {
		w.say("%v", err)
	}
}

func (w *window) installLog(format string, args ...any) {
	w.append(apruntime.Line{At: time.Now(), Source: "install", Text: fmt.Sprintf(format, args...)})
}

func (w *window) refresh() {
	running := w.supervisor.Running()
	s := w.supervisor.Settings()

	w.mu.Lock()
	busy := w.busy
	w.mu.Unlock()

	switch {
	case busy && !running:
		w.status.SetText("starting")
	case running:
		w.status.SetText("running")
	default:
		w.status.SetText("stopped")
	}
	room := settings.Room{Host: s.APHost, Port: s.APPort}
	if room.Port == 0 {
		w.room.SetText("no room set")
	} else {
		w.room.SetText(fmt.Sprintf("room %s   map %s", room, s.SrcdsStartMap))
	}
	if running {
		w.startStop.SetText("Stop")
	} else {
		w.startStop.SetText("Start")
	}
	w.restart.SetEnabled(running)
	w.command.SetEnabled(running)
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
	w.say("> %s", command)

	s := w.supervisor.Settings()
	go func() {
		address := fmt.Sprintf("127.0.0.1:%d", s.SrcdsPort)
		client, err := rcon.Dial(address, s.SrcdsRconPw)
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
	}()
}

// editSettings opens the dialog with the handful of values a player changes.
// The rest stay in `tf2ap.exe -configure`.
func (w *window) editSettings() {
	s := w.supervisor.Settings()
	next, ok, err := runSettingsDialog(w.main, s)
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
	w.refresh()
	w.say("settings saved. Restart to apply them to a running server.")
	if !w.supervisor.Running() {
		go w.start()
	}
}

// mapNames lists the maps for the dialog's combo box. gamedata owns the list.
func mapNames() []string {
	names := make([]string, 0, len(gamedata.Maps))
	for _, m := range gamedata.Maps {
		names = append(names, m.Name)
	}
	return names
}

// Available reports whether this build has a window.
func Available() bool { return true }
