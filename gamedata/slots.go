package gamedata

// WeaponSlotID identifies one loadout slot. Explicit literals, append-only.
type WeaponSlotID uint8

const (
	WeaponSlotPrimary   WeaponSlotID = 1
	WeaponSlotSecondary WeaponSlotID = 2
	WeaponSlotMelee     WeaponSlotID = 3
)

// WeaponSlot is one loadout slot. Key is what crosses the wire to the plugin
// in a grant.
type WeaponSlot struct {
	ID   WeaponSlotID
	Key  string
	Name string
}

// WeaponSlots is the three slots, in the order the progressive weapon slot
// item hands them out: the first copy grants Primary, the second Secondary,
// the third Melee. Reordering this changes what an already generated seed
// gives a player, so it is as frozen as an id.
var WeaponSlots = []WeaponSlot{
	{WeaponSlotPrimary, "primary", "Primary"},
	{WeaponSlotSecondary, "secondary", "Secondary"},
	{WeaponSlotMelee, "melee", "Melee"},
}
