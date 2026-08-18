package settings

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Room is what a player has to hand: the one line the Archipelago room page
// shows, such as `archipelago.gg:12345`.
type Room struct {
	Host string
	Port int
	TLS  bool
}

// ParseRoom reads a room address. It accepts what people actually paste: the
// bare `host:port` from the room page, either websocket scheme in front of it,
// and a trailing path or slash.
//
// TLS follows the host unless a scheme said otherwise. A name on the internet
// serves wss; a machine on your own network does not have a certificate for
// its address, so it serves ws.
func ParseRoom(address string) (Room, error) {
	value := strings.TrimSpace(address)
	if value == "" {
		return Room{}, fmt.Errorf("the address is empty")
	}

	tls, scheme := true, false
	for _, prefix := range []string{"wss://", "ws://", "https://", "http://"} {
		if rest, found := strings.CutPrefix(value, prefix); found {
			value = rest
			tls = prefix == "wss://" || prefix == "https://"
			scheme = true
			break
		}
	}
	value = strings.TrimSuffix(strings.SplitN(value, "/", 2)[0], "/")

	host, portText, err := net.SplitHostPort(value)
	if err != nil {
		return Room{}, fmt.Errorf("write it as host:port, like archipelago.gg:12345")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return Room{}, fmt.Errorf("%q is not a port number", portText)
	}
	if host == "" {
		return Room{}, fmt.Errorf("the address has no host")
	}
	if !scheme {
		tls = !isLocal(host)
	}
	return Room{Host: host, Port: port, TLS: tls}, nil
}

// String renders the room the way the room page writes it.
func (r Room) String() string {
	if r.Port == 0 {
		return ""
	}
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

func isLocal(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

// NewRconPassword returns a password for the admin console. The launcher
// generates one rather than asking, because the answer is a secret nobody
// wants to invent, and rcon is open on the network the moment the server
// starts.
func NewRconPassword() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("cannot generate an RCON password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
