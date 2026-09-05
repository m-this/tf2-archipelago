// Package lanaddr answers one question: which of this machine's addresses do
// the friends on the network reach it on. The join lines and the download
// URL both need the same answer.
package lanaddr

import (
	"context"
	"net"
	"sort"
	"time"
)

// All lists this machine's IPv4 addresses, skipping loopback and the ones
// Windows hands out when a network is not really there. The address this
// machine sends from comes first.
func All() []string {
	preferred := Preferred()
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
	sort.SliceStable(found, func(i, j int) bool {
		return found[i] == preferred && found[j] != preferred
	})
	return found
}

// routeLookupTimeout bounds the route question below.
const routeLookupTimeout = time.Second

/* Preferred is the one address this machine would send from, asked of the OS.
 *
 * The order net.Interfaces returns is arbitrary, and a machine with Docker,
 * WSL or a virtual machine on it has several addresses that answer nothing.
 * One bundle carried four: 192.168.50.105 alongside 192.168.34.1,
 * 192.168.222.1 and 172.25.192.1. Taking the first of those for the join link
 * sends the player at an adapter with no server behind it, which is
 * "connection failed after 4 retries" and a stall at two bars while the LAN
 * tab of the server browser finds the same server first try.
 *
 * A UDP dial to a routable address sends nothing. It only makes the kernel
 * choose a route, and the local address of that choice is the interface this
 * machine really reaches the network on. Empty when there is no route at all.
 */
func Preferred() string {
	// Bounded, though a UDP dial only consults the routing table and does not
	// wait for anybody: a machine with no route at all should not hold the
	// interface up while it finds that out.
	ctx, cancel := context.WithTimeout(context.Background(), routeLookupTimeout)
	defer cancel()
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "udp4", "192.0.2.1:9")
	if err != nil {
		return ""
	}
	defer func() { _ = conn.Close() }()
	local, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || local.IP == nil {
		return ""
	}
	ip := local.IP.To4()
	if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return ""
	}
	return ip.String()
}
