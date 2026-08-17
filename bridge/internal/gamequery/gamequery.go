// Package gamequery asks the game server how many people are on it.
//
// A2S_INFO over UDP, the same thing a server browser sends, and no credentials —
// unlike rcon, which would mean holding the rcon password to count players.
//
// Address it by the game server's name on the docker network, not 127.0.0.1.
// srcds binds 0.0.0.0:27015 and answers a query sent to any of its interface
// addresses, but drops one sent to loopback: measured on a running server, where
// 127.0.0.1 timed out while `srcds:27015` and the container's own eth0 address
// both answered. The bridge shares that network namespace, so either of those
// works from here.
//
// Only what a dashboard needs is read: the player counts and the map. The rest
// of the payload (steam id, keywords, ports) is skipped rather than modelled.
package gamequery

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

// request is A2S_INFO. The trailing NUL is part of the payload, not padding.
var request = []byte("\xff\xff\xff\xffTSource Engine Query\x00")

const (
	headerSimple = 0xffffffff

	replyInfo      = 'I'
	replyChallenge = 'A'

	// A challenge is answered exactly once. Two rounds is the protocol; a third
	// would be a server that has decided not to answer, and this is a metrics
	// scrape, not something worth retrying at.
	challengeRounds = 1

	// readMax is larger than any A2S_INFO reply. It bounds the read rather than
	// describing the payload.
	readMax = 4096
)

// Info is what a scrape needs out of the reply.
type Info struct {
	// Players counts everyone the server reports, robots included: in Mann vs
	// Machine the enemy waves are bots and they are in this number.
	Players int
	Bots    int

	// MaxPlayers is what the server advertises, which in MvM is the six RED
	// slots (sv_visiblemaxplayers) and not the 32 the process was started with.
	MaxPlayers int

	Map string
}

// Humans is the count a "who is playing" graph wants.
func (i Info) Humans() int {
	if i.Bots > i.Players {
		return 0
	}
	return i.Players - i.Bots
}

// Query sends one A2S_INFO to addr and parses the reply. The timeout covers the
// whole exchange, challenge included; ctx cancels it earlier, so a scrape that
// is abandoned does not leave this waiting.
func Query(ctx context.Context, addr string, timeout time.Duration) (Info, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "udp", addr)
	if err != nil {
		return Info{}, err
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return Info{}, err
	}

	payload := request
	for round := 0; round <= challengeRounds; round++ {
		if _, err := conn.Write(payload); err != nil {
			return Info{}, err
		}
		buffer := make([]byte, readMax)
		read, err := conn.Read(buffer)
		if err != nil {
			return Info{}, err
		}
		reply := buffer[:read]

		kind, body, err := split(reply)
		if err != nil {
			return Info{}, err
		}
		switch kind {
		case replyInfo:
			return parseInfo(body)
		case replyChallenge:
			if len(body) < 4 {
				return Info{}, fmt.Errorf("challenge is %d bytes", len(body))
			}
			// The challenge is echoed back appended to the same request.
			payload = append(append([]byte{}, request...), body[:4]...)
		default:
			return Info{}, fmt.Errorf("unexpected reply type %q", kind)
		}
	}
	return Info{}, errors.New("the server kept asking for a challenge")
}

// split checks the simple-response header and returns the reply type with the
// bytes after it.
func split(reply []byte) (byte, []byte, error) {
	if len(reply) < 5 {
		return 0, nil, fmt.Errorf("reply is %d bytes", len(reply))
	}
	if binary.LittleEndian.Uint32(reply[:4]) != headerSimple {
		return 0, nil, errors.New("not a single-packet response")
	}
	return reply[4], reply[5:], nil
}

// parseInfo reads the fields up to the bot count and stops. Everything after it
// is optional, differs by game, and no caller here reads it.
func parseInfo(body []byte) (Info, error) {
	// body starts at the protocol byte, followed by four NUL-terminated strings.
	if len(body) < 1 {
		return Info{}, errors.New("info reply is empty")
	}
	rest := body[1:]

	var strings [4]string // name, map, folder, game
	for i := range strings {
		value, remainder, err := cstring(rest)
		if err != nil {
			return Info{}, err
		}
		strings[i], rest = value, remainder
	}

	// appid (2) + players + maxplayers + bots
	if len(rest) < 5 {
		return Info{}, fmt.Errorf("info reply ends %d bytes early", 5-len(rest))
	}
	return Info{
		Players:    int(rest[2]),
		MaxPlayers: int(rest[3]),
		Bots:       int(rest[4]),
		Map:        strings[1],
	}, nil
}

func cstring(data []byte) (string, []byte, error) {
	for i, b := range data {
		if b == 0 {
			return string(data[:i]), data[i+1:], nil
		}
	}
	return "", nil, errors.New("unterminated string in info reply")
}
