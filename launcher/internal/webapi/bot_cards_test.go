package webapi

import (
	"reflect"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func TestForgetOldRoomCardsPreservesManualSeats(t *testing.T) {
	before := settings.Defaults()
	before.APHost, before.APPort, before.APSlotName = "old.example", 38281, "team"
	before.SrcdsBotTeamComp = []string{"scout", "spy", "medic"}
	before.SrcdsBotSeatNames = []string{"HandPicked", "Mentlegen", "Herr Doktor"}
	before.SrcdsBotSeatLoadouts = []string{"stock", "diamondback", "kritz"}
	before.SrcdsBotCardRolls = map[string]string{"mentlegen": "Bot: Mentlegen | Legendary | Robot"}
	before.SrcdsBotGiantCards = []string{"mentlegen"}
	next := before
	next.APPort++
	forgetOldRoomCards(before, &next)
	if !reflect.DeepEqual(next.SrcdsBotSeatNames, []string{"HandPicked"}) ||
		!reflect.DeepEqual(next.SrcdsBotTeamComp, []string{"scout"}) ||
		!reflect.DeepEqual(next.SrcdsBotSeatLoadouts, []string{"stock"}) {
		t.Fatalf("new-room seats = %v / %v / %v", next.SrcdsBotTeamComp, next.SrcdsBotSeatNames, next.SrcdsBotSeatLoadouts)
	}
	if next.SrcdsBotCardRolls != nil || next.SrcdsBotGiantCards != nil {
		t.Fatalf("new room retained card rewards: %+v", next)
	}
	if before.SrcdsBotSeatNames[1] != "Mentlegen" {
		t.Fatal("the saved before-state was mutated")
	}
}

func TestForgetOldRoomCardsKeepsLineupAcrossMaps(t *testing.T) {
	before := settings.Defaults()
	before.SrcdsBotSeatNames = []string{"Mentlegen"}
	before.SrcdsBotTeamComp = []string{"spy"}
	before.SrcdsBotCardRolls = map[string]string{"mentlegen": "Bot: Mentlegen | Legendary | Robot"}
	next := before
	forgetOldRoomCards(before, &next)
	if !reflect.DeepEqual(next, before) {
		t.Fatal("same-room lineup changed")
	}
}
