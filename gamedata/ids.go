package gamedata

import "fmt"

// BaseID is where this project's Archipelago id space starts. Ids are global
// across every game in a multiworld, so this number was picked once, at
// random, to sit where no other apworld is likely to sit. Moving it renumbers
// every location and every item, which silently breaks every seed already
// generated. It never moves.
const BaseID int64 = 7_442_000_000

const (
	// WavesMax is the longest mission this id scheme can hold. Wave 99 is the
	// mission-clear slot and each mission gets a block of 100, so a mission
	// with 99 waves would collide with its own completion.
	WavesMax uint8 = 98

	// MissionIDMax keeps the location space below the item space.
	MissionIDMax MissionID = MissionID(itemSpaceOffset/locationsPerMission - 1)

	locationsPerMission int64 = 100
	locationSlotClear   int64 = 99

	// itemSpaceOffset separates items from locations. Archipelago keeps the
	// two in separate namespaces and would tolerate an overlap, but an id that
	// means one thing and one thing only is worth the arithmetic when reading
	// a log.
	itemSpaceOffset int64 = 1_000_000

	itemBlockTicket     int64 = 1_000
	itemBlockClass      int64 = 2_000
	itemBlockWeaponSlot int64 = 3_000
	itemBlockCredits    int64 = 4_000
)

// Location ids. base + mission*100 + wave for a wave clear, + 99 for the
// mission clear.

// WaveLocationID is the id of the check for clearing wave n of this mission.
func (m Mission) WaveLocationID(wave uint8) int64 {
	if wave < 1 || wave > m.Waves {
		panic(fmt.Sprintf("gamedata: wave %d out of range for %s (%d waves)", wave, m.PopFile, m.Waves))
	}
	return BaseID + int64(m.ID)*locationsPerMission + int64(wave)
}

// ClearLocationID is the id of the check for clearing the whole mission.
func (m Mission) ClearLocationID() int64 {
	return BaseID + int64(m.ID)*locationsPerMission + locationSlotClear
}

// WaveLocationName is what the spoiler log calls that wave.
func (m Mission) WaveLocationName(wave uint8) string {
	return fmt.Sprintf("%s Wave %d", m.Name, wave)
}

// ClearLocationName is what the spoiler log calls the mission clear.
func (m Mission) ClearLocationName() string {
	return m.Name + " Complete"
}

// Item ids. One block per item kind, each entry keyed by the entity id it
// belongs to, so an item id is as append-only as the entity behind it.

// TicketItemID is the id of the item that unlocks this mission.
func (m Mission) TicketItemID() int64 {
	return BaseID + itemSpaceOffset + itemBlockTicket + int64(m.ID)
}

// TicketItemName is what the multiworld calls that item.
func (m Mission) TicketItemName() string {
	return "Mission Ticket: " + m.Name
}

// ItemID is the id of the item that unlocks this class.
func (c Class) ItemID() int64 {
	return BaseID + itemSpaceOffset + itemBlockClass + int64(c.ID)
}

// ItemName is what the multiworld calls that item.
func (c Class) ItemName() string {
	return "Class: " + c.Name
}
