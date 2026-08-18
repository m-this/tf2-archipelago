package runtime

import (
	"fmt"
	"net"
	"strconv"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// ConnectLines are what a player types in the game's developer console to join
// this server: one line for the machine running it, one for every address the
// friends on the network can reach, and the password when there is one.
//
// The public address is left out on purpose. Nothing here can see what a
// router does with the port, and printing a guess would send people to an
// address that does not answer.
func ConnectLines(s settings.Settings) []string {
	port := strconv.Itoa(s.SrcdsPort)
	lines := []string{"connect " + net.JoinHostPort("127.0.0.1", port) + "   (on this machine)"}
	for _, address := range localAddresses() {
		lines = append(lines, "connect "+net.JoinHostPort(address, port)+"   (from your network)")
	}
	if s.SrcdsPw != "" {
		lines = append(lines, fmt.Sprintf("password %s   (before connect, the server asks for it)", s.SrcdsPw))
	}
	return lines
}

// localAddresses lists this machine's IPv4 addresses, skipping loopback and
// the ones Windows hands out when a network is not really there.
func localAddresses() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var found []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			found = append(found, ip.String())
		}
	}
	return found
}
