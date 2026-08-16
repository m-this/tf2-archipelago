package apclient

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	versionsFile = "../../../deploy/env/versions.env"
	manifestFile = "../../../apworld/tf2_mvm/archipelago.json"
)

// pinnedArchipelagoVersion is the one this project builds and runs against.
// deploy/env/versions.env says nothing else may hardcode a version; these tests
// are what makes that true for the two places that must.
func pinnedArchipelagoVersion(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile(versionsFile)
	if err != nil {
		t.Fatal(err)
	}
	for line := range strings.Lines(string(body)) {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if found && key == "ARCHIPELAGO_VERSION" {
			return value
		}
	}
	t.Fatalf("%s has no ARCHIPELAGO_VERSION", versionsFile)
	return ""
}

func TestTheHandshakeAnnouncesThePinnedVersion(t *testing.T) {
	pinned := pinnedArchipelagoVersion(t)
	announced := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Build)

	// Archipelago compares this against its own version at Connect. A bridge
	// that claims a version the server was not built for is refused, or worse,
	// accepted while speaking a protocol that has moved.
	if announced != pinned {
		t.Errorf("the handshake announces %s, the project pins %s", announced, pinned)
	}
}

func TestTheApworldAsksForThePinnedVersion(t *testing.T) {
	body, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		MinimumAPVersion string `json:"minimum_ap_version"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	if pinned := pinnedArchipelagoVersion(t); manifest.MinimumAPVersion != pinned {
		t.Errorf("the manifest asks for %s, the project pins %s", manifest.MinimumAPVersion, pinned)
	}
}
