package gamedata

import (
	"os"
	"strings"
	"testing"
)

func sourceFunction(t *testing.T, path, signature string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	start := strings.Index(text, signature)
	if start < 0 {
		t.Fatalf("%s has no %s", path, signature)
	}
	open := strings.IndexByte(text[start:], '{')
	if open < 0 {
		t.Fatalf("%s has no body for %s", path, signature)
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
	t.Fatalf("%s has an unterminated body for %s", path, signature)
	return ""
}

func TestWeaponBuffsStayOutOfMvMShopping(t *testing.T) {
	station := sourceFunction(t, "../plugin/scripting/tf2_archipelago/mvm.inc",
		"stock void MvM_OnCommandKeyValues")
	for _, command := range []string{"MVM_Upgrade", "MVM_Respec"} {
		if !strings.Contains(station, command) {
			t.Fatalf("upgrade-station handler no longer handles %s", command)
		}
	}
	if !strings.Contains(station, "WeaponBuffs_ApplyNextFrame(client)") {
		t.Fatal("upgrade-station handler does not recalculate the independent provider")
	}

	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	source, err := os.ReadFile(buffs)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "WeaponBuffs_BeginStationTransaction") {
		t.Fatal("obsolete station transaction still couples Archipelago effects to shopping")
	}
	provider := sourceFunction(t, buffs, "static int WeaponBuffs_Provider")
	for _, required := range []string{
		`CreateEntityByName("tf_wearable")`, "RENDER_NONE",
		`SetEntProp(provider, Prop_Send, "m_ProviderType", 0)`,
		`SetEntPropEnt(provider, Prop_Send, "m_hOwnerEntity", client)`,
		`AcceptEntityInput(provider, "RunScriptCode")`,
		`TF2Attrib_HookValueFloat(1.0, "mult_dmg", weaponEntity)`,
		"TF2Attrib_ClearCache(weaponEntity)",
		"g_WeaponBuffProviderRef",
	} {
		if !strings.Contains(provider, required) {
			t.Fatalf("private attribute provider has no %s", required)
		}
	}
	apply := sourceFunction(t, buffs, "void WeaponBuffs_Apply(int client, bool refill")
	for _, required := range []string{
		"TF2Attrib_RemoveAll(provider)",
		"TF2Attrib_ClearCache(entity)",
		"TF2Attrib_HookValueFloat",
		"TF2Attrib_SetByName(provider",
	} {
		if !strings.Contains(apply, required) {
			t.Fatalf("independent attribute composition has no %s", required)
		}
	}
	weaponCheck := strings.Index(apply, "if (weapon < 0)")
	providerCreate := strings.Index(apply, "WeaponBuffs_Provider(client, entity)")
	if weaponCheck < 0 || providerCreate < 0 || weaponCheck > providerCreate {
		t.Fatal("buff application creates a provider before validating the active weapon")
	}
	for _, forbidden := range []string{
		"TF2Attrib_SetByName(entity", "GetPlayerWeaponSlot", "g_WeaponBuffWaveActive",
	} {
		if strings.Contains(apply, forbidden) {
			t.Fatalf("Archipelago apply path still uses station-owned state: %s", forbidden)
		}
	}
	if strings.Contains(string(source), "TF2Econ_") || strings.Contains(string(source), "TF2Util_") {
		t.Fatal("private provider still depends on update-sensitive TF Econ Data or TF2 Utils natives")
	}

	for _, signature := range []string{
		"public Action Command_TestWeaponBuff", "public Action Command_GiveWeaponBuff",
		"public Action Command_GiveSlotWeaponBuff",
	} {
		body := sourceFunction(t, buffs, signature)
		if strings.Contains(body, "WeaponBuffs_ApplyEntity") {
			t.Fatalf("%s bypasses the shopping-period guard", signature)
		}
		if !strings.Contains(body, "WeaponBuffs_Apply(") {
			t.Fatalf("%s does not apply through the guarded client path", signature)
		}
	}
}

func TestDisposableSentryBuffHasAnIndependentBuildAllowance(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	init := sourceFunction(t, buffs, "static void WeaponBuffs_InitDisposableSentryHook")
	for _, required := range []string{
		`DynamicDetour.FromConf(config, "CTFPlayer::CanBuild")`,
		"hook.Enable(Hook_Post, WeaponBuffs_CanBuildPost)",
	} {
		if !strings.Contains(init, required) {
			t.Fatalf("disposable sentry hook setup has no %s", required)
		}
	}

	allowance := sourceFunction(t, buffs, "static bool WeaponBuffs_HasDisposableSentryRoom")
	for _, required := range []string{
		"WeaponBuffs_NativeDisposableSentryLevels(client)",
		`FindEntityByClassname(sentry, "obj_sentrygun")`,
		`"m_bDisposableBuilding"`,
		"BuffLevels_DisposableRoom(",
	} {
		if !strings.Contains(allowance, required) {
			t.Fatalf("independent disposable allowance has no %s", required)
		}
	}

	native := sourceFunction(t, buffs, "static int WeaponBuffs_NativeDisposableSentryLevels")
	if strings.Contains(native, "TF2Attrib_HookValue") ||
		!strings.Contains(native, "WeaponBuffs_DirectDisposableSentryLevels") {
		t.Fatal("native disposable allowance is not read directly from Valve-owned item attributes")
	}

	// Which items predict how many is TestDisposableSentriesShareTheConstructionPDAsPool.
	apply := sourceFunction(t, buffs, "void WeaponBuffs_Apply(int client, bool refill")
	if !strings.Contains(apply, "WeaponBuffs_ApplyDisposableSentryPrediction(client, provider)") {
		t.Fatal("client prediction mirror is not installed with the provider")
	}
	prediction := sourceFunction(t, buffs,
		"static void WeaponBuffs_ApplyDisposableSentryPrediction")
	for _, required := range []string{
		"BuffLevels_DisposablePrediction(definition,",
		"g_WeaponBuffAtUpgradeStation[client]",
		`TF2Attrib_SetByName(provider,`,
		`"engy disposable sentries"`,
	} {
		if !strings.Contains(prediction, required) {
			t.Fatalf("loadout-wide disposable prediction mirror has no %s", required)
		}
	}
	if strings.Contains(prediction, "TF2Attrib_SetByName(item") {
		t.Fatal("disposable prediction writes into the MvM station's item list")
	}
	switchPost := sourceFunction(t, buffs,
		"public void WeaponBuffs_WeaponSwitchPost")
	if !strings.Contains(switchPost, "WeaponBuffs_ApplyNextFrame(client)") {
		t.Fatal("weapon switching applies before m_hActiveWeapon settles")
	}
	listing := sourceFunction(t, buffs, "static int WeaponBuffs_Lines")
	for _, required := range []string{
		"bool seen[WeaponCount]", "weapon < 0 || seen[weapon]", "seen[weapon] = true",
	} {
		if !strings.Contains(listing, required) {
			t.Fatalf("weapon-family listing deduplication has no %s", required)
		}
	}

	station := sourceFunction(t, "../plugin/scripting/tf2_archipelago/mvm.inc",
		"stock void MvM_OnCommandKeyValues")
	for _, required := range []string{
		`StrEqual(name, "MvM_UpgradesBegin", false)`,
		`StrEqual(name, "MvM_UpgradesDone", false)`,
		"WeaponBuffs_EnterUpgradeStation(client)",
		"WeaponBuffs_LeaveUpgradeStation(client)",
	} {
		if !strings.Contains(station, required) {
			t.Fatalf("upgrade station does not isolate the prediction mirror: missing %s", required)
		}
	}

	gamedata, err := os.ReadFile("../plugin/gamedata/tf2_archipelago.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(gamedata), `"CTFPlayer::CanBuild"`) < 2 {
		t.Fatal("plugin gamedata has no CTFPlayer::CanBuild signature/function pair")
	}
}

func TestAttributeProviderFailureIsRateLimited(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	source, err := os.ReadFile(buffs)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"#define WeaponBuffProviderRetrySeconds 5.0",
		"g_WeaponBuffProviderRetryAt[MAXPLAYERS + 1]",
		"GetGameTime() < g_WeaponBuffProviderRetryAt[client]",
		"GetGameTime() + WeaponBuffProviderRetrySeconds",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("provider retry guard has no %q", required)
		}
	}
}

func TestAttributeProviderDoesNotSurvivePluginReload(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	init := sourceFunction(t, buffs, "void WeaponBuffs_Init()")
	if !strings.Contains(init, "WeaponBuffs_RemoveStaleProviders()") {
		t.Fatal("weapon-buff startup does not remove orphaned providers")
	}
	provider := sourceFunction(t, buffs, "static int WeaponBuffs_Provider")
	if !strings.Contains(provider,
		`DispatchKeyValue(provider, "targetname", WeaponBuffProviderTargetName)`) {
		t.Fatal("weapon-buff provider has no reload-stable identity")
	}
	pluginEnd := sourceFunction(t, "../plugin/scripting/tf2_archipelago.sp",
		"public void OnPluginEnd()")
	if !strings.Contains(pluginEnd, "WeaponBuffs_Shutdown()") {
		t.Fatal("plugin unload does not remove the private attribute provider")
	}
}

func TestSlotBuffCommandFindsWeaponWearables(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	resolver := sourceFunction(t, buffs, "static int WeaponBuffs_EntityInLoadoutSlot")
	for _, required := range []string{
		"GetPlayerWeaponSlot(client, slot)",
		`FindEntityByClassname(entity, "tf_wearable*")`,
		"Unlocks_WearableSlot(class, definition) == slot",
	} {
		if !strings.Contains(resolver, required) {
			t.Errorf("loadout slot resolver has no %q", required)
		}
	}

	command := sourceFunction(t, buffs, "public Action Command_GiveSlotWeaponBuff")
	for _, required := range []string{
		"WeaponBuffs_ParseLoadoutSlot(rawSlot)",
		"WeaponBuffs_EntityInLoadoutSlot(target, slot)",
		"WeaponBuffs_AddTestEffects(weapon, wanted, levels)",
	} {
		if !strings.Contains(command, required) {
			t.Errorf("slot buff command has no %q", required)
		}
	}
}

func TestJoiningClientSnapshotIsNotChurnedByBotRemoval(t *testing.T) {
	plugin := "../plugin/scripting/tf2_archipelago.sp"
	putInServer := sourceFunction(t, plugin, "public void OnClientPutInServer")
	if strings.Contains(putInServer, "Bots_MakeRoom()") {
		t.Fatal("initial client sign-on still removes a bot during the entity snapshot")
	}
	joinRed := sourceFunction(t, plugin, "public Action Command_JoinRed")
	if !strings.Contains(joinRed, "Bots_MakeRoom()") {
		t.Fatal("join-team path no longer frees a bot seat for the loaded player")
	}
}

func TestNonMinigunProjectileDestructionUsesConfirmedShots(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	postThink := sourceFunction(t, buffs, "public void WeaponBuffs_PostThinkPost")
	// What counts as a shot is TestProjectileSweepsFollowConfirmedShots.
	for _, required := range []string{
		"BuffLevels_Shot(client, catalog, reference, clip, ammo, energy, nextAttack)",
		"WeaponBuffs_AttackEnemyProjectiles(client, levels)",
	} {
		if !strings.Contains(postThink, required) {
			t.Errorf("confirmed-shot detector has no %q", required)
		}
	}

	sweep := sourceFunction(t, buffs, "static void WeaponBuffs_AttackEnemyProjectiles")
	for _, required := range []string{
		"ProjectileDestructionRange",
		"ProjectileDestructionRadius",
		"WeaponBuffs_ProjectileDestructionCooldown(levels)",
		"WeaponBuffs_IsDestroyableProjectile(projectile)",
		"WeaponBuffs_ProjectileVisible(client, start, point)",
		"chosen = projectile",
		"RemoveEntity(chosen)",
		"WeaponBuffs_ShowProjectileDud(chosen, chosenPoint)",
		`EmitGameSoundToAll("Halloween.HeadlessBossAxeHitWorld", chosen)`,
		"TE_SetupSparks(chosenPoint, sparkDirection, 2, 1)",
		`WeaponBuffs_DebugLog("[AP destroy]`,
	} {
		if !strings.Contains(sweep, required) {
			t.Errorf("projectile sweep has no %q", required)
		}
	}
	visible := sourceFunction(t, buffs, "static bool WeaponBuffs_ProjectileVisible")
	for _, required := range []string{
		"ScaleVector(direction, 48.0)",
		"fraction >= 0.99",
		"TR_TraceRayFilterEx",
		"WeaponBuffs_ProjectileVisibilityFilter, client",
		`WeaponBuffs_DebugLog("[AP destroy] candidate blocked at trace fraction`,
	} {
		if !strings.Contains(visible, required) {
			t.Errorf("projectile visibility test has no %q", required)
		}
	}
	filter := sourceFunction(t, buffs, "public bool WeaponBuffs_ProjectileVisibilityFilter")
	for _, required := range []string{
		"entity == client",
		`HasEntProp(entity, Prop_Send, "m_hOwnerEntity")`,
		`GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity") == client`,
	} {
		if !strings.Contains(filter, required) {
			t.Errorf("projectile visibility filter has no %q", required)
		}
	}
	target := sourceFunction(t, buffs, "static bool WeaponBuffs_IsDestroyableProjectile")
	if !strings.Contains(target, `StrContains(classname, "tf_projectile_", false) == 0`) {
		t.Fatal("projectile destruction does not accept every TF2 projectile class")
	}

	text, err := os.ReadFile(buffs)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"#define ProjectileDestructionRange 2000.0",
		"#define ProjectileDestructionRadius 256.0",
	} {
		if !strings.Contains(string(text), required) {
			t.Errorf("projectile destruction configuration has no %q", required)
		}
	}

	/* The cooldown itself is not asked about here any more. It moved to
	   weapon_buff_levels.inc and TestProjectileDestructionCooldownIsTheDeclaredCurve
	   runs it, which catches the floor being inverted. Reading the source for
	   "float(levels - 1) * ProjectileDestructionCooldownStep" did not. */
	dud := sourceFunction(t, buffs, "static void WeaponBuffs_ShowProjectileDud")
	for _, required := range []string{
		`StartMessageAll("BreakModelRocketDud", USERMSG_RELIABLE)`,
		"BfWriteShort(message, model)",
		"BfWriteVecCoord(message, point)",
		"BfWriteAngles(message, angles)",
	} {
		if !strings.Contains(dud, required) {
			t.Errorf("projectile dud effect has no %q", required)
		}
	}
}

func TestWeaponBuffOwnershipPolicyDefaultsToPlayers(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	source, err := os.ReadFile(buffs)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		`CreateConVar("tf2ap_mirror_buffs_to_robots", "0"`,
		"g_WeaponBuffMirrorRobots.AddChangeHook",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("robot mirroring toggle has no %s", required)
		}
	}

	policy := sourceFunction(t, buffs, "static bool WeaponBuffs_CanUseRunBuffs(int client)")
	for _, required := range []string{
		"MvM_IsPlayer(client)",
		"g_WeaponBuffMirrorRobots.BoolValue",
		"WeaponBuffs_IsEnemyRobot(client)",
		"g_WeaponBuffDefenderBots.BoolValue",
		"WeaponBuffs_IsDefenderBot(client)",
	} {
		if !strings.Contains(policy, required) {
			t.Fatalf("weapon-buff ownership policy has no %s", required)
		}
	}

	for _, signature := range []string{
		"void WeaponBuffs_Apply(int client, bool refill",
		"public void WeaponBuffs_OnTakeDamagePost",
		"static void WeaponBuffs_FireExtraProjectiles(any reference)",
		"public void WeaponBuffs_SubstanceProjectileSpawned(int entity)",
	} {
		body := sourceFunction(t, buffs, signature)
		if !strings.Contains(body, "WeaponBuffs_CanUseRunBuffs(") {
			t.Fatalf("%s bypasses the run-buff ownership policy", signature)
		}
	}

	for _, signature := range []string{
		"public void WeaponBuffs_MirrorRobotsChanged", "public void WeaponBuffs_DefenderBotsChanged",
	} {
		changed := sourceFunction(t, buffs, signature)
		for _, required := range []string{"WeaponBuffs_Apply(client)", "WeaponBuffs_Remove(client)"} {
			if !strings.Contains(changed, required) {
				t.Fatalf("%s has no %s", signature, required)
			}
		}
	}
	if !strings.Contains(sourceFunction(t, buffs, "void WeaponBuffs_Init()"), `CreateConVar("tf2ap_buffs_for_defender_bots", "0"`) {
		t.Fatal("the defender-bot toggle is not off by default")
	}
}

// A jar splash is not a weapon attack, so the game never reads heal on hit or
// speed on hit off it, and the on-kill attributes are read through the
// provider, which carries the held weapon's buffs rather than the killing
// weapon's. Both are paid by the plugin, on the path that already knows the
// weapon.
func TestJarHitsAndHolsteredKillsArePaidByThePlugin(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	splash := sourceFunction(t, buffs, "public void WeaponBuffs_ApplySubstances(any rawPack)")
	if !strings.Contains(splash, "WeaponBuffs_ApplyJarHitRewards(attacker, weapon)") {
		t.Fatal("a jar splash pays no heal or speed on hit")
	}
	jar := sourceFunction(t, buffs, "static void WeaponBuffs_ApplyJarHitRewards")
	// What a splash and a kill pay is TestKillsAndJarSplashesPayTheirLevels.
	for _, required := range []string{
		"BuffLevels_JarReward(weapon, reward)",
		"WeaponBuffs_HealClient(attacker",
		"TF2_AddCondition(attacker, TFCond_SpeedBuffAlly, SpeedOnHitSeconds",
	} {
		if !strings.Contains(jar, required) {
			t.Fatalf("jar hit rewards have no %s", required)
		}
	}

	death := sourceFunction(t, buffs, "public void WeaponBuffs_PlayerDeath")
	for _, required := range []string{
		`event.GetInt("weapon_def_index")`,
		"WeaponBuffs_CanUseRunBuffs(attacker)",
		"GetClientTeam(attacker) == GetClientTeam(victim)",
		"BuffLevels_GamePaidKill(weapon, active)",
		"WeaponBuffs_ApplyKillRewards(attacker, weapon)",
		"WeaponBuffs_CreditGasBurn(attacker, victim, weapon)",
	} {
		if !strings.Contains(death, required) {
			t.Fatalf("holstered kill path has no %s", required)
		}
	}
	kill := sourceFunction(t, buffs, "static void WeaponBuffs_ApplyKillRewards")
	for _, required := range []string{
		"BuffLevels_KillReward(weapon, reward)", "TFCond_CritOnKill", "TFCond_MiniCritOnKill",
		"TFCond_SpeedBuffAlly, SpeedOnKillSeconds",
	} {
		if !strings.Contains(kill, required) {
			t.Fatalf("kill rewards have no %s", required)
		}
	}
	if !strings.Contains(sourceFunction(t, buffs, "void WeaponBuffs_Init()"), `HookEvent("player_death", WeaponBuffs_PlayerDeath)`) {
		t.Fatal("the plugin does not watch kills")
	}
	gas := sourceFunction(t, buffs, "static void WeaponBuffs_CreditGasBurn")
	if !strings.Contains(gas, "BuffLevels_GasKill(victim, attacker, killedWith, GetGameTime())") {
		t.Fatal("gas burn credit does not ask the levels who to pay")
	}
	if !strings.Contains(sourceFunction(t, buffs, "public Action Timer_WeaponBuffGasWatch"), "TF2_IsPlayerInCondition(client, TFCond_Gas)") {
		t.Fatal("nothing watches robots turning gassed")
	}
}

func TestSelfBlastBuffsPreserveTheNativeExplosionAndPush(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	hook := sourceFunction(t, buffs, "void WeaponBuffs_HookClient(int client)")
	if !strings.Contains(hook, "SDKHook_OnTakeDamage, WeaponBuffs_OnTakeDamage") {
		t.Fatal("players are not hooked for modifiable damage")
	}
	if !strings.Contains(hook, "SDKHook_OnTakeDamageAlivePost, WeaponBuffs_OnTakeDamageAlivePost") {
		t.Fatal("players are not hooked to restore self-blast health")
	}

	damage := sourceFunction(t, buffs, "public Action WeaponBuffs_OnTakeDamage")
	for _, required := range []string{
		"victim != attacker",
		"!(damagetype & DMG_BLAST)",
		"WeaponBuffs_WeaponOfHit(attacker, inflictor, weapon, damagecustom)",
		"WeaponBuffs_ForEntity(weapon)",
		"BuffLevels_SelfBlastKept(catalog)",
		"damage *= kept",
	} {
		if !strings.Contains(damage, required) {
			t.Fatalf("self-blast damage path has no %q", required)
		}
	}
	for _, required := range []string{
		"BuffLevels_Of(catalog, NoSelfBlastEffect)",
		"BuffLevels_SelfBlastHit(victim, health)",
		"WeaponBuffs_SelfBlastGuardHealth(health, damage)",
		"RequestFrame(Frame_RestoreSelfBlastHealth, GetClientUserId(victim))",
	} {
		if !strings.Contains(damage, required) {
			t.Fatalf("self-blast health refund path has no %q", required)
		}
	}
	if strings.Contains(damage, "damage = 0.0") {
		t.Fatal("no-self-blast zeroes the SDKHook event and suppresses blast movement")
	}
	post := sourceFunction(t, buffs, "public void WeaponBuffs_OnTakeDamageAlivePost")
	if !strings.Contains(post, "WeaponBuffs_RestorePendingSelfBlastHealth(victim)") {
		t.Fatal("self-blast health is not restored after alive damage")
	}
	frame := sourceFunction(t, buffs, "public void Frame_RestoreSelfBlastHealth")
	if !strings.Contains(frame, "WeaponBuffs_RestorePendingSelfBlastHealth(client)") {
		t.Fatal("cancelled self-blast damage can leave temporary guard health behind")
	}
	restore := sourceFunction(t, buffs, "void WeaponBuffs_RestoreSelfBlastHealth")
	if !strings.Contains(restore, "SetEntityHealth(victim, health)") {
		t.Fatal("self-blast health is not restored after the engine applies push")
	}

	native := sourceFunction(t, buffs, "static void WeaponBuffs_ClearOldSelfBlastAttributes")
	for _, required := range []string{
		"GetPlayerWeaponSlot(client, slot)",
		"TF2Attrib_RemoveByName(entity, LegacySelfBlastDamageAttribute)",
		"TF2Attrib_RemoveByName(entity, NoSelfBlastAttribute)",
		"TF2Attrib_ClearCache(entity)",
	} {
		if !strings.Contains(native, required) {
			t.Fatalf("old self-blast attribute cleanup has no %q", required)
		}
	}
	if strings.Contains(native, "TF2Attrib_SetByName") {
		t.Fatal("no-self-blast still applies an attribute that changes explosion visuals")
	}

	apply := sourceFunction(t, buffs, "void WeaponBuffs_Apply(int client, bool refill")
	if !strings.Contains(apply, "WeaponBuffs_ClearOldSelfBlastAttributes(client)") {
		t.Fatal("buff application does not clear old self-blast attributes")
	}
	remove := sourceFunction(t, buffs, "static void WeaponBuffs_Remove(int client)")
	if !strings.Contains(remove, "WeaponBuffs_ClearOldSelfBlastAttributes(client)") {
		t.Fatal("buff removal leaves old no-self-blast attributes behind")
	}
}

func TestExtraHealingBoltsUseNativeHealingAndPreservePenetration(t *testing.T) {
	buffs := "../plugin/scripting/tf2_archipelago/weapon_buffs.inc"
	native := sourceFunction(t, buffs, "static int WeaponBuffs_FireNativeArrow")
	for _, required := range []string{
		"g_WeaponBuffArrowCreate", "m_iProjectileType",
		"SDKCall(g_WeaponBuffArrowCreate", "g_WeaponBuffProjectileSetLauncher",
		"SDKCall(g_WeaponBuffProjectileSetLauncher, extra, launcher)",
		`GetEntPropFloat(source, Prop_Data, "m_flDamage")`,
		"WeaponBuffs_CopyArrowPenetration(source, extra)",
	} {
		if !strings.Contains(native, required) {
			t.Fatalf("native arrow fanout has no %s", required)
		}
	}

	for _, forbidden := range []string{"g_WeaponBuffArrowFire", "SDKCall(g_WeaponBuffArrowFire"} {
		if strings.Contains(native, forbidden) {
			t.Fatalf("extra arrows still patch private penetration state with %s", forbidden)
		}
	}

	for _, path := range []string{
		"../plugin/gamedata/tf2_archipelago.txt",
		"../launcher/internal/assets/embedded/tf2_archipelago.txt",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text,
			"@_ZN19CTFProjectile_Arrow6CreateERK6VectorRK6QAngleff16ProjectileType_tP11CBaseEntityS8_") {
			t.Fatalf("%s has no Linux native arrow factory", path)
		}
		if !strings.Contains(text, `"CBaseProjectile::SetLauncher"`) {
			t.Fatalf("%s has no native SetLauncher offset", path)
		}
	}
}
