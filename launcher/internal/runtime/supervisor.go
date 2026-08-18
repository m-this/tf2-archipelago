package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/m-this/tf2-archipelago/bridge"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

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
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.cancel, s.done, s.running, s.stopped = cancel, done, true, false
	current := s.settings
	s.mu.Unlock()

	s.emit("starting the bridge and the game server")

	bridgeErr := make(chan error, 1)
	go func() {
		bridgeErr <- bridge.Run(ctx, cfg, s.bridgeLogger())
	}()

	srcdsErr := make(chan error, 1)
	go func() {
		srcdsErr <- runSrcdsWithSink(ctx, current, s.logger, s.sink)
	}()

	go func() {
		defer close(done)
		var reason error
		select {
		case err := <-bridgeErr:
			reason = wrapExit("bridge", err)
		case err := <-srcdsErr:
			reason = wrapExit("game server", err)
		}
		cancel()

		s.mu.Lock()
		asked := s.stopped
		s.running = false
		s.mu.Unlock()

		if asked {
			reason = nil
			s.emit("stopped")
		} else if reason != nil {
			s.emit(reason.Error())
		}
		if onExit != nil {
			onExit(reason)
		}
	}()
	return nil
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
	return fmt.Errorf("%s stopped: %w", what, err)
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
