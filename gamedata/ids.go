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

	// MissionIDMax keeps the location space below the item space, and below
	// the block the milestones take: they are checks with no mission behind
	// them, so they borrow the highest block a mission could have had.
	MissionIDMax       MissionID = milestoneMissionID - 1
	milestoneMissionID MissionID = MissionID(itemSpaceOffset/locationsPerMission - 10)

	locationsPerMission int64 = 100
	locationSlotTank    int64 = 90
	locationSlotGiant   int64 = 91
	locationSlotClear   int64 = 99

	// The victory caches sit between the giant and the clear: slot 92 is the
	// first, and VictoryCachesMax bounds them so 96 to 98 stay free.
	locationSlotCacheFirst int64 = 92
	VictoryCachesMax       uint8 = 4

	// itemSpaceOffset keeps item ids clear of location ids, which Archipelago
	// namespaces separately and would let overlap.
	itemSpaceOffset int64 = 1_000_000

	// The per-kill checks, every giant and every tank of every wave, sit in
	// a block of their own above the items: a mission's block of 100 could
	// never hold the eighty giants of Caliginous Caper's one wave. Ten
	// thousand per mission, a hundred per wave: giants 1 to 90, tanks 91 to
	// 99. WaveKillsMax bound what a wave may hold before these overflow.
	waveKillSpaceOffset int64 = 2_000_000
	waveKillsPerMission int64 = 10_000
	waveKillsPerWave    int64 = 100
	waveKillSlotTank    int64 = 90
	WaveGiantsMax       uint8 = 90
	WaveTanksMax        uint8 = 9
	WaveKillWavesMax    uint8 = 99

	itemBlockTicket     int64 = 1_000
	itemBlockClass      int64 = 2_000
	itemBlockWeaponSlot int64 = 3_000
	itemBlockCredits    int64 = 4_000
	itemBlockWeaponBuff int64 = 5_000
	itemBlockTrap       int64 = 6_000
	itemBlockSetting    int64 = 7_000
	itemBlockTrophy     int64 = 8_000
	itemBlockClassSlot  int64 = 9_000

	// Ten ids per class, so a class's named slots sit together and a slot id
	// picks one. Past the progressive block above, which holds one per class.
	itemBlockClassSlotNamed int64 = 9_100
	itemBlockBotCard        int64 = 11_000
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

// VictoryCacheLocationID is the id of the nth extra check the mission clear
// pays when victory caches are on, n counted from 1. Only missions whose tier
// pays that many have one.
func (m Mission) VictoryCacheLocationID(n uint8) int64 {
	if n < 1 || n > VictoryCachesMax {
		panic(fmt.Sprintf("gamedata: victory cache %d out of range for %s", n, m.PopFile))
	}
	return BaseID + int64(m.ID)*locationsPerMission + locationSlotCacheFirst + int64(n-1)
}

// VictoryCacheLocationName is what the spoiler log calls that check.
func (m Mission) VictoryCacheLocationName(n uint8) string {
	return fmt.Sprintf("%s Victory Cache %d", m.Name, n)
}

// WaveLocationName is what the spoiler log calls that wave.
func (m Mission) WaveLocationName(wave uint8) string {
	return fmt.Sprintf("%s Wave %d", m.Name, wave)
}

// ClearLocationName is what the spoiler log calls the mission clear.
func (m Mission) ClearLocationName() string {
	return m.Name + " Complete"
}

// milestoneID is where a milestone lives: the reserved mission block, a
// hundred wide, with a row of ten per total and the ladder index down it.
func milestoneID(kind MilestoneKind, index uint8) int64 {
	if index < 1 || index > 9 {
		panic(fmt.Sprintf("gamedata: milestone index %d out of range", index))
	}
	return BaseID + int64(milestoneMissionID)*locationsPerMission + int64(kind)*10 + int64(index)
}

// WaveGiantLocationID is the id of the check for the nth giant killed in
// wave w of this mission, n counted from 1.
func (m Mission) WaveGiantLocationID(wave, n uint8) int64 {
	if wave < 1 || wave > WaveKillWavesMax || n < 1 || n > WaveGiantsMax {
		panic(fmt.Sprintf("gamedata: giant %d of wave %d out of range for %s", n, wave, m.PopFile))
	}
	return BaseID + waveKillSpaceOffset + int64(m.ID)*waveKillsPerMission + int64(wave)*waveKillsPerWave + int64(n)
}

// WaveTankLocationID is the id of the check for the nth tank destroyed in
// wave w of this mission, n counted from 1.
func (m Mission) WaveTankLocationID(wave, n uint8) int64 {
	if wave < 1 || wave > WaveKillWavesMax || n < 1 || n > WaveTanksMax {
		panic(fmt.Sprintf("gamedata: tank %d of wave %d out of range for %s", n, wave, m.PopFile))
	}
	return BaseID + waveKillSpaceOffset + int64(m.ID)*waveKillsPerMission + int64(wave)*waveKillsPerWave + waveKillSlotTank + int64(n)
}

// WaveGiantLocationName and WaveTankLocationName are what the spoiler log
// calls those checks.
func (m Mission) WaveGiantLocationName(wave, n uint8) string {
	return fmt.Sprintf("%s Wave %d Giant %d", m.Name, wave, n)
}

func (m Mission) WaveTankLocationName(wave, n uint8) string {
	return fmt.Sprintf("%s Wave %d Tank %d", m.Name, wave, n)
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

// SlotItemID is the id of the progressive item that opens this class's own
// loadout slots, in a block of its own so nothing shipped moves.
func (c Class) SlotItemID() int64 {
	return BaseID + itemSpaceOffset + itemBlockClassSlot + int64(c.ID)
}

// SlotItemName is what the multiworld calls that item.
func (c Class) SlotItemName() string {
	return "Progressive Weapon Slot: " + c.Name
}

/*
NamedSlotItemID is the id of the item that opens one named slot of this class,
for a seed whose slots are found in any order rather than in the class's own.

Its own row inside the class slot block: ten ids per class, the slot's own id
picking one of them. A progressive copy and a named slot are different items
and a seed holds one kind or the other, so neither may take the other's id.
*/
func (c Class) NamedSlotItemID(slot WeaponSlotID) int64 {
	return BaseID + itemSpaceOffset + itemBlockClassSlotNamed + int64(c.ID)*10 + int64(slot)
}

// NamedSlotItemName is what the multiworld calls it. The slot rather than the
// count, because that is the whole difference: finding this one opens the
// Scout's melee whether or not his secondary ever turned up.
func (c Class) NamedSlotItemName(slot WeaponSlot) string {
	return c.Name + " " + slot.Name + " Slot"
}

// ItemID is the id of the item that fires this trap.
func (t Trap) ItemID() int64 {
	return BaseID + itemSpaceOffset + itemBlockTrap + int64(t.ID)
}

// TrophyItemID is the id of the medal locked onto this mission's clear. Keyed
// by mission, like the ticket, so a mission carries its own trophy.
func (m Mission) TrophyItemID() int64 {
	return BaseID + itemSpaceOffset + itemBlockTrophy + int64(m.ID)
}

/*
TrophyItemName is what the multiworld calls that medal.

Named per mission rather than one name in several copies, because the two goals
ask different questions of it: Final Boss asks whether one named mission is
done, and missionsanity counts. A single name answers the second and cannot
answer the first.
*/
func (m Mission) TrophyItemName() string {
	return "Australium Medal: " + m.Name
}

// ItemID is where this setting's item lives in the id space. Its own block, so
// adding one renumbers nothing that has shipped.
func (s ServerSetting) ItemID() int64 {
	return BaseID + itemSpaceOffset + itemBlockSetting + int64(s.ID)
}
