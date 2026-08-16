package gamedata

// ItemKind is what an item does when it lands. It is also the plugin's grant
// vocabulary: the bridge sends the kind and its payload, never an item id.
type ItemKind uint8

const (
	ItemMissionTicket ItemKind = iota + 1
	ItemClass
	ItemWeaponSlot
	ItemCredits
)

var itemKindKeys = [...]string{
	ItemMissionTicket: "mission_ticket",
	ItemClass:         "class",
	ItemWeaponSlot:    "weapon_slot",
	ItemCredits:       "credits",
}

// Key is the string on the wire between the bridge and the plugin.
func (k ItemKind) Key() string { return itemKindKeys[k] }

// Item is one entry in the multiworld's item pool. Mission, Class and Credits
// are the payload of the kind that uses them and zero elsewhere; Count is zero
// for filler, whose copy count is decided at generation time.
type Item struct {
	ID             int64
	Name           string
	Kind           ItemKind
	Classification Classification
	Count          uint8
	Mission        MissionID
	Class          ClassID
	Credits        uint16
}

// ProgressiveWeaponSlotName is the one item that unlocks loadout slots: copy n
// grants WeaponSlots[n-1].
const ProgressiveWeaponSlotName = "Progressive Weapon Slot"

// progressiveWeaponSlotID takes offset zero, leaving 1 to 3 free for per-slot items later.
var progressiveWeaponSlotID = BaseID + itemSpaceOffset + itemBlockWeaponSlot

// cashBundleCredits is kept low so a filler-heavy sphere cannot buy a wave outright.
const cashBundleCredits uint16 = 200

var cashBundleID = BaseID + itemSpaceOffset + itemBlockCredits + 1

// Items is the whole item pool template: a ticket per mission, a class item
// per class, the progressive weapon slot, and the filler that pads the rest.
//
// Weapons, upgrades, canteens, robot templates and traps are out of scope for
// v1: slots and classes alone already make a progression.
var Items = buildItems()

var itemsByID = indexItems()

func indexItems() map[int64]Item {
	byID := make(map[int64]Item, len(Items))
	for _, it := range Items {
		byID[it.ID] = it
	}
	return byID
}

// ItemByID resolves an id from a ReceivedItems payload into the kind the
// plugin is told about.
func ItemByID(id int64) (Item, bool) {
	it, ok := itemsByID[id]
	return it, ok
}

func buildItems() []Item {
	all := make([]Item, 0, len(Missions)+len(Classes)+2)
	for _, m := range Missions {
		all = append(all, Item{
			ID:             m.TicketItemID(),
			Name:           m.TicketItemName(),
			Kind:           ItemMissionTicket,
			Classification: Progression,
			Count:          1,
			Mission:        m.ID,
		})
	}
	for _, c := range Classes {
		all = append(all, Item{
			ID:             c.ItemID(),
			Name:           c.ItemName(),
			Kind:           ItemClass,
			Classification: Progression,
			Count:          1,
			Class:          c.ID,
		})
	}
	all = append(all, Item{
		ID:             progressiveWeaponSlotID,
		Name:           ProgressiveWeaponSlotName,
		Kind:           ItemWeaponSlot,
		Classification: Progression,
		Count:          uint8(len(WeaponSlots)),
	})
	all = append(all, Item{
		ID:             cashBundleID,
		Name:           "Cash Bundle",
		Kind:           ItemCredits,
		Classification: Filler,
		Credits:        cashBundleCredits,
	})
	return all
}
