package gamedata

import (
	"slices"
	"testing"
)

// The Spy's revolver is a schema secondary held where the run's primary lock
// sits. Every other class keeps the schema's order, and the sappers, which are
// one weapon, and the PDAs that build or disguise are not offered.
func TestLoadoutUnlockSlotFollowsTheLocks(t *testing.T) {
	got := driver{body: `
    for (int class = 1; class <= 9; class++)
    {
        for (int position = 0; position < LoadoutPositionCount; position++)
        {
            printnum(Loadout_UnlockSlot(class, position));
        }
    }
`}.run(t)
	everyone := []int32{0, 1, 2, -1, -1, -1, -1}
	spy := []int32{-1, 0, 2, -1, -1, -1, 3}
	for class := 1; class <= 9; class++ {
		row := got[(class-1)*7 : class*7]
		want := everyone
		if class == 8 {
			want = spy
		}
		if !slices.Equal(row, want) {
			t.Errorf("class %d: unlock slots %v, want %v", class, row, want)
		}
	}
}

// Most buffs first, and weapons with as many buffs keep the catalogue's order.
func TestLoadoutOrdersByBuffsAndKeepsTies(t *testing.T) {
	got := driver{body: `
    int buffs[6] = { 0, 3, 1, 3, 0, 7 };
    int order[6];
    Loadout_OrderByBuffs(buffs, 6, order);
    for (int index = 0; index < 6; index++)
    {
        printnum(order[index]);
    }
`}.run(t)
	want := []int32{5, 1, 3, 2, 0, 4}
	if !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
}
