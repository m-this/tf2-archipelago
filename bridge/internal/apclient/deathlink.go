package apclient

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// deathLinkTag is the tag a client holds to take part. The server delivers a
// bounce only to clients holding one of its tags, so a slot that did not ask
// for DeathLink never hears one.
const deathLinkTag = "DeathLink"

// deathIntervalMin is how often the game server may kill the multiworld. A
// wave is lost once, and two reports of it seconds apart are the same loss.
const deathIntervalMin = 5 * time.Second

// ErrDeathLinkOff is a death reported by a slot whose seed did not ask for DeathLink.
var ErrDeathLinkOff = errors.New("this seed does not have DeathLink on")

// ErrDiedTooMuch is a second death inside deathIntervalMin.
var ErrDiedTooMuch = errors.New("the last death was a moment ago")

// deaths guards the outbound side.
type deaths struct {
	mu   sync.Mutex
	last time.Time
	now  func() time.Time
}

func (d *deaths) take() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	at := d.now()
	if !d.last.IsZero() && at.Sub(d.last) < deathIntervalMin {
		return false
	}
	d.last = at
	return true
}

// claimDeathLink tells the server this slot takes part, once the seed has said
// it does. It runs after Connected: the tag cannot go on Connect, because the
// slot data that decides it is what Connected carries.
func (c *Client) claimDeathLink(ctx context.Context, conn *websocket.Conn) error {
	c.opts.Logger.InfoContext(ctx, "the seed asks for DeathLink: a lost wave kills the multiworld, and a death there fails the wave")
	return c.send(ctx, conn, connectUpdateMessage{
		Cmd:           "ConnectUpdate",
		ItemsHandling: itemsHandling,
		Tags:          []string{deathLinkTag},
	})
}

// onBounced hands a DeathLink from somewhere else to the plugin. The server
// echoes our own bounces back, so those are dropped by name.
func (c *Client) onBounced(ctx context.Context, message []byte) error {
	var payload bounced
	if err := json.Unmarshal(message, &payload); err != nil {
		return err
	}
	if !slices.Contains(payload.Tags, deathLinkTag) {
		return nil
	}
	c.mu.Lock()
	wanted := c.slot.DeathLink
	c.mu.Unlock()
	if !wanted {
		return nil
	}
	var death deathLinkData
	if err := json.Unmarshal(payload.Data, &death); err != nil {
		c.opts.Logger.WarnContext(ctx, "a DeathLink arrived that this bridge cannot read", "error", err)
		return nil
	}
	if death.Source == c.opts.SlotName {
		return nil
	}
	c.opts.Logger.InfoContext(ctx, "death link received", "source", death.Source, "cause", death.Cause)
	c.opts.Deaths.Append(death.Source, death.Cause)
	return nil
}

// Die tells the multiworld this slot died. Like a chat line and unlike a check,
// it is refused rather than queued: a death delivered after the reconnect is a
// different death.
func (c *Client) Die(ctx context.Context, cause string) error {
	c.mu.Lock()
	conn, connected, wanted := c.conn, c.connected, c.slot.DeathLink
	c.mu.Unlock()
	if !connected || conn == nil {
		return ErrNotConnected
	}
	if !wanted {
		return ErrDeathLinkOff
	}
	if !c.died.take() {
		return ErrDiedTooMuch
	}
	c.opts.Logger.InfoContext(ctx, "death link sent", "cause", cause)
	return c.send(ctx, conn, bounceMessage{
		Cmd:  "Bounce",
		Tags: []string{deathLinkTag},
		Data: deathLinkData{
			Time:   float64(c.died.now().UnixMilli()) / 1000,
			Source: c.opts.SlotName,
			Cause:  cause,
		},
	})
}
