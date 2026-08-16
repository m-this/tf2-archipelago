package gamedata

import "fmt"

// Validate re-checks every invariant the id scheme rests on. The exporter runs
// it first: a broken table that reaches the apworld is a broken seed nobody can
// detect at play time.
func Validate() error {
	if err := validateEntities(); err != nil {
		return err
	}
	if err := validateMissions(); err != nil {
		return err
	}
	return validateIDs()
}

func validateEntities() error {
	if err := unique("map id", len(Maps), func(i int) any { return Maps[i].ID }); err != nil {
		return err
	}
	if err := unique("map name", len(Maps), func(i int) any { return Maps[i].Name }); err != nil {
		return err
	}
	if err := unique("class id", len(Classes), func(i int) any { return Classes[i].ID }); err != nil {
		return err
	}
	if err := unique("class key", len(Classes), func(i int) any { return Classes[i].Key }); err != nil {
		return err
	}
	if err := unique("weapon slot id", len(WeaponSlots), func(i int) any { return WeaponSlots[i].ID }); err != nil {
		return err
	}
	return unique("weapon slot key", len(WeaponSlots), func(i int) any { return WeaponSlots[i].Key })
}

func validateMissions() error {
	if err := unique("mission id", len(Missions), func(i int) any { return Missions[i].ID }); err != nil {
		return err
	}
	if err := unique("mission pop file", len(Missions), func(i int) any { return Missions[i].PopFile }); err != nil {
		return err
	}
	if err := unique("mission name", len(Missions), func(i int) any { return Missions[i].Name }); err != nil {
		return err
	}
	for _, m := range Missions {
		if m.ID < 1 || m.ID > MissionIDMax {
			return fmt.Errorf("mission %q: id %d outside 1..%d", m.PopFile, m.ID, MissionIDMax)
		}
		if m.Waves < 1 || m.Waves > WavesMax {
			return fmt.Errorf("mission %q: %d waves, outside 1..%d", m.PopFile, m.Waves, WavesMax)
		}
		if _, ok := MapByID(m.Map); !ok {
			return fmt.Errorf("mission %q: unknown map id %d", m.PopFile, m.Map)
		}
		if int(m.Difficulty) >= len(difficultyKeys) || m.Difficulty.Key() == "" {
			return fmt.Errorf("mission %q: unknown difficulty %d", m.PopFile, m.Difficulty)
		}
	}
	return nil
}

func validateIDs() error {
	if err := unique("location name", len(Locations), func(i int) any { return Locations[i].Name }); err != nil {
		return err
	}
	if err := unique("item name", len(Items), func(i int) any { return Items[i].Name }); err != nil {
		return err
	}
	seen := make(map[int64]string, len(Locations)+len(Items))
	for _, l := range Locations {
		if other, clash := seen[l.ID]; clash {
			return fmt.Errorf("id %d used by both %q and %q", l.ID, other, l.Name)
		}
		seen[l.ID] = l.Name
	}
	for _, it := range Items {
		if other, clash := seen[it.ID]; clash {
			return fmt.Errorf("id %d used by both %q and %q", it.ID, other, it.Name)
		}
		seen[it.ID] = it.Name
	}
	return nil
}

func unique(what string, n int, at func(int) any) error {
	seen := make(map[any]struct{}, n)
	for i := range n {
		key := at(i)
		if _, clash := seen[key]; clash {
			return fmt.Errorf("duplicate %s: %v", what, key)
		}
		seen[key] = struct{}{}
	}
	return nil
}
