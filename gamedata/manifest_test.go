package gamedata

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

// manifestFile is the apworld's own manifest, in Archipelago's format. It is
// hand-written because Archipelago owns its shape, so the one fact it shares
// with this package is checked rather than trusted.
const manifestFile = "../apworld/tf2_mvm/archipelago.json"

// pluginFile declares the same release version. Neither file can read the
// other: one is JSON for Archipelago, the other a #define spcomp needs as a
// literal.
const pluginFile = "../plugin/scripting/tf2_archipelago.sp"

func TestTheManifestNamesTheSameGame(t *testing.T) {
	body, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Game string `json:"game"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	// A slot connects by game name. A manifest that disagrees with this package
	// generates a seed the bridge cannot join, and nothing says why.
	if manifest.Game != GameName {
		t.Errorf("%s calls the game %q, this package calls it %q",
			manifestFile, manifest.Game, GameName)
	}
}

var pluginVersion = regexp.MustCompile(`#define PLUGIN_VERSION "([^"]+)"`)

// The plugin reports its version to the bridge and in `sm_ap_status`. A version
// left behind at the last release names a build nobody shipped.
func TestTheManifestAndThePluginNameTheSameVersion(t *testing.T) {
	body, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		WorldVersion string `json:"world_version"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}

	source, err := os.ReadFile(pluginFile)
	if err != nil {
		t.Fatal(err)
	}
	found := pluginVersion.FindSubmatch(source)
	if found == nil {
		t.Fatalf("%s declares no PLUGIN_VERSION", pluginFile)
	}

	if plugin := string(found[1]); plugin != manifest.WorldVersion {
		t.Errorf("%s is at %q, %s is at %q",
			pluginFile, plugin, manifestFile, manifest.WorldVersion)
	}
}
