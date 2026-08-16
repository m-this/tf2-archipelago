// Package apclient holds the Archipelago session: one websocket, reconnected
// for as long as the bridge runs.
//
// Everything durable lives in the state store. This package decides what to
// say and when, never what to remember.
package apclient

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"

	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/chat"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/state"
	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

const (
	// readLimitBytes bounds one message. RoomInfo carries a checksum per game
	// in the multiworld, so the default 32 KiB is not enough.
	readLimitBytes = 4 << 20

	dialTimeout  = 30 * time.Second
	writeTimeout = 10 * time.Second
	pingEvery    = 30 * time.Second

	backoffFirst = time.Second
	backoffMax   = 30 * time.Second
)

// version is the Archipelago release this client is written against.
var version = Version{Class: "Version", Major: 0, Minor: 6, Build: 7}

// permanentError wraps a failure that reconnecting cannot fix: a wrong slot
// name, a password the operator has to change, a seed this binary cannot read.
// The bridge stops rather than hammering the server forever.
type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// ErrNotConnected is what a chat line gets when there is no multiworld to say
// it to. Unlike a check, it is not worth queueing: a message that arrives ten
// minutes late is worse than one that was refused.
var ErrNotConnected = errors.New("not connected to archipelago")

// Options is everything the session needs.
type Options struct {
	URL      string
	SlotName string
	Password string
	Store    *state.Store
	Chat     *chat.Log
	Logger   *slog.Logger
}

// Client is one Archipelago session, reconnected as needed.
type Client struct {
	opts Options
	uuid string

	writeMu sync.Mutex

	mu        sync.Mutex
	conn      *websocket.Conn
	connected bool
	slot      SlotData
	lastError string
}

// Health is what /healthz reports.
type Health struct {
	Connected bool     `json:"connected"`
	Slot      string   `json:"slot"`
	Missions  []string `json:"missions"`
	LastError string   `json:"last_error,omitempty"`
}

func New(opts Options) *Client {
	if opts.Chat == nil {
		// A session with nowhere to put chat still has to run. One line of
		// history is enough to keep the append path honest.
		opts.Chat = chat.New(1)
	}
	return &Client{opts: opts, uuid: randomUUID()}
}

// Health reports the session state. The plugin uses it to tell a player the
// difference between "your check did not register" and "the multiworld is
// unreachable and your check is queued".
func (c *Client) Health() Health {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Health{
		Connected: c.connected,
		Slot:      c.opts.SlotName,
		Missions:  c.slot.Missions,
		LastError: c.lastError,
	}
}

// Run holds the session open until the context ends or something unfixable
// happens. Everything else is retried.
func (c *Client) Run(ctx context.Context) error {
	backoff := backoffFirst
	for {
		start := time.Now()
		err := c.session(ctx)
		if ctx.Err() != nil {
			// The bridge is shutting down, and whatever the session was doing
			// when it was cancelled is not a failure.
			return nil //nolint:nilerr // a cancelled context is a clean stop
		}
		if _, unfixable := errors.AsType[permanentError](err); unfixable {
			return err
		}
		c.setDisconnected(err)

		// A session that stayed up was not a bad address or a busy server, so
		// the next attempt starts from the short delay again.
		if time.Since(start) > backoffMax {
			backoff = backoffFirst
		}
		c.opts.Logger.WarnContext(ctx, "archipelago session ended, retrying",
			"error", err, "in", backoff)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, backoffMax)
	}
}

func (c *Client) session(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dialCtx, cancelDial := context.WithTimeout(ctx, dialTimeout)
	defer cancelDial()
	conn, handshake, err := websocket.Dial(dialCtx, c.opts.URL, &websocket.DialOptions{
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		return fmt.Errorf("dial %s: %w", c.opts.URL, err)
	}
	if handshake != nil && handshake.Body != nil {
		_ = handshake.Body.Close()
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(readLimitBytes)

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
	}()

	ready := make(chan struct{})
	pumped := make(chan error, 1)
	go func() { pumped <- c.pump(ctx, conn, ready) }()

	err = c.readLoop(ctx, conn, ready)
	cancel()
	<-pumped
	return err
}

// readLoop consumes the server's messages until the connection dies. The
// handshake is part of it: RoomInfo arrives first, Connect goes back, and
// everything after that is a running session.
func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn, ready chan struct{}) error {
	for {
		kind, body, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if kind != websocket.MessageText {
			continue
		}
		var messages []json.RawMessage
		if err := json.Unmarshal(body, &messages); err != nil {
			return fmt.Errorf("unreadable packet: %w", err)
		}
		for _, message := range messages {
			if err := c.handle(ctx, conn, message, ready); err != nil {
				return err
			}
		}
	}
}

func (c *Client) handle(
	ctx context.Context, conn *websocket.Conn, message json.RawMessage, ready chan struct{},
) error {
	var head header
	if err := json.Unmarshal(message, &head); err != nil {
		return fmt.Errorf("packet without a cmd: %w", err)
	}
	switch head.Cmd {
	case "RoomInfo":
		return c.onRoomInfo(ctx, conn, message)
	case "Connected":
		return c.onConnected(message, ready)
	case "ConnectionRefused":
		return c.onConnectionRefused(message)
	case "ReceivedItems":
		return c.onReceivedItems(ctx, conn, message)
	case "PrintJSON":
		var printed printJSON
		if err := json.Unmarshal(message, &printed); err == nil {
			text := printed.text()
			c.opts.Logger.InfoContext(ctx, "archipelago", "message", text)
			c.opts.Chat.Append(text)
		}
		return nil
	case "Bounced":
		// DeathLink lands here. What a death means in Mann vs Machine is not
		// settled (docs/spec.md, open question 5), so the bridge does not
		// claim the tag and nothing acts on this yet.
		return nil
	default:
		return nil
	}
}

func (c *Client) onRoomInfo(ctx context.Context, conn *websocket.Conn, message json.RawMessage) error {
	var room roomInfo
	if err := json.Unmarshal(message, &room); err != nil {
		return err
	}
	wiped, err := c.opts.Store.BindSeed(room.SeedName)
	if err != nil {
		return err
	}
	if wiped {
		c.opts.Logger.WarnContext(ctx, "new seed, dropped the state of the previous run",
			"seed", room.SeedName)
	}
	return c.send(ctx, conn, connectMessage{
		Cmd:           "Connect",
		Password:      c.opts.Password,
		Game:          gamedata.GameName,
		Name:          c.opts.SlotName,
		UUID:          c.uuid,
		Version:       version,
		ItemsHandling: itemsHandling,
		Tags:          []string{},
		SlotData:      true,
	})
}

// onConnected records the seed's shape. Nothing is sent from here: the pump
// wakes on ready and reports everything the run holds.
func (c *Client) onConnected(message json.RawMessage, ready chan struct{}) error {
	var payload connected
	if err := json.Unmarshal(message, &payload); err != nil {
		return err
	}
	var slot SlotData
	if err := json.Unmarshal(payload.SlotData, &slot); err != nil {
		return permanentError{fmt.Errorf("unreadable slot data: %w", err)}
	}
	if err := slot.validate(); err != nil {
		return permanentError{err}
	}
	if slot.DeathLink {
		c.opts.Logger.Warn("the seed asks for DeathLink, which this bridge does not implement")
	}

	c.mu.Lock()
	c.connected = true
	c.slot = slot
	c.lastError = ""
	c.mu.Unlock()

	c.opts.Logger.Info("connected to archipelago",
		"slot", c.opts.SlotName,
		"missions", len(slot.Missions),
		"goal", slot.Goal,
		"server_checks", len(payload.CheckedLocations),
	)
	select {
	case <-ready:
	default:
		close(ready)
	}
	return nil
}

func (c *Client) onConnectionRefused(message json.RawMessage) error {
	var refused connectionRefused
	if err := json.Unmarshal(message, &refused); err != nil {
		return err
	}
	return permanentError{fmt.Errorf("archipelago refused the connection: %v", refused.Errors)}
}

func (c *Client) onReceivedItems(ctx context.Context, conn *websocket.Conn, message json.RawMessage) error {
	var payload receivedItems
	if err := json.Unmarshal(message, &payload); err != nil {
		return err
	}
	ids := make([]int64, len(payload.Items))
	for i, item := range payload.Items {
		ids[i] = item.Item
	}
	err := c.opts.Store.ApplyItems(payload.Index, ids)
	if errors.Is(err, state.ErrDesync) {
		c.opts.Logger.WarnContext(ctx, "item list diverged, asking for a resend",
			"index", payload.Index)
		return c.send(ctx, conn, syncMessage{Cmd: "Sync"})
	}
	return err
}

// report tells the server everything we hold: every check, and the goal if it
// has been met. It runs on connect and on every change afterwards. Sending the
// whole set each time is what makes a reconnect mid-wave a non-event.
func (c *Client) report(ctx context.Context, conn *websocket.Conn) error {
	checks := c.opts.Store.Checks()
	if len(checks) > 0 {
		if err := c.send(ctx, conn, locationChecksMessage{
			Cmd:       "LocationChecks",
			Locations: checks,
		}); err != nil {
			return err
		}
	}

	c.mu.Lock()
	slot := c.slot
	c.mu.Unlock()
	if c.opts.Store.GoalSent() || !slot.goalReached(checks) {
		return nil
	}
	if err := c.send(ctx, conn, statusUpdateMessage{Cmd: "StatusUpdate", Status: statusGoal}); err != nil {
		return err
	}
	c.opts.Logger.InfoContext(ctx, "goal reached, told archipelago", "goal", slot.Goal)
	return c.opts.Store.MarkGoalSent()
}

// pump pushes state upstream and keeps the connection honest. It waits for the
// handshake first: a LocationChecks sent before Connected is dropped.
//
// The watch channel is taken before reporting, not after. A check recorded
// while the report is in flight would otherwise close a channel nobody was
// holding, and sit there until the next unrelated change.
func (c *Client) pump(ctx context.Context, conn *websocket.Conn, ready chan struct{}) error {
	select {
	case <-ready:
	case <-ctx.Done():
		return nil
	}

	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	for {
		changed := c.opts.Store.Watch()
		if err := c.report(ctx, conn); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-changed:
		case <-ping.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return err
			}
		}
	}
}

// Say passes a line to the multiworld. Anything starting with ! is a server
// command there, which is how a player runs !hint or !status from inside the
// game.
func (c *Client) Say(ctx context.Context, text string) error {
	c.mu.Lock()
	conn, connected := c.conn, c.connected
	c.mu.Unlock()
	if !connected || conn == nil {
		return ErrNotConnected
	}
	return c.send(ctx, conn, sayMessage{Cmd: "Say", Text: text})
}

func (c *Client) send(ctx context.Context, conn *websocket.Conn, messages ...any) error {
	body, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return conn.Write(writeCtx, websocket.MessageText, body)
}

func (c *Client) setDisconnected(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	if err != nil {
		c.lastError = err.Error()
	}
}

// randomUUID identifies this bridge process to Archipelago. It is not
// persisted: the server uses it to recognise a reconnect, and a fresh one
// after a restart costs nothing.
func randomUUID() string {
	return rand.Text()
}
