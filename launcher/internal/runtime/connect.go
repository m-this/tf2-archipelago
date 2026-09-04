package runtime

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/lanaddr"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// ConnectLines are what a player types in the game's developer console to join
// this server: one line for the machine running it, one for every address the
// friends on the network can reach, and the password when there is one.
//
// The public address is left out on purpose. Nothing here can see what a
// router does with the port, and printing a guess would send people to an
// address that does not answer. The relayed address is left out for the same
// reason: it does not exist yet at this point. FakeIPAddress picks it out of
// the server's own output once Valve has handed one over.
func ConnectLines(s settings.Settings) []string {
	port := strconv.Itoa(s.SrcdsPort)
	lines := []string{"connect " + net.JoinHostPort("127.0.0.1", port) + "   (on this machine)"}
	for _, address := range LocalAddresses() {
		lines = append(lines, "connect "+net.JoinHostPort(address, port)+"   (from your network)")
	}
	switch s.SrcdsReach {
	case settings.ReachLan:
		// Nothing to add: the lines above are already the whole answer.
	case settings.ReachSteam:
		lines = append(lines, "the address for friends elsewhere follows, once Steam hands one over")
	case settings.ReachPort:
		lines = append(lines, fmt.Sprintf("friends elsewhere need your public address, with port %s forwarded to this machine", port))
	}
	if s.SrcdsPw != "" {
		lines = append(lines, fmt.Sprintf("password %s   (before connect, the server asks for it)", s.SrcdsPw))
	}
	return lines
}

// fakeIPPrefix is what srcds prints when Valve has handed it a relayed
// address. The rest of the line is "169.254.13.42:20232, 20233": the first
// port carries the game and the second is the query port, which nobody types.
const fakeIPPrefix = "FakeIP allocation succeeded:"

// FakeIPAddress picks the relayed address out of one line of server output, or
// returns "" for every other line. Over Steam this address is the only way in,
// it is new on every start, and it exists nowhere else: reading it back out of
// the log is how the launcher learns it.
func FakeIPAddress(line string) string {
	_, rest, found := strings.Cut(line, fakeIPPrefix)
	if !found {
		return ""
	}
	address, _, _ := strings.Cut(rest, ",")
	address = strings.TrimSpace(address)
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return ""
	}
	// Valve allocates out of 169.254.0.0/16. Anything else on this line is a
	// message that changed shape, and a wrong address is worse than none.
	if ip := net.ParseIP(host); ip == nil || !ip.IsLinkLocalUnicast() {
		return ""
	}
	if _, err := strconv.Atoi(port); err != nil {
		return ""
	}
	return address
}

// LocalAddresses lists this machine's IPv4 addresses, the one friends reach it
// on first. See lanaddr.
func LocalAddresses() []string { return lanaddr.All() }

// RconAddresses are the addresses this machine's own game server may answer
// rcon on, in the order worth trying.
//
// srcds binds rcon to the address it believes it is on: 0.0.0.0 where the
// launcher passed -ip, and otherwise whatever the hostname resolves to, which
// is 127.0.1.1 on Debian and this machine's address on the network elsewhere.
// A LAN server is not passed -ip, because that address is also the one it
// compares joining players against, so which of these answers depends on the
// reach. Trying them in turn costs a refused connection on loopback, which is
// immediate.
func RconAddresses(s settings.Settings) []string {
	port := strconv.Itoa(s.SrcdsPort)
	addresses := []string{net.JoinHostPort("127.0.0.1", port)}
	host, err := os.Hostname()
	if err != nil {
		return addresses
	}
	// Bounded: a name server that never answers must not hold up the command
	// the player typed. Loopback is already in the list to fall back on.
	ctx, cancel := context.WithTimeout(context.Background(), hostLookupTimeout)
	defer cancel()
	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return addresses
	}
	for _, ip := range resolved {
		if ip.IP.To4() == nil {
			continue
		}
		address := net.JoinHostPort(ip.IP.String(), port)
		if !slices.Contains(addresses, address) {
			addresses = append(addresses, address)
		}
	}
	return addresses
}

// hostLookupTimeout is how long RconAddresses waits for this machine's own
// name to resolve. It is a hosts file lookup on every machine that has one.
const hostLookupTimeout = 2 * time.Second

// missionSwitchedPrefix is what the plugin logs whenever it loads a mission of
// the run, which the launcher reads to learn which one is on. The rest of the
// line is "Doe's Drill (mvm_decoy) on mvm_decoy".
const missionSwitchedPrefix = "mission switched to "

// LoadedMission picks the mission the plugin just loaded out of one line of
// server output, or returns "" for every other line.
//
// The run picks its own missions, so the mission the settings name is only
// ever the first one. Nothing else on this side is told when that changes: the
// plugin says so in the server's log and this is what listens.
func LoadedMission(line string) string {
	_, rest, found := strings.Cut(line, missionSwitchedPrefix)
	if !found {
		return ""
	}
	name, _, found := strings.Cut(rest, " (")
	if !found {
		return ""
	}
	return strings.TrimSpace(name)
}

// sourcemodUpdatedPrefix is what SourceMod's updater prints once it has
// fetched new gamedata. The files are on disk by then and none of them is in
// use until SourceMod loads again.
const sourcemodUpdatedPrefix = "[UPDATER] SourceMod has been updated"

// SourceModWasUpdated reports whether this line is the updater asking for a
// restart. It asks the operator, who is watching a log scroll past, so the
// launcher reads it instead and restarts on their behalf.
func SourceModWasUpdated(line string) bool {
	return strings.Contains(line, sourcemodUpdatedPrefix)
}

/* itemSchemaPrefix is srcds saying it reached Steam's item server.
 *
 * That server is what hands out weapons. Without it every player and every bot
 * plays full stock, and the game says nothing about why: a player reported
 * running a whole evening on stock without being able to tell an outage from a
 * setup step he had missed.
 *
 * The success line is matched rather than a failure, because the failure has
 * several wordings and the success has one. Anything else on the same subject
 * is passed through as-is by ItemServerLine, which is enough to say that
 * something happened without claiming to know what.
 */
const itemSchemaPrefix = "Current item schema is up-to-date"

// ItemServerLine turns one line of server output into something worth showing a
// player, or "" for every other line.
func ItemServerLine(line string) string {
	if strings.Contains(line, itemSchemaPrefix) {
		return "the item server answered: weapons are available"
	}
	if !strings.Contains(strings.ToLower(line), "item schema") {
		return ""
	}
	// Not the up-to-date line, so it is the item server having something to
	// say. Passed through rather than interpreted.
	return "item server: " + strings.TrimSpace(line)
}

// GameAppID is Team Fortress 2 in the Steam client. The dedicated server is a
// different application, 232250, and the two are not interchangeable here.
const GameAppID = "440"

// SteamConnectURL is the link that starts Team Fortress 2 and joins this
// server in one step. Steam owns the steam:// scheme and hands the game the
// connect and the password itself.
//
// It names the game rather than using steam://connect, which asks the server
// which game it is. Ours answers with something the Steam client will not
// launch, and the client says "app id specified by server is invalid": the
// server is application 232250, and 440 is the one anybody owns. A server that
// never logged in to Steam has no better answer to give, and a game on the
// same machine needs no login token to be worth joining. So this names 440 and
// asks the server nothing.
//
// The address is this machine's own address on the network, not 127.0.0.1.
// Loopback looks like the obvious choice for a server on the same machine and
// does not work: a connect to it times out, while the same server joined from
// the LAN tab of the server browser answers on the network address, which is
// also the address that tab shows. So the link uses the first address the
// machine has, and keeps loopback only for a machine that has none.
func SteamConnectURL(s settings.Settings, steamAddress string) string {
	address := steamAddress
	// Over Steam the relayed address is the server's address, and the machine
	// running the launcher can reach it as readily as anybody else. Joining a
	// local address instead worked here and nowhere else, so the link the log
	// carries, which is the one a player pastes to a friend, went to a machine
	// that friend has never heard of.
	//
	// Empty until Valve hands one out, and the local address is what there is
	// until then.
	if s.SrcdsReach != settings.ReachSteam || address == "" {
		host := "127.0.0.1"
		if local := LocalAddresses(); len(local) > 0 {
			host = local[0]
		}
		address = net.JoinHostPort(host, strconv.Itoa(s.SrcdsPort))
	}
	arguments := "+connect " + address
	if s.SrcdsPw != "" {
		arguments += " +password " + s.SrcdsPw
	}
	// The arguments are one field of the URL, so the spaces inside them are
	// escaped rather than ending it.
	return "steam://run/" + GameAppID + "//" + url.PathEscape(arguments)
}
