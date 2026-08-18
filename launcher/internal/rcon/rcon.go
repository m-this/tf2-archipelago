// Package rcon is a Source RCON client: the protocol the game server speaks on
// its own port for admin commands. It is the Go port of deploy/rcon.py, and it
// is what the launcher's command box sends through.
package rcon

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	typeResponse    int32 = 0
	typeExecCommand int32 = 2
	typeAuth        int32 = 3

	// 4096 is Valve's limit. The count cap is ours: an endless reply is a bug,
	// not a long answer.
	packetBytesMax = 4096
	packetCountMax = 128

	// The empty command sent behind every real one. The server answers in
	// order, so its reply marks the end of the real one.
	terminatorOffset = 1000

	authRequestID int32 = 1
	timeout             = 10 * time.Second
)

// ErrBadPassword is the server refusing the password. It reads the password at
// boot, so a value changed since then needs a restart.
var ErrBadPassword = errors.New("the server refused the RCON password")

// Client is one authenticated connection. Not safe for concurrent use: the
// protocol matches replies to requests by order on the wire.
type Client struct {
	conn      net.Conn
	requestID int32
}

// Dial connects and authenticates.
func Dial(address, password string) (*Client, error) {
	return DialContext(context.Background(), address, password)
}

// DialContext connects and authenticates, and gives up when ctx is done.
func DialContext(ctx context.Context, address, password string) (*Client, error) {
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("cannot reach %s: %w", address, err)
	}
	client := &Client{conn: conn, requestID: authRequestID}
	if err := client.authenticate(password); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

// Close drops the connection.
func (c *Client) Close() error { return c.conn.Close() }

// Exec runs one command and returns what the server printed.
func (c *Client) Exec(command string) (string, error) {
	if err := c.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}
	c.requestID++
	id := c.requestID
	terminator := id + terminatorOffset
	if err := c.send(id, typeExecCommand, command); err != nil {
		return "", err
	}
	if err := c.send(terminator, typeExecCommand, ""); err != nil {
		return "", err
	}

	var reply strings.Builder
	for range packetCountMax {
		answeredID, _, body, err := c.read()
		if err != nil {
			return "", err
		}
		if answeredID == terminator {
			return strings.TrimSpace(reply.String()), nil
		}
		reply.WriteString(body)
	}
	return "", fmt.Errorf("the server never finished answering %q", command)
}

func (c *Client) authenticate(password string) error {
	if err := c.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	if err := c.send(authRequestID, typeAuth, password); err != nil {
		return err
	}
	for range packetCountMax {
		id, kind, _, err := c.read()
		if err != nil {
			return err
		}
		// The server sends an empty value packet before the verdict.
		if kind == typeResponse {
			continue
		}
		if id == authRequestID {
			return nil
		}
		return ErrBadPassword
	}
	return errors.New("the server never answered the authentication")
}

func (c *Client) send(id, kind int32, body string) error {
	payload := new(bytes.Buffer)
	_ = binary.Write(payload, binary.LittleEndian, id)
	_ = binary.Write(payload, binary.LittleEndian, kind)
	payload.WriteString(body)
	payload.Write([]byte{0, 0})

	packet := new(bytes.Buffer)
	if err := binary.Write(packet, binary.LittleEndian, int32(payload.Len())); err != nil {
		return err
	}
	packet.Write(payload.Bytes())
	if _, err := c.conn.Write(packet.Bytes()); err != nil {
		return fmt.Errorf("cannot send the command: %w", err)
	}
	return nil
}

func (c *Client) read() (id, kind int32, body string, err error) {
	var size int32
	if err := binary.Read(c.conn, binary.LittleEndian, &size); err != nil {
		return 0, 0, "", fmt.Errorf("cannot read the reply: %w", err)
	}
	if size < 10 || size > packetBytesMax {
		return 0, 0, "", fmt.Errorf("the server sent a %d byte packet", size)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return 0, 0, "", fmt.Errorf("cannot read the reply: %w", err)
	}
	id = int32(binary.LittleEndian.Uint32(payload[0:4]))
	kind = int32(binary.LittleEndian.Uint32(payload[4:8]))
	return id, kind, string(bytes.TrimRight(payload[8:], "\x00")), nil
}
