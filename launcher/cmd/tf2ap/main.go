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
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/generate"
	"github.com/m-this/tf2-archipelago/launcher/internal/gui"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	"github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/srcdsconfig"
	"github.com/m-this/tf2-archipelago/launcher/internal/tui"
	"github.com/m-this/tf2-archipelago/launcher/internal/ui"
)

const version = "dev"

// tokenAsks bounds the retries on the login token prompt. Three is a
// typo and a paste; past that the answer is not coming.
const tokenAsks = 3

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("launcher stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	// Linked for the windows subsystem, so a run from a terminal has no
	// streams until this hands them back. Every flag below prints.
	if len(os.Args) > 1 {
		ui.AttachConsole()
	}

	installFlag := flag.Bool("install", false, "install or repair the server, then exit")
	configureFlag := flag.Bool("configure", false, "edit the configuration, then exit")
	statusFlag := flag.Bool("status", false, "show the configuration and install state, then exit")
	roomFlag := flag.String("room", "", "the Archipelago room address, as host:port")
	yamlFlag := flag.String("yaml", "", "write the Archipelago player file to this path, then exit")
	envFlag := flag.Bool("env", false, "list the environment variables that override the configuration, then exit")
	consoleFlag := flag.Bool("console", false, "print the log and nothing else, with no interface over it")
	tuiFlag := flag.Bool("tui", false, "the terminal interface, on a platform whose default is the window")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		printVersion()
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

	if *roomFlag != "" {
		room, err := settings.ParseRoom(*roomFlag)
		if err != nil {
			return fmt.Errorf("-room %q: %w", *roomFlag, err)
		}
		s.APHost, s.APPort, s.APTls = room.Host, room.Port, room.TLS
	}

	if *yamlFlag != "" {
		return writeYAML(s, *yamlFlag)
	}

	if *statusFlag {
		showStatus(s)
		return nil
	}

	if *installFlag {
		_, err := installer.Ensure(context.Background(), s.InstallRoot, settings.CommunityArchives(s), logf(logger))
		return err
	}

	if *configureFlag {
		s = configure(ui.New(), s)
		if _, err := settings.CheckRunSelection(s); err != nil {
			return err
		}
		if err := settings.Save(s); err != nil {
			return fmt.Errorf("cannot save the configuration: %w", err)
		}
		fmt.Println("saved.")
		return nil
	}

	if gui.Available() && !*consoleFlag && !*tuiFlag {
		return gui.Run(s, nil)
	}
	return guided(logger, s, !*consoleFlag)
}

func printVersion() {
	v := assets.Versions()
	fmt.Printf("tf2ap %s\n", version)
	for _, name := range []string{"metamod", "sourcemod", "ripext", "archipelago"} {
		fmt.Printf("  %-12s %s\n", name+":", v[name])
	}
}

/*
guided is the no-args path: ask for the room if this is a first run, install
whatever is missing, write the server configs, then start.

The questions and the install print as they go, because a 14 GB download with a
progress line is not something to hide behind an interface. What comes after is
either the terminal interface or the plain log, which is what -console asks for
and what a service or a CI job wants: an interface that draws over the whole
screen writes nothing useful into a file.
*/
func guided(logger *slog.Logger, s settings.Settings, interactive bool) error {
	// The question comes before the 14 GB, so a player who mistyped the address
	// finds out in a second rather than after the download.
	s, err := ensureConfigured(ui.New(), s)
	if err != nil {
		return err
	}
	if err := settings.Save(s); err != nil {
		return fmt.Errorf("cannot save the configuration: %w", err)
	}
	s = ensureInstalled(s, logger)
	if err := srcdsconfig.Install(s); err != nil {
		return fmt.Errorf("cannot write the server configs: %w", err)
	}
	if err := writeStarterYAML(s); err != nil {
		return err
	}
	summary(s)

	if interactive && tui.Available() {
		return tui.Run(s, logger)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	fmt.Println("starting the server. press Ctrl-C to stop.")
	return runtime.Run(ctx, s, logger)
}

// writeStarterYAML drops the player file next to the game files, for the
// Archipelago app to generate from.
func writeStarterYAML(s settings.Settings) error {
	path, err := settings.WritePlayerFile(s, assets.ArchipelagoVersion)
	if err != nil {
		return err
	}
	fmt.Printf("wrote %s for the Archipelago app to generate from\n", path)
	return nil
}

func ensureInstalled(s settings.Settings, logger *slog.Logger) settings.Settings {
	result, err := installer.Ensure(context.Background(), s.InstallRoot, settings.CommunityArchives(s), logf(logger))
	if err != nil {
		logger.Error("install failed", "error", err, "advice", installer.RepairAdvice)
		os.Exit(1)
	}
	if result.Done.Message != "" {
		fmt.Println(result.Done.Message)
	}
	return s
}

// ensureConfigured asks for the one value nobody can guess, and fills the rest.
// Everything else has a working default, and `-configure` is there for the
// player who wants to change one.
func ensureConfigured(p *ui.Prompt, s settings.Settings) (settings.Settings, error) {
	// Test mode serves its own multiworld on loopback, so there is no room to
	// ask for. Asking anyway is how a test run stopped before it started.
	if s.APPort == 0 && !s.TestMode {
		fmt.Println()
		fmt.Println("Paste the address from your Archipelago room page.")
		fmt.Println("It looks like archipelago.gg:12345.")
		for {
			room, err := settings.ParseRoom(p.Text("  Room address", ""))
			if err == nil {
				s.APHost, s.APPort, s.APTls = room.Host, room.Port, room.TLS
				break
			}
			fmt.Printf("  %v\n", err)
			// Nobody is typing: a pipe, a service, a redirect from
			// /dev/null. Asking again would ask forever.
			if p.Closed() {
				return s, errors.New(
					"no room address, and stdin has ended. Pass one with -room " +
						"or AP_ROOM, or set TF2AP_TEST_MODE=1 to play without a room")
			}
		}
	}
	if s.SrcdsRconPw == "" {
		password, err := settings.NewRconPassword()
		if err != nil {
			return s, err
		}
		s.SrcdsRconPw = password
	}
	return s, nil
}

// summary is what the player sees before the server starts: the room they
// gave, and the defaults they did not have to answer for.
func summary(s settings.Settings) {
	room := settings.Room{Host: s.APHost, Port: s.APPort, TLS: s.APTls}
	scheme := "ws"
	if s.APTls {
		scheme = "wss"
	}
	path, _ := settings.Path()
	fmt.Println()
	fmt.Printf("  room     %s (%s), slot %s\n", room, scheme, s.APSlotName)
	fmt.Printf("  server   %q on port %d, %s\n", s.SrcdsHostname, s.SrcdsPort, s.EffectiveReach().Label())
	fmt.Printf("  mission  %s\n", runshape.MissionLabel(s.SrcdsStartMission))
	fmt.Printf("  bots     %s\n", botsStatus(s))
	fmt.Printf("  run      %d missions, %s, goal %s\n", s.MvmMissionCount, s.MvmDifficulty, s.MvmGoal)
	fmt.Printf("  rcon     %s\n", s.SrcdsRconPw)
	fmt.Printf("\ntf2ap.exe -configure changes any of this. It is saved in %s.\n", path)
}

// configure walks every setting, section by section, the way the window's
// tabs do. One function per section: the whole list read as one is longer than
// a screen, and each section stands on its own.
func configure(p *ui.Prompt, s settings.Settings) settings.Settings {
	s = configureRoom(p, s)
	s = configureServer(p, s)
	s = configureBots(p, s)
	s = configureRun(p, s)
	return configureInstall(p, s)
}

// askReach asks where the players come from, and for the login token when the
// answer needs one. A reach that leaves the local network with no token is a
// server that refuses every player who tries to join, with a message that says
// nothing about a token, so this does not let one through.
func askReach(p *ui.Prompt, s settings.Settings) settings.Settings {
	s.SrcdsReach, _ = settings.ParseReach(p.Select("Who can reach the server", reachOptions(), string(s.SrcdsReach)))
	if !s.SrcdsReach.NeedsToken() {
		s.SrcdsToken = "0"
		return s
	}

	fmt.Println("  This needs a Game Server Login Token for app id 440.")
	fmt.Println("  Get one at steamcommunity.com/dev/managegameservers.")
	// Bounded: a closed stdin gives the same answer every time, and a prompt
	// that never gives up is a program that never exits.
	for range tokenAsks {
		s.SrcdsToken = p.Text("Game Server Login Token", s.SrcdsToken)
		if settings.HasToken(s.SrcdsToken) {
			return s
		}
		fmt.Println("  Without one nobody can join.")
	}
	fmt.Println("  No token given: keeping the server on the local network.")
	s.SrcdsReach, s.SrcdsToken = settings.ReachLan, "0"
	return s
}

func reachOptions() []ui.Option {
	options := make([]ui.Option, 0, len(settings.Reaches()))
	for _, reach := range settings.Reaches() {
		options = append(options, ui.Option{Value: string(reach), Label: reach.Label()})
	}
	return options
}

func configureRoom(p *ui.Prompt, s settings.Settings) settings.Settings {
	fmt.Println("--- Archipelago room ---")
	current := settings.Room{Host: s.APHost, Port: s.APPort, TLS: s.APTls}
	for {
		answer := p.Text("Room address (host:port)", current.String())
		room, err := settings.ParseRoom(answer)
		if err != nil {
			fmt.Printf("  %v\n", err)
			if p.Closed() {
				return s
			}
			continue
		}
		s.APHost, s.APPort, s.APTls = room.Host, room.Port, room.TLS
		break
	}
	s.APTls = p.Bool("Use TLS (wss)", s.APTls)
	s.APSlotName = p.Text("Slot name", s.APSlotName)
	s.APPassword = p.Password("Room password", s.APPassword)
	return s
}

func configureServer(p *ui.Prompt, s settings.Settings) settings.Settings {
	fmt.Println("\n--- Game server ---")
	s.SrcdsHostname = p.Text("Server hostname", s.SrcdsHostname)
	s.SrcdsRconPw = p.Password("RCON password", s.SrcdsRconPw)
	s.SrcdsPw = p.Password("Player password (blank for none)", s.SrcdsPw)
	s.SrcdsPort = p.Int("Game port", s.SrcdsPort)
	s.SrcdsStartMission = p.Select("Start mission", missionOptions(s), s.SrcdsStartMission)
	s.SrcdsAdminSteamIDs = p.Text("Admin Steam IDs (comma-separated, blank for none)", s.SrcdsAdminSteamIDs)
	for _, reach := range settings.Reaches() {
		fmt.Printf("  %-6s %s\n", reach, reach.Help())
	}
	s = askReach(p, s)
	return s
}

func configureBots(p *ui.Prompt, s settings.Settings) settings.Settings {
	fmt.Println("\n--- Defender bots ---")
	s.SrcdsBots = p.Bool("Fill the RED team with bots", s.SrcdsBots)
	if s.SrcdsBots {
		s.SrcdsBotTeamSize = p.Int("Fill RED to how many players", s.SrcdsBotTeamSize)
		s.SrcdsBotClassBlacklist = settings.SplitList(p.Text(
			"Classes the bots never play (comma-separated: sniper,spy)",
			strings.Join(s.SrcdsBotClassBlacklist, ",")))
		s.BotUpgradesChat = p.Bool("Say what the bots buy in the chat", s.BotUpgradesChat)
	}
	return s
}

func configureRun(p *ui.Prompt, s settings.Settings) settings.Settings {
	fmt.Println("\n--- Run shape ---")
	fmt.Println("These go in the player file the Archipelago app generates from.")
	fmt.Println("Change them here, then generate again, for a new seed.")

	fmt.Println("The easiest tier a mission may come from. Harder tiers are")
	fmt.Println("always in as well, so the pool shrinks as the floor rises.")
	tiers := runshape.Tiers()
	s.MvmDifficulty = p.Select("Difficulty floor", tierOptions(tiers), s.MvmDifficulty)

	pool := runshape.MissionsInPool(s.MvmDifficulty)
	s.MvmMissionCount = p.IntRange("Missions the run uses, out of that pool",
		s.MvmMissionCount, 1, pool)
	fmt.Printf("  about %d waves.\n", wavesFor(tiers, s.MvmDifficulty, s.MvmMissionCount))

	s.MvmGoal = p.Select("Goal", goalOptions(), s.MvmGoal)
	if s.MvmGoal == "missionsanity" {
		s.MvmMissionsanityPct = p.IntRange("Share of the missions to clear, in percent",
			s.MvmMissionsanityPct, 10, 100)
	}
	s.MvmDeathLink = p.Bool("Death Link, a lost wave kills every linked player and their deaths wipe the team", s.MvmDeathLink)
	fmt.Println("\n--- Rewards ---")
	importance := []ui.Option{{Value: "useful", Label: "Useful"}, {Value: "progression", Label: "Required for progression"}}
	s.MvmMissionTicketImportance = p.Select("Mission tickets", importance, s.MvmMissionTicketImportance)
	s.MvmClassUnlockImportance = p.Select("Class unlocks", importance, s.MvmClassUnlockImportance)
	s.MvmWeaponSlotImportance = p.Select("Weapon slots", importance, s.MvmWeaponSlotImportance)
	s.MvmWeaponBuffImportance = p.Select("Weapon buffs", importance, s.MvmWeaponBuffImportance)
	s.MvmCashRewards = p.Bool("Include temporary cash rewards", s.MvmCashRewards)
	if s.MvmCashRewards {
		s.MvmWeaponBuffPct = p.IntRange("Percent of spare checks that award buffs", s.MvmWeaponBuffPct, 0, 100)
	}
	s.MvmWeaponBuffStackChance = p.IntRange("Chance to stack an existing numeric buff", s.MvmWeaponBuffStackChance, 0, 100)
	s.MvmTrapPct = p.IntRange("Percent of spare checks that hold a trap instead of a reward", s.MvmTrapPct, 0, 100)
	s.MvmExcludedMissions = settings.SplitList(p.Text(
		"Missions the run never draws (comma-separated popfiles, e.g. mvm_ghost_town_666)",
		strings.Join(s.MvmExcludedMissions, ",")))
	return s
}

func configureInstall(p *ui.Prompt, s settings.Settings) settings.Settings {
	fmt.Println("\n--- Install location ---")
	s.InstallRoot = p.Text("Install root (14 GB of game files)", s.InstallRoot)
	fmt.Println("Where the Archipelago app is, for generating seeds. Blank looks in:")
	for _, dir := range generate.SearchPath("") {
		fmt.Printf("  %s\n", dir)
	}
	s.ArchipelagoDir = p.Text("Archipelago app (blank for the list above)", s.ArchipelagoDir)
	return s
}

// missionOptions lists Valve missions plus missions whose community archives
// validate locally. gamedata owns the catalog; see ADR 0001.
func missionOptions(s settings.Settings) []ui.Option {
	availablePaths := installer.AvailableCommunityArchives(settings.KnownCommunityArchives(s.CommunityContentDir))
	availablePacks := make([]string, 0, len(availablePaths))
	for _, path := range availablePaths {
		availablePacks = append(availablePacks, filepath.Base(path))
	}
	choices := runshape.MissionChoicesForPacks(availablePacks)
	options := make([]ui.Option, 0, len(choices))
	for _, choice := range choices {
		options = append(options, ui.Option{Value: choice.PopFile, Label: choice.Label})
	}
	return options
}

func tierOptions(tiers []runshape.Tier) []ui.Option {
	options := make([]ui.Option, 0, len(tiers))
	for _, tier := range tiers {
		options = append(options, ui.Option{Value: tier.Key, Label: tier.Label()})
	}
	return options
}

func goalOptions() []ui.Option {
	goals := runshape.Goals()
	options := make([]ui.Option, 0, len(goals))
	for _, goal := range goals {
		options = append(options, ui.Option{Value: goal.Key, Label: goal.Label()})
	}
	return options
}

func wavesFor(tiers []runshape.Tier, key string, missions int) int {
	for _, tier := range tiers {
		if tier.Key == key {
			return tier.WavesFor(missions)
		}
	}
	return 0
}

func showStatus(s settings.Settings) {
	fmt.Printf("Install root:  %s\n", s.InstallRoot)
	// Two different things: where the seed generator is installed, and which
	// room to play in. Both said "Archipelago", which read as one line
	// repeated.
	fmt.Printf("Generator app: %s\n", appDirStatus(s.ArchipelagoDir))
	fmt.Printf("Room:          %s (tls=%v)\n", settings.Room{Host: s.APHost, Port: s.APPort}, s.APTls)
	fmt.Printf("Slot:          %s\n", s.APSlotName)
	fmt.Printf("Server:        %s on port %d (reach=%s)\n", s.SrcdsHostname, s.SrcdsPort, reachStatus(s))
	fmt.Printf("Start mission: %s\n", runshape.MissionLabel(s.SrcdsStartMission))
	fmt.Printf("Run:           %d missions, %s, goal=%s\n", s.MvmMissionCount, s.MvmDifficulty, s.MvmGoal)
	fmt.Printf("Bots:          %s\n", botsStatus(s))
	fmt.Printf("RCON password: %s\n", masked(s.SrcdsRconPw))
}

// appDirStatus says where the Archipelago app is, or that it was not found and
// where the launcher looked. -status is what a player reads before asking why
// Generate seed does nothing.
func appDirStatus(given string) string {
	if dir, err := generate.FindApp(given); err == nil {
		return dir
	}
	return "not found in " + strings.Join(generate.SearchPath(given), ", ")
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

// reachStatus is the reach, and what the server can do with the token it has.
// A reach that needs a login token and has none is a server on the local
// network whatever it was asked for, and a status that does not say so sends
// the reader looking at their router.
func reachStatus(s settings.Settings) string {
	if effective := s.EffectiveReach(); effective != s.SrcdsReach {
		return fmt.Sprintf("%s, %s until it has a login token", s.SrcdsReach, effective)
	}
	return string(s.SrcdsReach)
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
