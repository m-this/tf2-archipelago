package gamedata

import (
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestWeaponBuffCatalogPreservesEveryWeaponEffectPermutation(t *testing.T) {
	if len(BuffWeapons) < 212 {
		t.Fatalf("catalog has %d weapons, want at least 212", len(BuffWeapons))
	}
	if got, want := len(WeaponBuffs), len(BuffWeapons)*len(WeaponEffects); got != want {
		t.Fatalf("catalog has %d permutations, want %d", got, want)
	}
	var definitions []int
	for _, weapon := range BuffWeapons {
		if weapon.ID == 0 || weapon.Key == "" || weapon.Name == "" {
			t.Errorf("incomplete weapon: %+v", weapon)
		}
		if len(weapon.DefIndexes) == 0 {
			t.Errorf("%s has no item definition", weapon.Name)
		}
		if weapon.ApplyID != weapon.ID {
			continue
		}
		for _, definition := range weapon.DefIndexes {
			if slices.Contains(definitions, definition) {
				t.Errorf("item definition %d belongs to more than one weapon", definition)
			}
			definitions = append(definitions, definition)
		}
	}
	seen := make(map[[2]uint16]bool, len(WeaponBuffs))
	for _, buff := range WeaponBuffs {
		if buff.ID == 0 || buff.Key == "" || buff.Weapon == "" || buff.Attribute == "" || buff.Description == "" {
			t.Errorf("incomplete weapon buff: %+v", buff)
		}
		pair := [2]uint16{buff.WeaponID, uint16(buff.EffectID)}
		if seen[pair] {
			t.Errorf("duplicate weapon/effect permutation %v", pair)
		}
		seen[pair] = true
	}
	if len(definitions) < 400 {
		t.Fatalf("catalog covers %d concrete item definitions, want at least 400", len(definitions))
	}
}

func weaponNamed(t *testing.T, name string) BuffWeapon {
	t.Helper()
	for _, weapon := range BuffWeapons {
		if weapon.Name == name {
			return weapon
		}
	}
	t.Fatalf("weapon %q is absent", name)
	return BuffWeapon{}
}

func buffNamed(t *testing.T, weaponName, effectKey string) WeaponBuff {
	t.Helper()
	weapon := weaponNamed(t, weaponName)
	for _, buff := range WeaponBuffs {
		if buff.WeaponID == weapon.ID && WeaponEffects[buff.EffectID-1].Key == effectKey {
			return buff
		}
	}
	t.Fatalf("buff %s/%s is absent", weaponName, effectKey)
	return WeaponBuff{}
}

func TestOnlyCuratedPermutationsAreEligibleRewards(t *testing.T) {
	eligible := 0
	for _, buff := range WeaponBuffs {
		if buff.Eligible {
			eligible++
		}
	}
	if eligible == 0 || eligible >= len(WeaponBuffs) {
		t.Fatalf("eligible buffs = %d of %d, want a non-empty curated subset", eligible, len(WeaponBuffs))
	}
}

func TestAmmoOnHitEligibilityIncludesShieldsButNotManmelter(t *testing.T) {
	for _, shield := range []string{"Chargin' Targe", "Splendid Screen", "Tide Turner"} {
		if !buffNamed(t, shield, "ammo-on-hit").Eligible {
			t.Errorf("%s/ammo-on-hit is not eligible", shield)
		}
	}
	if buffNamed(t, "Manmelter", "ammo-on-hit").Eligible {
		t.Error("Manmelter/ammo-on-hit is eligible")
	}
	for _, weapon := range []string{"Mantreads", "Thermal Thruster"} {
		if !buffNamed(t, weapon, "ammo-on-hit").Eligible {
			t.Errorf("%s/ammo-on-hit is not eligible", weapon)
		}
	}
}

func TestStompWeaponSpecialBuffsAreEligibleAndDescribed(t *testing.T) {
	for _, weapon := range []string{"Mantreads", "Thermal Thruster"} {
		buff := buffNamed(t, weapon, "damage")
		if !buff.Eligible {
			t.Errorf("%s/damage is not eligible", weapon)
		}
		if buff.Description != "+5× fall-damage multiplier" {
			t.Errorf("%s/damage description = %q", weapon, buff.Description)
		}
		for _, effect := range []string{"base-health-on-kill", "crits-on-kill", "minicrits-on-kill", "speed-on-kill"} {
			if !buffNamed(t, weapon, effect).Eligible {
				t.Errorf("%s/%s is not eligible", weapon, effect)
			}
		}
		if buffNamed(t, weapon, "heal-on-kill").Eligible {
			t.Errorf("%s still draws the ordinary heal-on-kill effect", weapon)
		}
	}
	for _, weapon := range BuffWeapons {
		if stompWeapons[weapon.Name] {
			continue
		}
		if buffNamed(t, weapon.Name, "base-health-on-kill").Eligible {
			t.Errorf("non-stomp weapon %s draws base-health-on-kill", weapon.Name)
		}
	}
	clip := buffNamed(t, "Thermal Thruster", "clip-size")
	if !clip.Eligible {
		t.Error("Thermal Thruster/clip-size is not eligible")
	}
	if clip.Description != "+1 launch charge" {
		t.Errorf("Thermal Thruster/clip-size description = %q", clip.Description)
	}
}

func TestPluginAttributesStompKillEffectsToTheWearable(t *testing.T) {
	body, err := os.ReadFile("../plugin/scripting/tf2_archipelago/weapon_buffs.inc")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	death := substanceSourceFunction(t, "public void WeaponBuffs_PlayerDeath")
	for _, required := range []string{
		`event.GetString("weapon", eventWeapon`,
		`event.GetString("weapon_logclassname", logWeapon`,
		`StrEqual(eventWeapon, "mantreads")`,
		`StrEqual(logWeapon, "rocketpack_stomp")`,
		"WeaponBuffs_EntityInLoadoutSlot(attacker, Slot_Secondary)",
		`StrEqual(g_WeaponNames[weapon], "Mantreads")`,
		`StrEqual(g_WeaponNames[weapon], "Thermal Thruster")`,
		"WeaponBuffs_ApplyStompKillEffects(attacker, entity, weapon)",
	} {
		if !strings.Contains(death, required) {
			t.Errorf("stomp death attribution has no %q", required)
		}
	}
	for _, required := range []string{
		`HookEvent("player_death", WeaponBuffs_PlayerDeath)`,
		"g_WeaponEffectLevels[weapon][BaseHealthOnKillEffect]",
		"WeaponBuffs_ClassBaseHealth(attacker)",
		"g_WeaponEffectIncrements[BaseHealthOnKillEffect]",
		"float(maximum) * 1.5",
		"int maximum = baseHealth",
		"SetEntityHealth(attacker, health + healed)",
		"TFCond_CritOnKill",
		"TFCond_MiniCritOnKill",
		"TFCond_SpeedBuffAlly",
		"effect == DamageEffect || WeaponBuffs_IsOnKillEffect(effect)",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("stomp kill-effect implementation has no %q", required)
		}
	}
}

func TestPluginImplementsStompDamageAndThermalClipSize(t *testing.T) {
	body, err := os.ReadFile("../plugin/scripting/tf2_archipelago/weapon_buffs.inc")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		`HookEvent("rocketpack_launch", WeaponBuffs_RocketPackLaunch)`,
		"g_WeaponEffectLevels[weapon][ClipSizeEffect]",
		"WeaponBuffs_RocketPackLaunchCost()",
		"WeaponBuffs_ThermalThrusterEffectiveCost(weapon)",
		"stockCharges + levels - 1",
		`SetEntPropFloat(client, Prop_Send, "m_flItemChargeMeter"`,
		"damagecustom == TF_CUSTOM_BOOTS_STOMP",
		"g_WeaponEffectLevels[catalog][DamageEffect]",
		"StompFallDamagePerLevel * float(levels)",
		`strcopy(description, maxlength, "+1 launch charge")`,
		`strcopy(description, maxlength, "+5x fall-damage multiplier")`,
	} {
		if !strings.Contains(text, required) {
			t.Errorf("special stomp buff implementation has no %q", required)
		}
	}
}

func TestWeaponFamiliesShareOneRewardPool(t *testing.T) {
	for _, family := range weaponFamilies {
		canonical := weaponNamed(t, family[0])
		for _, name := range family {
			member := weaponNamed(t, name)
			if member.ApplyID != canonical.ID {
				t.Errorf("%s applies to weapon %d, want %s (%d)", name, member.ApplyID, canonical.Name, canonical.ID)
			}
			for _, definition := range member.DefIndexes {
				if !slices.Contains(canonical.DefIndexes, definition) {
					t.Errorf("%s family does not include definition %d from %s", canonical.Name, definition, name)
				}
			}
			if name != canonical.Name && buffNamed(t, name, "damage").Eligible {
				t.Errorf("family member %s has a separate eligible reward pool", name)
			}
		}
	}
}

func TestRequestedReskinsUseTheirMechanicalWeaponPool(t *testing.T) {
	for member, canonicalName := range map[string]string{
		"Holy Mackerel":          "Bat",
		"Unarmed Combat":         "Bat",
		"Mutated Milk":           "Mad Milk",
		"Self-Aware Beauty Mark": "Flying Guillotine",
		"Red-Tape Recorder":      "Sapper",
	} {
		memberWeapon := weaponNamed(t, member)
		canonical := weaponNamed(t, canonicalName)
		if memberWeapon.ApplyID != canonical.ID {
			t.Errorf("%s applies to weapon %d, want %s (%d)", member, memberWeapon.ApplyID, canonicalName, canonical.ID)
		}
	}
}

func TestDecoratedWeaponDefinitionsUseTheirUnderlyingWeaponPool(t *testing.T) {
	// TF2's current item schema has one decorated weapon definition at every
	// index from 15000 through 15158 except the unused 15093. Keep the snapshot
	// exhaustive so adding one example cannot leave the rest silently broken.
	seen := make(map[int]string)
	for _, weapon := range legacyWeaponBuffs {
		for _, definition := range weapon.DefIndexes {
			if definition < 15000 || definition > 15158 {
				continue
			}
			if previous, ok := seen[definition]; ok {
				t.Errorf("decorated definition %d belongs to both %s and %s", definition, previous, weapon.Weapon)
			}
			seen[definition] = weapon.Weapon
		}
	}
	for definition := 15000; definition <= 15158; definition++ {
		if definition == 15093 {
			continue
		}
		if _, ok := seen[definition]; !ok {
			t.Errorf("decorated definition %d has no weapon buff pool", definition)
		}
	}

	for definition, want := range map[int]string{
		15008: "Medi Gun",
		15029: "Scattergun", // Backcountry Blaster
		15030: "Flame Thrower",
		15157: "Scattergun", // Corsair
	} {
		if got := seen[definition]; got != want {
			t.Errorf("decorated definition %d uses %q buffs, want %q", definition, got, want)
		}
	}
}

func TestBuilderToolboxSharesTheConstructionPDABuffPool(t *testing.T) {
	construction := weaponNamed(t, "Construction PDA")
	builder := weaponNamed(t, "PDA")
	if builder.ApplyID != construction.ID {
		t.Fatalf("builder toolbox applies to weapon %d, want Construction PDA %d", builder.ApplyID, construction.ID)
	}
	if !slices.Contains(construction.DefIndexes, 28) {
		t.Errorf("Construction PDA definitions = %v, want builder toolbox definition 28", construction.DefIndexes)
	}
	if !buffNamed(t, "Construction PDA", "disposable-sentry").Eligible {
		t.Error("Construction PDA cannot draw the disposable-sentry buff")
	}
	if buffNamed(t, "PDA", "disposable-sentry").Eligible {
		t.Error("builder toolbox has a separate disposable-sentry reward pool")
	}
}

func TestMechanicSpecificEffectsStayOnTheirWeapons(t *testing.T) {
	cases := []struct {
		effects []string
		allowed map[string]bool
	}{
		{[]string{"airblast-power", "airblast-rate", "charged-airblast", "airblast-cost"}, airblastWeapons},
		{[]string{"building-health", "sentry-fire-rate", "disposable-sentry", "metal-regen", "max-metal", "construction-rate", "repair-rate"}, engineerWeapons},
		{[]string{"healing", "healing-received", "uber-rate", "uber-duration"}, mediguns},
		{[]string{"banner-duration"}, banners},
		{[]string{"cloak-duration", "cloak-regen"}, watches},
		{[]string{"cloak-on-hit", "cloak-on-kill"}, spyAttackWeapons},
	}
	for _, test := range cases {
		for _, weapon := range BuffWeapons {
			if weapon.ApplyID != weapon.ID {
				continue
			}
			for _, effect := range test.effects {
				if got, want := buffNamed(t, weapon.Name, effect).Eligible, test.allowed[weapon.Name]; got != want {
					t.Errorf("%s/%s eligible = %t, want %t", weapon.Name, effect, got, want)
				}
			}
		}
	}
}

func TestPassiveAndConsumableItemsDoNotDrawDamageBuffs(t *testing.T) {
	for _, name := range []string{"Razorback", "Sandvich", "Bonk! Atomic Punch", "Medi Gun", "Buff Banner"} {
		if buffNamed(t, name, "damage").Eligible {
			t.Errorf("%s still draws damage buffs", name)
		}
	}
	for _, buff := range WeaponBuffs {
		if buff.WeaponID == weaponNamed(t, "Razorback").ID && buff.Eligible {
			t.Errorf("passive Razorback still draws %s", WeaponEffects[buff.EffectID-1].Key)
		}
	}
}

func TestThrownMetersAndProjectileMeleesKeepProjectileUpgrades(t *testing.T) {
	for _, name := range []string{"Gas Passer", "Jarate", "Mad Milk", "Sandman", "Wrap Assassin"} {
		for _, effect := range []string{"projectile-count", "projectile-speed"} {
			if !buffNamed(t, name, effect).Eligible {
				t.Errorf("%s lost useful %s", name, effect)
			}
		}
	}
}

func TestJarateAndMadMilkOnlyDrawProjectileRechargeAndSubstanceBuffs(t *testing.T) {
	for _, name := range []string{"Jarate", "Mad Milk"} {
		for _, effect := range WeaponEffects {
			want := jarProjectileEffects[effect.Key] || substanceEffects[effect.Key] || effect.Key == "meter-recharge"
			if got := buffNamed(t, name, effect.Key).Eligible; got != want {
				t.Errorf("%s/%s eligible = %t, want %t", name, effect.Key, got, want)
			}
		}
		for _, effect := range []string{"bleed", "mad-milk", "mark-for-death", "jarate"} {
			if !buffNamed(t, name, effect).Eligible {
				t.Errorf("%s lost substance effect %s", name, effect)
			}
		}
	}
}

func TestDirectHitWeaponsDrawSubstanceBuffs(t *testing.T) {
	for _, name := range []string{"Minigun", "Pistol", "Scattergun"} {
		for _, effect := range []string{"bleed", "ignite", "mad-milk", "mark-for-death"} {
			if !buffNamed(t, name, effect).Eligible {
				t.Errorf("%s lost direct-hit substance effect %s", name, effect)
			}
		}
	}
	if !buffNamed(t, "Sniper Rifle", "jarate").Eligible {
		t.Error("Sniper Rifle lost direct-hit Jarate")
	}
}

// A jar that lands on somebody registers as a hit, so the on-hit attributes
// fire from it. Cowser checked each in game, gh-32.
func TestThrownJarsDrawTheOnHitBuffs(t *testing.T) {
	for _, name := range []string{"Jarate", "Mad Milk"} {
		for _, effect := range []string{"ignite", "heal-on-hit"} {
			if !buffNamed(t, name, effect).Eligible {
				t.Errorf("%s lost on-hit effect %s", name, effect)
			}
		}
		// Neither jar kills, so nothing fires an on-kill attribute.
		for _, effect := range []string{"heal-on-kill", "crits-on-kill", "speed-on-kill"} {
			if buffNamed(t, name, effect).Eligible {
				t.Errorf("%s draws %s and cannot get a kill", name, effect)
			}
		}
	}
}

// The Gas Passer performs no attack and still gets kills, through the afterburn
// its gas leaves behind. gh-32, note 7.
func TestTheGasPasserDrawsTheOnKillBuffs(t *testing.T) {
	for _, effect := range []string{"heal-on-kill", "crits-on-kill", "minicrits-on-kill", "speed-on-kill"} {
		if !buffNamed(t, "Gas Passer", effect).Eligible {
			t.Errorf("Gas Passer lost on-kill effect %s", effect)
		}
	}
	// It still swings at nobody, so the attack effects stay off it.
	for _, effect := range []string{"damage", "fire-rate", "reload-rate"} {
		if buffNamed(t, "Gas Passer", effect).Eligible {
			t.Errorf("Gas Passer draws %s and performs no attack", effect)
		}
	}
}

func TestCliplessWeaponsDoNotDrawClipOrReloadBuffs(t *testing.T) {
	for _, name := range []string{"Flame Thrower", "Minigun", "Sniper Rifle", "Huntsman"} {
		for _, effect := range []string{"clip-size", "reload-rate"} {
			if buffNamed(t, name, effect).Eligible {
				t.Errorf("clipless %s still draws %s", name, effect)
			}
		}
	}
}

func TestReserveShooterHasAConcreteBuff(t *testing.T) {
	for _, weapon := range BuffWeapons {
		if weapon.Name == "Reserve Shooter" {
			if !slices.Contains(weapon.DefIndexes, 415) {
				t.Fatalf("Reserve Shooter definitions = %v, want 415", weapon.DefIndexes)
			}
			return
		}
	}
	t.Fatal("Reserve Shooter is absent from the weapon buff catalog")
}

func TestGeneratedPluginCatalogContainsEveryBuffKey(t *testing.T) {
	body, err := os.ReadFile("../plugin/scripting/tf2_archipelago/weapon_buffs_data.inc")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, buff := range WeaponBuffs {
		if !strings.Contains(text, `"`+buff.Key+`"`) {
			t.Errorf("generated plugin catalog has no key %q", buff.Key)
		}
	}
}

func substanceSourceFunction(t *testing.T, signature string) string {
	t.Helper()
	body, err := os.ReadFile("../plugin/scripting/tf2_archipelago/weapon_buffs.inc")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	start := strings.Index(text, signature)
	if start < 0 {
		t.Fatalf("weapon buffs have no %s", signature)
	}
	open := strings.IndexByte(text[start:], '{')
	if open < 0 {
		t.Fatalf("weapon buffs have no body for %s", signature)
	}
	open += start
	depth := 0
	for index := open; index < len(text); index++ {
		switch text[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : index+1]
			}
		}
	}
	t.Fatalf("weapon buffs have an unterminated body for %s", signature)
	return ""
}

func TestWeaponBuffSubstancesUseTheSharedHitPath(t *testing.T) {
	apply := substanceSourceFunction(t, "static void WeaponBuffs_ApplyHitEffects")
	for _, effect := range []string{
		"TF2_MakeBleed", "TF2_IgnitePlayer", "TFCond_Milked", "TFCond_Gas",
		"TFCond_MarkedForDeath", "TFCond_Jarated",
	} {
		if !strings.Contains(apply, effect) {
			t.Fatalf("shared hit path does not apply %s", effect)
		}
	}

	jar := substanceSourceFunction(t, "public void WeaponBuffs_ApplySubstances")
	if !strings.Contains(jar, "WeaponBuffs_ApplyHitEffects(victim, attacker, weapon)") {
		t.Fatal("jar splashes bypass the shared hit-effect path")
	}

	damage := substanceSourceFunction(t, "public void WeaponBuffs_OnTakeDamagePost")
	for _, guard := range []string{
		"GetClientTeam(victim) == GetClientTeam(attacker)",
		"WeaponBuffs_IsDamageOverTime(attacker, inflictor, damagecustom)",
		"WeaponBuffs_IsSubstanceProjectile(inflictorClass)",
	} {
		if !strings.Contains(damage, guard) {
			t.Fatalf("direct-hit path lost guard %q", guard)
		}
	}
	if !strings.Contains(damage, "WeaponBuffs_ForEntity(weapon)") ||
		!strings.Contains(damage, "WeaponBuffs_ApplyHitEffects(victim, attacker, catalog)") {
		t.Fatal("direct hits do not resolve the canonical weapon and apply its hit effects")
	}
}

func TestPluginImplementsActiveHealthRegenInsteadOfBrokenSchemaHealing(t *testing.T) {
	body, err := os.ReadFile("../plugin/scripting/tf2_archipelago/weapon_buffs.inc")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		// ActiveHealthRegenEffect's number moved to weapon_buffs_math.inc with
		// the predicate that reads it, and the predicate is run rather than
		// read: TestOnlyTheNamedEffectsArePassive.
		"CreateTimer(1.0, Timer_WeaponBuffHealthRegen",
		"WeaponBuffs_LoadoutLevels(client, ActiveHealthRegenEffect)",
		"GetEntProp(resource, Prop_Send, \"m_iMaxHealth\", 4, client)",
		"SetEntityHealth(client, health + healed)",
		"if (WeaponBuffs_IsPassiveEffect(effect))",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("active health regeneration implementation has no %q", required)
		}
	}
}

func TestPluginImplementsAmmoOnHitForClippedAndMeleeWeapons(t *testing.T) {
	body, err := os.ReadFile("../plugin/scripting/tf2_archipelago/weapon_buffs.inc")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		"#define AmmoOnHitEffect 69",
		"WeaponBuffs_AddAmmoOnHit(attacker, weapon, catalog)",
		"GetPlayerWeaponSlot(attacker, TFWeaponSlot_Primary)",
		"GetEntProp(ammoWeapon, Prop_Send, \"m_iPrimaryAmmoType\")",
		"GivePlayerAmmo(attacker, amount, ammoType, true)",
		"WeaponBuffs_SuppliesPrimaryAmmo(entity)",
		"tf_wearable_demoshield",
		"Mantreads",
		"if (effect == AmmoOnHitEffect)",
		"FindEntityByClassname(tank, \"tank_boss\")",
		"SDKHook(tank, SDKHook_OnTakeDamagePost, WeaponBuffs_OnTankTakeDamagePost)",
		"WeaponBuffs_AddAmmoOnHit(attacker, weapon, catalog)",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("ammo-on-hit implementation has no %q", required)
		}
	}

	thermal := substanceSourceFunction(t, "static void WeaponBuffs_AddThermalThrusterCharge")
	for _, required := range []string{
		`GetEntPropFloat(attacker, Prop_Send,`,
		`"m_flItemChargeMeter", Slot_Secondary)`,
		"WeaponBuffs_ThermalThrusterEffectiveCost(weapon) * float(amount)",
		`SetEntPropFloat(attacker, Prop_Send, "m_flItemChargeMeter"`,
	} {
		if !strings.Contains(thermal, required) {
			t.Errorf("Thermal Thruster charge restoration has no %q", required)
		}
	}

	resolver := substanceSourceFunction(t, "static int WeaponBuffs_WeaponOfHit")
	for _, required := range []string{
		"damagecustom == TF_CUSTOM_BOOTS_STOMP",
		"WeaponBuffs_EntityInLoadoutSlot(attacker, Slot_Secondary)",
		"Mantreads",
		"Thermal Thruster",
	} {
		if !strings.Contains(resolver, required) {
			t.Errorf("stomp hit resolver has no %q", required)
		}
	}

	energy := substanceSourceFunction(t, "static float WeaponBuffs_EnergyShotCost")
	for _, required := range []string{
		"tf_weapon_particle_cannon",
		"tf_weapon_raygun",
		"tf_weapon_drg_pomson",
		"return 5.0",
	} {
		if !strings.Contains(energy, required) {
			t.Errorf("energy-weapon shot cost has no %q", required)
		}
	}
	for _, required := range []string{
		"WeaponBuffs_EnergyShotCost(ammoWeapon)",
		"energyPerShot * float(amount)",
		"WeaponBuffs_EnergyMaxCharge(ammoWeapon, energyPerShot)",
		"EnergyClipSizeAttributeClass",
		"SetEntPropFloat(ammoWeapon, Prop_Send, \"m_flEnergy\", energy)",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("energy ammo-on-hit implementation has no %q", required)
		}
	}

	apply := substanceSourceFunction(t, "void WeaponBuffs_Apply(int client)")
	for _, required := range []string{
		"effect == ClipSizeEffect",
		"EnergyClipSizeAttributeClass",
		"TF2Attrib_SetByName(provider, EnergyClipSizeAttribute, value)",
	} {
		if !strings.Contains(apply, required) {
			t.Errorf("energy clip-size implementation has no %q", required)
		}
	}

	tank := substanceSourceFunction(t, "public void WeaponBuffs_OnTankTakeDamagePost")
	for _, guard := range []string{
		"damage <= 0.0",
		"GetClientTeam(attacker) == GetEntProp(tank, Prop_Send, \"m_iTeamNum\")",
		"WeaponBuffs_IsDamageOverTime(attacker, inflictor, damagecustom)",
	} {
		if !strings.Contains(tank, guard) {
			t.Errorf("tank ammo-on-hit path lost guard %q", guard)
		}
	}

	damageOverTime := substanceSourceFunction(t, "static bool WeaponBuffs_IsDamageOverTime")
	for _, required := range []string{
		"damagecustom == TF_CUSTOM_BLEEDING",
		"damagecustom == TF_CUSTOM_BURNING && inflictor == attacker",
	} {
		if !strings.Contains(damageOverTime, required) {
			t.Errorf("damage-over-time classifier has no %q", required)
		}
	}
}

func TestEveryRequestedSillyEffectKeepsItsStackingMode(t *testing.T) {
	wanted := map[string]BuffMode{
		"projectile-count":    BuffPercentage,
		"projectile-speed":    BuffPercentage,
		"bleed":               BuffAdd,
		"afterburn-damage":    BuffPercentage,
		"airborne-crits":      BuffToggle,
		"ignite":              BuffToggle,
		"gasoline":            BuffToggle,
		"mad-milk":            BuffToggle,
		"no-self-blast":       BuffToggle,
		"heal-on-kill":        BuffAdd,
		"slow-on-hit":         BuffToggle,
		"gesture-speed":       BuffPercentage,
		"base-health-on-kill": BuffAdd,
	}
	for _, effect := range WeaponEffects {
		if mode, ok := wanted[effect.Key]; ok {
			if effect.Mode != mode {
				t.Errorf("%s mode = %d, want %d", effect.Key, effect.Mode, mode)
			}
			delete(wanted, effect.Key)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("missing requested effects: %v", wanted)
	}
}

func TestRequestedSecondPassEffectValues(t *testing.T) {
	wanted := map[string]float32{
		"heal-on-kill":        15,
		"no-self-blast":       1,
		"slow-on-hit":         1,
		"gesture-speed":       0.50,
		"base-health-on-kill": 50,
	}
	for _, effect := range WeaponEffects {
		if value, ok := wanted[effect.Key]; ok {
			if effect.Increment != value {
				t.Errorf("%s increment = %.2f, want %.2f", effect.Key, effect.Increment, value)
			}
			delete(wanted, effect.Key)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("missing requested second-pass effects: %v", wanted)
	}
}

func TestWeaponEffectsUseDistinctSchemaAttributes(t *testing.T) {
	seen := make(map[string]string, len(WeaponEffects))
	for _, effect := range WeaponEffects {
		if previous, duplicate := seen[effect.Attribute]; duplicate {
			t.Errorf("effects %q and %q both use schema attribute %q",
				previous, effect.Key, effect.Attribute)
		}
		seen[effect.Attribute] = effect.Key
	}
	if got := seen["bullets per shot bonus"]; got != "projectile-count" {
		t.Errorf("projectile-count schema mapping = %q, want bullets per shot bonus", got)
	}
}

func TestWeaponEffectAttributeClassesComplete(t *testing.T) {
	if got, want := len(WeaponEffectAttributeClasses), len(WeaponEffects); got != want {
		t.Fatalf("attribute classes: got %d, want %d", got, want)
	}
	for index, class := range WeaponEffectAttributeClasses {
		if class == "" {
			t.Errorf("%s has no engine attribute class", WeaponEffects[index].Key)
		}
	}
}

func TestLegacyPermutationKeepsItsIDAndEveryEffectExists(t *testing.T) {
	for _, old := range legacyWeaponBuffs {
		seenEffects := make(map[uint8]bool, len(WeaponEffects))
		keptLegacy := false
		for _, buff := range WeaponBuffs {
			if buff.WeaponID != old.ID {
				continue
			}
			seenEffects[buff.EffectID] = true
			if buff.Attribute == old.Attribute {
				keptLegacy = buff.ID == old.ID && buff.Key == old.Key
			}
		}
		if !keptLegacy {
			t.Errorf("%s did not retain legacy id %d and key %q", old.Weapon, old.ID, old.Key)
		}
		if len(seenEffects) != len(WeaponEffects) {
			t.Errorf("%s has %d effects, want %d", old.Weapon, len(seenEffects), len(WeaponEffects))
		}
	}
}

func TestItemExportMarksOnlyNumericBuffsStackable(t *testing.T) {
	for _, item := range buildItemsFile().Items {
		if item.WeaponBuffID == 0 {
			if item.Stackable {
				t.Errorf("non-buff item %q is stackable", item.Name)
			}
			continue
		}
		buff, ok := WeaponBuffByID(item.WeaponBuffID)
		if !ok {
			t.Errorf("item %q has unknown weapon buff %d", item.Name, item.WeaponBuffID)
			continue
		}
		want := buff.Mode != BuffToggle
		if item.Stackable != want {
			t.Errorf("%s stackable = %t, want %t for mode %d",
				buff.Key, item.Stackable, want, buff.Mode)
		}
	}
}

func TestEveryEligibleBuffNamesTheClassesThatCanEquipIt(t *testing.T) {
	for _, buff := range WeaponBuffs {
		if buff.Eligible && len(weaponClassNames(buff.Weapon, buff.DefIndexes)) == 0 {
			t.Errorf("eligible buff %q has no equipping class", buff.ItemName())
		}
	}
	for _, test := range []struct {
		weapon string
		want   []string
	}{
		{"Air Strike", []string{"Soldier"}},
		{"Mad Milk", []string{"Scout"}},
		{"Crusader's Crossbow", []string{"Medic"}},
		{"Shotgun", []string{"Soldier", "Pyro", "Heavy", "Engineer"}},
		{"Saxxy", []string{"Scout", "Soldier", "Pyro", "Demoman", "Heavy", "Engineer", "Medic", "Sniper", "Spy"}},
	} {
		buff := buffNamed(t, test.weapon, "damage")
		if got := weaponClassNames(buff.Weapon, buff.DefIndexes); !slices.Equal(got, test.want) {
			t.Errorf("%s classes = %v, want %v", test.weapon, got, test.want)
		}
	}
}

func TestWeaponClassExportCarriesWeaponClasses(t *testing.T) {
	icons := make(map[string]string)
	for _, weapon := range buildWeaponClassesFile().Weapons {
		icons[weapon.Name] = weapon.Icon
		if weapon.Name != "Air Strike" {
			continue
		}
		if !slices.Equal(weapon.Classes, []string{"Soldier"}) {
			t.Fatalf("Air Strike export classes = %v, want Soldier", weapon.Classes)
		}
		if want := "assets/tf2/items/f87faf790afc0d04056479f1566f09f1.png"; weapon.Icon != want {
			t.Fatalf("Air Strike icon = %q, want %q", weapon.Icon, want)
		}
	}
	if icons["Air Strike"] == "" {
		t.Fatal("Air Strike is missing from weapon class export")
	}
	for weapon, filename := range tfWikiItemIconNames {
		want := trackerItemIconPath(filename)
		if got := icons[weapon]; got != want {
			t.Errorf("%s icon = %q, want %q", weapon, got, want)
		}
	}
}

func TestWeaponClassExportIconsAreBundled(t *testing.T) {
	for _, weapon := range buildWeaponClassesFile().Weapons {
		icon, err := url.PathUnescape(weapon.Icon)
		if err != nil {
			t.Fatalf("%s icon path %q: %v", weapon.Name, weapon.Icon, err)
		}
		if !strings.HasPrefix(icon, "assets/tf2/items/") {
			t.Errorf("%s icon is not a bundled asset: %q", weapon.Name, weapon.Icon)
			continue
		}
		name := strings.TrimSuffix(filepath.Base(icon), ".png")
		if len(name) != 32 || strings.Trim(name, "0123456789abcdef") != "" {
			t.Errorf("%s icon has an embed-unsafe filename: %q", weapon.Name, weapon.Icon)
		}
		if _, err := os.Stat(filepath.Join("../launcher/web/src", filepath.FromSlash(icon))); err != nil {
			t.Errorf("%s icon %q is not bundled: %v", weapon.Name, weapon.Icon, err)
		}
	}
}

// Explode on ignite ended waves on its own once substances landed on direct
// hits (gh-17). It is out of the pool everywhere and keeps its ID.
func TestExplodeOnIgniteIsOfferedNowhere(t *testing.T) {
	for _, buff := range WeaponBuffs {
		if buff.EffectID == 16 && buff.Eligible {
			t.Errorf("%s still offers explode on ignite", buff.Weapon)
		}
	}
	if _, ok := WeaponBuffByID(buffNamed(t, "Minigun", "gasoline").ID); !ok {
		t.Error("the effect lost its ID, which seeds hold")
	}
}

// A rocket that penetrates does not explode where it was aimed (gh-21).
func TestProjectilePenetrationStaysOffExplosives(t *testing.T) {
	for name, want := range map[string]bool{
		"Rocket Launcher": false, "Grenade Launcher": false, "Loose Cannon": false, "Scorch Shot": false,
		"Huntsman": true, "Crusader's Crossbow": true, "Syringe Gun": true, "Flare Gun": true,
	} {
		if got := buffNamed(t, name, "projectile-penetration").Eligible; got != want {
			t.Errorf("%s penetration eligible = %t, want %t", name, got, want)
		}
	}
}

func TestProjectileDestructionOnlyDrawsOnBulletAndEnergyGuns(t *testing.T) {
	for _, weapon := range BuffWeapons {
		buff := buffNamed(t, weapon.Name, "destroy-projectiles")
		want := weapon.ID == weapon.ApplyID && projectileDestructionWeapons[weapon.Name]
		if buff.Eligible != want {
			t.Errorf("%s destroy-projectiles eligibility = %v, want %v", weapon.Name, buff.Eligible, want)
		}
	}

	for _, name := range []string{
		"Rocket Launcher", "Grenade Launcher", "Stickybomb Launcher", "Flare Gun",
		"Jarate", "Mad Milk", "Huntsman", "Syringe Gun", "Knife", "Mantreads",
		"Thermal Thruster", "Wrangler",
	} {
		if buffNamed(t, name, "destroy-projectiles").Eligible {
			t.Errorf("%s draws projectile destruction", name)
		}
	}

	for _, name := range []string{"Scattergun", "Shotgun", "Sniper Rifle", "Pistol", "Pomson 6000", "Righteous Bison", "Short Circuit", "Minigun"} {
		if !buffNamed(t, name, "destroy-projectiles").Eligible {
			t.Errorf("%s does not draw projectile destruction", name)
		}
	}
}

// The game reads armor piercing on a backstab and nowhere else (gh-25).
func TestArmorPiercingIsAKnifeBuff(t *testing.T) {
	for name, want := range map[string]bool{
		"Knife": true, "Your Eternal Reward": true, "Conniver's Kunai": true, "Big Earner": true, "Spy-Cicle": true,
		"Revolver": false, "Minigun": false, "Rocket Launcher": false, "Kukri": false,
	} {
		if got := buffNamed(t, name, "armor-piercing").Eligible; got != want {
			t.Errorf("%s armor piercing eligible = %t, want %t", name, got, want)
		}
	}
}

// Every pair the sheet cuts names a weapon and an effect the tables know, and
// the cut only ever removes: a pair the rules already refuse is redundant here
// and worth a line less.
func TestSheetCutsNameRealWeaponsAndEffects(t *testing.T) {
	effects := map[string]bool{}
	for _, effect := range WeaponEffects {
		effects[effect.Key] = true
	}
	for name, keys := range sheetCuts {
		weaponNamed(t, name)
		for key := range keys {
			if !effects[key] {
				t.Errorf("%s cuts %q, which is not an effect", name, key)
			}
			if buffNamed(t, name, key).Eligible {
				t.Errorf("%s/%s is still eligible", name, key)
			}
		}
	}
	for name, want := range map[string]bool{"Rocket Launcher": false, "Scattergun": true} {
		if got := buffNamed(t, name, "accuracy").Eligible; got != want {
			t.Errorf("%s accuracy eligible = %t, want %t", name, got, want)
		}
	}
}

// Über on hit needs a hit and an ÜberCharge to put it in. A medigun heals and
// never hits, so the sheet marks it N on all four; the Medic's syringe guns and
// saws do both, at the two rates a player asked for (gh-32).
func TestUberOnHitFollowsTheMedicsAttackWeapons(t *testing.T) {
	want := map[string]string{
		"Syringe Gun":         "+1% ÜberCharge on hit",
		"Blutsauger":          "+1% ÜberCharge on hit",
		"Overdose":            "+1% ÜberCharge on hit",
		"Übersaw":             "+5% ÜberCharge on hit",
		"Bonesaw":             "+5% ÜberCharge on hit",
		"Vita-Saw":            "+5% ÜberCharge on hit",
		"Amputator":           "+5% ÜberCharge on hit",
		"Solemn Vow":          "+5% ÜberCharge on hit",
		"Crusader's Crossbow": "+5% ÜberCharge on hit",
	}
	for _, buff := range WeaponBuffs {
		if buff.EffectID != 40 {
			continue
		}
		description, wanted := want[buff.Weapon]
		if buff.Eligible != wanted {
			t.Errorf("%s über on hit eligible = %t, want %t", buff.Weapon, buff.Eligible, wanted)
		}
		if wanted && buff.Description != description {
			t.Errorf("%s über on hit reads %q, want %q", buff.Weapon, buff.Description, description)
		}
	}
	// A medigun is the one Medic weapon it stays off.
	for _, name := range []string{"Medi Gun", "Kritzkrieg", "Quick-Fix", "Vaccinator"} {
		if buffNamed(t, name, "uber-on-hit").Eligible {
			t.Errorf("%s offers über on hit and never lands one", name)
		}
	}
}

// The plugin pays the syringe guns' one percent itself, because one increment
// per effect is all the generated table carries.
func TestPluginSplitsTheUberOnHitRate(t *testing.T) {
	increment := substanceSourceFunction(t, "static float WeaponBuffs_EffectIncrement")
	for _, required := range []string{
		"effect == UberOnHitEffect",
		"WeaponBuffs_IsSyringeGun(weapon)",
		"return SyringeUberOnHitFraction",
		"return g_WeaponEffectIncrements[effect]",
	} {
		if !strings.Contains(increment, required) {
			t.Errorf("per-weapon increment has no %q", required)
		}
	}
	syringe := substanceSourceFunction(t, "static bool WeaponBuffs_IsSyringeGun")
	for _, required := range []string{"Blutsauger", "Overdose", "Syringe Gun"} {
		if !strings.Contains(syringe, required) {
			t.Errorf("syringe gun test has no %q", required)
		}
	}
	body, err := os.ReadFile("../plugin/scripting/tf2_archipelago/weapon_buffs.inc")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"#define UberOnHitEffect 39",
		"#define SyringeUberOnHitFraction 0.01",
		"WeaponBuffs_EffectIncrement(weapon, effect) * float(levels)",
	} {
		if !strings.Contains(string(body), required) {
			t.Errorf("über on hit rate split has no %q", required)
		}
	}
}

/*
Movement, jump height and health regeneration read off the whole loadout, the
way MvM's own class upgrades do, rather than off the weapon in hand.

Which effects are passive is no longer asked here. That set moved to
weapon_buffs_math.inc and TestOnlyTheNamedEffectsArePassive runs the predicate
over every effect the generated table holds, which is a better answer than
looking for three names in a function body. What is left is the wiring around
it, and the wiring reads entity properties and calls TF2Attrib, so reading the
source is still the best that can be done for it.
*/
func TestPassiveBuffsReadTheWholeLoadout(t *testing.T) {
	body, err := os.ReadFile("../plugin/scripting/tf2_archipelago/weapon_buffs.inc")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		"WeaponBuffs_ApplyPassives(client, provider)",
		"if (WeaponBuffs_IsPassiveEffect(effect))",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("passive buff wiring has no %q", required)
		}
	}
	levels := substanceSourceFunction(t, "static int WeaponBuffs_LoadoutLevels")
	for _, required := range []string{
		"for (int slot = 0; slot <= 5; slot++)",
		"WeaponBuffs_EntityInLoadoutSlot(client, slot)",
		"levels += g_WeaponEffectLevels[weapon][effect]",
	} {
		if !strings.Contains(levels, required) {
			t.Errorf("loadout level sum has no %q", required)
		}
	}
	apply := substanceSourceFunction(t, "static void WeaponBuffs_ApplyPassives")
	for _, required := range []string{
		"WeaponBuffs_LoadoutLevels(client, effect)",
		"g_WeaponEffectAttributeClasses[effect], client",
		"TF2Attrib_SetByName(provider, g_WeaponEffectAttributes[effect]",
	} {
		if !strings.Contains(apply, required) {
			t.Errorf("passive application has no %q", required)
		}
	}
}
