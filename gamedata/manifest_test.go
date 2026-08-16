package gamedata

import (
	"encoding/json"
	"os"
	"testing"
)

// manifestFile is the apworld's own manifest, in Archipelago's format. It is
// hand-written because Archipelago owns its shape, so the one fact it shares
// with this package is checked rather than trusted.
const manifestFile = "../apworld/tf2_mvm/archipelago.json"

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
