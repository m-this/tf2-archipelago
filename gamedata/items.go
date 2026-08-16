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

// Item is one entry in the multiworld's item pool.
//
// Mission, Class and Credits are the payload of the kind that uses them and
// zero everywhere else. Count is how many copies the pool must hold; a filler
// item carries zero, because filler pads the pool to the location count and
// how many that takes is decided at generation time.
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

// ProgressiveWeaponSlotName is the one item that unlocks loadout slots. It is
// progressive: copy n grants WeaponSlots[n-1], so the three copies hand out
// Primary, then Secondary, then Melee.
const ProgressiveWeaponSlotName = "Progressive Weapon Slot"

// progressiveWeaponSlotID sits at offset zero of the weapon slot block, which
// leaves offsets 1 to 3 free for per-slot items keyed by WeaponSlotID if v2
// ever drops the progressive shape.
var progressiveWeaponSlotID = BaseID + itemSpaceOffset + itemBlockWeaponSlot

// cashBundleCredits is what one filler item is worth at the next wave start.
// Small enough that a filler-heavy sphere does not buy a wave outright.
const cashBundleCredits uint16 = 200

var cashBundleID = BaseID + itemSpaceOffset + itemBlockCredits + 1

// Items is the whole item pool template: 29 mission tickets, 9 classes, the
// progressive weapon slot, and the filler that pads the rest.
//
// Weapons, upgrade lines, canteens, robot templates and traps are all v1
// omissions, not oversights. Slots and classes alone already make a
// progression, and the weapon table is the largest data-entry job in the
// project.
var Items = buildItems()

var itemsByID = indexItems()

func indexItems() map[int64]Item {
	byID := make(map[int64]Item, len(Items))
	for _, it := range Items {
		byID[it.ID] = it
	}
	return byID
}

// ItemByID is how the bridge reads a received item: Archipelago sends an id
// and nothing else, and the kind behind it is what the plugin is told about.
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
