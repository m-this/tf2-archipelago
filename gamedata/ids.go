package gamedata

import "fmt"

// BaseID is where this project's Archipelago id space starts. Ids are global
// across a multiworld, so this was picked once at random to sit where no other
// apworld does. Moving it renumbers everything and breaks every seed already
// generated.
const BaseID int64 = 7_442_000_000

const (
	// WavesMax is the longest mission this id scheme holds. Each mission gets a
	// block of 100, and the slots above the waves are named objectives: 90 is
	// the tank, 91 the giant, 99 the mission clear. Valve's longest mission is
	// 8 waves, so the ceiling is a bound on the id scheme, not on the game.
	WavesMax uint8 = 89

	// MissionIDMax keeps the location space below the item space.
	MissionIDMax MissionID = MissionID(itemSpaceOffset/locationsPerMission - 1)

	locationsPerMission int64 = 100
	locationSlotTank    int64 = 90
	locationSlotGiant   int64 = 91
	locationSlotClear   int64 = 99

	// itemSpaceOffset keeps item ids clear of location ids, which Archipelago
	// namespaces separately and would let overlap.
	itemSpaceOffset int64 = 1_000_000

	itemBlockTicket     int64 = 1_000
	itemBlockClass      int64 = 2_000
	itemBlockWeaponSlot int64 = 3_000
	itemBlockCredits    int64 = 4_000
	itemBlockWeaponBuff int64 = 5_000
)

// Location ids: base + mission*100 + wave, or + 99 for the mission clear.

// WaveLocationID is the id of the check for clearing wave n of this mission.
func (m Mission) WaveLocationID(wave uint8) int64 {
	if wave < 1 || wave > m.Waves {
		panic(fmt.Sprintf("gamedata: wave %d out of range for %s (%d waves)", wave, m.PopFile, m.Waves))
	}
	return BaseID + int64(m.ID)*locationsPerMission + int64(wave)
}

// TankLocationID is the id of the check for destroying a tank in this mission.
// Only missions with HasTank have one.
func (m Mission) TankLocationID() int64 {
	return BaseID + int64(m.ID)*locationsPerMission + locationSlotTank
}

// TankLocationName is what the spoiler log calls that check.
func (m Mission) TankLocationName() string {
	return m.Name + " Tank"
}

// GiantLocationID is the id of the check for killing a giant in this mission.
// Only missions with HasGiant have one.
func (m Mission) GiantLocationID() int64 {
	return BaseID + int64(m.ID)*locationsPerMission + locationSlotGiant
}

// GiantLocationName is what the spoiler log calls that check.
func (m Mission) GiantLocationName() string {
	return m.Name + " Giant"
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

// Item ids: one block per kind, keyed by entity id, so an item id is as
// append-only as the entity behind it.

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
