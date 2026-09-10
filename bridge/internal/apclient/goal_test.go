package apclient

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// The old message named two numbers and no action, and two players a week
// apart asked the same question about it in Discord.
func TestAFormatMismatchSaysWhatToDoAboutIt(t *testing.T) {
	err := SlotData{FormatVersion: gamedata.FormatVersion - 1}.validate()
	if err == nil {
		t.Fatal("a seed from another apworld was accepted")
	}
	for _, want := range []string{"apworld", "Regenerate"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal never says %q: %v", want, err)
		}
	}
}
