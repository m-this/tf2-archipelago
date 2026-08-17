package gamequery

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeServer answers on loopback with whatever the script says, one reply per
// request, and reports the address to query.
func fakeServer(t *testing.T, replies ...[]byte) string {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	go func() {
		buffer := make([]byte, 1024)
		for _, reply := range replies {
			_, from, err := conn.ReadFrom(buffer)
			if err != nil {
				return
			}
			if _, err := conn.WriteTo(reply, from); err != nil {
				return
			}
		}
	}()
	return conn.LocalAddr().String()
}

// infoReply builds an A2S_INFO reply with the counts a test cares about.
func infoReply(players, maxPlayers, bots byte) []byte {
	reply := []byte("\xff\xff\xff\xffI\x11")
	reply = append(reply, "Mann vs Archipelago\x00"...)
	reply = append(reply, "mvm_decoy\x00"...)
	reply = append(reply, "tf\x00"...)
	reply = append(reply, "Team Fortress\x00"...)
	// appid 440, little endian, then the three counts.
	return append(reply, 0xb8, 0x01, players, maxPlayers, bots)
}

func challengeReply() []byte {
	return []byte("\xff\xff\xff\xffA\x01\x02\x03\x04")
}

func TestQueryReadsTheCounts(t *testing.T) {
	addr := fakeServer(t, infoReply(7, 6, 5))

	got, err := Query(context.Background(), addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got.Players != 7 || got.MaxPlayers != 6 || got.Bots != 5 {
		t.Fatalf("info = %+v", got)
	}
	// MvM counts its robot waves as players, so the people on the server are
	// what is left after the bots.
	if got.Humans() != 2 {
		t.Fatalf("humans = %d", got.Humans())
	}
	if got.Map != "mvm_decoy" {
		t.Fatalf("map = %q", got.Map)
	}
}

func TestQueryAnswersAChallenge(t *testing.T) {
	// Since 2020 a Source server answers the first A2S_INFO with a challenge and
	// only replies once it is echoed back.
	addr := fakeServer(t, challengeReply(), infoReply(1, 6, 0))

	got, err := Query(context.Background(), addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got.Players != 1 {
		t.Fatalf("info = %+v", got)
	}
}

func TestQueryGivesUpOnAnEndlessChallenge(t *testing.T) {
	addr := fakeServer(t, challengeReply(), challengeReply())

	if _, err := Query(context.Background(), addr, time.Second); err == nil {
		t.Fatal("a server that only ever challenges was accepted")
	}
}

func TestQueryRefusesATruncatedReply(t *testing.T) {
	for name, reply := range map[string][]byte{
		"empty":             {},
		"header only":       []byte("\xff\xff\xff\xff"),
		"not single packet": []byte("\xfe\xff\xff\xffI\x11rest"),
		"unknown type":      []byte("\xff\xff\xff\xffZ"),
		"unterminated name": []byte("\xff\xff\xff\xffI\x11Mann vs Archipelago"),
		"cut before counts": append([]byte("\xff\xff\xff\xffI\x11"), "n\x00m\x00f\x00g\x00\xb8"...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Query(context.Background(), fakeServer(t, reply), time.Second); err == nil {
				t.Fatalf("%s was accepted", name)
			}
		})
	}
}

func TestQueryTimesOutOnSilence(t *testing.T) {
	addr := fakeServer(t) // accepts the packet, never answers

	start := time.Now()
	_, err := Query(context.Background(), addr, 150*time.Millisecond)
	if err == nil {
		t.Fatal("silence was accepted")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("error = %v, want a timeout", err)
	}
	// The deadline covers the whole exchange, so it cannot hold a scrape open.
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("waited %s for a 150ms deadline", elapsed)
	}
}
