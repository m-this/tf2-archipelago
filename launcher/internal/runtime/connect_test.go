package runtime

import (
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/lanaddr"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func TestFakeIPAddress(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{
			name: "the line srcds prints",
			line: "FakeIP allocation succeeded: 169.254.134.215:57528, 57529",
			want: "169.254.134.215:57528",
		},
		{
			name: "with the timestamp the console adds",
			line: "08/19 04:09:13 FakeIP allocation succeeded: 169.254.13.42:20232, 20233",
			want: "169.254.13.42:20232",
		},
		{
			name: "one port only",
			line: "FakeIP allocation succeeded: 169.254.13.42:20232",
			want: "169.254.13.42:20232",
		},
		{name: "another line entirely", line: "Connection to Steam servers successful.", want: ""},
		{name: "the request, not the answer", line: "Requesting FakeIP as per -enablefakeip", want: ""},
		// Anything outside 169.254.0.0/16 is not a relayed address. Printing a
		// real one here would hand out the address the relay exists to hide.
		{name: "a real address", line: "FakeIP allocation succeeded: 203.0.113.7:27015, 27016", want: ""},
		{name: "no port", line: "FakeIP allocation succeeded: 169.254.13.42, 20233", want: ""},
		{name: "not a port", line: "FakeIP allocation succeeded: 169.254.13.42:none, 20233", want: ""},
		{name: "nothing after it", line: "FakeIP allocation succeeded:", want: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FakeIPAddress(c.line); got != c.want {
				t.Errorf("FakeIPAddress(%q) = %q, want %q", c.line, got, c.want)
			}
		})
	}
}

func TestConnectLinesSayWhatEachReachNeeds(t *testing.T) {
	s := settings.Settings{SrcdsPort: 27015, SrcdsReach: settings.ReachLan}

	joined := strings.Join(ConnectLines(s), "\n")
	if !strings.Contains(joined, "connect 127.0.0.1:27015") {
		t.Error("no loopback line")
	}
	if strings.Contains(joined, "forwarded") || strings.Contains(joined, "Steam") {
		t.Errorf("a LAN server talked about getting out:\n%s", joined)
	}

	s.SrcdsReach = settings.ReachSteam
	if joined := strings.Join(ConnectLines(s), "\n"); !strings.Contains(joined, "Steam") {
		t.Errorf("nothing said about the address to come:\n%s", joined)
	}

	s.SrcdsReach = settings.ReachPort
	if joined := strings.Join(ConnectLines(s), "\n"); !strings.Contains(joined, "27015 forwarded") {
		t.Errorf("nothing said about forwarding the port:\n%s", joined)
	}
}

// The Join button hands this to Steam, which starts the game and connects.
// The password rides in the URL, so one that holds a space or a slash has to
// survive the trip.
//
// The link names 440. steam://connect asks the server which game it is, and
// ours answers with the dedicated server's own application, which the client
// refuses with "app id specified by server is invalid".
func TestSteamConnectURL(t *testing.T) {
	s := settings.Settings{SrcdsPort: 27015}

	// This machine's own address, not loopback: a connect to 127.0.0.1 times
	// out against a server on the same machine, and the LAN tab of the server
	// browser lists that server at its network address.
	host := "127.0.0.1"
	if local := LocalAddresses(); len(local) > 0 {
		host = local[0]
	}

	if got := SteamConnectURL(s, ""); got != "steam://run/440//+connect%20"+host+":27015" {
		t.Errorf("with no password = %q", got)
	}

	s.SrcdsPw = "friends only/2"
	want := "steam://run/440//+connect%20" + host + ":27015%20+password%20friends%20only%2F2"
	if got := SteamConnectURL(s, ""); got != want {
		t.Errorf("with a password = %q, want %q", got, want)
	}

	// A port that is not the default has to reach the URL, or the button joins
	// a server that is not the one running.
	s.SrcdsPw, s.SrcdsPort = "", 27045
	if got := SteamConnectURL(s, ""); !strings.Contains(got, host+":27045") {
		t.Errorf("the port did not reach the link: %q", got)
	}
}

// The run moves itself from mission to mission, and the plugin's log line is
// the only place that says which one is on. The window's header showed the
// mission the settings named instead, which stayed on the first one all
// evening.
func TestLoadedMission(t *testing.T) {
	line := `L 08/19/2026 - 17:28:26: [tf2_archipelago.smx] mission switched to Doe's Drill (mvm_decoy) on mvm_decoy`
	if got := LoadedMission(line); got != "Doe's Drill" {
		t.Errorf("LoadedMission = %q, want %q", got, "Doe's Drill")
	}
	// A name with a space and a plus in it, which several missions have.
	line = `[tf2_archipelago.smx] mission switched to Ctrl+Alt+Destruction (mvm_coaltown_advanced) on mvm_coaltown`
	if got := LoadedMission(line); got != "Ctrl+Alt+Destruction" {
		t.Errorf("LoadedMission = %q", got)
	}
	for _, other := range []string{
		"-------- Mapchange to mvm_decoy --------",
		`[UPDATER] Successfully updated gamedata file "sm-tf2.games.txt"`,
		"mission switched to",
		"",
	} {
		if got := LoadedMission(other); got != "" {
			t.Errorf("read a mission out of %q: %q", other, got)
		}
	}
}

// The updater writes its gamedata and then asks whoever is watching the log to
// restart. Reading that line is what lets the launcher do it instead.
func TestSourceModWasUpdated(t *testing.T) {
	for _, line := range []string{
		"L 08/19/2026 - 17:28:35: [UPDATER] SourceMod has been updated, please reload it or restart your server.",
		"[UPDATER] SourceMod has been updated",
	} {
		if !SourceModWasUpdated(line) {
			t.Errorf("missed the updater asking for a restart: %q", line)
		}
	}
	// The same updater says this a few hundred times per start, and none of
	// them is a reason to bring the server round.
	for _, line := range []string{
		`L 08/19/2026 - 17:28:34: [UPDATER] Successfully updated gamedata file "sm-tf2.games.txt"`,
		"L 08/19/2026 - 17:28:26: [tf2_archipelago.smx] mission switched to Doe's Drill",
		"",
	} {
		if SourceModWasUpdated(line) {
			t.Errorf("restarted for an ordinary line: %q", line)
		}
	}
}

// Over Steam the relayed address is the server's address, so the link goes
// there rather than to an address only this network knows. It used to join a
// local address whatever the reach was, which worked on the machine that ran
// the launcher and nowhere else.
func TestSteamConnectURLPrefersTheRelayedAddress(t *testing.T) {
	s := settings.Settings{SrcdsPort: 27015, SrcdsReach: settings.ReachSteam}

	got := SteamConnectURL(s, "169.254.13.42:20232")
	if !strings.Contains(got, "169.254.13.42:20232") {
		t.Errorf("the relayed address is not in %q", got)
	}

	// Valve hands it out a moment after the server starts, and until it does
	// there is only the local address.
	if got := SteamConnectURL(s, ""); strings.Contains(got, "169.254") {
		t.Errorf("an address nobody handed out reached %q", got)
	}

	// Any other reach ignores it: a relayed address on a LAN server is one
	// nothing routes to.
	s.SrcdsReach = settings.ReachLan
	if got := SteamConnectURL(s, "169.254.13.42:20232"); strings.Contains(got, "169.254") {
		t.Errorf("a LAN server was joined over Steam: %q", got)
	}
}

/*
The item server is what hands out weapons, and its absence is silent.

A player ran a whole evening on stock and could not tell an outage from a setup
step he had missed. The success line has one wording and the failures have
several, so the success is what is matched.
*/
func TestItemServerLine(t *testing.T) {
	up := "Current item schema is up-to-date with version 760AF0C1."
	if got := ItemServerLine(up); got == "" || !strings.Contains(got, "weapons are available") {
		t.Errorf("the up-to-date line said %q", got)
	}
	// Anything else about the schema is passed through rather than read.
	other := "Failed to load item schema"
	if got := ItemServerLine(other); !strings.Contains(got, "Failed to load item schema") {
		t.Errorf("a schema failure said %q", got)
	}
	for _, line := range []string{
		"Server is hibernating",
		"L 08/26/2026 - 07:58:45: [tf2_archipelago.smx] anything",
	} {
		if got := ItemServerLine(line); got != "" {
			t.Errorf("%q was read as an item server line: %q", line, got)
		}
	}
}

/*
The join link has to name the address the machine really answers on.

A bundle carried four: 192.168.50.105 beside 192.168.34.1, 192.168.222.1 and
172.25.192.1, which are the adapters Docker, WSL and a virtual machine leave
behind. net.Interfaces returns them in no useful order, and taking the first
sent players at an adapter with nothing behind it: "connection failed after 4
retries", a stall at two bars, while the LAN tab of the server browser found
the same server first try.
*/
func TestTheRoutableAddressComesFirst(t *testing.T) {
	preferred := lanaddr.Preferred()
	if preferred == "" {
		t.Skip("no route out of this machine, so there is no preference to check")
	}
	addresses := LocalAddresses()
	if len(addresses) == 0 {
		t.Skip("no addresses on this machine")
	}
	if !slices.Contains(addresses, preferred) {
		t.Fatalf("the routable address %s is not in %v", preferred, addresses)
	}
	if addresses[0] != preferred {
		t.Errorf("addresses = %v, want %s first", addresses, preferred)
	}
}

// The dial must not actually talk to anything: 192.0.2.1 is the documentation
// range and nothing routes there, which is the point.
func TestThePreferredAddressIsNotLoopback(t *testing.T) {
	if got := lanaddr.Preferred(); got != "" && strings.HasPrefix(got, "127.") {
		t.Errorf("preferred address = %q, which is loopback", got)
	}
}

/*
An address a client is told to look at is not the same question as whether this
machine answers, and the settings that hold them are separate.

A server reached over a forwarded port needs both: the launcher writes the
address it prints in the connect lines, which is this machine on the local
network, and a friend outside cannot reach that. Filling in the public address
used to turn off the server it was addressing, so the fix stopped the file
server and pointed the players at nothing.
*/
func TestTheDownloadURLDoesNotDecideWhetherWeServe(t *testing.T) {
	for _, c := range []struct {
		name string
		s    settings.Settings
		want string
	}{
		{"default", settings.Settings{FastDLPort: 27080}, ":27080"},
		{"public address named", settings.Settings{FastDLPort: 27080, SrcdsDownloadURL: "http://198.51.100.7:27080/tf"}, ":27080"},
		{"somebody else's host", settings.Settings{FastDLPort: 27080, SrcdsDownloadURL: "https://example.test/tf"}, ":27080"},
		{"turned off", settings.Settings{FastDLPort: 0, SrcdsDownloadURL: "https://example.test/tf"}, ""},
		{"turned off, nothing named", settings.Settings{FastDLPort: 0}, ""},
	} {
		if got := FastDLListen(c.s); got != c.want {
			t.Errorf("%s: FastDLListen = %q, want %q", c.name, got, c.want)
		}
	}
}
