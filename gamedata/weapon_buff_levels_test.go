package gamedata

import (
	"fmt"
	"strings"
	"testing"
)

/*
The weapon buff levels, run through their interface.

weapon_buff_levels.inc owns the levels the run granted and every decision that
reads them: what goes on the provider and at what value, what a kill, a hit or a
jar pays, what a stomp and a self-inflicted blast become, how many disposable
sentries fit and when a shot was fired. weapon_buffs.inc only carries the answers
to the engine. These scenarios drive the module the way the plugin does, grant by
grant and think by think, and read back what it decided.

A scenario names weapons as {Name} and effects as [key]; levelsDriver turns both
into the indexes the generated tables use.
*/

// levelsDriver runs body after substituting {Weapon Name} and [effect-key].
func levelsDriver(t *testing.T, body string) []int32 {
	t.Helper()
	var pairs []string
	for _, weapon := range BuffWeapons {
		pairs = append(pairs, "{"+weapon.Name+"}", fmt.Sprint(int(weapon.ID)-1))
	}
	for _, effect := range WeaponEffects {
		pairs = append(pairs, "["+effect.Key+"]", fmt.Sprint(int(effect.ID)-1))
	}
	return driver{body: strings.NewReplacer(pairs...).Replace(body)}.run(t)
}

func wantCells(t *testing.T, what string, got []int32, want ...int32) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%s: got %v, want %v", what, got, want)
	}
}

func wantFloats(t *testing.T, what string, got []int32, want ...float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d answers for %d expected", what, len(got), len(want))
	}
	for i := range want {
		if bitsToFloat(got[i]) != want[i] {
			t.Errorf("%s[%d]: got %v, want %v", what, i, bitsToFloat(got[i]), want[i])
		}
	}
}

func toggleEffect(t *testing.T) WeaponEffect {
	t.Helper()
	for _, effect := range WeaponEffects {
		if effect.Mode == BuffToggle {
			return effect
		}
	}
	t.Fatal("no toggle effect in the table")
	return WeaponEffect{}
}

// A copy of a toggle changes nothing; any other copy is one more level. A
// resync clears every level before the new set is granted.
func TestBuffLevelsStackExceptToggles(t *testing.T) {
	toggle := toggleEffect(t)
	got := levelsDriver(t, fmt.Sprintf(`
    BuffLevels_Add({Shotgun}, %d, 1);
    BuffLevels_Add({Shotgun}, %d, 1);
    BuffLevels_Add({Shotgun}, [damage], 1);
    BuffLevels_Add({Shotgun}, [damage], 2);
    printnum(BuffLevels_Of({Shotgun}, %d));
    printnum(BuffLevels_Of({Shotgun}, [damage]));
    printnum(BuffLevels_Count({Shotgun}));
    printnum(BuffLevels_Total());
    printnum(BuffLevels_Of(-1, [damage]));
    BuffLevels_Clear();
    printnum(BuffLevels_Total());
`, toggle.ID-1, toggle.ID-1, toggle.ID-1))
	wantCells(t, "toggle, stack, count, total, no weapon, cleared", got, 1, 3, 4, 4, 0, 0)
}

// A grant names a key's row in g_WeaponBuffKeys and lands on its weapon and effect.
func TestBuffLevelsGrantByKeyIndex(t *testing.T) {
	buff := buffNamed(t, "Scattergun", "damage")
	row := -1
	for index, key := range unquoted(literals(t, dataSource, "g_WeaponBuffKeys")) {
		if key == buff.Key {
			row = index
		}
	}
	if row < 0 {
		t.Fatalf("the plugin's key table has no %s", buff.Key)
	}
	got := levelsDriver(t, fmt.Sprintf(`
    BuffLevels_Grant(%d);
    BuffLevels_Grant(%d);
    printnum(BuffLevels_Of({Scattergun}, [damage]));
    printnum(BuffLevels_Total());
`, row, row))
	wantCells(t, "two grants of one key", got, 2, 2)
}

// Passives are summed over the loadout, an empty slot included, and a buff
// rolled on two weapons counts twice (gh-32).
func TestPassiveLevelsAreTheWholeLoadout(t *testing.T) {
	got := levelsDriver(t, `
    BuffLevels_Add({Scattergun}, [move-speed], 2);
    BuffLevels_Add({Pistol}, [move-speed], 1);
    BuffLevels_Add({Bat}, [max-health], 1);
    int loadout[4] = { {Scattergun}, {Pistol}, -1, {Bat} };
    printnum(BuffLevels_Loadout(loadout, sizeof(loadout), [move-speed]));
    printnum(BuffLevels_Loadout(loadout, sizeof(loadout), [max-health]));
    printnum(BuffLevels_Loadout(loadout, sizeof(loadout), [jump-height]));
`)
	wantCells(t, "move speed, max health, jump height", got, 3, 1, 0)
}

/*
The provider carries an effect only when no other path pays it.

Passives follow the loadout, hit substances and ammo on hit are paid on a
confirmed enemy hit (as attributes TF2 also applied them to the owner's own
blast), no-self-blast is the damage hook's, disposable sentries are predicted
from the PDA, projectile destruction is native only on a minigun, and the stomp
items' damage, on-kill and Thermal launches are the plugin's.
*/
func TestTheProviderCarriesOnlyWhatNoOtherPathPays(t *testing.T) {
	got := levelsDriver(t, `
    BuffLevels_Add({Shotgun}, [damage], 2);
    BuffLevels_Add({Shotgun}, [move-speed], 2);
    BuffLevels_Add({Shotgun}, [bleed], 2);
    BuffLevels_Add({Shotgun}, [ammo-on-hit], 2);
    BuffLevels_Add({Shotgun}, [destroy-projectiles], 2);
    BuffLevels_Add({Minigun}, [destroy-projectiles], 2);
    BuffLevels_Add({Mantreads}, [damage], 2);
    BuffLevels_Add({Mantreads}, [heal-on-kill], 2);
    BuffLevels_Add({Thermal Thruster}, [clip-size], 2);
    BuffLevels_Add({Rocket Launcher}, [no-self-blast], 1);
    printnum(BuffLevels_OnProvider({Shotgun}, [damage], true, 0));
    printnum(BuffLevels_OnProvider({Shotgun}, [damage], true, 1));
    printnum(BuffLevels_OnProvider({Shotgun}, [damage], false, 1));
    printnum(BuffLevels_OnProvider({Shotgun}, [move-speed], true, 0));
    printnum(BuffLevels_OnProvider({Shotgun}, [bleed], true, 0));
    printnum(BuffLevels_OnProvider({Shotgun}, [ammo-on-hit], true, 0));
    printnum(BuffLevels_OnProvider({Shotgun}, [destroy-projectiles], true, 0));
    printnum(BuffLevels_OnProvider({Minigun}, [destroy-projectiles], true, 0));
    printnum(BuffLevels_OnProvider({Mantreads}, [damage], true, 0));
    printnum(BuffLevels_OnProvider({Mantreads}, [heal-on-kill], true, 0));
    printnum(BuffLevels_OnProvider({Thermal Thruster}, [clip-size], true, 0));
    printnum(BuffLevels_OnProvider({Rocket Launcher}, [no-self-blast], true, 0));
`)
	wantCells(t, "provider levels", got, 2, 3, 1, 0, 0, 0, 0, 2, 0, 0, 0, 0)
}

/*
What the provider's attribute is worth.

A percentage moves the hooked value the item and MvM already give, so the
provider holds the ratio. Fire rate never reaches zero, and a melee swing stops
at +50% because past that the swing outruns its animation and lands nothing.
A syringe gun's uber on hit is a fifth of the saw's. A hooked value of zero is
treated as one rather than divided by.
*/
func TestProviderValuesComposeWithWhatTheWeaponHas(t *testing.T) {
	damage := effectByKey(t, "damage").Increment
	uber := effectByKey(t, "uber-on-hit").Increment
	melee := floatDefine(t, "MeleeFireRateFloor")
	syringe := floatDefine(t, "SyringeUberOnHitFraction")
	toggle := toggleEffect(t)
	got := levelsDriver(t, fmt.Sprintf(`
    printnum(view_as<int>(BuffLevels_ProviderValue({Shotgun}, [damage], 2, 1.0, false)));
    printnum(view_as<int>(BuffLevels_ProviderValue({Shotgun}, [damage], 2, 0.5, false)));
    printnum(view_as<int>(BuffLevels_ProviderValue({Shotgun}, [damage], 2, 0.0, false)));
    printnum(view_as<int>(BuffLevels_ProviderValue({Shotgun}, [fire-rate], 20, 1.0, false)));
    printnum(view_as<int>(BuffLevels_ProviderValue({Bat}, [fire-rate], 20, 1.0, true)));
    printnum(view_as<int>(BuffLevels_ProviderValue({Bat}, [fire-rate], 1, 1.0, true)));
    printnum(view_as<int>(BuffLevels_ProviderValue({Übersaw}, [uber-on-hit], 3, 1.0, false)));
    printnum(view_as<int>(BuffLevels_ProviderValue({Syringe Gun}, [uber-on-hit], 3, 1.0, false)));
    printnum(view_as<int>(BuffLevels_ProviderValue({Shotgun}, %d, 4, 1.0, false)));
    printnum(view_as<int>(BuffLevels_PassiveValue([move-speed], 2, 1.0)));
    printnum(BuffLevels_ReadsBase([damage]) ? 1 : 0);
    printnum(BuffLevels_ReadsBase([heal-on-kill]) ? 1 : 0);
`, toggle.ID-1))
	wantFloats(t, "provider values", got[:10],
		1.0+2*damage, (0.5+2*damage)/0.5, 1.0+2*damage,
		0.2, melee, 1.0+effectByKey(t, "fire-rate").Increment,
		3*uber, 3*syringe, 1.0, 1.0+2*effectByKey(t, "move-speed").Increment)
	wantCells(t, "reads base", got[10:], 1, 0)
}

// A kill pays heal, crits a second longer than the attribute, mini-crits and
// speed. A jar splash pays heal and speed on hit. Nothing levelled pays nothing.
func TestKillsAndJarSplashesPayTheirLevels(t *testing.T) {
	heal := effectByKey(t, "heal-on-kill").Increment
	crits := effectByKey(t, "crits-on-kill").Increment
	minicrits := effectByKey(t, "minicrits-on-kill").Increment
	healOnHit := effectByKey(t, "heal-on-hit").Increment
	extra := floatDefine(t, "CritsOnKillExtraSeconds")
	got := levelsDriver(t, `
    BuffReward reward;
    printnum(BuffLevels_KillReward({Shotgun}, reward) ? 1 : 0);
    BuffLevels_Add({Shotgun}, [heal-on-kill], 2);
    BuffLevels_Add({Shotgun}, [crits-on-kill], 3);
    BuffLevels_Add({Shotgun}, [minicrits-on-kill], 1);
    BuffLevels_Add({Shotgun}, [speed-on-kill], 1);
    printnum(BuffLevels_KillReward({Shotgun}, reward) ? 1 : 0);
    printnum(view_as<int>(reward.heal));
    printnum(view_as<int>(reward.crits));
    printnum(view_as<int>(reward.minicrits));
    printnum(reward.speed ? 1 : 0);
    printnum(BuffLevels_JarReward({Jarate}, reward) ? 1 : 0);
    BuffLevels_Add({Jarate}, [heal-on-hit], 2);
    printnum(BuffLevels_JarReward({Jarate}, reward) ? 1 : 0);
    printnum(view_as<int>(reward.heal));
    printnum(reward.speed ? 1 : 0);
`)
	wantCells(t, "kill pays", []int32{got[0], got[1], got[5], got[6], got[7], got[9]}, 0, 1, 1, 0, 1, 0)
	wantFloats(t, "kill reward", got[2:5], 2*heal, 3*crits+extra, minicrits)
	wantFloats(t, "jar heal", got[8:9], 2*healOnHit)
}

/*
A kill by a holstered weapon is the plugin's to pay.

The game reads on-kill buffs through the provider, which carries the weapon in
hand, so a robot that burns after the Pyro put the Gas Passer away was paid as
the flamethrower's. The plugin pays the weapon that killed when it is not the
one held, and only then: paying a held weapon's kill as well doubles it.
*/
func TestTheGamePaysOnlyKillsByTheHeldWeapon(t *testing.T) {
	got := levelsDriver(t, `
    printnum(BuffLevels_GamePaidKill({Gas Passer}, {Flame Thrower}) ? 1 : 0);
    printnum(BuffLevels_GamePaidKill({Flame Thrower}, {Flame Thrower}) ? 1 : 0);
    printnum(BuffLevels_GamePaidKill(-1, {Flame Thrower}) ? 1 : 0);
`)
	wantCells(t, "holstered, held, unknown", got, 0, 1, 1)
}

// A burn death of a robot the attacker gassed is the Gas Passer's kill, once,
// within the credit window, and not when the Gas Passer killed it itself.
func TestGasCreditsOneBurnKillToTheThrower(t *testing.T) {
	window := floatDefine(t, "GasCreditSeconds")
	got := levelsDriver(t, fmt.Sprintf(`
    BuffLevels_Gassed(5, 1, 10.0);
    printnum(BuffLevels_GasKill(5, 2, {Flame Thrower}, 12.0));
    printnum(BuffLevels_GasKill(5, 1, {Gas Passer}, 12.0));
    printnum(BuffLevels_GasKill(5, 1, {Flame Thrower}, 12.0));
    printnum(BuffLevels_GasKill(5, 1, {Flame Thrower}, 13.0));
    BuffLevels_Gassed(6, 1, 10.0);
    printnum(BuffLevels_GasKill(6, 1, {Flame Thrower}, %.1f));
`, 10+window+0.5))
	gasPasser := int32(weaponNamed(t, "Gas Passer").ID - 1)
	wantCells(t, "other attacker, gas passer itself, credited, again, too late", got, -1, -1, gasPasser, -1, -1)
}

// Hit substances: bleed lasts three seconds a level, Jarate ten plus two.
func TestConfirmedHitsCarryTheirSubstances(t *testing.T) {
	got := levelsDriver(t, `
    BuffHit hit;
    printnum(BuffLevels_HitEffects({SMG}, hit) ? 1 : 0);
    BuffLevels_Add({SMG}, [bleed], 2);
    BuffLevels_Add({SMG}, [jarate], 3);
    BuffLevels_Add({SMG}, [mad-milk], 1);
    printnum(BuffLevels_HitEffects({SMG}, hit) ? 1 : 0);
    printnum(view_as<int>(hit.bleed));
    printnum(view_as<int>(hit.jarate));
    printnum(hit.milk ? 1 : 0);
    printnum(hit.ignite ? 1 : 0);
`)
	wantCells(t, "nothing, something", got[:2], 0, 1)
	wantFloats(t, "bleed, jarate", got[2:4], 6, 16)
	wantCells(t, "milk, ignite", got[4:], 1, 0)
}

// Stomp damage is three times the fall plus five a level; a lethal stomp gives
// back a share of the class's base health. Neither touches another weapon.
func TestStompsScaleWithTheirLevels(t *testing.T) {
	base := floatDefine(t, "StompBaseFallDamageMultiplier")
	per := floatDefine(t, "StompFallDamagePerLevel")
	heal := effectByKey(t, "base-health-on-kill").Increment
	got := levelsDriver(t, `
    BuffLevels_Add({Mantreads}, [damage], 2);
    BuffLevels_Add({Shotgun}, [damage], 2);
    BuffLevels_Add({Thermal Thruster}, [base-health-on-kill], 1);
    printnum(view_as<int>(BuffLevels_StompDamageScale({Mantreads})));
    printnum(view_as<int>(BuffLevels_StompDamageScale({Shotgun})));
    printnum(view_as<int>(BuffLevels_StompDamageScale({Thermal Thruster})));
    printnum(view_as<int>(BuffLevels_StompKillHeal({Thermal Thruster}, 200)));
    printnum(view_as<int>(BuffLevels_StompKillHeal({Mantreads}, 200)));
    printnum(view_as<int>(BuffLevels_StompKillHeal({Thermal Thruster}, 0)));
`)
	wantFloats(t, "stomps", got, (base+2*per)/base, 1, 1, 200*heal/100, 0, 0)
}

/*
No-self-blast gives back the health from before the first blast of the frame.

Two blasts landing before the refund must restore the health from before the
first, not from between them, and the refund pays once (#146). Rocket-jump
protection keeps a quarter less a level and never goes below nothing.
*/
func TestSelfBlastRefundsTheHealthBeforeTheFirstHit(t *testing.T) {
	got := levelsDriver(t, `
    printnum(BuffLevels_SelfBlastHit(3, 100) ? 1 : 0);
    printnum(BuffLevels_SelfBlastHit(3, 40) ? 1 : 0);
    printnum(BuffLevels_SelfBlastRefund(3));
    printnum(BuffLevels_SelfBlastRefund(3));
    printnum(BuffLevels_SelfBlastHit(3, 70) ? 1 : 0);
    BuffLevels_ForgetClient(3);
    printnum(BuffLevels_SelfBlastRefund(3));
    BuffLevels_Add({Rocket Launcher}, [rocket-jump-protection], 1);
    BuffLevels_Add({Direct Hit}, [rocket-jump-protection], 5);
    printnum(view_as<int>(BuffLevels_SelfBlastKept({Rocket Launcher})));
    printnum(view_as<int>(BuffLevels_SelfBlastKept({Direct Hit})));
    printnum(view_as<int>(BuffLevels_SelfBlastKept({Shotgun})));
`)
	wantCells(t, "first, overlap, refund, refund again, next frame, forgotten", got[:6], 1, 0, 100, 0, 1, 0)
	wantFloats(t, "kept", got[6:], 0.75, 0, 1)
}

/*
Disposable sentries: the Construction PDA's levels, shared with the toolbox.

The client predicts one more from the toolbox, because placing takes a slot,
and nothing at the upgrade station, where the shop must see only what it sold.
An Engineer needs his primary sentry, and the allowance is the run's levels on
top of the station's.
*/
func TestDisposableSentriesShareTheConstructionPDAsPool(t *testing.T) {
	got := levelsDriver(t, `
    printnum(BuffLevels_DisposablePrediction(25, false));
    printnum(BuffLevels_DisposableRoom(3, true, 0) ? 1 : 0);
    BuffLevels_Add({Construction PDA}, [disposable-sentry], 2);
    printnum(BuffLevels_DisposableSentries());
    printnum(BuffLevels_DisposablePrediction(25, false));
    printnum(BuffLevels_DisposablePrediction(737, false));
    printnum(BuffLevels_DisposablePrediction(28, false));
    printnum(BuffLevels_DisposablePrediction(15028, false));
    printnum(BuffLevels_DisposablePrediction(28, true));
    printnum(BuffLevels_DisposablePrediction(9, false));
    printnum(BuffLevels_DisposableRoom(1, true, 2) ? 1 : 0);
    printnum(BuffLevels_DisposableRoom(1, true, 3) ? 1 : 0);
    printnum(BuffLevels_DisposableRoom(1, false, 0) ? 1 : 0);
`)
	wantCells(t, "prediction and room", got, 0, 0, 2, 2, 2, 3, 3, 0, 0, 1, 0, 0)
}

/*
A gun destroys projectiles on a confirmed shot, think by think.

A shot is a clip, reserve or energy that went down, or the next attack pushed
back while there was something to fire. A weapon switch is not a shot, an
empty gun pushing its next attack is not one, and the miniguns are the game's.
*/
func TestProjectileSweepsFollowConfirmedShots(t *testing.T) {
	got := levelsDriver(t, `
    BuffLevels_Add({Shotgun}, [destroy-projectiles], 2);
    BuffLevels_Add({Minigun}, [destroy-projectiles], 2);
    printnum(BuffLevels_Shot(1, {Shotgun}, 100, 6, 32, -1.0, 1.0));
    printnum(BuffLevels_Shot(1, {Shotgun}, 100, 5, 32, -1.0, 1.6));
    printnum(BuffLevels_Shot(1, {Shotgun}, 100, 5, 32, -1.0, 1.6));
    printnum(BuffLevels_Shot(1, {Shotgun}, 100, 5, 32, -1.0, 2.2));
    printnum(BuffLevels_Shot(1, {Shotgun}, 101, 4, 32, -1.0, 2.2));
    printnum(BuffLevels_Shot(1, {Shotgun}, 101, 0, 32, -1.0, 2.2));
    printnum(BuffLevels_Shot(1, {Shotgun}, 101, 0, 32, -1.0, 3.0));
    BuffLevels_ForgetShot(1);
    printnum(BuffLevels_Shot(1, {Shotgun}, 101, 0, 30, -1.0, 3.0));
    printnum(BuffLevels_Shot(2, {Minigun}, 200, -1, 200, -1.0, 1.0));
    printnum(BuffLevels_Shot(2, {Minigun}, 200, -1, 199, -1.0, 1.1));
    printnum(BuffLevels_Shot(3, {Pomson 6000}, 300, -1, -1, 20.0, 1.0));
    BuffLevels_Add({Pomson 6000}, [destroy-projectiles], 1);
    printnum(BuffLevels_Shot(3, {Pomson 6000}, 300, -1, -1, 15.0, 1.0));
`)
	wantCells(t, "first think, shot, idle, pushed, switch, emptied, empty push, forgotten, minigun, minigun, energy, energy shot",
		got, 0, 2, 0, 2, 0, 2, 0, 0, 0, 0, 0, 1)
}

// The cooldown curve has its own test next door; a weapon with no levels
// sweeps nothing even when it fires.
func TestAShotWithoutTheBuffSweepsNothing(t *testing.T) {
	got := levelsDriver(t, `
    printnum(BuffLevels_Shot(1, {Scattergun}, 100, 6, 32, -1.0, 1.0));
    printnum(BuffLevels_Shot(1, {Scattergun}, 100, 5, 32, -1.0, 1.6));
`)
	wantCells(t, "unbuffed shot", got, 0, 0)
}

// A rebuild in the middle of a life keeps the ammo it found, however far the
// attribute scaled it (apw-b14o). A spawn or a resupply keeps up to the buffed
// maximum, so a max-metal buff starts the Engineer on 300 (apw-hntl), and a
// second resupply on a full reserve still cannot climb past it.
func TestBuffLevelsAmmoKeptAcrossRebuild(t *testing.T) {
	got := levelsDriver(t, `
    printnum(BuffLevels_AmmoKept(31, 46, -1));
    printnum(BuffLevels_AmmoKept(200, 300, -1));
    printnum(BuffLevels_AmmoKept(200, 300, 300));
    printnum(BuffLevels_AmmoKept(300, 450, 300));
    printnum(BuffLevels_AmmoKept(120, 180, 300));
    printnum(BuffLevels_AmmoKept(150, 100, 300));
`)
	wantCells(t, "ammo kept", got, 31, 200, 300, 300, 180, 100)
}
