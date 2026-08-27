// Initial buff catalog imported from TF2's item schema. IDs and keys are append-only.
package gamedata

// legacyWeaponBuffs is the first shipped one-buff-per-weapon catalog. Its IDs
// remain the first permutation for each weapon; weapon_buffs.go expands this
// into the complete weapon x positive-effect catalog.
var legacyWeaponBuffs = []WeaponBuff{
	{
		ID: 1, Key: "weapon-001", Weapon: "Air Strike",
		DefIndexes: []int{1104}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 2, Key: "weapon-002", Weapon: "Ali Baba's Wee Booties",
		DefIndexes: []int{405}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 3, Key: "weapon-003", Weapon: "Ambassador",
		DefIndexes: []int{61, 1006, 15061, 15068, 15069, 15073, 15088}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 4, Key: "weapon-004", Weapon: "Amputator",
		DefIndexes: []int{304}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 5, Key: "weapon-005", Weapon: "Ap-Sap",
		DefIndexes: []int{933}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 6, Key: "weapon-006", Weapon: "Apoco-Fists",
		DefIndexes: []int{587}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 7, Key: "weapon-007", Weapon: "Atomizer",
		DefIndexes: []int{450}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 8, Key: "weapon-008", Weapon: "AWPer Hand",
		DefIndexes: []int{851}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 9, Key: "weapon-009", Weapon: "Axtinguisher",
		DefIndexes: []int{38, 1000, 15038}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 10, Key: "weapon-010", Weapon: "B.A.S.E. Jumper",
		DefIndexes: []int{1101}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 11, Key: "weapon-011", Weapon: "Baby Face's Blaster",
		DefIndexes: []int{772}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 12, Key: "weapon-012", Weapon: "Back Scatter",
		DefIndexes: []int{1103}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 13, Key: "weapon-013", Weapon: "Back Scratcher",
		DefIndexes: []int{326}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 14, Key: "weapon-014", Weapon: "Backburner",
		DefIndexes: []int{40, 1146, 15040}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 15, Key: "weapon-015", Weapon: "Bat",
		DefIndexes: []int{0, 190, 660}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 16, Key: "weapon-016", Weapon: "Bat Outta Hell",
		DefIndexes: []int{939}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 17, Key: "weapon-017", Weapon: "Battalion's Backup",
		DefIndexes: []int{226}, Attribute: "increase buff duration", Value: 1.20,
		Description: "+20% banner duration", Additive: false,
	},
	{
		ID: 18, Key: "weapon-018", Weapon: "Bazaar Bargain",
		DefIndexes: []int{402}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 19, Key: "weapon-019", Weapon: "Beggar's Bazooka",
		DefIndexes: []int{730}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 20, Key: "weapon-020", Weapon: "Big Earner",
		DefIndexes: []int{461}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 21, Key: "weapon-021", Weapon: "Big Kill",
		DefIndexes: []int{161}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 22, Key: "weapon-022", Weapon: "Black Box",
		DefIndexes: []int{228, 1085}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 23, Key: "weapon-023", Weapon: "Black Rose",
		DefIndexes: []int{727}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 24, Key: "weapon-024", Weapon: "Blutsauger",
		DefIndexes: []int{36, 15036}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 25, Key: "weapon-025", Weapon: "Bonesaw",
		DefIndexes: []int{8, 198, 1143, 15009}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 26, Key: "weapon-026", Weapon: "Bonk! Atomic Punch",
		DefIndexes: []int{46, 1145, 15046}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 27, Key: "weapon-027", Weapon: "Bootlegger",
		DefIndexes: []int{608}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 28, Key: "weapon-028", Weapon: "Boston Basher",
		DefIndexes: []int{325}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 29, Key: "weapon-029", Weapon: "Bottle",
		DefIndexes: []int{1, 191, 15014}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 30, Key: "weapon-030", Weapon: "Brass Beast",
		DefIndexes: []int{312}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 31, Key: "weapon-031", Weapon: "Bread Bite",
		DefIndexes: []int{1100}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 32, Key: "weapon-032", Weapon: "Buff Banner",
		DefIndexes: []int{129, 1001}, Attribute: "increase buff duration", Value: 1.20,
		Description: "+20% banner duration", Additive: false,
	},
	{
		ID: 33, Key: "weapon-033", Weapon: "Buffalo Steak Sandvich",
		DefIndexes: []int{311}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 34, Key: "weapon-034", Weapon: "Bushwacka",
		DefIndexes: []int{232}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 35, Key: "weapon-035", Weapon: "Candy Cane",
		DefIndexes: []int{317}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 36, Key: "weapon-036", Weapon: "Chargin' Targe",
		DefIndexes: []int{131, 1144}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 37, Key: "weapon-037", Weapon: "Claidheamh Mòr",
		DefIndexes: []int{327}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 38, Key: "weapon-038", Weapon: "Classic",
		DefIndexes: []int{1098}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 39, Key: "weapon-039", Weapon: "Cleaner's Carbine",
		DefIndexes: []int{751}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 40, Key: "weapon-040", Weapon: "Cloak and Dagger",
		DefIndexes: []int{60, 15066, 15074, 15077, 15081, 15082, 15085}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 41, Key: "weapon-041", Weapon: "Concheror",
		DefIndexes: []int{354}, Attribute: "increase buff duration", Value: 1.20,
		Description: "+20% banner duration", Additive: false,
	},
	{
		ID: 42, Key: "weapon-042", Weapon: "Conniver's Kunai",
		DefIndexes: []int{356}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 43, Key: "weapon-043", Weapon: "Conscientious Objector",
		DefIndexes: []int{474}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 44, Key: "weapon-044", Weapon: "Construction PDA",
		DefIndexes: []int{25, 737, 15025}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 45, Key: "weapon-045", Weapon: "Cow Mangler 5000",
		DefIndexes: []int{441}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 46, Key: "weapon-046", Weapon: "Cozy Camper",
		DefIndexes: []int{642}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 47, Key: "weapon-047", Weapon: "Crit-a-Cola",
		DefIndexes: []int{163}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 48, Key: "weapon-048", Weapon: "Crossing Guard",
		DefIndexes: []int{1127}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 49, Key: "weapon-049", Weapon: "Crusader's Crossbow",
		DefIndexes: []int{305, 1079}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 50, Key: "weapon-050", Weapon: "Dalokohs Bar",
		DefIndexes: []int{159}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 51, Key: "weapon-051", Weapon: "Darwin's Danger Shield",
		DefIndexes: []int{231}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 52, Key: "weapon-052", Weapon: "Dead Ringer",
		DefIndexes: []int{59, 15059}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 53, Key: "weapon-053", Weapon: "Deflector",
		DefIndexes: []int{850}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 54, Key: "weapon-054", Weapon: "Degreaser",
		DefIndexes: []int{215}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 55, Key: "weapon-055", Weapon: "Destruction PDA",
		DefIndexes: []int{26, 15026}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 56, Key: "weapon-056", Weapon: "Detonator",
		DefIndexes: []int{351}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 57, Key: "weapon-057", Weapon: "Diamondback",
		DefIndexes: []int{525}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 58, Key: "weapon-058", Weapon: "Direct Hit",
		DefIndexes: []int{127}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 59, Key: "weapon-059", Weapon: "Disciplinary Action",
		DefIndexes: []int{447}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 60, Key: "weapon-060", Weapon: "Disguise Kit",
		DefIndexes: []int{27, 15027}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 61, Key: "weapon-061", Weapon: "Dragon's Fury",
		DefIndexes: []int{1178}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 62, Key: "weapon-062", Weapon: "Enforcer",
		DefIndexes: []int{460}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 63, Key: "weapon-063", Weapon: "Enthusiast's Timepiece",
		DefIndexes: []int{297}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 64, Key: "weapon-064", Weapon: "Equalizer",
		DefIndexes: []int{128}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 65, Key: "weapon-065", Weapon: "Escape Plan",
		DefIndexes: []int{775}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 66, Key: "weapon-066", Weapon: "Eureka Effect",
		DefIndexes: []int{589}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 67, Key: "weapon-067", Weapon: "Eviction Notice",
		DefIndexes: []int{426}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 68, Key: "weapon-068", Weapon: "Eyelander",
		DefIndexes: []int{132, 1082}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 69, Key: "weapon-069", Weapon: "Family Business",
		DefIndexes: []int{425}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 70, Key: "weapon-070", Weapon: "Fan O'War",
		DefIndexes: []int{355}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 71, Key: "weapon-071", Weapon: "Fire Axe",
		DefIndexes: []int{2, 192, 15010}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 72, Key: "weapon-072", Weapon: "Fishcake",
		DefIndexes: []int{433}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 73, Key: "weapon-073", Weapon: "Fists",
		DefIndexes: []int{5, 195, 15008}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 74, Key: "weapon-074", Weapon: "Fists of Steel",
		DefIndexes: []int{331}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 75, Key: "weapon-075", Weapon: "Flame Thrower",
		DefIndexes: []int{21, 208, 659, 798, 807, 887, 896, 905, 914, 963, 972, 15021}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 76, Key: "weapon-076", Weapon: "Flare Gun",
		DefIndexes: []int{39, 1081, 15039}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 77, Key: "weapon-077", Weapon: "Flying Guillotine",
		DefIndexes: []int{812, 833}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 78, Key: "weapon-078", Weapon: "Force-a-Nature",
		DefIndexes: []int{45, 1078, 15045}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 79, Key: "weapon-079", Weapon: "Fortified Compound",
		DefIndexes: []int{1092}, Attribute: "Projectile speed increased", Value: 1.15,
		Description: "+15% projectile speed", Additive: false,
	},
	{
		ID: 80, Key: "weapon-080", Weapon: "Freedom Staff",
		DefIndexes: []int{880}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 81, Key: "weapon-081", Weapon: "Frontier Justice",
		DefIndexes: []int{141, 1004}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 82, Key: "weapon-082", Weapon: "Frying Pan",
		DefIndexes: []int{264}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 83, Key: "weapon-083", Weapon: "Gas Passer",
		DefIndexes: []int{1180}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 84, Key: "weapon-084", Weapon: "Gloves of Running Urgently",
		DefIndexes: []int{239, 1084, 1184}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 85, Key: "weapon-085", Weapon: "Golden Frying Pan",
		DefIndexes: []int{1071}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 86, Key: "weapon-086", Weapon: "Golden Wrench",
		DefIndexes: []int{169}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 87, Key: "weapon-087", Weapon: "Grenade Launcher",
		DefIndexes: []int{19, 206, 1007, 15019}, Attribute: "Projectile speed increased", Value: 1.15,
		Description: "+15% projectile speed", Additive: false,
	},
	{
		ID: 88, Key: "weapon-088", Weapon: "Gunboats",
		DefIndexes: []int{133}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 89, Key: "weapon-089", Weapon: "Gunslinger",
		DefIndexes: []int{142}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 90, Key: "weapon-090", Weapon: "Half-Zatoichi",
		DefIndexes: []int{357}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 91, Key: "weapon-091", Weapon: "Ham Shank",
		DefIndexes: []int{1013}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 92, Key: "weapon-092", Weapon: "Hitman's Heatmaker",
		DefIndexes: []int{752}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 93, Key: "weapon-093", Weapon: "Holiday Punch",
		DefIndexes: []int{656}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 94, Key: "weapon-094", Weapon: "Holy Mackerel",
		DefIndexes: []int{221, 999}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 95, Key: "weapon-095", Weapon: "Homewrecker",
		DefIndexes: []int{153}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 96, Key: "weapon-096", Weapon: "Horseless Headless Horsemann's Headtaker",
		DefIndexes: []int{266}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 97, Key: "weapon-097", Weapon: "Hot Hand",
		DefIndexes: []int{1181}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 98, Key: "weapon-098", Weapon: "Huntsman",
		DefIndexes: []int{56, 1005, 15056}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 99, Key: "weapon-099", Weapon: "Huo-Long Heater",
		DefIndexes: []int{811, 832}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 100, Key: "weapon-100", Weapon: "Invis Watch",
		DefIndexes: []int{30, 212, 15030}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 101, Key: "weapon-101", Weapon: "Iron Bomber",
		DefIndexes: []int{1151}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 102, Key: "weapon-102", Weapon: "Iron Curtain",
		DefIndexes: []int{298}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 103, Key: "weapon-103", Weapon: "Jag",
		DefIndexes: []int{329}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 104, Key: "weapon-104", Weapon: "Jarate",
		DefIndexes: []int{58, 1083, 15058}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 105, Key: "weapon-105", Weapon: "Killing Gloves of Boxing",
		DefIndexes: []int{43, 15043}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 106, Key: "weapon-106", Weapon: "Knife",
		DefIndexes: []int{4, 194, 665, 794, 803, 883, 892, 901, 910, 959, 968, 15012}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 107, Key: "weapon-107", Weapon: "Kritzkrieg",
		DefIndexes: []int{35, 15035}, Attribute: "heal rate bonus", Value: 1.10,
		Description: "+10% healing", Additive: false,
	},
	{
		ID: 108, Key: "weapon-108", Weapon: "Kukri",
		DefIndexes: []int{3, 193, 15011}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 109, Key: "weapon-109", Weapon: "L'Étranger",
		DefIndexes: []int{224}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 110, Key: "weapon-110", Weapon: "Liberty Launcher",
		DefIndexes: []int{414}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 111, Key: "weapon-111", Weapon: "Loch-n-Load",
		DefIndexes: []int{308}, Attribute: "Projectile speed increased", Value: 1.15,
		Description: "+15% projectile speed", Additive: false,
	},
	{
		ID: 112, Key: "weapon-112", Weapon: "Lollichop",
		DefIndexes: []int{739}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 113, Key: "weapon-113", Weapon: "Loose Cannon",
		DefIndexes: []int{996}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 114, Key: "weapon-114", Weapon: "Lugermorph",
		DefIndexes: []int{160, 294}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 115, Key: "weapon-115", Weapon: "Machina",
		DefIndexes: []int{526}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 116, Key: "weapon-116", Weapon: "Mad Milk",
		DefIndexes: []int{222}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 117, Key: "weapon-117", Weapon: "Manmelter",
		DefIndexes: []int{595}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 118, Key: "weapon-118", Weapon: "Mantreads",
		DefIndexes: []int{444}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 119, Key: "weapon-119", Weapon: "Market Gardener",
		DefIndexes: []int{416}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 120, Key: "weapon-120", Weapon: "Maul",
		DefIndexes: []int{466}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 121, Key: "weapon-121", Weapon: "Medi Gun",
		DefIndexes: []int{29, 211, 663, 796, 805, 885, 894, 903, 912, 961, 970, 15029}, Attribute: "heal rate bonus", Value: 1.10,
		Description: "+10% healing", Additive: false,
	},
	{
		ID: 122, Key: "weapon-122", Weapon: "Memory Maker",
		DefIndexes: []int{954}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 123, Key: "weapon-123", Weapon: "Minigun",
		DefIndexes: []int{15, 202, 654, 793, 802, 882, 891, 900, 909, 958, 967, 15015}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 124, Key: "weapon-124", Weapon: "Mutated Milk",
		DefIndexes: []int{1121}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 125, Key: "weapon-125", Weapon: "Natascha",
		DefIndexes: []int{41, 15041}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 126, Key: "weapon-126", Weapon: "Necro Smasher",
		DefIndexes: []int{1123}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 127, Key: "weapon-127", Weapon: "Neon Annihilator",
		DefIndexes: []int{813, 834}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 128, Key: "weapon-128", Weapon: "Nessie's Nine Iron",
		DefIndexes: []int{482}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 129, Key: "weapon-129", Weapon: "Original",
		DefIndexes: []int{513}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 130, Key: "weapon-130", Weapon: "Overdose",
		DefIndexes: []int{412}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 131, Key: "weapon-131", Weapon: "Pain Train",
		DefIndexes: []int{154}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 132, Key: "weapon-132", Weapon: "Panic Attack",
		DefIndexes: []int{1153}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 133, Key: "weapon-133", Weapon: "PDA",
		DefIndexes: []int{28, 15028}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 134, Key: "weapon-134", Weapon: "Persian Persuader",
		DefIndexes: []int{404}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 135, Key: "weapon-135", Weapon: "Phlogistinator",
		DefIndexes: []int{594}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 136, Key: "weapon-136", Weapon: "Pistol",
		DefIndexes: []int{22, 23, 209, 15013, 15022, 15023}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 137, Key: "weapon-137", Weapon: "Pomson 6000",
		DefIndexes: []int{588}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 138, Key: "weapon-138", Weapon: "Postal Pummeler",
		DefIndexes: []int{457}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 139, Key: "weapon-139", Weapon: "Powerjack",
		DefIndexes: []int{214}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 140, Key: "weapon-140", Weapon: "Pretty Boy's Pocket Pistol",
		DefIndexes: []int{773}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 141, Key: "weapon-141", Weapon: "Quick-Fix",
		DefIndexes: []int{411}, Attribute: "heal rate bonus", Value: 1.10,
		Description: "+10% healing", Additive: false,
	},
	{
		ID: 142, Key: "weapon-142", Weapon: "Quickiebomb Launcher",
		DefIndexes: []int{1150}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 143, Key: "weapon-143", Weapon: "Quäckenbirdt",
		DefIndexes: []int{947}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 144, Key: "weapon-144", Weapon: "Rainblower",
		DefIndexes: []int{741}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 145, Key: "weapon-145", Weapon: "Razorback",
		DefIndexes: []int{57, 15057}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 146, Key: "weapon-146", Weapon: "Red-Tape Recorder",
		DefIndexes: []int{810, 831}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 147, Key: "weapon-147", Weapon: "Reissued Enthusiast's Timepiece",
		DefIndexes: []int{1205}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 148, Key: "weapon-148", Weapon: "Reissued Iron Curtain",
		DefIndexes: []int{1206}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 149, Key: "weapon-149", Weapon: "Reissued Lugermorph",
		DefIndexes: []int{1202}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 150, Key: "weapon-150", Weapon: "Rescue Ranger",
		DefIndexes: []int{997}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 151, Key: "weapon-151", Weapon: "Reserve Shooter",
		DefIndexes: []int{415}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 152, Key: "weapon-152", Weapon: "Revolver",
		DefIndexes: []int{24, 210, 1142, 15024}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 153, Key: "weapon-153", Weapon: "Righteous Bison",
		DefIndexes: []int{442}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 154, Key: "weapon-154", Weapon: "Robo-Sandvich",
		DefIndexes: []int{863}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 155, Key: "weapon-155", Weapon: "Rocket Jumper",
		DefIndexes: []int{237}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 156, Key: "weapon-156", Weapon: "Rocket Launcher",
		DefIndexes: []int{18, 205, 658, 800, 809, 889, 898, 907, 916, 965, 974, 15018}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 157, Key: "weapon-157", Weapon: "Sandman",
		DefIndexes: []int{44, 15044}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 158, Key: "weapon-158", Weapon: "Sandvich",
		DefIndexes: []int{42, 1002, 15042}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 159, Key: "weapon-159", Weapon: "Sapper",
		DefIndexes: []int{735, 736, 1080}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 160, Key: "weapon-160", Weapon: "Saxxy",
		DefIndexes: []int{423}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 161, Key: "weapon-161", Weapon: "Scattergun",
		DefIndexes: []int{13, 200, 669, 799, 808, 888, 897, 906, 915, 964, 973, 15001}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 162, Key: "weapon-162", Weapon: "Scorch Shot",
		DefIndexes: []int{740}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 163, Key: "weapon-163", Weapon: "Scotsman's Skullcutter",
		DefIndexes: []int{172}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 164, Key: "weapon-164", Weapon: "Scottish Handshake",
		DefIndexes: []int{609}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 165, Key: "weapon-165", Weapon: "Scottish Resistance",
		DefIndexes: []int{130}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 166, Key: "weapon-166", Weapon: "Second Banana",
		DefIndexes: []int{1190}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 167, Key: "weapon-167", Weapon: "Self-Aware Beauty Mark",
		DefIndexes: []int{1105}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 168, Key: "weapon-168", Weapon: "Shahanshah",
		DefIndexes: []int{401}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 169, Key: "weapon-169", Weapon: "Sharp Dresser",
		DefIndexes: []int{638}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 170, Key: "weapon-170", Weapon: "Sharpened Volcano Fragment",
		DefIndexes: []int{348}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 171, Key: "weapon-171", Weapon: "Short Circuit",
		DefIndexes: []int{528}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 172, Key: "weapon-172", Weapon: "Shortstop",
		DefIndexes: []int{220}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 173, Key: "weapon-173", Weapon: "Shotgun",
		DefIndexes: []int{9, 10, 11, 12, 199, 1141, 15002, 15003, 15004, 15005}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 174, Key: "weapon-174", Weapon: "Shovel",
		DefIndexes: []int{6, 196, 15006}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 175, Key: "weapon-175", Weapon: "SMG",
		DefIndexes: []int{16, 203, 1149, 15016}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 176, Key: "weapon-176", Weapon: "Snack Attack",
		DefIndexes: []int{1102}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 177, Key: "weapon-177", Weapon: "Sniper Rifle",
		DefIndexes: []int{14, 201, 664, 792, 801, 881, 890, 899, 908, 957, 966, 15000}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 178, Key: "weapon-178", Weapon: "Soda Popper",
		DefIndexes: []int{448}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 179, Key: "weapon-179", Weapon: "Solemn Vow",
		DefIndexes: []int{413}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 180, Key: "weapon-180", Weapon: "Southern Hospitality",
		DefIndexes: []int{155}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 181, Key: "weapon-181", Weapon: "Splendid Screen",
		DefIndexes: []int{406}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 182, Key: "weapon-182", Weapon: "Spy-Cicle",
		DefIndexes: []int{649}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 183, Key: "weapon-183", Weapon: "Sticky Jumper",
		DefIndexes: []int{265}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 184, Key: "weapon-184", Weapon: "Stickybomb Launcher",
		DefIndexes: []int{20, 207, 661, 797, 806, 886, 895, 904, 913, 962, 971, 15020}, Attribute: "faster reload rate", Value: 0.85,
		Description: "+15% reload speed", Additive: false,
	},
	{
		ID: 185, Key: "weapon-185", Weapon: "Sun-on-a-Stick",
		DefIndexes: []int{349}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 186, Key: "weapon-186", Weapon: "Sydney Sleeper",
		DefIndexes: []int{230}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 187, Key: "weapon-187", Weapon: "Syringe Gun",
		DefIndexes: []int{17, 204, 15017}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 188, Key: "weapon-188", Weapon: "Thermal Thruster",
		DefIndexes: []int{1179}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 189, Key: "weapon-189", Weapon: "Third Degree",
		DefIndexes: []int{593}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 190, Key: "weapon-190", Weapon: "Three-Rune Blade",
		DefIndexes: []int{452}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 191, Key: "weapon-191", Weapon: "Tide Turner",
		DefIndexes: []int{1099}, Attribute: "major move speed bonus", Value: 1.05,
		Description: "+5% movement speed while active", Additive: false,
	},
	{
		ID: 192, Key: "weapon-192", Weapon: "Tomislav",
		DefIndexes: []int{424}, Attribute: "clip size bonus", Value: 1.25,
		Description: "+25% clip size", Additive: false,
	},
	{
		ID: 193, Key: "weapon-193", Weapon: "Tribalman's Shiv",
		DefIndexes: []int{171}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 194, Key: "weapon-194", Weapon: "Ullapool Caber",
		DefIndexes: []int{307}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 195, Key: "weapon-195", Weapon: "Unarmed Combat",
		DefIndexes: []int{572}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 196, Key: "weapon-196", Weapon: "Vaccinator",
		DefIndexes: []int{998}, Attribute: "heal rate bonus", Value: 1.10,
		Description: "+10% healing", Additive: false,
	},
	{
		ID: 197, Key: "weapon-197", Weapon: "Vita-Saw",
		DefIndexes: []int{173}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 198, Key: "weapon-198", Weapon: "Wanga Prick",
		DefIndexes: []int{574}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 199, Key: "weapon-199", Weapon: "Warrior's Spirit",
		DefIndexes: []int{310}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 200, Key: "weapon-200", Weapon: "Widowmaker",
		DefIndexes: []int{527}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 201, Key: "weapon-201", Weapon: "Winger",
		DefIndexes: []int{449}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 202, Key: "weapon-202", Weapon: "Wrangler",
		DefIndexes: []int{140, 1086}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 203, Key: "weapon-203", Weapon: "Wrap Assassin",
		DefIndexes: []int{648}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 204, Key: "weapon-204", Weapon: "Wrench",
		DefIndexes: []int{7, 197, 662, 795, 804, 884, 893, 902, 911, 960, 969, 15007}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 205, Key: "weapon-205", Weapon: "Your Eternal Reward",
		DefIndexes: []int{225}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 206, Key: "weapon-206", Weapon: "Übersaw",
		DefIndexes: []int{37, 1003, 15037}, Attribute: "heal on kill", Value: 15.00,
		Description: "+15 health on kill", Additive: true,
	},
	{
		ID: 207, Key: "weapon-207", Weapon: "Batsaber",
		DefIndexes: []int{30667}, Attribute: "increased jump height from weapon", Value: 1.15,
		Description: "+15% jump height", Additive: false,
	},
	{
		ID: 208, Key: "weapon-208", Weapon: "C.A.P.P.E.R",
		DefIndexes: []int{30666}, Attribute: "critboost on kill", Value: 2.00,
		Description: "2 seconds of critical hits on kill", Additive: true,
	},
	{
		ID: 209, Key: "weapon-209", Weapon: "Giger Counter",
		DefIndexes: []int{30668}, Attribute: "fire rate bonus", Value: 0.90,
		Description: "+10% firing speed", Additive: false,
	},
	{
		ID: 210, Key: "weapon-210", Weapon: "Nostromo Napalmer",
		DefIndexes: []int{30474}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
	{
		ID: 211, Key: "weapon-211", Weapon: "Prinny Machete",
		DefIndexes: []int{30758}, Attribute: "bleeding duration", Value: 3.00,
		Description: "inflicts 3 seconds of bleed", Additive: true,
	},
	{
		ID: 212, Key: "weapon-212", Weapon: "Shooting Star",
		DefIndexes: []int{30665}, Attribute: "damage bonus", Value: 1.10,
		Description: "+10% damage", Additive: false,
	},
}
