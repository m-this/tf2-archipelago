// Command tf2ap is the all-in-one Windows launcher for tf2-archipelago.
//
// It installs SteamCMD, the TF2 dedicated server, SourceMod, ripext and the
// plugin; asks for the configuration an evening needs (room address, RCON
// password, run shape); and runs the bridge in-process alongside the srcds
// subprocess. One exe, no Docker, no clone.
//
// Seed generation stays with the official Archipelago app: install it once,
// drop the tf2_mvm.apworld from the same release into its custom_worlds/, and
// generate there. This launcher only owns the TF2 server side.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/srcdsconfig"
	"github.com/m-this/tf2-archipelago/launcher/internal/ui"
)

const version = "dev"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("launcher stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	installFlag := flag.Bool("install", false, "install or repair the server, then exit")
	configureFlag := flag.Bool("configure", false, "edit the configuration, then exit")
	statusFlag := flag.Bool("status", false, "show the configuration and install state, then exit")
	yamlFlag := flag.String("yaml", "", "write the Archipelago player file to this path, then exit")
	envFlag := flag.Bool("env", false, "list the environment variables that override the configuration, then exit")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		v := assets.Versions()
		fmt.Printf("tf2ap %s\n", version)
		for _, name := range []string{"metamod", "sourcemod", "ripext", "archipelago"} {
			fmt.Printf("  %-12s %s\n", name+":", v[name])
		}
		return nil
	}

	if *envFlag {
		showEnv()
		return nil
	}

	saved, err := settings.Load()
	if err != nil {
		return fmt.Errorf("cannot load the configuration: %w", err)
	}
	// The environment wins over the file, and is never written back: an
	// override for one run must not become the saved answer.
	s := settings.ApplyEnv(saved)

	if *yamlFlag != "" {
		return writeYAML(s, *yamlFlag)
	}

	if *statusFlag {
		showStatus(s)
		return nil
	}

	if *installFlag {
		_, err := installer.Ensure(context.Background(), s.InstallRoot, logf(logger))
		return err
	}

	if *configureFlag {
		s = configure(ui.New(), s)
		if err := settings.Save(s); err != nil {
			return fmt.Errorf("cannot save the configuration: %w", err)
		}
		fmt.Println("saved.")
		return nil
	}

	return guided(logger, s)
}

// guided is the no-args path: install if needed, configure if missing required
// values, write the server configs, then start.
func guided(logger *slog.Logger, s settings.Settings) error {
	s = ensureInstalled(s, logger)
	prompt := ui.New()
	s = ensureConfigured(prompt, s)
	if err := settings.Save(s); err != nil {
		return fmt.Errorf("cannot save the configuration: %w", err)
	}
	if err := srcdsconfig.Install(s); err != nil {
		return fmt.Errorf("cannot write the server configs: %w", err)
	}
	if err := writeStarterYAML(s); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	fmt.Println("starting the server. press Ctrl-C to stop.")
	return runtime.Run(ctx, s, logger)
}

// writeStarterYAML drops the player file next to the game files the first time
// only. Generation happens in the Archipelago app, and this is the file it
// needs; rewriting it on every start would undo an edit made there.
func writeStarterYAML(s settings.Settings) error {
	path := filepath.Join(s.InstallRoot, "tf2.yaml")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.WriteFile(path, []byte(settings.PlayerYAML(s, assets.ArchipelagoVersion)), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	fmt.Printf("wrote %s for the Archipelago app to generate from\n", path)
	return nil
}

func ensureInstalled(s settings.Settings, logger *slog.Logger) settings.Settings {
	result, err := installer.Ensure(context.Background(), s.InstallRoot, logf(logger))
	if err != nil {
		logger.Error("install failed", "error", err)
		os.Exit(1)
	}
	if result.Done.Message != "" {
		fmt.Println(result.Done.Message)
	}
	return s
}

func ensureConfigured(p *ui.Prompt, s settings.Settings) settings.Settings {
	fmt.Println("\n--- Configuration ---")
	if s.SrcdsRconPw == "" {
		s.SrcdsRconPw = p.Password("Choose an RCON password (admin console, keep this secret)", "")
	}
	if s.APPort == 0 {
		fmt.Println("\nThe Archipelago room address goes here.")
		fmt.Println("Create one at https://archipelago.gg after generating a seed with the Archipelago app.")
		s.APHost = p.Text("  AP host", s.APHost)
		s.APPort = p.Int("  AP port", s.APPort)
		s.APTls = p.Bool("  Use TLS (wss, yes for archipelago.gg)", s.APTls)
	}
	return s
}

func configure(p *ui.Prompt, s settings.Settings) settings.Settings {
	fmt.Println("--- Archipelago room ---")
	s.APHost = p.Text("AP host", s.APHost)
	s.APPort = p.Int("AP port", s.APPort)
	s.APTls = p.Bool("Use TLS (wss)", s.APTls)
	s.APSlotName = p.Text("Slot name", s.APSlotName)
	s.APPassword = p.Password("Room password", s.APPassword)

	fmt.Println("\n--- Game server ---")
	s.SrcdsHostname = p.Text("Server hostname", s.SrcdsHostname)
	s.SrcdsRconPw = p.Password("RCON password", s.SrcdsRconPw)
	s.SrcdsPw = p.Password("Player password (blank for none)", s.SrcdsPw)
	s.SrcdsPort = p.Int("Game port", s.SrcdsPort)
	s.SrcdsStartMap = p.Text("Start map", s.SrcdsStartMap)
	s.SrcdsAdminSteamIDs = p.Text("Admin Steam IDs (comma-separated, blank for none)", s.SrcdsAdminSteamIDs)
	s.SrcdsLan = p.Bool("LAN mode (yes for friends, no for public)", s.SrcdsLan)
	if s.SrcdsLan {
		s.SrcdsToken = "0"
	} else {
		s.SrcdsToken = p.Text("Game Server Login Token", s.SrcdsToken)
	}

	fmt.Println("\n--- Defender bots ---")
	s.SrcdsBots = p.Bool("Fill the RED team with bots", s.SrcdsBots)
	if s.SrcdsBots {
		s.SrcdsBotTeamSize = p.Int("Fill RED to how many players", s.SrcdsBotTeamSize)
	}

	fmt.Println("\n--- Run shape (for seed generation) ---")
	s.MvmMissionCount = p.Int("Mission count", s.MvmMissionCount)
	s.MvmDifficulty = p.Choice("Difficulty", []string{"normal", "intermediate", "advanced", "expert"}, s.MvmDifficulty)
	s.MvmGoal = p.Choice("Goal", []string{"final_boss", "missionsanity"}, s.MvmGoal)
	if s.MvmGoal == "missionsanity" {
		s.MvmMissionsanityPct = p.Int("Missionsanity percentage", s.MvmMissionsanityPct)
	}
	s.MvmDeathLink = p.Bool("Death Link", s.MvmDeathLink)

	fmt.Println("\n--- Install location ---")
	s.InstallRoot = p.Text("Install root (14 GB of game files)", s.InstallRoot)
	return s
}

func showStatus(s settings.Settings) {
	fmt.Printf("Install root:  %s\n", s.InstallRoot)
	fmt.Printf("Archipelago:   %s:%d (tls=%v)\n", s.APHost, s.APPort, s.APTls)
	fmt.Printf("Slot:          %s\n", s.APSlotName)
	fmt.Printf("Server:        %s on port %d (lan=%v)\n", s.SrcdsHostname, s.SrcdsPort, s.SrcdsLan)
	fmt.Printf("Start map:     %s\n", s.SrcdsStartMap)
	fmt.Printf("Run:           %d missions, %s, goal=%s\n", s.MvmMissionCount, s.MvmDifficulty, s.MvmGoal)
	fmt.Printf("Bots:          %s\n", botsStatus(s))
	fmt.Printf("RCON password: %s\n", masked(s.SrcdsRconPw))
}

func botsStatus(s settings.Settings) string {
	if !s.SrcdsBots {
		return "off"
	}
	return fmt.Sprintf("RED filled to %d", s.SrcdsBotTeamSize)
}

// showEnv prints the override names. Every one of them is a value the guided
// flow would otherwise ask for, so a shortcut or a .bat can start a server
// with no prompt at all.
func showEnv() {
	fmt.Println("These override the saved configuration for one run:")
	for _, name := range settings.EnvNames {
		if value, ok := os.LookupEnv(name); ok && value != "" {
			fmt.Printf("  %-30s (set)\n", name)
			continue
		}
		fmt.Printf("  %s\n", name)
	}
}

func writeYAML(s settings.Settings, path string) error {
	content := settings.PlayerYAML(s, assets.ArchipelagoVersion)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	fmt.Printf("wrote %s\n", path)
	fmt.Println("drop it in the Archipelago app's Players folder, then generate.")
	return nil
}

func masked(s string) string {
	if s == "" {
		return "(not set)"
	}
	return "(set)"
}

// logf returns a progress callback for the installer, routed through the logger
// with a static message key so sloglint's static-msg rule holds.
func logf(logger *slog.Logger) func(format string, args ...any) {
	return func(format string, args ...any) {
		logger.Info("installer", "message", fmt.Sprintf(format, args...))
	}
}
