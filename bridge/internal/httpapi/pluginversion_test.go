package httpapi

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// The plugin is the one component that cannot be run here, so the wire contract
// is checked by reading its source as text. gamedata does the same for the keys
// that cross the barrier; this covers the version those keys are spelled in.
//
// The failure it prevents is a bridge and a plugin from different releases in
// one compose file, agreeing on nothing and saying so only mid-wave.
const bridgeSource = "../../../plugin/scripting/tf2_archipelago/bridge.inc"

func TestThePluginSpeaksThisAPIVersion(t *testing.T) {
	body, err := os.ReadFile(bridgeSource)
	if err != nil {
		t.Fatal(err)
	}
	found := regexp.MustCompile(`#define BridgeAPIVersion\s+(\d+)`).FindSubmatch(body)
	if found == nil {
		t.Fatalf("%s does not define BridgeAPIVersion", bridgeSource)
	}
	spoken, err := strconv.Atoi(string(found[1]))
	if err != nil {
		t.Fatal(err)
	}
	if spoken != APIVersion {
		t.Fatalf("the plugin speaks API version %d, this bridge serves %d", spoken, APIVersion)
	}
}

func TestThePluginCallsEveryRouteThisBridgeServes(t *testing.T) {
	body, err := os.ReadFile(bridgeSource)
	if err != nil {
		t.Fatal(err)
	}
	called := string(body)
	for _, route := range []string{"/objective", "/unlocks", "/grants", "/grants/ack", "/messages", "/say", "/healthz"} {
		if !regexp.MustCompile(regexp.QuoteMeta(`"` + route)).MatchString(called) {
			t.Errorf("the plugin never calls %s", route)
		}
	}
}
