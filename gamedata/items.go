package gamedata

// ItemKind is what an item does when it lands. It is also the plugin's grant
// vocabulary: the bridge sends the kind and its payload, never an item id.
type ItemKind uint8

const (
	ItemMissionTicket ItemKind = iota + 1
	ItemClass
	ItemWeaponSlot
	ItemCredits
	ItemWeaponBuff
	ItemTrap
	ItemServerSetting
	ItemTrophy
	ItemClassWeaponSlot
	ItemBotCard
)

var itemKindKeys = [...]string{
	ItemMissionTicket:   "mission_ticket",
	ItemClass:           "class",
	ItemWeaponSlot:      "weapon_slot",
	ItemCredits:         "credits",
	ItemWeaponBuff:      "weapon_buff",
	ItemTrap:            "trap",
	ItemServerSetting:   "server_setting",
	ItemTrophy:          "trophy",
	ItemClassWeaponSlot: "class_weapon_slot",
	ItemBotCard:         "bot_card",
}

// ItemKinds is every kind that exists, in id order. The bridge walks it to
// build the unlock set, so a kind added here needs no second list anywhere.
var ItemKinds = []ItemKind{
	ItemMissionTicket, ItemClass, ItemWeaponSlot, ItemCredits, ItemWeaponBuff, ItemTrap,
	ItemServerSetting, ItemTrophy, ItemClassWeaponSlot, ItemBotCard,
}

// Key is the string on the wire between the bridge and the plugin.
func (k ItemKind) Key() string { return itemKindKeys[k] }

// OneShot reports whether applying the item a second time differs from applying
// it once. A class is state: granting it again changes nothing, so it can be
// replayed after any reload. Credits are an effect: granting them again pays a
// second time.
//
// The distinction is what decides how the bridge delivers an item. State goes
// in the unlock set and is resent freely; an effect is sent once and is not
// sent again until the plugin says it applied it. A trap is an effect for the
// same reason: firing it twice soaks a team that only earned it once.
func (k ItemKind) OneShot() bool { return k == ItemCredits || k == ItemTrap }

/*
Granted reports whether the plugin ever sees this kind.

A trophy is not a grant. It is locked onto a mission clear so that generation
can ask whether the mission is done, and nothing happens in the game when it
lands: the bridge drops it and the plugin is never told. Every other kind is
something a player receives, and one the plugin does not handle is an item the
seed loses in silence, which is what the plugin-keys test is for.
*/
func (k ItemKind) Granted() bool { return k != ItemTrophy }

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
	WeaponBuff     uint16
	Trap           TrapID
	ServerSetting  ServerSettingID
	BotName        string
	BotTier        string
	BotForm        string
	BotStock       bool

	// Slot is the loadout slot a named class slot item opens, and zero on
	// every other item, progressive ones included: a progressive copy opens
	// whichever slot is next, which is not a thing an item can carry.
	Slot WeaponSlotID
}

// BotCardTemplates are unique identities. Stock cards use the stock loadout;
// rarity and form are separate seed rolls, not properties of the name.
var BotCardTemplates = []struct {
	Name  string
	Class string
	Stock bool
}{
	{"Chucklenuts", "scout", true},
	{"Maggot", "soldier", true},
	{"BeepBeepBoop", "pyro", true},
	{"Kaboom!", "demoman", true},
	{"Nom Nom Nom", "heavyweapons", true},
	{"MoreGun", "engineer", true},
	{"Archimedes!", "medic", true},
	{"A Professional With Standards", "sniper", true},
	{"Gentlemanne of Leisure", "spy", true}, //nolint:misspell // A deliberate BOT-list name.
	{"CreditToTeam", "scout", false},
	{"Screamin' Eagles", "soldier", false},
	{"IvanTheSpaceBiker", "heavyweapons", false},
	{"Herr Doktor", "medic", false},
	{"Chell", "engineer", false},
	{"Mentlegen", "spy", false},
}

// Every possible roll has a stable AP item ID. A generated seed includes at
// most one variant of each identity, so finding a card never duplicates it.
func botCardItems() []Item {
	items := make([]Item, 0, len(BotCardTemplates)*9)
	for index, card := range BotCardTemplates {
		for tierIndex, tier := range []string{"Common", "Elite", "Legendary"} {
			for formIndex, form := range []string{"Human", "Robot", "Giant"} {
				items = append(items, Item{
					ID:   BaseID + itemSpaceOffset + itemBlockBotCard + int64(index*9+tierIndex*3+formIndex+1),
					Name: "Bot: " + card.Name + " | " + tier + " | " + form,
					Kind: ItemBotCard, Classification: Useful, Count: 1,
					BotName: card.Name, BotTier: tier, BotForm: form, BotStock: card.Stock,
				})
			}
		}
	}
	return items
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
// Weapon ownership, canteens and robot templates remain out of scope.
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

// slotItems is the one progressive weapon slot for everybody, then the one per
// class each copy of which opens that class's next slot in its own order. The
// class's first slot is free with the class, so a class earns the other two.
// A seed holds one set or the other: class_weapon_slots decides which.
func slotItems() []Item {
	items := make([]Item, 0, len(Classes)+1)
	items = append(items, Item{
		ID:             progressiveWeaponSlotID,
		Name:           ProgressiveWeaponSlotName,
		Kind:           ItemWeaponSlot,
		Classification: Progression,
		Count:          uint8(len(WeaponSlots)),
	})
	for _, c := range Classes {
		items = append(items, Item{
			ID:             c.SlotItemID(),
			Name:           c.SlotItemName(),
			Kind:           ItemClassWeaponSlot,
			Classification: Progression,
			Count:          ClassSlotsEarned,
			Class:          c.ID,
		})
		// And the same two slots as items that name them, for a seed that
		// hands them out in any order. A seed holds one kind or the other.
		for _, slot := range c.EarnedSlots() {
			items = append(items, Item{
				ID:             c.NamedSlotItemID(slot.ID),
				Name:           c.NamedSlotItemName(slot),
				Kind:           ItemClassWeaponSlot,
				Classification: Progression,
				Count:          1,
				Class:          c.ID,
				Slot:           slot.ID,
			})
		}
	}
	return items
}

func buildItems() []Item {
	all := make([]Item, 0, len(Missions)+len(Classes)+len(WeaponBuffs)+len(Traps)+2)
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
	all = append(all, slotItems()...)
	all = append(all, Item{
		ID:             cashBundleID,
		Name:           "Cash Bundle",
		Kind:           ItemCredits,
		Classification: Filler,
		Credits:        cashBundleCredits,
	})
	for _, buff := range WeaponBuffs {
		all = append(all, Item{
			ID:             buff.ItemID(),
			Name:           buff.ItemName(),
			Kind:           ItemWeaponBuff,
			Classification: Useful,
			WeaponBuff:     buff.ID,
		})
	}
	for _, trap := range Traps {
		all = append(all, Item{
			ID:             trap.ItemID(),
			Name:           trap.ItemName(),
			Kind:           ItemTrap,
			Classification: TrapClassification,
			Trap:           trap.ID,
		})
	}
	/* Useful and never progression

	A wave has to stay winnable without one, so no access rule may ever need
	a setting: a seed that puts the hook behind a check nobody can reach
	would still be beatable. */
	for _, setting := range ServerSettings {
		all = append(all, Item{
			ID:             setting.ItemID(),
			Name:           setting.ItemName(),
			Kind:           ItemServerSetting,
			Classification: Useful,
			Count:          1,
			ServerSetting:  setting.ID,
		})
	}
	all = append(all, botCardItems()...)
	return append(all, trophyItems()...)
}

/*
trophyItems is a medal per mission, locked onto that mission's clear.

Never in the pool: the world places one on each clear when the option is on and
adds none otherwise, so a seed without the option never sees these ids.
Progression because both goals read them.
*/
func trophyItems() []Item {
	all := make([]Item, 0, len(Missions))
	for _, m := range Missions {
		all = append(all, Item{
			ID:             m.TrophyItemID(),
			Name:           m.TrophyItemName(),
			Kind:           ItemTrophy,
			Classification: Progression,
			Count:          1,
			Mission:        m.ID,
		})
	}
	return all
}
