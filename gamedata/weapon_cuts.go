package gamedata

/*
sheetCuts are the weapon and effect pairs Cowser's sheet marks N: the buff is
offered and does nothing on that weapon (gh-20). The sheet is the source, one
row per effect and one column per weapon, and this is its N cells where the
rules here still said yes. A cell that says Native is a buff the weapon already
carries, which still works and stays.

Three of its columns are not taken, because the plugin implements what the
attribute alone would not: the jars' substances and projectile buffs, which
PlayerJarated and the projectile fanout carry; projectile count and speed on the
Sandman and the Wrap Assassin, the same fanout; and the engineer's building
buffs on his shotgun and pistol, which the player's attribute walk reads off
any held item.

Data rather than rules on purpose. The rules in weapon_eligibility.go say what
a mechanic needs; this says what one person checked in game, weapon by weapon,
and a wrong cell is one line to remove. IDs are untouched, a cut pair leaves
the pool and nothing else.
*/
var sheetCuts = map[string]map[string]bool{
	"Air Strike":           names("accuracy"),
	"Axtinguisher":         names("afterburn-damage", "afterburn-duration"),
	"Backburner":           names("fire-rate"),
	"Bazaar Bargain":       names("accuracy", "fire-rate"),
	"Black Box":            names("accuracy"),
	"Blutsauger":           names("secondary-ammo"),
	"Chargin' Targe":       names("deploy-speed"),
	"Classic":              names("accuracy", "fire-rate"),
	"Cow Mangler 5000":     names("accuracy", "max-ammo"),
	"Crusader's Crossbow":  names("secondary-ammo"),
	"Degreaser":            names("fire-rate"),
	"Detonator":            names("accuracy", "clip-size"),
	"Direct Hit":           names("accuracy"),
	"Dragon's Fury":        names("accuracy"),
	"Flame Thrower":        names("fire-rate"),
	"Flare Gun":            names("accuracy", "clip-size"),
	"Flying Guillotine":    names("accuracy"),
	"Grenade Launcher":     names("accuracy"),
	"Hitman's Heatmaker":   names("accuracy", "fire-rate"),
	"Huntsman":             names("accuracy", "fire-rate"),
	"Iron Bomber":          names("accuracy"),
	"Kritzkrieg":           names("uber-on-hit"),
	"Liberty Launcher":     names("accuracy"),
	"Loch-n-Load":          names("accuracy"),
	"Loose Cannon":         names("accuracy"),
	"Machina":              names("accuracy", "fire-rate"),
	"Manmelter":            names("accuracy", "clip-size"),
	"Medi Gun":             names("uber-on-hit"),
	"Overdose":             names("secondary-ammo"),
	"Panic Attack":         names("reload-rate"),
	"Phlogistinator":       names("fire-rate"),
	"Pomson 6000":          names("accuracy", "max-ammo", "projectile-range"),
	"Quick-Fix":            names("uber-on-hit"),
	"Quickiebomb Launcher": names("accuracy"),
	"Rescue Ranger":        names("accuracy", "projectile-range", "projectile-speed"),
	"Righteous Bison":      names("accuracy", "max-ammo", "secondary-ammo"),
	"Rocket Jumper":        names("accuracy", "airborne-crits", "crits-on-kill", "crits-vs-burning", "damage", "drop-health-pack", "heal-on-kill", "minicrits-on-kill", "minicrits-to-crits", "speed-on-kill"),
	"Rocket Launcher":      names("accuracy"),
	"Scorch Shot":          names("accuracy", "clip-size"),
	"Scottish Resistance":  names("accuracy"),
	"Short Circuit":        names("accuracy", "clip-size", "max-ammo", "projectile-count", "reload-rate", "secondary-ammo"),
	"Sniper Rifle":         names("accuracy", "fire-rate"),
	"Splendid Screen":      names("deploy-speed"),
	"Sticky Jumper":        names("accuracy", "airborne-crits", "airborne-minicrits", "crits-on-kill", "crits-vs-burning", "damage", "drop-health-pack", "heal-on-kill", "mad-milk", "minicrits-on-kill", "minicrits-to-crits", "speed-on-kill"),
	"Stickybomb Launcher":  names("accuracy"),
	"Sydney Sleeper":       names("accuracy", "fire-rate"),
	"Syringe Gun":          names("secondary-ammo"),
	"Tide Turner":          names("deploy-speed"),
	"Vaccinator":           names("uber-on-hit"),
	"Widowmaker":           names("clip-size", "max-ammo", "reload-rate"),
}

func cutBySheet(name, key string) bool {
	return sheetCuts[name][key]
}
