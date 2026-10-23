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
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/m-this/tf2-archipelago/bridge"
	"github.com/m-this/tf2-archipelago/bridge/config"
	"github.com/m-this/tf2-archipelago/fakeroom"
	"github.com/m-this/tf2-archipelago/gamedata/availability"
	"github.com/m-this/tf2-archipelago/launcher/internal/fastdl"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/provision"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/srcdsconfig"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// consoleWait bounds the wait for the console the game server reads from.
const consoleWait = 5 * time.Second

// closeTestRoom stops the room of one on a context of its own. Run defers it,
// and by then the context Run was given is the one that ended, so passing it
// down would ask the room to shut down with no time to do it in.
func closeTestRoom(room *fakeroom.Room) {
	ctx, cancel := context.WithTimeout(context.Background(), roomCloseGrace)
	defer cancel()
	_ = room.Close(ctx)
}

// Run starts the bridge in-process and the game server as a subprocess, and
// blocks until the context is cancelled or one of them stops. The bridge gets
// a head start so the plugin can reach /unlocks on first load.
func Run(ctx context.Context, s settings.Settings, logger *slog.Logger) error {
	prepared, err := prepareTailscaleFastDL(ctx, s, func(text string) {
		logger.InfoContext(ctx, "download server", "detail", text)
	})
	if err != nil {
		return err
	}
	s = prepared
	if s.TailscaleFastDL {
		defer stopTailscaleFastDL(ctx, s, func(cleanupCtx context.Context, text string) {
			logger.InfoContext(cleanupCtx, "download server", "detail", text)
		})
	}
	// Console mode has no interface to change a setting from, but it shares
	// this entry point with anything that does. Rendering here keeps the one
	// rule: a server that starts, starts from the settings it was given.
	if err := srcdsconfig.Install(s); err != nil {
		return err
	}
	bridgeCfg, err := bridgeConfigReplacingStale(ctx, s, func(text string) {
		logger.InfoContext(ctx, "bridge", "detail", text)
	})
	if err != nil {
		return err
	}

	bridgeCtx, cancelBridge := context.WithCancel(ctx)
	defer cancelBridge()

	// Before the bridge dials anything: in test mode this is what it dials.
	room, err := StartTestRoom(bridgeCtx, s, &bridgeCfg, func(text string) {
		logger.InfoContext(ctx, "test room", "message", text)
	})
	if err != nil {
		return err
	}
	if room != nil {
		//nolint:contextcheck // the caller's context is what is being cancelled here
		defer closeTestRoom(room)
	}
	/* Both outlive this call, and the headless path has the same exposure the
	   supervised one had: a panic on either takes the process down with the
	   server it is running. */
	say := func(text string) { logger.ErrorContext(ctx, "contained a panic", "detail", text) }

	bridgeErr := make(chan error, 1)
	go Guard("the bridge", say, func() {
		bridgeErr <- bridge.Run(bridgeCtx, bridgeCfg, logger)
	})

	srcdsErr := make(chan error, 1)
	srcdsCtx, cancelSrcds := context.WithCancel(ctx)
	defer cancelSrcds()
	go Guard("the game server", say, func() {
		srcdsErr <- runSrcds(srcdsCtx, s, logger)
	})
	go Guard("the download server", say, func() {
		serveFastDL(srcdsCtx, s, func(text string) { logger.InfoContext(ctx, "download server", "detail", text) })
	})

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

/*
FastDLListen is where the content server binds, and empty is not serving.

The port decides that on its own, and SrcdsDownloadURL does not come into it.
The two answer different questions -- where a client is told to look, and
whether this machine answers -- and an operator who fills in a public address
so that friends outside the network can reach this server would otherwise turn
off the server they were addressing.
*/
func FastDLListen(s settings.Settings) string {
	if s.FastDLPort <= 0 {
		return ""
	}
	if s.TailscaleFastDL {
		return "127.0.0.1:" + strconv.Itoa(s.FastDLPort)
	}

	return ":" + strconv.Itoa(s.FastDLPort)
}

// serveFastDL runs the content file server for the life of ctx. A port that
// is taken costs the players their HTTP download and nothing else: the game
// server's own transfer is still on, so it is a line in the log, not a stop.
func serveFastDL(ctx context.Context, s settings.Settings, say func(string)) {
	listen := FastDLListen(s)
	if listen == "" {
		return
	}
	gameDir := filepath.Join(s.InstallRoot, "tf-dedicated", "tf")
	if err := fastdl.Serve(ctx, gameDir, listen); err != nil {
		say("the download server did not start, so maps come from the game server itself: " + err.Error())
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
		Missions:       new(missionServer(s)),
	}
	if s.MetricsPort > 0 {
		cfg.MetricsListen = "127.0.0.1:" + fmt.Sprintf("%d", s.MetricsPort)
	}
	return cfg, nil
}

// missionServer is what the bridge refuses a seed against: the packs and mods
// these settings tick, and what Start installed of them.
func missionServer(s settings.Settings) availability.Server {
	return availability.Server{
		Platform:      goruntime.GOOS,
		PacksSelected: s.CommunityPacks,
		PacksPresent:  installer.AvailableCommunityPacks(settings.KnownCommunityArchives(s.CommunityContentDir)),
		ModsSelected:  settings.ServerModKeys(s),
		ModsVerified:  installer.ReadyServerMods(s.InstallRoot),
	}
}

// RunServer starts the game server alone, with the configuration Run would
// write and no bridge or room beside it. It is what the unattended wave probe
// plays against: the plugin finds no bridge and stays out of the way.
func RunServer(ctx context.Context, s settings.Settings, logger *slog.Logger) error {
	if err := srcdsconfig.Install(s); err != nil {
		return err
	}
	return runSrcds(ctx, s, logger)
}

func runSrcds(ctx context.Context, s settings.Settings, logger *slog.Logger) error {
	return runSrcdsWithSink(ctx, s, logger, nil)
}

// srcdsEnv is the environment the game server runs with: this process's own,
// with HOME pointed at the install root.
//
// srcds_run dlopens $HOME/.steam/sdk32/steamclient.so and segfaults without
// it. The installer puts a link there, under the install root rather than in
// the operator's home, and this is what makes the server look in the same
// place. Windows has neither the file nor the problem, so its environment is
// left alone.
func srcdsEnv(s settings.Settings) []string {
	if goruntime.GOOS == "windows" {
		return nil
	}
	env := os.Environ()
	kept := env[:0]
	for _, entry := range env {
		if !strings.HasPrefix(entry, "HOME=") {
			kept = append(kept, entry)
		}
	}
	return append(kept, "HOME="+installer.SteamHome(s.InstallRoot))
}

func runSrcdsWithSink(ctx context.Context, s settings.Settings, logger *slog.Logger, sink Sink) error {
	gameDir := filepath.Join(s.InstallRoot, "tf-dedicated")
	exeName := "srcds.exe"
	if _, err := os.Stat(filepath.Join(gameDir, "srcds_run")); err == nil {
		exeName = "srcds_run"
	}
	if exeName == "srcds_run" {
		if err := installer.InspectSteamClient(s.InstallRoot).Err; err != nil {
			return fmt.Errorf("the game server would crash at replay_srv.so: %w. %s", err, installer.RepairAdvice)
		}
	}
	// Before the server opens it: -condebug appends, and one file holding every
	// evening ever played is a file nobody can send anywhere.
	RotateConsoleLog(gameDir)
	server, err := provision.For(s, srcdsconfig.Host(exeName))
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, filepath.Join(gameDir, exeName), server.Args...)
	cmd.Dir = gameDir
	cmd.Env = srcdsEnv(s)
	winproc.KillGroup(cmd)
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
	// Most srcds output reaches stdout, but the engine occasionally writes the
	// FakeIP allocation only to the -condebug file. That address controls the
	// Join button and is the only address remote players can use in Steam relay
	// mode, so follow the durable console for that one signal as well. Both
	// sources share a deduplicating sink because the usual case writes it twice.
	outputSink := sink
	watchCtx, stopWatching := context.WithCancel(ctx)
	if sink != nil {
		outputSink = dedupeFakeIPSink(sink)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go pipeLines(stdout, "srcds", logger, outputSink, &wg)
	go pipeLines(stderr, "srcds", logger, outputSink, &wg)
	if outputSink != nil {
		wg.Add(1)
		go watchConsoleFakeIP(watchCtx, filepath.Join(gameDir, "tf", ConsoleLogName), outputSink, &wg)
	}
	waitErr := cmd.Wait()
	stopWatching()
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
