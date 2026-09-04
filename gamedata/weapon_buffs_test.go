package gamedata

import (
	"os"
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

func TestFunctionalReskinsShareOneRewardPool(t *testing.T) {
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
				t.Errorf("reskin %s has a separate eligible reward pool", name)
			}
		}
	}
}

func TestMechanicSpecificEffectsStayOnTheirWeapons(t *testing.T) {
	cases := []struct {
		effects []string
		allowed map[string]bool
	}{
		{[]string{"airblast-power", "airblast-rate", "charged-airblast", "airblast-cost"}, airblastWeapons},
		{[]string{"building-health", "sentry-fire-rate", "disposable-sentry", "metal-regen", "max-metal", "construction-rate", "repair-rate"}, engineerWeapons},
		{[]string{"healing", "healing-received", "uber-rate", "uber-on-hit", "uber-duration"}, mediguns},
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
		for _, effect := range []string{"bleed", "mad-milk", "gasoline", "mark-for-death", "jarate"} {
			if !buffNamed(t, name, effect).Eligible {
				t.Errorf("%s lost substance effect %s", name, effect)
			}
		}
	}
}

func TestDirectHitWeaponsDrawSubstanceBuffs(t *testing.T) {
	for _, name := range []string{"Minigun", "Pistol", "Scattergun"} {
		for _, effect := range []string{"bleed", "ignite", "mad-milk", "gasoline", "mark-for-death"} {
			if !buffNamed(t, name, effect).Eligible {
				t.Errorf("%s lost direct-hit substance effect %s", name, effect)
			}
		}
	}
	if !buffNamed(t, "Sniper Rifle", "jarate").Eligible {
		t.Error("Sniper Rifle lost direct-hit Jarate")
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
		"damagecustom == TF_CUSTOM_BURNING",
		"damagecustom == TF_CUSTOM_BLEEDING",
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
		"#define ActiveHealthRegenEffect 66",
		"CreateTimer(1.0, Timer_WeaponBuffHealthRegen",
		"g_WeaponEffectLevels[weapon][ActiveHealthRegenEffect]",
		"GetEntProp(resource, Prop_Send, \"m_iMaxHealth\", 4, client)",
		"SetEntityHealth(client, health + healed)",
		"if (effect == ActiveHealthRegenEffect)",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("active health regeneration implementation has no %q", required)
		}
	}
}

func TestEveryRequestedSillyEffectKeepsItsStackingMode(t *testing.T) {
	wanted := map[string]BuffMode{
		"projectile-count": BuffPercentage,
		"projectile-speed": BuffPercentage,
		"bleed":            BuffAdd,
		"afterburn-damage": BuffPercentage,
		"airborne-crits":   BuffToggle,
		"ignite":           BuffToggle,
		"gasoline":         BuffToggle,
		"mad-milk":         BuffToggle,
		"no-self-blast":    BuffToggle,
		"heal-on-kill":     BuffAdd,
		"slow-on-hit":      BuffToggle,
		"gesture-speed":    BuffPercentage,
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
		"heal-on-kill":  15,
		"no-self-blast": 1,
		"slow-on-hit":   1,
		"gesture-speed": 0.50,
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
