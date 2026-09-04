package gamedata

import "slices"

// weaponFamilies are mechanically identical item definitions. The first name
// owns the reward pool; every member resolves to that pool in the plugin. This
// keeps a reward useful when a player swaps a stock weapon for its promotional
// or cosmetic reskin.
var weaponFamilies = [][]string{
	{"Sapper", "Ap-Sap", "Snack Attack"},
	{"Minigun", "Iron Curtain", "Reissued Iron Curtain", "Deflector"},
	{"Pistol", "Lugermorph", "Reissued Lugermorph", "C.A.P.P.E.R"},
	{"Invis Watch", "Enthusiast's Timepiece", "Reissued Enthusiast's Timepiece", "Quäckenbirdt"},
	{"Sandvich", "Robo-Sandvich"},
	{"Flame Thrower", "Rainblower", "Nostromo Napalmer"},
	{"Rocket Launcher", "Original"},
	{"Huntsman", "Fortified Compound"},
	{"Mad Milk", "Mutated Milk"},
	{"Flying Guillotine", "Self-Aware Beauty Mark"},
	{"Sniper Rifle", "AWPer Hand"},
	{"Machina", "Shooting Star"},
	{"Revolver", "Big Kill"},
	{"Axtinguisher", "Postal Pummeler"},
	{"Homewrecker", "Maul"},
	{"Fire Axe", "Lollichop"},
	{"Eyelander", "Nessie's Nine Iron", "Horseless Headless Horsemann's Headtaker"},
	{"Bottle", "Scottish Handshake"},
	{"Bat", "Holy Mackerel", "Unarmed Combat", "Batsaber"},
	{"Knife", "Black Rose", "Sharp Dresser"},
	{"Your Eternal Reward", "Wanga Prick"},
	{"Wrench", "Golden Wrench"},
	{"Fists", "Apoco-Fists"},
	{"Gloves of Running Urgently", "Bread Bite"},
	{"Dalokohs Bar", "Fishcake"},
	// These all-class promotional melees inherit the same stock-melee
	// behavior for the class holding them.
	{"Saxxy", "Bat Outta Hell", "Conscientious Objector", "Crossing Guard", "Freedom Staff", "Frying Pan", "Golden Frying Pan", "Ham Shank", "Memory Maker", "Necro Smasher", "Prinny Machete"},
	{"Wrangler", "Giger Counter"},
}

func mergeWeaponFamilies(weapons []BuffWeapon) {
	byName := make(map[string]int, len(weapons))
	for index := range weapons {
		byName[weapons[index].Name] = index
	}
	for _, family := range weaponFamilies {
		canonicalIndex, ok := byName[family[0]]
		if !ok {
			panic("unknown canonical weapon family member: " + family[0])
		}
		canonicalID := weapons[canonicalIndex].ID
		for _, name := range family {
			index, found := byName[name]
			if !found {
				panic("unknown weapon family member: " + name)
			}
			weapons[index].ApplyID = canonicalID
			for _, definition := range weapons[index].DefIndexes {
				if !slices.Contains(weapons[canonicalIndex].DefIndexes, definition) {
					weapons[canonicalIndex].DefIndexes = append(weapons[canonicalIndex].DefIndexes, definition)
				}
			}
		}
		slices.Sort(weapons[canonicalIndex].DefIndexes)
	}
}

func names(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

var passiveWeapons = names(
	"Ali Baba's Wee Booties", "B.A.S.E. Jumper", "Cozy Camper",
	"Darwin's Danger Shield", "Gunboats", "Mantreads", "Razorback",
)

var nonAttackingWeapons = names(
	"Battalion's Backup", "Bonk! Atomic Punch", "Buff Banner", "Buffalo Steak Sandvich",
	"Chargin' Targe", "Cloak and Dagger", "Concheror", "Construction PDA",
	"Crit-a-Cola", "Dalokohs Bar", "Dead Ringer", "Destruction PDA", "Disguise Kit",
	"Gas Passer", "Invis Watch", "Jarate", "Kritzkrieg", "Mad Milk", "Medi Gun", "PDA",
	"Quick-Fix", "Red-Tape Recorder", "Sandvich", "Sapper", "Second Banana", "Splendid Screen",
	"Thermal Thruster", "Tide Turner", "Vaccinator", "Wrangler",
)

var attackRequiredEffects = names(
	"damage", "fire-rate", "reload-rate", "clip-size", "projectile-speed", "projectile-count",
	"heal-on-kill", "crits-on-kill", "bleed", "ignite", "afterburn-damage",
	"afterburn-duration", "airborne-crits", "airborne-minicrits", "mad-milk", "gasoline",
	"mark-for-death", "explosive-shots", "projectile-penetration", "destroy-projectiles",
	"knockback", "heal-on-hit", "speed-on-kill", "max-ammo", "armor-piercing",
	"cloak-on-hit", "cloak-on-kill", "crits-vs-burning", "back-crits", "sniper-charge",
	"pellet-count", "aiming-speed", "secondary-ammo", "max-stickies", "blast-radius",
	"projectile-range", "accuracy", "flame-size", "flame-range", "jarate",
	"minicrits-to-crits", "no-self-blast", "drop-health-pack", "slow-on-hit", "ammo-on-hit",
	"minicrits-on-kill", "rocket-jump-protection", "self-blast-force", "spinup-speed",
	"reveal-cloaked", "reveal-disguised", "speed-on-hit", "ammo-regen",
)

var (
	banners         = names("Battalion's Backup", "Buff Banner", "Concheror")
	mediguns        = names("Kritzkrieg", "Medi Gun", "Quick-Fix", "Vaccinator")
	airblastWeapons = names("Backburner", "Degreaser", "Dragon's Fury", "Flame Thrower")
	flamethrowers   = names("Backburner", "Degreaser", "Dragon's Fury", "Flame Thrower", "Phlogistinator")
	fireWeapons     = names(
		"Axtinguisher", "Backburner", "Degreaser", "Detonator", "Dragon's Fury", "Flame Thrower",
		"Flare Gun", "Manmelter", "Phlogistinator", "Scorch Shot", "Sharpened Volcano Fragment",
	)
)

var (
	miniguns         = names("Brass Beast", "Huo-Long Heater", "Minigun", "Natascha", "Tomislav")
	sniperRifles     = names("Bazaar Bargain", "Classic", "Hitman's Heatmaker", "Machina", "Sniper Rifle", "Sydney Sleeper")
	stickyLaunchers  = names("Quickiebomb Launcher", "Scottish Resistance", "Sticky Jumper", "Stickybomb Launcher")
	explosiveWeapons = names(
		"Air Strike", "Beggar's Bazooka", "Black Box", "Cow Mangler 5000", "Detonator",
		"Direct Hit", "Grenade Launcher", "Iron Bomber", "Liberty Launcher", "Loch-n-Load",
		"Loose Cannon", "Quickiebomb Launcher", "Rocket Jumper", "Rocket Launcher", "Scorch Shot",
		"Scottish Resistance", "Sticky Jumper", "Stickybomb Launcher", "Ullapool Caber",
	)
)

var engineerWeapons = names(
	"Construction PDA", "Destruction PDA", "Eureka Effect", "Frontier Justice", "Gunslinger",
	"Jag", "Panic Attack", "PDA", "Pistol", "Pomson 6000", "Rescue Ranger", "Short Circuit",
	"Shotgun", "Southern Hospitality", "Widowmaker", "Wrangler", "Wrench",
)

// spyKnives are the weapons a backstab comes from, which is the only hit the
// game reads armor piercing on.
var spyKnives = names("Big Earner", "Conniver's Kunai", "Knife", "Spy-Cicle", "Your Eternal Reward")

var (
	watches          = names("Cloak and Dagger", "Dead Ringer", "Invis Watch")
	spyAttackWeapons = names(
		"Ambassador", "Big Earner", "Diamondback", "Enforcer", "Knife", "L'Étranger",
		"Revolver", "Spy-Cicle", "Your Eternal Reward",
	)
)

var meterWeapons = names(
	"Battalion's Backup", "Bonk! Atomic Punch", "Buff Banner", "Concheror", "Crit-a-Cola",
	"Dalokohs Bar", "Flying Guillotine", "Gas Passer", "Jarate", "Mad Milk", "Sandman",
	"Sandvich", "Second Banana", "Wrap Assassin",
)

var consumables = names(
	"Bonk! Atomic Punch", "Buffalo Steak Sandvich", "Crit-a-Cola", "Dalokohs Bar", "Gas Passer",
	"Jarate", "Mad Milk", "Sandvich", "Second Banana",
)

var meleeWeapons = names(
	"Amputator", "Atomizer", "Axtinguisher", "Back Scratcher", "Bat", "Big Earner", "Bonesaw",
	"Boston Basher", "Bottle", "Bushwacka", "Candy Cane", "Claidheamh Mòr", "Conniver's Kunai",
	"Disciplinary Action", "Equalizer", "Escape Plan", "Eureka Effect", "Eviction Notice",
	"Eyelander", "Fan O'War", "Fire Axe", "Fists", "Fists of Steel", "Gloves of Running Urgently",
	"Gunslinger", "Half-Zatoichi", "Holiday Punch", "Homewrecker", "Hot Hand", "Jag",
	"Killing Gloves of Boxing", "Knife", "Kukri", "Market Gardener", "Neon Annihilator",
	"Pain Train", "Persian Persuader", "Powerjack", "Saxxy", "Sandman", "Scotsman's Skullcutter",
	"Shahanshah", "Sharpened Volcano Fragment", "Shovel", "Solemn Vow", "Southern Hospitality",
	"Spy-Cicle", "Sun-on-a-Stick", "Third Degree", "Three-Rune Blade", "Tribalman's Shiv",
	"Ullapool Caber", "Vita-Saw", "Warrior's Spirit", "Wrap Assassin", "Wrench",
	"Your Eternal Reward", "Übersaw",
)

var rangedOnlyEffects = names(
	"reload-rate", "clip-size", "destroy-projectiles", "max-ammo", "secondary-ammo",
	"aiming-speed", "max-stickies", "blast-radius", "accuracy", "ammo-regen",
)

var projectileWeapons = names(
	"Air Strike", "Beggar's Bazooka", "Black Box", "Blutsauger", "Cow Mangler 5000",
	"Crusader's Crossbow", "Detonator", "Direct Hit", "Dragon's Fury", "Flare Gun",
	"Flying Guillotine", "Gas Passer", "Grenade Launcher", "Huntsman", "Iron Bomber", "Jarate",
	"Liberty Launcher", "Loch-n-Load", "Loose Cannon", "Mad Milk", "Manmelter", "Pomson 6000",
	"Quickiebomb Launcher", "Rescue Ranger", "Righteous Bison", "Rocket Jumper", "Rocket Launcher",
	"Sandman", "Scorch Shot", "Scottish Resistance", "Short Circuit", "Sticky Jumper",
	"Stickybomb Launcher", "Syringe Gun", "Wrap Assassin",
)

var projectileCountExtras = names(
	"Gas Passer", "Jarate", "Mad Milk", "Sandman", "Wrap Assassin",
)

var (
	thrownSubstances     = names("Jarate", "Mad Milk")
	substanceEffects     = names("bleed", "mad-milk", "mark-for-death", "jarate")
	jarProjectileEffects = names(
		"projectile-count", "projectile-speed", "projectile-range", "projectile-penetration",
	)
)

var cliplessWeapons = names(
	"Backburner", "Bazaar Bargain", "Brass Beast", "Classic", "Degreaser", "Dragon's Fury",
	"Flame Thrower", "Hitman's Heatmaker", "Huo-Long Heater", "Huntsman", "Machina", "Minigun",
	"Natascha", "Phlogistinator", "Sniper Rifle", "Sydney Sleeper", "Tomislav",
)

func weaponEffectEligible(weapon BuffWeapon, effect WeaponEffect) bool {
	name, key := weapon.Name, effect.Key
	if decided, eligible := eligibilityByShape(name, key); decided {
		return eligible
	}
	switch key {
	case "clip-size", "reload-rate":
		return !cliplessWeapons[name]
	case "pellet-count":
		return meleeWeapons[name]
	case "explosive-shots":
		return sniperRifles[name]
	case "banner-duration":
		return banners[name]
	case "healing", "healing-received", "uber-rate", "uber-on-hit", "uber-duration":
		return mediguns[name]
	case "airblast-power", "airblast-rate", "charged-airblast", "airblast-cost":
		return airblastWeapons[name]
	case "building-health", "sentry-fire-rate", "disposable-sentry", "metal-regen", "max-metal", "construction-rate", "repair-rate":
		return engineerWeapons[name]
	case "flame-size", "flame-range", "back-crits":
		return flamethrowers[name]
	case "afterburn-damage", "afterburn-duration":
		return fireWeapons[name]
	case "sniper-charge", "jarate":
		return sniperRifles[name]
	case "aiming-speed", "spinup-speed":
		return miniguns[name]
	case "max-stickies":
		return stickyLaunchers[name]
	case "blast-radius", "no-self-blast", "rocket-jump-protection", "self-blast-force":
		return explosiveWeapons[name]
	case "cloak-duration", "cloak-regen":
		return watches[name]
	case "cloak-on-hit", "cloak-on-kill":
		return spyAttackWeapons[name]
	case "armor-piercing":
		return spyKnives[name]
	case "meter-recharge":
		return meterWeapons[name]
	case "gesture-speed":
		return consumables[name]
	default:
		return true
	}
}

/*
eligibilityByShape answers what the weapon's shape decides before its mechanic
is looked at, and says whether it answered.

Passives draw nothing. Explode on ignite is out of the pool on every weapon: on
a minigun it ended a wave on its own, and the effect keeps its ID because seeds
hold it. Jars do not perform weapon attacks, so most item attributes are inert
on them; their pool is their thrown projectile, their recharge meter and the
splash substances the plugin's PlayerJarated hook applies. Projectile count is
implemented for bullets and entities alike, and the thrown meters and the two
projectile melees are the deliberate exceptions to the non-attacking and melee
filters. A rocket that penetrates does not explode where it was aimed, so
penetration stays off explosives and on arrows, syringes and flares.
*/
func eligibilityByShape(name, key string) (decided, eligible bool) {
	switch {
	case passiveWeapons[name], key == "gasoline":
		return true, false
	case thrownSubstances[name]:
		return true, jarProjectileEffects[key] || substanceEffects[key] || key == "meter-recharge"
	case key == "projectile-count":
		return true, projectileCountExtras[name] ||
			(!nonAttackingWeapons[name] && !meleeWeapons[name] && (!flamethrowers[name] || name == "Dragon's Fury"))
	case key == "projectile-speed", key == "projectile-range":
		return true, projectileWeapons[name]
	case key == "projectile-penetration":
		return true, projectileWeapons[name] && !explosiveWeapons[name]
	case nonAttackingWeapons[name] && attackRequiredEffects[key],
		meleeWeapons[name] && rangedOnlyEffects[key]:
		return true, false
	}
	return false, false
}
