package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// staleBridgeWait bounds the wait for a stopped bridge to let go of its port.
const staleBridgeWait = 5 * time.Second

/*
replaceStaleBridge stops a bridge that is already answering where this one is
about to listen.

This process starts its bridge only after this runs, so anything answering is
another launcher's: usually the previous version, still running after its tab
was closed. The new bridge then fails to bind while the game server it started
beside it loads the new plugin and talks to the old bridge, which speaks an
older API (apw-glb). Something on the port that is not a bridge is left alone.
*/
func replaceStaleBridge(listen string, say func(string)) error {
	version, ok := bridgeAt(listen)
	if !ok {
		return nil
	}
	_, portText, err := net.SplitHostPort(listen)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return err
	}
	pid, found := winproc.ListenerPID(port)
	if !found || pid == os.Getpid() {
		return fmt.Errorf("a bridge (API version %d) from another launcher is already on %s: close the other launcher and press Start again", version, listen)
	}
	say(fmt.Sprintf("stopping a bridge left running by another launcher (API version %d, process %d)", version, pid))
	process, err := os.FindProcess(pid)
	if err == nil {
		err = process.Kill()
	}
	if err != nil {
		return fmt.Errorf("cannot stop the other launcher's bridge on %s (process %d): %w", listen, pid, err)
	}
	for deadline := time.Now().Add(staleBridgeWait); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if _, still := bridgeAt(listen); !still {
			return nil
		}
	}
	return fmt.Errorf("the other launcher's bridge on %s did not stop", listen)
}

// bridgeAt reports whether a bridge answers /healthz at listen, and its API
// version.
func bridgeAt(listen string) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listen+"/healthz", nil)
	if err != nil {
		return 0, false
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, false
	}
	defer func() { _ = response.Body.Close() }()
	var health struct {
		APIVersion int `json:"api_version"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&health) != nil || health.APIVersion == 0 {
		return 0, false
	}
	return health.APIVersion, true
}
