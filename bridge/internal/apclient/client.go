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

	"github.com/m-this/tf2-archipelago/bridge/internal/chat"
	"github.com/m-this/tf2-archipelago/bridge/internal/deathlink"
	"github.com/m-this/tf2-archipelago/bridge/internal/state"
	"github.com/m-this/tf2-archipelago/gamedata"
)

const (
	// readLimitBytes: RoomInfo carries a checksum per game, past the 32 KiB default.
	readLimitBytes = 4 << 20

	dialTimeout  = 30 * time.Second
	writeTimeout = 10 * time.Second
	pingEvery    = 30 * time.Second

	backoffFirst = time.Second
	backoffMax   = 30 * time.Second
)

// version is the Archipelago release this client is written against.
var version = Version{Class: "Version", Major: 0, Minor: 6, Build: 7}

// permanentError wraps a failure reconnecting cannot fix; the bridge stops rather than retry.
type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// ErrNotConnected is what a chat line gets when there is no multiworld to say
// it to. Unlike a check it is refused rather than queued: a line that lands ten
// minutes late is worse than one that never landed.
var ErrNotConnected = errors.New("not connected to archipelago")

// Options is everything the session needs.
type Options struct {
	URL      string
	SlotName string
	Password string
	Store    *state.Store
	Chat     *chat.Log
	Deaths   *deathlink.Feed
	Logger   *slog.Logger
}

// Client is one Archipelago session, reconnected as needed.
type Client struct {
	opts Options
	uuid string

	// said and died bound what the game server can pour into the multiworld.
	said *bucket
	died *deaths

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
	DeathLink bool     `json:"death_link"`
	LastError string   `json:"last_error,omitempty"`
}

func New(opts Options) *Client {
	if opts.Chat == nil {
		// A session with nowhere to put chat still has to run.
		opts.Chat = chat.New(1)
	}
	if opts.Deaths == nil {
		opts.Deaths = deathlink.New(1)
	}
	return &Client{opts: opts, uuid: randomUUID(), said: newBucket(time.Now), died: &deaths{now: time.Now}}
}

// Health reports the session state. The plugin uses it to tell a player the
// difference between a check that did not register and one that is queued.
func (c *Client) Health() Health {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Health{
		Connected: c.connected,
		Slot:      c.opts.SlotName,
		Missions:  c.slot.Missions,
		DeathLink: c.slot.DeathLink,
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
			return nil //nolint:nilerr // a cancelled context is a clean stop
		}
		if _, unfixable := errors.AsType[permanentError](err); unfixable {
			return err
		}
		c.setDisconnected(err)

		// A session that stayed up was not a bad address, so the next attempt starts short.
		if time.Since(start) > backoffMax {
			backoff = backoffFirst
		}
		c.opts.Logger.WarnContext(ctx, "archipelago session ended, will retry",
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

// readLoop reads messages until the connection dies, starting with the RoomInfo/Connect handshake.
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
			return fmt.Errorf("cannot read the packet: %w", err)
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
		return fmt.Errorf("packet has no cmd: %w", err)
	}
	switch head.Cmd {
	case "RoomInfo":
		return c.onRoomInfo(ctx, conn, message)
	case "Connected":
		return c.onConnected(ctx, conn, message, ready)
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
		return c.onBounced(ctx, message)
	default:
		return nil
	}
}

func (c *Client) onRoomInfo(ctx context.Context, conn *websocket.Conn, message json.RawMessage) error {
	var room roomInfo
	if err := json.Unmarshal(message, &room); err != nil {
		return err
	}
	archive, err := c.opts.Store.BindSeed(room.SeedName)
	if err != nil {
		return err
	}
	if archive != "" {
		c.opts.Logger.WarnContext(ctx, "new seed, set the previous run aside",
			"seed", room.SeedName, "archive", archive)
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

// onConnected records the seed's shape. The pump wakes on ready and reports;
// the only thing sent from here is the DeathLink tag, which cannot go on
// Connect because the slot data that decides it is what Connected carries.
func (c *Client) onConnected(
	ctx context.Context, conn *websocket.Conn, message json.RawMessage, ready chan struct{},
) error {
	var payload connected
	if err := json.Unmarshal(message, &payload); err != nil {
		return err
	}
	var slot SlotData
	if err := json.Unmarshal(payload.SlotData, &slot); err != nil {
		return permanentError{fmt.Errorf("cannot read the slot data: %w", err)}
	}
	if err := slot.validate(); err != nil {
		return permanentError{err}
	}
	// The server holds the same check list for this slot, and the seed is
	// already bound, so this is the run coming back after a state file was lost
	// or rolled back. It happens before the handshake is announced ready, so
	// the first report upstream carries the whole set.
	//
	// A failure here does not end the session. The server still holds every one
	// of these checks and will offer them again on the next connect, whereas
	// returning the error reconnects into the same failing write for as long as
	// the disk stays full.
	adopted, err := c.opts.Store.AdoptChecks(payload.CheckedLocations)
	switch {
	case err != nil:
		c.opts.Logger.ErrorContext(ctx, "cannot hold the checks the server already had",
			"server_checks", len(payload.CheckedLocations), "error", err)
	case adopted > 0:
		c.opts.Logger.WarnContext(ctx, "the server knew checks this bridge did not, and they are held now",
			"adopted", adopted, "server_checks", len(payload.CheckedLocations))
	}

	c.mu.Lock()
	c.connected = true
	c.slot = slot
	c.lastError = ""
	c.mu.Unlock()

	c.opts.Logger.InfoContext(ctx, "connected to archipelago",
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
	if slot.DeathLink {
		return c.claimDeathLink(ctx, conn)
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
		c.opts.Logger.WarnContext(ctx, "item list diverged, asked for a resend",
			"index", payload.Index)
		return c.send(ctx, conn, syncMessage{Cmd: "Sync"})
	}
	return err
}

// report sends every check we hold, plus the goal once met. The whole set goes
// every time, which is what makes a reconnect mid-wave a non-event.
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

// pump pushes state upstream and pings. It waits for the handshake first: a
// LocationChecks sent before Connected is dropped.
//
// The watch channel is taken before reporting, not after: a check recorded
// while the report is in flight would otherwise sit until the next change.
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
// command there, which is how a player runs !hint from inside the game, and is
// also why the line is checked against a policy before it goes anywhere.
func (c *Client) Say(ctx context.Context, text string) error {
	if err := checkSayable(text); err != nil {
		return err
	}
	if !c.said.take() {
		return ErrSaidTooMuch
	}
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

// randomUUID identifies this process to Archipelago; a fresh one after a restart costs nothing.
func randomUUID() string {
	return rand.Text()
}
