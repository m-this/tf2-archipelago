package gamedata

// Stock weapons are deliberately absent from Weapons, because the loadout
// picker represents them with its default entry. Keep their class ownership
// here so consumers of the generated item catalogue can still answer which
// mercenary can equip a weapon buff. Non-stock and shared weapons are derived
// from Weapons below rather than repeated in this table.
var stockWeaponClassNames = map[string][]string{
	"Scattergun":             {"Scout"},
	"Pistol":                 {"Scout", "Engineer"},
	"Bat":                    {"Scout"},
	"Rocket Launcher":        {"Soldier"},
	"Rocket Jumper":          {"Soldier"},
	"Shotgun":                {"Soldier", "Pyro", "Heavy", "Engineer"},
	"Shovel":                 {"Soldier"},
	"Flame Thrower":          {"Pyro"},
	"Fire Axe":               {"Pyro"},
	"Grenade Launcher":       {"Demoman"},
	"Stickybomb Launcher":    {"Demoman"},
	"Bottle":                 {"Demoman"},
	"Minigun":                {"Heavy"},
	"Fists":                  {"Heavy"},
	"Buffalo Steak Sandvich": {"Heavy"},
	"Dalokohs Bar":           {"Heavy"},
	"Sandvich":               {"Heavy"},
	"Second Banana":          {"Heavy"},
	"Wrench":                 {"Engineer"},
	"Construction PDA":       {"Engineer"},
	"Destruction PDA":        {"Engineer"},
	"Syringe Gun":            {"Medic"},
	"Medi Gun":               {"Medic"},
	"Bonesaw":                {"Medic"},
	"Sniper Rifle":           {"Sniper"},
	"SMG":                    {"Sniper"},
	"Kukri":                  {"Sniper"},
	"Revolver":               {"Spy"},
	"Knife":                  {"Spy"},
	"Sapper":                 {"Spy"},
	"Invis Watch":            {"Spy"},
	"Disguise Kit":           {"Spy"},
}

// Weapon is one item the bot mod will hand a defender bot: the definition
// index the mod writes into its loadout file, and what to call it in a menu.
type Weapon struct {
	DefIndex int
	Name     string
	Class    string
	Slot     string
}

// WeaponsFor is what one class can hold in one slot, in the order a menu
// should show them. Empty for a pair the mod has no pool for, which is every
// slot the Spy does not have and the Spy's own pda2 for everybody else.
func WeaponsFor(class, slot string) []Weapon {
	var out []Weapon
	for _, weapon := range Weapons {
		if weapon.Class == class && weapon.Slot == slot {
			out = append(out, weapon)
		}
	}
	return out
}

// WeaponByIndex is the item with that definition index, and whether the
// catalogue carries it at all. A loadout naming an index this does not know is
// still legal, because the mod validates nothing: it just cannot be named.
func WeaponByIndex(defIndex int) (Weapon, bool) {
	for _, weapon := range Weapons {
		if weapon.DefIndex == defIndex {
			return weapon, true
		}
	}
	return Weapon{}, false
}

// weaponClassNames returns class display names in menu order. A definition
// index may appear in several loadout pools (the Shotgun and all-class melee
// weapons do), so the result can contain more than one class.
func weaponClassNames(name string, defIndexes []int) []string {
	owned := make(map[string]bool, len(Classes))
	for _, className := range stockWeaponClassNames[name] {
		owned[className] = true
	}
	for _, definition := range defIndexes {
		for _, weapon := range Weapons {
			if weapon.DefIndex != definition {
				continue
			}
			key := weapon.Class
			if key == "heavyweapons" {
				key = "heavy"
			}
			for _, class := range Classes {
				if class.Key == key {
					owned[class.Name] = true
					break
				}
			}
		}
	}
	var names []string
	for _, class := range Classes {
		if owned[class.Name] {
			names = append(names, class.Name)
		}
	}
	return names
}
