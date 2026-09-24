package botcards

import (
	"slices"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
)

func TestInnatesAreDistinctStackableAPEffectsForTheClass(t *testing.T) {
	for _, card := range Cards {
		t.Run(card.Name, func(t *testing.T) {
			class, ok := botloadout.ClassByKey(card.Class)
			if !ok {
				t.Fatalf("unknown class %q", card.Class)
			}
			loadout := class.LoadoutByKey(card.Loadout)
			definitions := []int{loadout.Primary, loadout.Second, loadout.Melee, loadout.PDA2}
			if card.StockDef != 0 {
				definitions = append(definitions, card.StockDef)
			}
			if len(card.BuffIDs) != card.Tier.Stacks() {
				t.Fatalf("%s has %d innates, want %d", card.Tier, len(card.BuffIDs), card.Tier.Stacks())
			}
			seen := map[int]bool{}
			for _, id := range card.BuffIDs {
				buff, found := gamedata.WeaponBuffByID(id)
				if !found || !buff.Eligible || buff.Mode == gamedata.BuffToggle {
					t.Errorf("innate %d is not an eligible stackable AP weapon buff", id)
					continue
				}
				if seen[int(buff.EffectID)] {
					t.Errorf("duplicate innate effect %d", buff.EffectID)
				}
				seen[int(buff.EffectID)] = true
				if !slices.ContainsFunc(definitions, func(definition int) bool {
					return slices.Contains(buff.DefIndexes, definition)
				}) {
					t.Errorf("%s cannot equip %s", card.Class, buff.ItemName())
				}
			}
		})
	}
}

func TestLegendaryUnusualEffectsBelongToCards(t *testing.T) {
	seen := map[int]bool{}
	for _, card := range Cards {
		if card.Tier != Legendary {
			if card.UnusualEffect != 0 {
				t.Errorf("%s is not Legendary but has effect %d", card.ID, card.UnusualEffect)
			}
			continue
		}
		if card.UnusualEffect <= 0 || seen[card.UnusualEffect] {
			t.Errorf("%s needs its own fixed Unusual effect, got %d", card.ID, card.UnusualEffect)
		}
		seen[card.UnusualEffect] = true
	}
}

func TestEverySeedTierHasItsOwnEligibleInnates(t *testing.T) {
	for _, card := range Cards {
		for _, tier := range []Tier{Common, Elite, Legendary} {
			buffs := BuffsFor(card, tier)
			if len(buffs) != tier.Stacks() {
				t.Errorf("%s %s: %d innates, want %d", card.Name, tier, len(buffs), tier.Stacks())
			}
			seen := map[uint8]bool{}
			for _, id := range buffs {
				buff, ok := gamedata.WeaponBuffByID(id)
				if !ok || !buff.Eligible || buff.Mode == gamedata.BuffToggle {
					t.Errorf("%s %s: invalid buff %d", card.Name, tier, id)
					continue
				}
				if seen[buff.EffectID] {
					t.Errorf("%s %s: repeated effect %d", card.Name, tier, buff.EffectID)
				}
				seen[buff.EffectID] = true
			}
		}
	}
}

// gamedata names the cards a seed can hold and this package says what each one
// is. They are two lists of the same fifteen identities, so a card added to one
// and not the other is an item the launcher cannot seat, or a seat no seed can
// ever unlock.
func TestTheCatalogueMatchesTheSeedsCards(t *testing.T) {
	if len(Cards) != len(gamedata.BotCardTemplates) {
		t.Fatalf("%d cards here, %d in gamedata", len(Cards), len(gamedata.BotCardTemplates))
	}
	for i, card := range Cards {
		seed := gamedata.BotCardTemplates[i]
		if card.Name != seed.Name || card.Class != seed.Class || (card.Loadout == "stock") != seed.Stock {
			t.Errorf("card %d is %s/%s/%s here and %s/%s/stock=%t in gamedata",
				i, card.Name, card.Class, card.Loadout, seed.Name, seed.Class, seed.Stock)
		}
	}
}
