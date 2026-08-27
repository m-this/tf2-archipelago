package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/m-this/tf2-archipelago/bridge"
	"github.com/m-this/tf2-archipelago/bridge/config"
	"github.com/m-this/tf2-archipelago/fakeroom"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/srcdsconfig"
)

// roomCloseGrace bounds the wait when the test room is asked to stop.
const roomCloseGrace = 3 * time.Second

// Line is one entry for the log view: where it came from and what it said.
type Line struct {
	At     time.Time
	Source string
	Text   string
}

// Sink receives every line the bridge and the game server produce. It is
// called from several goroutines, so an implementation that touches a window
// has to hand the line to the UI thread itself.
type Sink func(Line)

// Supervisor owns the pair of processes: the bridge in this process, and the
// game server beside it. Start and Stop are what the buttons call, so both are
// safe to call twice and safe to call from the UI thread.
type Supervisor struct {
	settings settings.Settings
	logger   *slog.Logger
	sink     Sink

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	running bool
	stopped bool // a Stop the operator asked for, so the exit is not an error
}

// NewSupervisor returns a stopped supervisor. sink may be nil.
func NewSupervisor(s settings.Settings, logger *slog.Logger, sink Sink) *Supervisor {
	if sink == nil {
		sink = func(Line) {}
	}
	return &Supervisor{settings: s, logger: logger, sink: sink}
}

// Settings returns the settings the next Start will use.
func (s *Supervisor) Settings() settings.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

// SetSettings replaces them. A running server keeps the ones it started with
// until the next Start.
func (s *Supervisor) SetSettings(next settings.Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = next
}

// Running reports whether the pair is up.
func (s *Supervisor) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Start brings up the bridge and the game server. It returns once both have
// been launched; onExit fires later, on the goroutine that watched them, with
// the reason they stopped (nil for a Stop the operator asked for).
func (s *Supervisor) Start(onExit func(error)) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("the server is already running")
	}
	cfg, err := bridgeConfig(s.settings)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	// The game server reads server.cfg once, at its own startup, so the file
	// has to hold the settings this start is using. Rendering it here rather
	// than once per launcher run is what makes a class unticked in the
	// interface reach the server it is unticked for.
	if err := srcdsconfig.Install(s.settings); err != nil {
		s.mu.Unlock()
		return err
	}
	ctxRoom, cancelRoom := context.WithCancel(context.Background())
	room, err := s.startTestRoom(ctxRoom, &cfg)
	if err != nil {
		cancelRoom()
		s.mu.Unlock()
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	stopRoom := func() {
		cancelRoom()
		if room != nil {
			stop, cancelStop := context.WithTimeout(context.Background(), roomCloseGrace)
			_ = room.Close(stop)
			cancelStop()
		}
	}
	s.cancel, s.done, s.running, s.stopped = cancel, done, true, false
	current := s.settings
	s.mu.Unlock()

	s.emit("starting the bridge and the game server")
	// The reach asked for is not always the one the server gets, and the
	// difference is invisible everywhere else: srcds says nothing about a
	// token it was never given.
	if current.EffectiveReach() != current.SrcdsReach {
		s.emit("no login token, so the server stays on the local network: " +
			string(current.SrcdsReach) + " needs one from steamcommunity.com/dev/managegameservers")
	}

	/* Each of these outlives this call, so a panic on any of them takes the
	   launcher down with the server it is supervising. guard turns that into a
	   line in the log a debug bundle carries. */
	bridgeErr := make(chan error, 1)
	go guard("the bridge", s.emit, func() {
		bridgeErr <- bridge.Run(ctx, cfg, s.bridgeLogger())
	})

	srcdsErr := make(chan error, 1)
	go guard("the game server", s.emit, func() {
		srcdsErr <- runSrcdsWithSink(ctx, current, s.logger, s.sink)
	})
	go guard("the SourceMod error watcher", s.emit, func() {
		watchSourcemodErrors(ctx, filepath.Join(current.InstallRoot, "tf-dedicated"), s.sink)
	})

	go guard("the supervisor", s.emit, func() {
		s.await(session{
			cancel:    cancel,
			done:      done,
			bridgeErr: bridgeErr,
			srcdsErr:  srcdsErr,
			stopRoom:  stopRoom,
			onExit:    onExit,
		})
	})
	return nil
}

// exitGrace bounds the wait for the second half to exit. Longer than the delay
// os/exec gives a signalled process group before it kills it, so the wait ends
// because the process is gone rather than because this gave up.
const exitGrace = 20 * time.Second

// session is what await needs to see one run through to its end.
type session struct {
	cancel    context.CancelFunc
	done      chan struct{}
	bridgeErr chan error
	srcdsErr  chan error
	stopRoom  func()
	onExit    func(error)
}

// await watches the pair and reports why they stopped. Whichever of the two
// ends first ends the other: a bridge with no server has nothing to serve, and
// a server with no bridge records nothing. It returns once both are gone.
func (s *Supervisor) await(run session) {
	defer close(run.done)
	defer run.stopRoom()

	var reason error
	select {
	case err := <-run.bridgeErr:
		reason = wrapExit("bridge", err)
	case err := <-run.srcdsErr:
		reason = wrapExit("game server", err)
	}
	run.cancel()

	/* Then the other one, because Stop waits on this and its caller was told
	the server is down when it returns.

	The bridge is almost always the one that ends first: it is a goroutine
	watching a context, where the game server is a shell script that has to be
	signalled, pass the signal to srcds, and be reaped. Closing here on the
	first of the two meant Stop returned while the game server still held the
	game port and the rcon port, and the Start after it bound neither. */
	select {
	case <-run.bridgeErr:
	case <-run.srcdsErr:
	case <-time.After(exitGrace):
		s.emit("the game server has not exited yet, carrying on without it")
	}

	s.mu.Lock()
	asked := s.stopped
	s.running = false
	s.mu.Unlock()

	switch {
	case asked:
		reason = nil
		s.emit("stopped")
	case reason != nil:
		s.emit(reason.Error())
	}
	if run.onExit != nil {
		run.onExit(reason)
	}
}

// Stop takes both down and waits for them. Stopping a stopped supervisor does
// nothing.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	cancel, done := s.cancel, s.done
	s.mu.Unlock()

	s.emit("stopping")
	cancel()
	<-done
}

// Restart is a Stop then a Start, which is what the button of that name does.
func (s *Supervisor) Restart(onExit func(error)) error {
	s.Stop()
	return s.Start(onExit)
}

// startTestRoom serves the multiworld of one and repoints the bridge at it,
// when the settings ask for test mode. It returns nil otherwise.
func (s *Supervisor) startTestRoom(ctx context.Context, cfg *config.Config) (*fakeroom.Room, error) {
	return StartTestRoom(ctx, s.settings, cfg, s.emit)
}

/*
StartTestRoom serves the multiworld of one and repoints the bridge at it, when
the settings ask for test mode. It returns nil otherwise.

The room's address replaces whatever the player set: a test run that quietly
dialled a real room would send checks to somebody's actual game.

Shared rather than a method, because the window is not the only way in. The
console flow, which is the whole of the Linux launcher, had no test room of its
own: TF2AP_TEST_MODE=1 set the flag, nothing read it out here, and the bridge
spent the evening dialling archipelago.gg on port zero.
*/
func StartTestRoom(
	ctx context.Context, s settings.Settings, cfg *config.Config, emit func(string),
) (*fakeroom.Room, error) {
	if !s.TestMode {
		return nil, nil //nolint:nilnil // no room and no error is the normal case
	}
	if _, err := settings.CheckRunSelection(s); err != nil {
		return nil, err
	}
	if emit == nil {
		emit = func(string) {}
	}
	room, address, err := fakeroom.Start(ctx, fakeroom.Options{
		SlotName:     s.APSlotName,
		Goal:         s.MvmGoal,
		MissionCount: s.MvmMissionCount,
		Excluded:     s.MvmExcludedMissions,
		Difficulty:   s.MvmDifficulty,
		StartMission: s.MvmStartMission,
		StartClass:   s.MvmStartClass,
		DeathLink:    s.MvmDeathLink,
		Log:          emit,
	})
	if err != nil {
		return nil, err
	}
	cfg.ArchipelagoURL = address
	return room, nil
}

func (s *Supervisor) emit(text string) {
	s.sink(Line{At: time.Now(), Source: "launcher", Text: text})
}

// bridgeLogger routes the bridge's own log through the sink, so the window
// shows one stream rather than sending half of it to a console nobody sees.
func (s *Supervisor) bridgeLogger() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.New(slog.NewTextHandler(&sinkWriter{sink: s.sink, source: "bridge"}, nil))
}

func wrapExit(what string, err error) error {
	if err == nil {
		return fmt.Errorf("%s stopped", what)
	}
	if note := crashNote(err); note != "" {
		return fmt.Errorf("%s CRASHED: %w (%s)", what, err, note)
	}
	return fmt.Errorf("%s stopped: %w", what, err)
}

/* crashNote names the exit statuses that mean the process died rather than
 * returned, and says so in words a player can repeat.
 *
 * It exists because "game server stopped: exit status 0xc0000005" reads like
 * every other stop. A crash was reported for weeks as "bridge stopping", which
 * is the launcher's own orderly shutdown printed on the way out: the last line
 * a player sees is the one they report, and it named the component that had not
 * failed. A crash has to outrank the shutdown noise around it.
 */
func crashNote(err error) string {
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return ""
	}
	// A signal is a Unix crash: SIGSEGV, SIGABRT, SIGBUS and the rest arrive
	// this way rather than as an exit code.
	if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return "killed by " + status.Signal().String() + ", so this is a crash and not a stop"
	}
	code := exit.ExitCode()
	if code < 0 {
		return ""
	}
	// Windows reports an unhandled exception as the NTSTATUS itself, and every
	// one of those has the top two bits set. 0xc0000005 is the access
	// violation the game dies with most often.
	if uint32(code)&0xC0000000 == 0xC0000000 {
		if name := ntStatusNames[uint32(code)]; name != "" {
			return name + ", so this is a crash and not a stop"
		}
		return "an unhandled exception, so this is a crash and not a stop"
	}
	return ""
}

// ntStatusNames covers the handful worth naming. Anything else still reports as
// a crash, just without the word for it.
var ntStatusNames = map[uint32]string{
	0xC0000005: "access violation",
	0xC0000006: "in-page error",
	0xC000001D: "illegal instruction",
	0xC0000025: "unrecoverable exception",
	0xC0000094: "integer divide by zero",
	0xC00000FD: "stack overflow",
	0xC0000409: "stack buffer overrun",
	0xC0000374: "corrupted heap",
}

// sinkWriter turns a stream of bytes into whole lines for the sink.
type sinkWriter struct {
	sink   Sink
	source string
	buf    []byte
}

func (w *sinkWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := string(bytes.TrimRight(w.buf[:i], "\r"))
		w.buf = w.buf[i+1:]
		if line != "" {
			w.sink(Line{At: time.Now(), Source: w.source, Text: line})
		}
	}
	return len(p), nil
}
