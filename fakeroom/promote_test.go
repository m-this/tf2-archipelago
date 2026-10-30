package fakeroom

import (
	"slices"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/gamedata"
)

/*
	Test mode has to hand out a buff early enough to try one.

The order is classes, then weapon slots, then everything else, and everything
else is fifty mission tickets ahead of sixteen thousand buffs. Without this the
first buff lands about sixty waves in, which is not a test.
*/
func TestABuffArrivesEarlyEnoughToTry(t *testing.T) {
	order := unlockOrder(nil, nil)

	kind := make(map[int64]gamedata.ItemKind, len(gamedata.Items))
	for _, item := range gamedata.Items {
		kind[item.ID] = item.Kind
	}

	first := -1
	for at, id := range order {
		if kind[id] == gamedata.ItemWeaponBuff {
			first = at
			break
		}
	}
	if first < 0 {
		t.Fatal("test mode hands out no weapon buffs at all")
	}
	if first > 16 {
		t.Errorf("the first buff is item %d, which is %d waves of waiting", first+1, first+1)
	}
}

// The ones brought forward are different weapons and effects, not sixteen
// levels of the same thing, which is what taking the first few would give.
func TestThePromotedBuffsAreNotAllOneWeapon(t *testing.T) {
	order := unlockOrder(nil, nil)

	buff := make(map[int64]uint16, len(gamedata.Items))
	for _, item := range gamedata.Items {
		if item.Kind == gamedata.ItemWeaponBuff {
			buff[item.ID] = item.WeaponBuff
		}
	}
	seen := map[uint16]bool{}
	for _, id := range order[:buffsUpFront+12] {
		if id, ok := buff[id]; ok {
			seen[id] = true
		}
	}
	if len(seen) < 4 {
		t.Errorf("only %d distinct buffs up front, so the sample is not a spread", len(seen))
	}
}

// Every item a seed could grant in this mode survives reordering, with its
// normal copy count. Ineligible permutations never enter the test pool.
func TestPromotingKeepsEveryItem(t *testing.T) {
	order := unlockOrder(nil, nil)

	counted := map[int64]int{}
	for _, id := range order {
		counted[id]++
	}
	var want int
	for _, item := range gamedata.Items {
		copies := 0
		switch item.Kind {
		case gamedata.ItemMissionTicket, gamedata.ItemClass, gamedata.ItemWeaponSlot:
			copies = max(int(item.Count), 1)
		case gamedata.ItemTrap:
			// A room that names no trap holds Team Jarate alone, as a seed does.
			if item.Trap == gamedata.TrapDefault {
				copies = 1
			}
		case gamedata.ItemWeaponBuff:
			buff, known := gamedata.WeaponBuffByID(item.WeaponBuff)
			if known && buff.Eligible {
				copies = 1
			}
		default:
			// Other kinds need options or a locked location.
		}
		if got := counted[item.ID]; got != copies {
			t.Errorf("%q has %d copies in the test pool, want %d", item.Name, got, copies)
		}
		want += copies
	}
	if len(order) != want {
		t.Errorf("the order holds %d items, the pool has %d", len(order), want)
	}
}

// The promotion must not be slow: it runs when a test-mode room starts.
func TestPromotingIsNotQuadratic(t *testing.T) {
	start := time.Now()
	unlockOrder(nil, nil)
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("building the order took %s", took)
	}
}

func TestTheTestPoolHoldsTheChosenTraps(t *testing.T) {
	order := unlockOrder(nil, []string{"team_gas", "team_bleed"})
	var got []string
	for _, id := range order {
		if item, known := gamedata.ItemByID(id); known && item.Kind == gamedata.ItemTrap {
			trap, _ := gamedata.TrapByID(item.Trap)
			got = append(got, trap.Key)
		}
	}
	if !slices.Equal(got, []string{"team_bleed", "team_gas"}) {
		t.Errorf("the test pool holds the traps %v, want team_bleed and team_gas", got)
	}
}
