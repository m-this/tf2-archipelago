package provision

import (
	"strconv"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// args is the command line the game server starts with. It is separate
// from starting it, and pure, because the reach decides four arguments at once
// and a wrong combination is a server nobody can join: a test says which
// arguments each reach produces without downloading 14 GB to find out.
//
// The dash flags come first and the console commands after, which is the order
// the game's own scripts use. The commands run in the order they are given,
// and sv_lan has to be settled before the map loads and the server tries to
// log in to Steam.
func args(s settings.Settings, exeName string) []string {
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
	/* -debug is what makes a crash readable.

	Two access violations arrived in a debug bundle with no dump beside them, so
	the function the server died in is unknown and the cause with it. Source
	writes debug.log on a crash when this is set, with a stack, and that file
	does not depend on Breakpad having started. It costs nothing on a server
	that never crashes. */
	flags := []string{
		"-game", "tf", "-usercon", "-console", "-condebug", "-debug",
		"-maxplayers", strconv.Itoa(s.SrcdsMaxPlayers),
	}
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
