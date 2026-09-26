// Command winbed prepares and runs a disposable game server for the wave probe
// on the machine it runs on, the way the launcher prepares a player's: the same
// installer, the same community packs, the same SigMod install and the same
// server configuration. It exists so the probe can play the SigMod missions on
// a real Windows server rather than only in the Linux container.
//
//	winbed -root C:\bed -sigmod package-windows.zip -probe tf2_waveprobe.smx -install
//	winbed -root C:\bed -serve
//
// -install leaves the server ready and exits. -serve runs it until killed,
// with the console in <root>/tf-dedicated/tf/console.log.
package main

import (
	"archive/zip"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	tfruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/sigmodpatch"
)

const sigmodKey = "sigsegv-mvm"

type options struct {
	root    string
	packs   string
	sigmod  string
	probe   string
	rconpw  string
	port    int
	install bool
	serve   bool
}

func main() {
	var opt options
	flag.StringVar(&opt.root, "root", "", "install root, as the launcher's install_root")
	flag.StringVar(&opt.packs, "packs", settings.CommunityPackPotato+","+settings.CommunityPackMoonlight, "community packs, comma separated")
	flag.StringVar(&opt.sigmod, "sigmod", "", "a SigMod package zip to install instead of the pinned one")
	flag.StringVar(&opt.probe, "probe", "", "the compiled tf2_waveprobe.smx")
	flag.StringVar(&opt.rconpw, "rconpw", os.Getenv("WAVEPROBE_RCONPW"), "rcon password")
	flag.IntVar(&opt.port, "port", 27015, "game port, also rcon")
	flag.BoolVar(&opt.install, "install", false, "install the server, the packs, SigMod and the probe, then exit")
	flag.BoolVar(&opt.serve, "serve", false, "run the game server until interrupted")
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(opt, logger); err != nil {
		fmt.Fprintln(os.Stderr, "winbed:", err)
		os.Exit(1)
	}
}

func run(opt options, logger *slog.Logger) error {
	if opt.root == "" || opt.rconpw == "" {
		return errors.New("-root and -rconpw (or WAVEPROBE_RCONPW) are required")
	}
	if opt.install == opt.serve {
		return errors.New("give one of -install and -serve")
	}
	s := bedSettings(opt)
	if opt.install {
		return install(opt, s, logger)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return tfruntime.RunServer(ctx, s, logger)
}

// bedSettings is the launcher's defaults with what the Docker probe sets:
// a LAN server with no bots and no room, SigMod loaded whatever the mission.
func bedSettings(opt options) settings.Settings {
	s := settings.Defaults()
	s.InstallRoot = opt.root
	s.CommunityContentDir = filepath.Join(opt.root, "community")
	s.CommunityPacks = strings.Split(opt.packs, ",")
	s.SrcdsMods = []string{sigmodKey}
	s.SrcdsModLoading = settings.ModLoadingAlways
	s.SrcdsRconPw = opt.rconpw
	s.SrcdsPort = opt.port
	s.SrcdsMaxPlayers = 32
	s.SrcdsReach = settings.ReachLan
	s.SrcdsBots = false
	s.TestMode = false
	s.FastDLPort = 0
	s.SrcdsStartMap = "mvm_decoy"
	s.SrcdsHostname = "TF2 Archipelago Wave Probe"
	return s
}

func install(opt options, s settings.Settings, logger *slog.Logger) error {
	ctx := context.Background()
	logf := func(format string, args ...any) { logger.Info("install", "step", fmt.Sprintf(format, args...)) }
	archives := settings.CommunityArchives(s)
	if err := os.MkdirAll(s.CommunityContentDir, 0o755); err != nil {
		return err
	}
	if err := installer.DownloadCommunityArchives(ctx, archives, logf); err != nil {
		return err
	}
	// The packs go in after SigMod, as they do for a player, because the
	// installer drops SigMod-only population files when SigMod is not ready.
	if opt.sigmod == "" {
		_, err := installer.Ensure(ctx, s.InstallRoot, archives, s.SrcdsMods, logf)
		return errors.Join(err, finish(opt, s))
	}
	if _, err := installer.Ensure(ctx, s.InstallRoot, nil, nil, logf); err != nil {
		return err
	}
	modDir := filepath.Join(s.InstallRoot, "tf-dedicated", "tf")
	logf("installing SigMod from %s", opt.sigmod)
	if err := unzip(opt.sigmod, modDir); err != nil {
		return fmt.Errorf("cannot install %s: %w", opt.sigmod, err)
	}
	if err := sigmodpatch.Patch(modDir, runtime.GOOS); err != nil {
		return err
	}
	if err := installer.InstallCommunityArchives(archives, modDir, s.SrcdsMods, logf); err != nil {
		return err
	}
	return finish(opt, s)
}

func finish(opt options, s settings.Settings) error {
	if err := installer.SetServerModLoading(s.InstallRoot, settings.ServerModsToLoad(s, runtime.GOOS)); err != nil {
		return err
	}
	if opt.probe == "" {
		return nil
	}
	body, err := os.ReadFile(opt.probe)
	if err != nil {
		return err
	}
	target := filepath.Join(s.InstallRoot, "tf-dedicated", "tf", "addons", "sourcemod", "plugins", "tf2_waveprobe.smx")
	return os.WriteFile(target, body, 0o644)
}

func unzip(path, dir string) error {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close() }()
	for _, file := range archive.File {
		target := filepath.Join(dir, filepath.FromSlash(file.Name))
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(os.PathSeparator)) {
			return fmt.Errorf("%s leaves the game directory", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := extract(file, target); err != nil {
			return err
		}
	}
	return nil
}

func extract(file *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
