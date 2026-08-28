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
	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
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
	// Console mode has no interface to change a setting from, but it shares
	// this entry point with anything that does. Rendering here keeps the one
	// rule: a server that starts, starts from the settings it was given.
	if err := srcdsconfig.Install(s); err != nil {
		return err
	}
	bridgeCfg, err := bridgeConfig(s)
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

// srcdsArgs is the command line the game server starts with. It is separate
// from starting it, and pure, because the reach decides four arguments at once
// and a wrong combination is a server nobody can join: a test says which
// arguments each reach produces without downloading 14 GB to find out.
//
// The dash flags come first and the console commands after, which is the order
// the game's own scripts use. The commands run in the order they are given,
// and sv_lan has to be settled before the map loads and the server tries to
// log in to Steam.
func srcdsArgs(s settings.Settings, exeName string) []string {
	// Not s.SrcdsReach: a reach with no login token behind it cannot log in,
	// and a server that cannot log in refuses every player. It stays local.
	reach := s.EffectiveReach()

	// -console on both, for the same reason spelled two ways. On Windows,
	// srcds.exe without it opens its own window and waits for a click on
	// Start, so the launcher sits there having apparently done nothing. On
	// Linux it is the flag every server runs with: srcds_linux without it
	// brings up an interactive text console, and this launcher gives it no
	// terminal to bring it up on. The server then holds its port, burns a
	// fifth of a core and never finishes loading the map.
	//
	// -condebug writes the same console to tf/console.log. The launcher already
	// pipes the console into its own window and its own log, so this looks like
	// the same thing twice, and it is not: the launcher's log covers the run the
	// launcher is having, and a player who hits a bug restarts before asking
	// about it. The game's file is the one that survives that restart, and it is
	// what the debug bundle promises.
	flags := []string{"-game", "tf", "-usercon", "-console", "-condebug"}
	// -ip 0.0.0.0 binds every interface. Without it srcds binds to whatever
	// its hostname resolves to, and on Debian that is 127.0.1.1: the game
	// answers on every address, the rcon port answers only on that one, and
	// the launcher's own rcon box gets "connection refused" from a server
	// running perfectly well.
	//
	// Only where sv_lan is off. The engine keeps whatever -ip says as the
	// address it believes it is on, and a LAN server compares every joining
	// player against it: same class C or refused. Told 0.0.0.0, it compares
	// them against 0.0.0.0, which nothing matches, and the server turns away
	// every player on the network with "LAN servers are restricted to local
	// clients (class C)". Only the loopback address gets in, because that is
	// checked before the comparison. RconAddresses covers the bind instead.
	if !reach.Lan() {
		flags = append(flags, "-ip", "0.0.0.0")
	}
	if exeName == "srcds.exe" {
		// A crash dialog is a Windows idea, and it waits for a click too.
		//
		// -nowatchdog does not belong here. The watchdog is POSIX only: tier0
		// arms it with alarm() and SIGALRM, and every one of the four
		// Plat_*WatchdogTimer functions is an empty stub in the Windows build.
		// Passing the flag would read like protection that was never there.
		flags = append(flags, "-nocrashdialog")
	} else {
		// The engine watchdog kills the server when a frame takes too long.
		// On Linux that fires under load the same box survives fine otherwise,
		// and it is most of why a native Linux server is reported as crashing
		// far more than Docker or Windows. A stress run that killed the
		// watchdog build inside nine minutes ran clean without it.
		//
		// What is lost is the kill on a genuinely hung frame, so a real
		// infinite loop now hangs instead of restarting. That is the better
		// failure: it leaves a process to attach to, and the hangs this mod
		// has actually produced were slow frames rather than hangs.
		//
		// There is no middle setting to reach for. tier0 parses this with a
		// strstr over the command line, the timeout is whatever the engine
		// passed capped at five minutes, and the one scale factor is 1 in a
		// release build and 10 in a debug one. On or off is the whole choice.
		flags = append(flags, "-nowatchdog")
	}
	if reach.SteamNetworking() {
		// Asks Valve for the relayed address. Without it sv_use_steam_networking
		// alone changes nothing a player outside the network can use.
		flags = append(flags, "-enablefakeip")
	}

	commands := []string{
		"+maxplayers", strconv.Itoa(s.SrcdsMaxPlayers),
		"+map", StartMap(s),
		"+hostport", strconv.Itoa(s.SrcdsPort),
		"+rcon_password", s.SrcdsRconPw,
		"+sv_lan", boolArg(reach.Lan()),
	}
	if reach.SteamNetworking() {
		commands = append(commands, "+sv_use_steam_networking", "1")
	}
	// A server in LAN mode never logs in, so a token left over from an earlier
	// evening is not passed and cannot put it on the public list by accident.
	if reach.NeedsToken() && settings.HasToken(s.SrcdsToken) {
		commands = append(commands, "+sv_setsteamaccount", s.SrcdsToken)
	}
	if s.SrcdsPw != "" {
		commands = append(commands, "+sv_password", s.SrcdsPw)
	}
	return append(flags, commands...)
}

func boolArg(on bool) string {
	if on {
		return "1"
	}
	return "0"
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
	// Before the server opens it: -condebug appends, and one file holding every
	// evening ever played is a file nobody can send anywhere.
	RotateConsoleLog(gameDir)
	cmd := exec.CommandContext(ctx, filepath.Join(gameDir, exeName), srcdsArgs(s, exeName)...)
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

// StartMap is the map the start mission runs on. A mission the tables do not
// know is taken as a map name, which is what an older setting held.
func StartMap(s settings.Settings) string {
	if mission, ok := gamedata.MissionByPopFile(s.SrcdsStartMission); ok {
		if played, ok := gamedata.MapByID(mission.Map); ok {
			return played.Name
		}
	}
	return s.SrcdsStartMission
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
