package runtime

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

// staleBridgeEnv makes the test binary play an older launcher's bridge.
const staleBridgeEnv = "TF2AP_STALE_BRIDGE_LISTEN"

func TestStaleBridgeHelper(t *testing.T) {
	listen := os.Getenv(staleBridgeEnv)
	if listen == "" {
		t.Skip("only runs as the stale bridge")
	}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"api_version":5}`))
	})
	_ = http.ListenAndServe(listen, nil) //nolint:gosec // a test helper on loopback
}

/*
A bridge another launcher left running is stopped before this one starts.

apw-glb: after an update the previous launcher was still running, and the new
plugin spent a server start talking to its version 5 bridge.
*/
func TestAStaleBridgeIsReplaced(t *testing.T) {
	probe, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listen := probe.Addr().String()
	_ = probe.Close()

	old := exec.Command(os.Args[0], "-test.run=^TestStaleBridgeHelper$")
	old.Env = append(os.Environ(), staleBridgeEnv+"="+listen)
	if err := old.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = old.Wait(); close(exited) }()
	t.Cleanup(func() { _ = old.Process.Kill() })

	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if _, up := bridgeAt(listen); up {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the stale bridge never answered")
		}
	}

	if err := replaceStaleBridge(listen, func(text string) { t.Log(text) }); err != nil {
		t.Fatalf("the stale bridge was not replaced: %v", err)
	}
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("the stale bridge is still running")
	}
	listener, err := net.Listen("tcp4", listen)
	if err != nil {
		t.Fatalf("the port is still held: %v", err)
	}
	_ = listener.Close()
}

// Nothing on the port is nothing to do.
func TestNoBridgeIsNoWork(t *testing.T) {
	probe, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listen := probe.Addr().String()
	_ = probe.Close()
	if err := replaceStaleBridge(listen, func(text string) { t.Log(text) }); err != nil {
		t.Fatal(err)
	}
}
