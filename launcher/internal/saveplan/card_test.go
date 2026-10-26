package saveplan

import (
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/botcards"
	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// A card pins a name on its seat, and botlive sends that name to the running
// server. Counting the name as read at start told a player who applied a card
// that the server still played the old team and needed a restart (apw-fjv8).
func TestChoosingACardIsATeamChangeAndNotARestart(t *testing.T) {
	before := settings.Defaults()
	next, err := form.Apply(form.NewState(before), form.Env{},
		form.Change{Field: "bots.seat.0.card", Value: botcards.Cards[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Settings.SrcdsBotSeatNames) == 0 {
		t.Fatal("the card pinned no name, so this tests nothing")
	}
	if plan := For(before, next.Settings); !plan.Team || plan.Restart {
		t.Fatalf("applying a card planned %+v, want a live team change only", plan)
	}
}
