package runtime

import (
	"net"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Whatever the server bound, the launcher has to find it: loopback first,
// because that is where a server told -ip 0.0.0.0 answers.
func TestRconAddressesStartAtLoopback(t *testing.T) {
	s := settings.Settings{SrcdsPort: 27015, SrcdsReach: settings.ReachLan}
	addresses := RconAddresses(s)
	if len(addresses) == 0 {
		t.Fatal("no address to try")
	}
	if addresses[0] != "127.0.0.1:27015" {
		t.Errorf("first address = %q, want 127.0.0.1:27015", addresses[0])
	}
	for _, address := range addresses {
		if _, port, err := net.SplitHostPort(address); err != nil || port != "27015" {
			t.Errorf("address %q: port %q, %v", address, port, err)
		}
	}
}
