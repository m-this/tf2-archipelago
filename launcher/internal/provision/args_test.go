package provision

import (
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func baseSettings() settings.Settings {
	return settings.Settings{
		SrcdsMaxPlayers:   32,
		SrcdsStartMission: "mvm_decoy",
		SrcdsPort:         27015,
		SrcdsRconPw:       "rcon-secret",
		SrcdsToken:        "0",
		SrcdsReach:        settings.ReachLan,
	}
}

// value returns what follows flag on the command line, or "" when the flag is
// not there at all.
func value(args []string, flag string) string {
	i := slices.Index(args, flag)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func TestLanArgsNeverLeaveTheNetwork(t *testing.T) {
	s := baseSettings()
	// A token left over from an evening spent online must not put a server
	// that is back on LAN mode into the public list.
	s.SrcdsToken = "C7A1B2E3D4F5A6B7C8D9E0F1A2B3C4D5"
	args := argsFor(t, s, "srcds_run")

	if got := value(args, "+sv_lan"); got != "1" {
		t.Errorf("+sv_lan = %q, want 1", got)
	}
	if slices.Contains(args, "-enablefakeip") {
		t.Error("-enablefakeip on a LAN server")
	}
	if slices.Contains(args, "+sv_setsteamaccount") {
		t.Error("the token was passed to a server that never logs in")
	}
	if slices.Contains(args, "+sv_use_steam_networking") {
		t.Error("+sv_use_steam_networking on a LAN server")
	}
}

func TestSteamArgsAskForARelayedAddress(t *testing.T) {
	s := baseSettings()
	s.SrcdsReach = settings.ReachSteam
	s.SrcdsToken = "C7A1B2E3D4F5A6B7C8D9E0F1A2B3C4D5"
	args := argsFor(t, s, "srcds_run")

	if !slices.Contains(args, "-enablefakeip") {
		t.Error("no -enablefakeip: Valve hands out no address without it")
	}
	if got := value(args, "+sv_use_steam_networking"); got != "1" {
		t.Errorf("+sv_use_steam_networking = %q, want 1", got)
	}
	if got := value(args, "+sv_lan"); got != "0" {
		t.Errorf("+sv_lan = %q, want 0", got)
	}
	if got := value(args, "+sv_setsteamaccount"); got != s.SrcdsToken {
		t.Errorf("+sv_setsteamaccount = %q, want the token", got)
	}
}

func TestPortArgsGoOnlineWithoutTheRelay(t *testing.T) {
	s := baseSettings()
	s.SrcdsReach = settings.ReachPort
	s.SrcdsToken = "C7A1B2E3D4F5A6B7C8D9E0F1A2B3C4D5"
	args := argsFor(t, s, "srcds_run")

	if slices.Contains(args, "-enablefakeip") {
		t.Error("-enablefakeip on a forwarded-port server")
	}
	if got := value(args, "+sv_lan"); got != "0" {
		t.Errorf("+sv_lan = %q, want 0", got)
	}
	if got := value(args, "+sv_setsteamaccount"); got != s.SrcdsToken {
		t.Errorf("+sv_setsteamaccount = %q, want the token", got)
	}
}

// "0" is how the compose file and the docs spell "no token". Passing it to
// sv_setsteamaccount is worse than passing nothing.
func TestNoTokenIsNotPassed(t *testing.T) {
	s := baseSettings()
	s.SrcdsReach = settings.ReachSteam
	s.SrcdsToken = "0"
	if args := argsFor(t, s, "srcds_run"); slices.Contains(args, "+sv_setsteamaccount") {
		t.Error("+sv_setsteamaccount 0 was passed")
	}
}

// srcds reads the dash flags as a set and the + commands in order, so the
// commands must not start before the flags are done.
func TestFlagsComeBeforeCommands(t *testing.T) {
	s := baseSettings()
	s.SrcdsReach = settings.ReachSteam
	args := argsFor(t, s, "srcds.exe")

	firstCommand := len(args)
	for i, arg := range args {
		if strings.HasPrefix(arg, "+") {
			firstCommand = i
			break
		}
	}
	for i, arg := range args[firstCommand:] {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "-1") {
			t.Errorf("flag %q at %d, after the commands started", arg, firstCommand+i)
		}
	}
	if !slices.Contains(args[:firstCommand], "-console") {
		t.Error("srcds.exe without -console waits for a click nobody makes")
	}
}

func TestSrcdsUsesTheSupportedPlayerLimit(t *testing.T) {
	args := argsFor(t, baseSettings(), "srcds_run")
	if slices.Contains(args, "-unrestricted_maxplayers") {
		t.Error("srcds starts with unsupported unrestricted slots, which reset to 101 on changelevel")
	}
	if got := value(args, "-maxplayers"); got != "32" {
		t.Errorf("-maxplayers = %q, want 32", got)
	}
	if slices.Contains(args, "+maxplayers") {
		t.Error("srcds uses +maxplayers, which resets to 101 on changelevel with unrestricted slots")
	}
}

// Both platforms need -console, for reasons that look different and are not.
// srcds.exe without it waits for a click on Start. srcds_linux without it
// brings up an interactive text console, and the launcher gives it no terminal
// to bring it up on: the server holds its port, burns a fifth of a core, and
// never finishes loading the map.
func TestSrcdsRunsWithAConsoleOnEitherPlatform(t *testing.T) {
	for _, exeName := range []string{"srcds.exe", "srcds_run"} {
		args := argsFor(t, settings.Settings{SrcdsPort: 27015}, exeName)
		if !slices.Contains(args, "-console") {
			t.Errorf("%s runs without -console", exeName)
		}
	}
	// The crash dialog is a Windows idea, and nothing on Linux answers it.
	if slices.Contains(argsFor(t, settings.Settings{}, "srcds_run"), "-nocrashdialog") {
		t.Error("srcds_run got -nocrashdialog, which is a Windows flag")
	}
}

// The watchdog goes off on Linux only. Windows is not reported as crashing and
// keeps the kill on a hung frame; Linux trades it away for a server that lives
// through load.
func TestTheWatchdogIsOffOnLinuxOnly(t *testing.T) {
	if !slices.Contains(argsFor(t, settings.Settings{SrcdsPort: 27015}, "srcds_run"), "-nowatchdog") {
		t.Error("srcds_run keeps the watchdog that kills it under load")
	}
	if slices.Contains(argsFor(t, settings.Settings{SrcdsPort: 27015}, "srcds.exe"), "-nowatchdog") {
		t.Error("srcds.exe lost its watchdog, and Windows was not the platform crashing")
	}
}

// The map is the start mission's, not the mission name. Ghost Town is the case
// that defeats trimming the popfile: mvm_ghost_town_666 runs on mvm_ghost_town.
func TestTheMapComesFromTheStartMission(t *testing.T) {
	s := baseSettings()
	s.SrcdsStartMission = "mvm_ghost_town_666"
	if got := value(argsFor(t, s, "srcds_run"), "+map"); got != "mvm_ghost_town" {
		t.Errorf("+map = %q, want mvm_ghost_town", got)
	}
}

func TestARemovedStartMissionUsesTheDefaultMap(t *testing.T) {
	s := baseSettings()
	s.SrcdsStartMission = "mvm_removed_mission"
	if got := value(argsFor(t, s, "srcds_run"), "+map"); got != "mvm_decoy" {
		t.Errorf("+map = %q, want mvm_decoy", got)
	}
}

// The default reach leaves the local network, and a fresh install has no token
// yet. That server has to come up on the local network rather than come up
// refusing everybody, which is what srcds does with sv_lan 0 and no Steam
// session behind it.
func TestNoTokenKeepsTheServerOnTheNetwork(t *testing.T) {
	s := baseSettings()
	s.SrcdsReach = settings.ReachPort
	s.SrcdsToken = "0"
	args := argsFor(t, s, "srcds_run")

	if got := value(args, "+sv_lan"); got != "1" {
		t.Errorf("+sv_lan = %q, want 1", got)
	}
	if slices.Contains(args, "+sv_setsteamaccount") {
		t.Error("a token was passed to a server that has none")
	}
}

// -ip 0.0.0.0 is also the address the engine believes it is on, and a LAN
// server refuses every player whose address is not in the same class C as
// that one. Told 0.0.0.0, it refuses the whole network: the player sees "LAN
// servers are restricted to local clients (class C)" and the server says
// nothing at all.
func TestALanServerKeepsItsOwnAddress(t *testing.T) {
	s := baseSettings()
	s.SrcdsReach = settings.ReachLan
	if args := argsFor(t, s, "srcds_run"); slices.Contains(args, "-ip") {
		t.Errorf("-ip on a LAN server: %v", args)
	}

	// A server with no token is a LAN server whatever it was asked for.
	s.SrcdsReach = settings.ReachPort
	s.SrcdsToken = "0"
	if args := argsFor(t, s, "srcds_run"); slices.Contains(args, "-ip") {
		t.Errorf("-ip on a server with no token: %v", args)
	}
}

// A server that leaves the network has no such comparison to make, and binding
// every interface is what keeps rcon answering on loopback.
func TestAnOpenServerBindsEveryInterface(t *testing.T) {
	s := baseSettings()
	s.SrcdsReach = settings.ReachPort
	s.SrcdsToken = "C7A1B2E3D4F5A6B7C8D9E0F1A2B3C4D5"
	if got := value(argsFor(t, s, "srcds_run"), "-ip"); got != "0.0.0.0" {
		t.Errorf("-ip = %q, want 0.0.0.0", got)
	}
}

// The game server writes its own console log, which is the one that survives
// the restart a player makes before asking anybody about the bug.
func TestSrcdsArgsAsksForTheConsoleLog(t *testing.T) {
	s := settings.Defaults()
	for _, exe := range []string{"srcds.exe", "srcds_run"} {
		if !slices.Contains(argsFor(t, s, exe), "-condebug") {
			t.Errorf("%s starts without -condebug", exe)
		}
	}
}

// argsFor is the command line For decides for s, started as exe.
func argsFor(t *testing.T, s settings.Settings, exe string) []string {
	t.Helper()
	server, err := For(s, Host{GOOS: "linux", Exe: exe})
	if err != nil {
		t.Fatal(err)
	}
	return server.Args
}

// sv_downloadurl names the launcher's own file server on the address friends
// reach this machine at. The operator's own host wins when they named one.
// A relayed server hands out no address of its own, so nothing is written and
// the game server's transfer stays the way in.
func TestDownloadURL(t *testing.T) {
	// Obviously not a token, so a secret scanner has nothing to flag.
	const token = "NOT-A-REAL-TOKEN-0000000000000000"
	for _, c := range []struct {
		name string
		s    settings.Settings
		host string
		want string
	}{
		{"lan", settings.Settings{SrcdsReach: settings.ReachLan, FastDLPort: 27080}, "192.168.1.10", "http://192.168.1.10:27080/tf"},
		{"port", settings.Settings{SrcdsReach: settings.ReachPort, SrcdsToken: token, FastDLPort: 27080}, "192.168.1.10", "http://192.168.1.10:27080/tf"},
		{"steam", settings.Settings{SrcdsReach: settings.ReachSteam, SrcdsToken: token, FastDLPort: 27080}, "192.168.1.10", ""},
		{"no route", settings.Settings{SrcdsReach: settings.ReachLan, FastDLPort: 27080}, "", ""},
		{"off", settings.Settings{SrcdsReach: settings.ReachLan, FastDLPort: 0}, "192.168.1.10", ""},
		{"own host", settings.Settings{SrcdsReach: settings.ReachSteam, SrcdsToken: token, SrcdsDownloadURL: "https://example.test/tf"}, "", "https://example.test/tf"},
	} {
		if got := DownloadURL(c.s, c.host); got != c.want {
			t.Errorf("%s: DownloadURL = %q, want %q", c.name, got, c.want)
		}
	}
}
