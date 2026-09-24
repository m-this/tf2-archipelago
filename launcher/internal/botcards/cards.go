// Package botcards holds the named defender identities available to a run.
// A card is an identity, not a bot client: its seat is rebuilt after map changes.
package botcards

import (
	"slices"
	"strings"

	"github.com/m-this/tf2-archipelago/gamedata"
)

type Card struct {
	ID            string
	Name          string
	Class         string
	Loadout       string
	Tier          Tier
	Cosmetic      int // TF2 item definition index from the installed item schema.
	UnusualEffect int // Fixed cosmetic particle ID; zero for non-Legendary cards.
	// BuffIDs identify normal eligible Archipelago effects through an item
	// the class can equip. The card grants each distinct effect to every
	// weapon it carries, once per tier stack.
	BuffIDs  []uint16
	StockDef int // Stock weapon definition used to choose eligible innates.
}

type Tier string

const (
	Common    Tier = "common"
	Elite     Tier = "elite"
	Legendary Tier = "legendary"
)

func (tier Tier) Stacks() int {
	switch tier {
	case Elite:
		return 2
	case Legendary:
		return 3
	default:
		return 1
	}
}

// Cards use names from the defender name list and loadouts the mod knows.
var Cards = []Card{
	{ID: "stock-scout", Name: "Chucklenuts", Class: "scout", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 13, BuffIDs: []uint16{161}},
	{ID: "stock-soldier", Name: "Maggot", Class: "soldier", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 18, BuffIDs: []uint16{156}},
	{ID: "stock-pyro", Name: "BeepBeepBoop", Class: "pyro", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 21, BuffIDs: []uint16{75}},
	{ID: "stock-demoman", Name: "Kaboom!", Class: "demoman", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 19, BuffIDs: []uint16{87}},
	{ID: "stock-heavy", Name: "Nom Nom Nom", Class: "heavyweapons", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 15, BuffIDs: []uint16{123}},
	{ID: "stock-engineer", Name: "MoreGun", Class: "engineer", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 9, BuffIDs: []uint16{173}},
	{ID: "stock-medic", Name: "Archimedes!", Class: "medic", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 17, BuffIDs: []uint16{187}},
	{ID: "stock-sniper", Name: "A Professional With Standards", Class: "sniper", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 14, BuffIDs: []uint16{10433}},
	{ID: "stock-spy", Name: "Gentlemanne of Leisure", Class: "spy", Loadout: "stock", Tier: Common, Cosmetic: 116, StockDef: 24, BuffIDs: []uint16{152}}, //nolint:misspell // A deliberate BOT-list name.
	{
		ID: "credit-to-team", Name: "CreditToTeam", Class: "scout", Loadout: "milk", Tier: Common, Cosmetic: 111,
		BuffIDs: []uint16{10434},
	}, // Soda Popper damage
	{
		ID: "screamin-eagles", Name: "Screamin' Eagles", Class: "soldier", Loadout: "beggar", Tier: Elite, Cosmetic: 378,
		BuffIDs: []uint16{10275, 10577},
	}, // Beggar damage, Escape Plan firing speed
	{
		ID: "ivan", Name: "IvanTheSpaceBiker", Class: "heavyweapons", Loadout: "brass", Tier: Elite, Cosmetic: 185,
		BuffIDs: []uint16{10542, 11093},
	}, // Brass Beast firing speed, Family Business clip size
	{ID: "herr-doktor", Name: "Herr Doktor", Class: "medic", Loadout: "kritz", Tier: Legendary, Cosmetic: 315, UnusualEffect: 13, // Burning Flames
		BuffIDs: []uint16{10305, 17531, 10718}}, // Crossbow damage, Kritz Über rate, Übersaw firing speed
	{
		ID: "chell", Name: "Chell", Class: "engineer", Loadout: "ranger", Tier: Common, Cosmetic: 484,
		BuffIDs: []uint16{10406},
	}, // Rescue Ranger damage
	{ID: "mentlegen", Name: "Mentlegen", Class: "spy", Loadout: "diamondback", Tier: Legendary, Cosmetic: 55, UnusualEffect: 14, // Scorching Flames
		BuffIDs: []uint16{10313, 10532, 18212}}, // Diamondback damage, Big Earner firing speed and armor
}

// ParseItemName returns the seed's immutable roll for one AP reward.
func ParseItemName(name string) (Card, Tier, string, bool) {
	parts := strings.Split(name, " | ")
	if len(parts) != 3 || !strings.HasPrefix(parts[0], "Bot: ") {
		return Card{}, "", "", false
	}
	var tier Tier
	switch parts[1] {
	case "Common":
		tier = Common
	case "Elite":
		tier = Elite
	case "Legendary":
		tier = Legendary
	default:
		return Card{}, "", "", false
	}
	if parts[2] != "Human" && parts[2] != "Robot" && parts[2] != "Giant" {
		return Card{}, "", "", false
	}
	for _, card := range Cards {
		if card.Name == strings.TrimPrefix(parts[0], "Bot: ") {
			return card, tier, strings.ToLower(parts[2]), true
		}
	}
	return Card{}, "", "", false
}

// BuffsFor takes distinct eligible AP effects for this class's kit. Existing
// curated innates stay first; a higher seed tier fills the additional slots.
func BuffsFor(card Card, tier Tier) []uint16 {
	want := tier.Stacks()
	ids := slices.Clone(card.BuffIDs)
	seen := map[uint8]bool{}
	for _, id := range ids {
		if buff, ok := gamedata.WeaponBuffByID(id); ok {
			seen[buff.EffectID] = true
		}
	}
	defs := []int{card.StockDef}
	if card.StockDef == 0 {
		for _, id := range card.BuffIDs {
			if buff, ok := gamedata.WeaponBuffByID(id); ok {
				defs = append(defs, buff.DefIndexes...)
			}
		}
	}
	for _, buff := range gamedata.WeaponBuffs {
		if len(ids) >= want {
			break
		}
		if !buff.Eligible || buff.Mode == gamedata.BuffToggle || seen[buff.EffectID] {
			continue
		}
		if !slices.ContainsFunc(defs, func(def int) bool { return slices.Contains(buff.DefIndexes, def) }) {
			continue
		}
		ids = append(ids, buff.ID)
		seen[buff.EffectID] = true
	}
	return ids[:min(want, len(ids))]
}

func ByID(id string) (Card, bool) {
	for _, card := range Cards {
		if card.ID == id {
			return card, true
		}
	}
	return Card{}, false
}

func BySeat(class, name string) (Card, bool) {
	for _, card := range Cards {
		if card.Class == class && card.Name == name {
			return card, true
		}
	}
	return Card{}, false
}
