package rcon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
)

// fakeServer speaks enough of the protocol to answer one client: it accepts
// the password it was given, refuses any other, and echoes each command back.
func fakeServer(t *testing.T, password string) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for {
			id, kind, body, err := readPacket(conn)
			if err != nil {
				return
			}
			switch kind {
			case typeAuth:
				writePacket(conn, 0, typeResponse, "")
				if body != password {
					writePacket(conn, -1, 2, "")
					return
				}
				writePacket(conn, id, 2, "")
			case typeExecCommand:
				if body == "" {
					writePacket(conn, id, typeResponse, "")
					continue
				}
				writePacket(conn, id, typeResponse, "you said "+body)
			}
		}
	}()
	return listener.Addr().String()
}

func writePacket(w io.Writer, id, kind int32, body string) {
	payload := new(bytes.Buffer)
	_ = binary.Write(payload, binary.LittleEndian, id)
	_ = binary.Write(payload, binary.LittleEndian, kind)
	payload.WriteString(body)
	payload.Write([]byte{0, 0})
	_ = binary.Write(w, binary.LittleEndian, int32(payload.Len()))
	_, _ = w.Write(payload.Bytes())
}

func readPacket(r io.Reader) (id, kind int32, body string, err error) {
	var size int32
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return 0, 0, "", err
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, 0, "", err
	}
	return int32(binary.LittleEndian.Uint32(payload[0:4])),
		int32(binary.LittleEndian.Uint32(payload[4:8])),
		string(bytes.TrimRight(payload[8:], "\x00")), nil
}

func TestExec(t *testing.T) {
	client, err := Dial(fakeServer(t, "secret"), "secret")
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer func() { _ = client.Close() }()

	for _, command := range []string{"sm_ap_status", "changelevel mvm_decoy"} {
		reply, err := client.Exec(command)
		if err != nil {
			t.Fatalf("Exec(%q): %v", command, err)
		}
		if want := "you said " + command; reply != want {
			t.Errorf("Exec(%q) = %q, want %q", command, reply, want)
		}
	}
}

func TestDialRejectsABadPassword(t *testing.T) {
	_, err := Dial(fakeServer(t, "secret"), "wrong")
	if !errors.Is(err, ErrBadPassword) {
		t.Fatalf("got %v, want ErrBadPassword", err)
	}
}

func TestDialReportsAServerThatIsNotThere(t *testing.T) {
	// Port 1 on loopback: nothing listens, and the refusal is immediate.
	_, err := Dial("127.0.0.1:1", "secret")
	if err == nil {
		t.Fatal("dialling a closed port succeeded")
	}
	if !strings.Contains(err.Error(), "cannot reach") {
		t.Errorf("unhelpful error: %v", err)
	}
}
