package webapi

import (
	"slices"

	"github.com/m-this/tf2-archipelago/launcher/internal/botcards"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// A reward belongs to the room that issued it, not to the player's launcher.
// Clearing only recognized card seats preserves manually configured bots.
func forgetOldRoomCards(before settings.Settings, next *settings.Settings) {
	if before.APHost == next.APHost && before.APPort == next.APPort &&
		before.APTls == next.APTls && before.APSlotName == next.APSlotName &&
		before.APRoomURL == next.APRoomURL && (!before.TestMode || next.TestMode) {
		return
	}
	next.SrcdsBotCardRolls = nil
	next.SrcdsBotCardForms = nil
	next.SrcdsBotTeamComp = slices.Clone(next.SrcdsBotTeamComp)
	next.SrcdsBotSeatNames = slices.Clone(next.SrcdsBotSeatNames)
	next.SrcdsBotSeatLoadouts = slices.Clone(next.SrcdsBotSeatLoadouts)
	for index, name := range next.SrcdsBotSeatNames {
		if _, found := botcards.BySeat(seatValue(next.SrcdsBotTeamComp, index), name); !found {
			continue
		}
		next.SrcdsBotSeatNames[index] = ""
		if index < len(next.SrcdsBotTeamComp) {
			next.SrcdsBotTeamComp[index] = ""
		}
		if index < len(next.SrcdsBotSeatLoadouts) {
			next.SrcdsBotSeatLoadouts[index] = ""
		}
	}
	next.SrcdsBotSeatNames = trimEmptySeats(next.SrcdsBotSeatNames)
	next.SrcdsBotTeamComp = trimEmptySeats(next.SrcdsBotTeamComp)
	next.SrcdsBotSeatLoadouts = trimEmptySeats(next.SrcdsBotSeatLoadouts)
}

func seatValue(values []string, index int) string {
	if index < len(values) {
		return values[index]
	}
	return ""
}

func trimEmptySeats(values []string) []string {
	for len(values) > 0 && values[len(values)-1] == "" {
		values = values[:len(values)-1]
	}
	return values
}
