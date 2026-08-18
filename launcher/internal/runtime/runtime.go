// Package runtime starts the game server subprocess and the bridge in-process,
// interleaves their logs, and shuts both down on Ctrl-C. It is the launcher's
// equivalent of `docker compose up`: one call blocks until something stops.
package runtime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/m-this/tf2-archipelago/bridge"
	"github.com/m-this/tf2-archipelago/bridge/config"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// consoleWait bounds the wait for the console the game server reads from.
const consoleWait = 5 * time.Second

// Run starts the bridge in-process and the game server as a subprocess, and
// blocks until the context is cancelled or one of them stops. The bridge gets
// a head start so the plugin can reach /unlocks on first load.
func Run(ctx context.Context, s settings.Settings, logger *slog.Logger) error {
	bridgeCfg, err := bridgeConfig(s)
	if err != nil {
		return err
	}

	bridgeCtx, cancelBridge := context.WithCancel(ctx)
	defer cancelBridge()
	bridgeErr := make(chan error, 1)
	go func() {
		bridgeErr <- bridge.Run(bridgeCtx, bridgeCfg, logger)
	}()

	srcdsErr := make(chan error, 1)
	srcdsCtx, cancelSrcds := context.WithCancel(ctx)
	defer cancelSrcds()
	go func() {
		srcdsErr <- runSrcds(srcdsCtx, s, logger)
	}()

	select {
	case <-ctx.Done():
		logger.InfoContext(ctx, "stopping")
		return nil
	case err := <-bridgeErr:
		logger.ErrorContext(ctx, "bridge stopped", "error", err)
		cancelSrcds()
		return fmt.Errorf("bridge stopped: %w", err)
	case err := <-srcdsErr:
		logger.ErrorContext(ctx, "game server stopped", "error", err)
		return fmt.Errorf("game server stopped: %w", err)
	}
}

func bridgeConfig(s settings.Settings) (config.Config, error) {
	if s.SrcdsRconPw == "" {
		return config.Config{}, fmt.Errorf("SRCDS_RCONPW is not set")
	}
	if s.APPort == 0 && !s.TestMode {
		return config.Config{}, fmt.Errorf("AP_PORT is not set; create a room on archipelago.gg first")
	}
	port := fmt.Sprintf("%d", s.APPort)
	scheme := "ws"
	if s.APTls {
		scheme = "wss"
	}
	statePath := filepath.Join(s.InstallRoot, "bridge-state", "bridge.json")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return config.Config{}, err
	}
	cfg := config.Config{
		ArchipelagoURL: scheme + "://" + s.APHost + ":" + port,
		SlotName:       s.APSlotName,
		Password:       s.APPassword,
		Listen:         "127.0.0.1:24680",
		StatePath:      statePath,
	}
	if s.MetricsPort > 0 {
		cfg.MetricsListen = "127.0.0.1:" + fmt.Sprintf("%d", s.MetricsPort)
	}
	return cfg, nil
}

func runSrcds(ctx context.Context, s settings.Settings, logger *slog.Logger) error {
	return runSrcdsWithSink(ctx, s, logger, nil)
}

func runSrcdsWithSink(ctx context.Context, s settings.Settings, logger *slog.Logger, sink Sink) error {
	gameDir := filepath.Join(s.InstallRoot, "tf-dedicated")
	exeName := "srcds.exe"
	if _, err := os.Stat(filepath.Join(gameDir, "srcds_run")); err == nil {
		exeName = "srcds_run"
	}
	args := []string{
		"-game", "tf",
		"-usercon",
		"+maxplayers", fmt.Sprintf("%d", s.SrcdsMaxPlayers),
		"+map", s.SrcdsStartMap,
		"+hostport", fmt.Sprintf("%d", s.SrcdsPort),
		"+rcon_password", s.SrcdsRconPw,
	}
	if exeName == "srcds.exe" {
		// Without -console, srcds.exe opens its own window and waits for a
		// click on Start, so the launcher would sit there having apparently
		// done nothing. -nocrashdialog keeps a crash from doing the same.
		args = append([]string{"-console", "-nocrashdialog"}, args...)
	}
	if s.SrcdsLan {
		args = append(args, "+sv_lan", "1")
	}
	if s.SrcdsPw != "" {
		args = append(args, "+sv_password", s.SrcdsPw)
	}
	cmd := exec.CommandContext(ctx, filepath.Join(gameDir, exeName), args...)
	cmd.Dir = gameDir
	// -console reads the console input buffer, so the server needs a real one
	// as its standard input. CREATE_NO_WINDOW would deny it any console at
	// all, and the server dies on its first read. Its output still comes back
	// through the pipes below.
	stdin, err := winproc.ConsoleStdinTimeout(consoleWait)
	switch {
	case errors.Is(err, winproc.ErrNoConsole):
	case err != nil:
		report(logger, sink, fmt.Sprintf("no console for the game server: %v", err))
	default:
		cmd.Stdin = stdin
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start the game server: %w", err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go pipeLines(stdout, "srcds", logger, sink, &wg)
	go pipeLines(stderr, "srcds", logger, sink, &wg)
	waitErr := cmd.Wait()
	wg.Wait()
	// A non-nil waitErr after context cancellation is the subprocess being
	// killed, which is expected and not an error to report.
	if ctx.Err() != nil {
		return nil //nolint:nilerr // context cancellation kills the subprocess; the wait error is expected
	}
	return waitErr
}

// report says something to whoever is listening. The window passes a sink and
// no logger, the console flow passes a logger and no sink, and neither is
// allowed to be assumed: calling a method on a nil *slog.Logger panics, and
// that took the whole launcher down once.
func report(logger *slog.Logger, sink Sink, text string) {
	if sink != nil {
		sink(Line{At: time.Now(), Source: "launcher", Text: text})
		return
	}
	if logger != nil {
		logger.Warn("launcher", "message", text)
	}
}

func pipeLines(r io.Reader, source string, logger *slog.Logger, sink Sink, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t\r")
		if line == "" {
			continue
		}
		if sink != nil {
			sink(Line{At: time.Now(), Source: source, Text: line})
			continue
		}
		if logger != nil {
			logger.Info("srcds output", "source", source, "line", line)
		}
	}
}
