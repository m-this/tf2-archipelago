/**
 * The loadout menu's decisions that need nothing but their arguments.
 *
 * Positions are TF2's loadout_positions_t, which is what TF2Econ_GetItemLoadoutSlot
 * answers. Classes are TFClassType. Neither enum is included here, so the
 * numbers are spelled out and gamedata/spdriver_test.go runs these as they are.
 */

#define LoadoutPositionPrimary 0
#define LoadoutPositionSecondary 1
#define LoadoutPositionMelee 2
#define LoadoutPositionPda2 6
#define LoadoutPositionCount 7

#define LoadoutClassSpy 8

// A position the run never locks: the Spy's watch.
#define LoadoutSlotAlwaysOpen 3

/* The unlock slot an item in this position sits in, or -1 for a position the
 * menu does not offer.
 *
 * The run locks weapon slots, and the Spy's are not his schema positions: his
 * revolver is a schema secondary held where everybody else holds a primary.
 * Every sapper is one functional weapon, so his secondary offers no choice, and
 * the PDAs that build and disguise are left to the game.
 */
int Loadout_UnlockSlot(int class, int position)
{
    if (class == LoadoutClassSpy)
    {
        switch (position)
        {
            case LoadoutPositionSecondary: return 0;
            case LoadoutPositionMelee: return 2;
            case LoadoutPositionPda2: return LoadoutSlotAlwaysOpen;
        }
        return -1;
    }
    switch (position)
    {
        case LoadoutPositionPrimary: return 0;
        case LoadoutPositionSecondary: return 1;
        case LoadoutPositionMelee: return 2;
    }
    return -1;
}

/* Fills order with 0..count-1, most buffs first, ties kept in catalogue order.
 *
 * Insertion sort: a slot holds a few dozen weapons at most.
 */
void Loadout_OrderByBuffs(const int[] buffs, int count, int[] order)
{
    for (int index = 0; index < count; index++)
    {
        int position = index;
        while (position > 0 && buffs[order[position - 1]] < buffs[index])
        {
            order[position] = order[position - 1];
            position--;
        }
        order[position] = index;
    }
}
