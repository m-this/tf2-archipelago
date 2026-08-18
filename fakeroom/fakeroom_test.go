package fakeroom

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// client drives the room's websocket directly, the way the bridge does, so the
// dedup and the seed can be checked without a whole apclient session.
type client struct {
	t    *testing.T
	conn *websocket.Conn
}

func dial(t *testing.T, address string) *client {
	t.Helper()
	conn, handshake, err := websocket.Dial(t.Context(), address, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if handshake != nil && handshake.Body != nil {
		_ = handshake.Body.Close()
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return &client{t: t, conn: conn}
}

func (c *client) send(message any) {
	c.t.Helper()
	body, err := json.Marshal([]any{message})
	if err != nil {
		c.t.Fatal(err)
	}
	if err := c.conn.Write(c.t.Context(), websocket.MessageText, body); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

// await reads messages until one with the wanted cmd shows up.
func (c *client) await(want string) map[string]json.RawMessage {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(c.t.Context(), 3*time.Second)
	defer cancel()
	for {
		_, body, err := c.conn.Read(ctx)
		if err != nil {
			c.t.Fatalf("waiting for %s: %v", want, err)
		}
		var messages []map[string]json.RawMessage
		if err := json.Unmarshal(body, &messages); err != nil {
			c.t.Fatal(err)
		}
		for _, message := range messages {
			var cmd string
			if err := json.Unmarshal(message["cmd"], &cmd); err != nil {
				c.t.Fatal(err)
			}
			if cmd == want {
				return message
			}
		}
	}
}

func startRoom(t *testing.T) string {
	t.Helper()
	room, address, err := Start(t.Context(), Options{
		SlotName:     "tester",
		MissionCount: 2,
		Log:          func(string) {},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = room.Close(context.Background()) })
	return address
}

func connect(t *testing.T, address string) *client {
	t.Helper()
	c := dial(t, address)
	c.await("RoomInfo")
	c.send(map[string]any{"cmd": "Connect"})
	c.await("Connected")
	return c
}

func itemCount(t *testing.T, message map[string]json.RawMessage) int {
	t.Helper()
	var items []json.RawMessage
	if err := json.Unmarshal(message["items"], &items); err != nil {
		t.Fatal(err)
	}
	return len(items)
}

// This is the bug: the bridge resends its whole check list on every report,
// which is correct against a real server because a repeat check there hands
// out nothing new. The room has to make that true itself, or a resend floods
// unlocks nobody earned.
func TestASecondReportOfTheSameChecksGrantsNothingMore(t *testing.T) {
	address := startRoom(t)
	c := connect(t, address)

	c.send(map[string]any{"cmd": "LocationChecks", "locations": []int64{111, 222}})
	first := c.await("ReceivedItems")
	if got := itemCount(t, first); got != 2 {
		t.Fatalf("first report granted %d item(s), want 2", got)
	}

	// The same two locations again, the way a reconnect or a ping-triggered
	// report resends them. A third, genuinely new one is mixed in so a
	// wrongly-silent room cannot be mistaken for one that dropped the message.
	c.send(map[string]any{"cmd": "LocationChecks", "locations": []int64{111, 222, 333}})
	second := c.await("ReceivedItems")
	if got := itemCount(t, second); got != 1 {
		t.Fatalf("the resend granted %d item(s), want 1 for the one new location", got)
	}
}

// Test mode's state on the bridge side is durable and keyed by seed name
// (BindSeed). A constant seed name would have every restart treat the
// bridge's already-recorded checks as still due a reward, and this
// memory-only room has no record of ever having paid them.
func TestEachStartUsesADifferentSeed(t *testing.T) {
	addressA := startRoom(t)
	addressB := startRoom(t)

	seedA := seedOf(t, addressA)
	seedB := seedOf(t, addressB)
	if seedA == seedB {
		t.Fatalf("two rooms shared the seed %q", seedA)
	}
}

func seedOf(t *testing.T, address string) string {
	t.Helper()
	c := dial(t, address)
	roomInfo := c.await("RoomInfo")
	var seed string
	if err := json.Unmarshal(roomInfo["seed_name"], &seed); err != nil {
		t.Fatal(err)
	}
	return seed
}
