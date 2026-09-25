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

/*
Cards is every card, in the order the seed draws them.

Who a card is (its name, its class, whether it carries the stock loadout) is
gamedata.BotCardTemplates, the list the apworld places items from, and nowhere
else. What the launcher needs beyond that is details, found by the card's name.
A template with no details stops the launcher at start rather than seating a
card with no weapons, and the test says so before that.
*/
var Cards = build()

// details is what a card is beyond who it is. Loadout is empty for a stock card,
// whose loadout is the class's stock one.
type details struct {
	ID            string
	Loadout       string
	Tier          Tier
	Cosmetic      int
	UnusualEffect int
	BuffIDs       []uint16
	StockDef      int
}

var byName = map[string]details{
	"Chucklenuts":                   {ID: "stock-scout", Tier: Common, Cosmetic: 116, StockDef: 13, BuffIDs: []uint16{161}},
	"Maggot":                        {ID: "stock-soldier", Tier: Common, Cosmetic: 116, StockDef: 18, BuffIDs: []uint16{156}},
	"BeepBeepBoop":                  {ID: "stock-pyro", Tier: Common, Cosmetic: 116, StockDef: 21, BuffIDs: []uint16{75}},
	"Kaboom!":                       {ID: "stock-demoman", Tier: Common, Cosmetic: 116, StockDef: 19, BuffIDs: []uint16{87}},
	"Nom Nom Nom":                   {ID: "stock-heavy", Tier: Common, Cosmetic: 116, StockDef: 15, BuffIDs: []uint16{123}},
	"MoreGun":                       {ID: "stock-engineer", Tier: Common, Cosmetic: 116, StockDef: 9, BuffIDs: []uint16{173}},
	"Archimedes!":                   {ID: "stock-medic", Tier: Common, Cosmetic: 116, StockDef: 17, BuffIDs: []uint16{187}},
	"A Professional With Standards": {ID: "stock-sniper", Tier: Common, Cosmetic: 116, StockDef: 14, BuffIDs: []uint16{10433}},
	"Gentlemanne of Leisure":        {ID: "stock-spy", Tier: Common, Cosmetic: 116, StockDef: 24, BuffIDs: []uint16{152}}, //nolint:misspell // A deliberate BOT-list name.
	// Soda Popper damage.
	"CreditToTeam": {ID: "credit-to-team", Loadout: "milk", Tier: Common, Cosmetic: 111, BuffIDs: []uint16{10434}},
	// Beggar damage, Escape Plan firing speed.
	"Screamin' Eagles": {ID: "screamin-eagles", Loadout: "beggar", Tier: Elite, Cosmetic: 378, BuffIDs: []uint16{10275, 10577}},
	// Brass Beast firing speed, Family Business clip size.
	"IvanTheSpaceBiker": {ID: "ivan", Loadout: "brass", Tier: Elite, Cosmetic: 185, BuffIDs: []uint16{10542, 11093}},
	// Crossbow damage, Kritz Über rate, Übersaw firing speed. Burning Flames.
	"Herr Doktor": {ID: "herr-doktor", Loadout: "kritz", Tier: Legendary, Cosmetic: 315, UnusualEffect: 13, BuffIDs: []uint16{10305, 17531, 10718}},
	// Rescue Ranger damage.
	"Chell": {ID: "chell", Loadout: "ranger", Tier: Common, Cosmetic: 484, BuffIDs: []uint16{10406}},
	// Diamondback damage, Big Earner firing speed and armor. Scorching Flames.
	"Mentlegen": {ID: "mentlegen", Loadout: "diamondback", Tier: Legendary, Cosmetic: 55, UnusualEffect: 14, BuffIDs: []uint16{10313, 10532, 18212}},
}

func build() []Card {
	cards := make([]Card, 0, len(gamedata.BotCardTemplates))
	for _, template := range gamedata.BotCardTemplates {
		d, ok := byName[template.Name]
		if !ok {
			panic("botcards: gamedata has a card named " + template.Name + " and this package has no details for it")
		}
		loadout := d.Loadout
		if template.Stock {
			loadout = "stock"
		}
		cards = append(cards, Card{
			ID: d.ID, Name: template.Name, Class: template.Class, Loadout: loadout, Tier: d.Tier,
			Cosmetic: d.Cosmetic, UnusualEffect: d.UnusualEffect, BuffIDs: d.BuffIDs, StockDef: d.StockDef,
		})
	}
	return cards
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
