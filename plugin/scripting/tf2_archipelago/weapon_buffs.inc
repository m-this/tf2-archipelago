/** Every positive weapon/effect permutation received from the multiworld. */

#define DamageEffect 0
#define ProjectileCountEffect 5
#define DestroyProjectilesEffect 19
#define DisposableSentryEffect 36
#define FireRateEffect 1
#define ClipSizeEffect 3
// The multiplier a melee weapon's fire rate never goes below: +50% and no more.
#define MeleeFireRateFloor 0.5
#define BleedEffect 8
#define IgniteEffect 9
#define MadMilkEffect 14
#define GasolineEffect 15
#define MarkForDeathEffect 16
#define JarateEffect 60
#define HealOnHitEffect 21
#define SpeedOnHitEffect 79
// Measured on a server: speed_boost_on_kill lasts one second, and critboost
// on kill lasts a second longer than the attribute says.
#define SpeedOnKillSeconds 1.0
#define SpeedOnHitSeconds 1.0
#define CritsOnKillExtraSeconds 1.0
#define GasPasserDefinition 1180
// Gas coats for ten seconds and the burn it starts lasts ten more.
#define GasCreditSeconds 25.0
#define UberOnHitEffect 39
// What a syringe lands instead of the saw's five percent. A syringe gun puts
// out hits far faster than a swing, so the same buff charges a fifth as much.
#define SyringeUberOnHitFraction 0.01
#define AmmoOnHitEffect 69
#define NoSelfBlastEffect 62
#define SelfBlastDamageAttribute "blast dmg to self increased"
#define RocketJumpProtectionEffect 72
// What one level of rocket-jump protection takes off self-inflicted blast damage.
#define RocketJumpProtectionPerLevel 0.25
#define ProjectileDestructionRange 2000.0
#define ProjectileDestructionRadius 256.0
#define EnergyWeaponMaxCharge 20.0
#define EnergyClipSizeAttribute "clip size bonus upgrade"
#define EnergyClipSizeAttributeClass "mult_clipsize_upgrade"
#define WeaponBuffProviderRetrySeconds 5.0
#define WeaponBuffProviderTargetName "tf2ap_weapon_buff_provider"
#define ItemChargeMeterMaximum 100.0
#define RocketPackReadyCharge 50.0
#define StompBaseFallDamageMultiplier 3.0
#define StompFallDamagePerLevel 5.0

#define BuffMode_Add 1
#define BuffMode_Percentage 2
#define BuffMode_Toggle 3

int g_WeaponEffectLevels[WeaponCount][WeaponEffectCount];
int g_WeaponBuffProviderRef[MAXPLAYERS + 1];
float g_WeaponBuffProviderRetryAt[MAXPLAYERS + 1];
bool g_WeaponBuffAtUpgradeStation[MAXPLAYERS + 1];
int g_DisposableSentryLastSwitchArgumentRef[MAXPLAYERS + 1];
int g_DisposableSentryLastSwitchActiveRef[MAXPLAYERS + 1];
int g_DisposableSentryCanBuildCalls[MAXPLAYERS + 1];
int g_DisposableSentryCanBuildOverrides[MAXPLAYERS + 1];
int g_DisposableSentryLastCanBuildResult[MAXPLAYERS + 1];
ConVar g_WeaponBuffProjectileDebug;
ConVar g_WeaponBuffMirrorRobots;
ConVar g_WeaponBuffDefenderBots;
bool g_WeaponBuffFiringExtra;
int g_WeaponBuffLastExtra;
Handle g_WeaponBuffFlareCreate;
Handle g_WeaponBuffArrowCreate;
Handle g_WeaponBuffProjectileGetDamage;
Handle g_WeaponBuffProjectileSetDamage;
Handle g_WeaponBuffProjectileSetLauncher;
Handle g_WeaponBuffFlareAdd;
bool g_WeaponBuffNativeExtra[2049];
float g_WeaponBuffNativeExtraBorn[2049];
bool g_WeaponBuffArrowExtra[2049];
bool g_WeaponBuffArrowTouched[2049];
float g_WeaponBuffArrowOrigin[2049][3];
int g_WeaponBuffStickyMimic[2049];
int g_WeaponBuffLastButtons[MAXPLAYERS + 1];
int g_WeaponBuffShotWeaponRef[MAXPLAYERS + 1];
int g_WeaponBuffShotClip[MAXPLAYERS + 1];
int g_WeaponBuffShotAmmo[MAXPLAYERS + 1];
float g_WeaponBuffShotEnergy[MAXPLAYERS + 1];
float g_WeaponBuffShotNextAttack[MAXPLAYERS + 1];
int g_WeaponBuffProjectileDestructionHits[2049];
float g_WeaponBuffProjectileDestructionReadyAt[MAXPLAYERS + 1];
int g_WeaponBuffLastSubstanceWeapon[MAXPLAYERS + 1];
float g_WeaponBuffLastSubstanceTime[MAXPLAYERS + 1];
float g_WeaponBuffLastGasThrow[MAXPLAYERS + 1];
int g_WeaponBuffGasFrom[MAXPLAYERS + 1];
float g_WeaponBuffGasTime[MAXPLAYERS + 1];
bool g_WeaponBuffGassed[MAXPLAYERS + 1];
#define ProjectileDebugLineCount 24
char g_WeaponBuffProjectileLines[ProjectileDebugLineCount][256];
int g_WeaponBuffProjectileLineNext;
int g_WeaponBuffProjectileLineHeld;

static void WeaponBuffs_DebugLog(const char[] format, any ...)
{
    char line[256];
    VFormat(line, sizeof(line), format, 2);
    LogMessage("%s", line);
    strcopy(g_WeaponBuffProjectileLines[g_WeaponBuffProjectileLineNext],
        sizeof(g_WeaponBuffProjectileLines[]), line);
    g_WeaponBuffProjectileLineNext = (g_WeaponBuffProjectileLineNext + 1) % ProjectileDebugLineCount;
    if (g_WeaponBuffProjectileLineHeld < ProjectileDebugLineCount)
    {
        g_WeaponBuffProjectileLineHeld++;
    }
}

void WeaponBuffs_Init()
{
    WeaponBuffs_RemoveStaleProviders();
    g_WeaponBuffProjectileDebug = CreateConVar("tf2ap_projectile_debug", "0",
        "Log extra-projectile creation and lifetime", FCVAR_NONE, true, 0.0, true, 1.0);
    CreateTimer(1.0, Timer_WeaponBuffHealthRegen, _, TIMER_REPEAT);
    CreateTimer(0.5, Timer_WeaponBuffGasWatch, _, TIMER_REPEAT);
    g_WeaponBuffMirrorRobots = CreateConVar("tf2ap_mirror_buffs_to_robots", "0",
        "Mirror Archipelago weapon buffs to enemy BLU robots", FCVAR_NONE, true, 0.0, true, 1.0);
    g_WeaponBuffMirrorRobots.AddChangeHook(WeaponBuffs_MirrorRobotsChanged);
    g_WeaponBuffDefenderBots = CreateConVar("tf2ap_buffs_for_defender_bots", "0",
        "Let the defender bots carry the run's weapon buffs, to measure a buff on a server with no player", FCVAR_NONE, true, 0.0, true, 1.0);
    g_WeaponBuffDefenderBots.AddChangeHook(WeaponBuffs_DefenderBotsChanged);
    HookEvent("player_death", WeaponBuffs_PlayerDeath);
    HookEvent("rocketpack_launch", WeaponBuffs_RocketPackLaunch);
    WeaponBuffs_InitDisposableSentryHook();
    WeaponBuffs_InitProjectileCalls();
    UserMsg jarated = GetUserMessageId("PlayerJarated");
    if (jarated == INVALID_MESSAGE_ID || GetUserMessageType() != UM_BitBuf)
    {
        LogError("Jar substance buffs disabled: PlayerJarated is unavailable or not a bit-buffer message.");
    }
    else
    {
        HookUserMessage(jarated, WeaponBuffs_PlayerJarated);
    }
    for (int client = 1; client <= MaxClients; client++)
    {
        if (IsClientInGame(client))
        {
            WeaponBuffs_HookClient(client);
        }
    }
    int tank = -1;
    while ((tank = FindEntityByClassname(tank, "tank_boss")) != -1)
    {
        WeaponBuffs_HookTank(tank);
    }
}

static bool WeaponBuffs_IsPrivateProvider(int entity)
{
    if (entity <= MaxClients || !IsValidEntity(entity)
        || !HasEntProp(entity, Prop_Send, "m_iItemDefinitionIndex")
        || GetEntProp(entity, Prop_Send, "m_iItemDefinitionIndex") != 940
        || !HasEntProp(entity, Prop_Send, "m_ProviderType")
        || GetEntProp(entity, Prop_Send, "m_ProviderType") != 0
        || GetEntityRenderMode(entity) != RENDER_NONE)
    {
        return false;
    }
    char targetname[48];
    if (HasEntProp(entity, Prop_Data, "m_iName"))
    {
        GetEntPropString(entity, Prop_Data, "m_iName", targetname,
            sizeof(targetname));
    }
    // Legacy providers had no targetname. Their invisible, generic, level-one
    // Ghostly Gibus shape is otherwise unused and safe to retire on startup.
    return StrEqual(targetname, WeaponBuffProviderTargetName)
        || (HasEntProp(entity, Prop_Send, "m_iEntityLevel")
            && GetEntProp(entity, Prop_Send, "m_iEntityLevel") == 1);
}

static int WeaponBuffs_PrivateProviderCount()
{
    int count;
    int entity = -1;
    while ((entity = FindEntityByClassname(entity, "tf_wearable")) != -1)
    {
        if (WeaponBuffs_IsPrivateProvider(entity))
        {
            count++;
        }
    }
    return count;
}

static void WeaponBuffs_RemoveStaleProviders()
{
    int entity = -1;
    while ((entity = FindEntityByClassname(entity, "tf_wearable")) != -1)
    {
        if (WeaponBuffs_IsPrivateProvider(entity))
        {
            RemoveEntity(entity);
        }
    }
}

void WeaponBuffs_Shutdown()
{
    for (int client = 1; client <= MaxClients; client++)
    {
        int provider = EntRefToEntIndex(g_WeaponBuffProviderRef[client]);
        g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
        if (provider > MaxClients && IsValidEntity(provider))
        {
            RemoveEntity(provider);
        }
    }
}

// CTFPlayer::CanBuild result values from tf_shareddefs.h. The hook changes only
// the ordinary one-sentry limit; every other refusal remains the game's.
#define CanBuild_Allowed 0
#define CanBuild_LimitReached 2

static void WeaponBuffs_InitDisposableSentryHook()
{
    GameData config = LoadGameConfigFile("tf2_archipelago");
    if (config == null)
    {
        LogError("Disposable-sentry buffs disabled: tf2_archipelago gamedata is missing.");
        return;
    }
    DynamicDetour hook = DynamicDetour.FromConf(config, "CTFPlayer::CanBuild");
    delete config;
    if (hook == null || !hook.Enable(Hook_Post, WeaponBuffs_CanBuildPost))
    {
        LogError("Disposable-sentry buffs disabled: CTFPlayer::CanBuild could not be hooked.");
    }
    delete hook;
}

static int WeaponBuffs_DisposableSentryLevels()
{
    int constructionPDA = WeaponBuffs_ForDefinition(25);
    return constructionPDA >= 0
        ? g_WeaponEffectLevels[constructionPDA][DisposableSentryEffect] : 0;
}

static int WeaponBuffs_DirectDisposableSentryLevels(int entity)
{
    if (entity < 1 || !IsValidEntity(entity))
    {
        return 0;
    }
    Address attribute = TF2Attrib_GetByName(entity, "engy disposable sentries");
    return attribute == Address_Null ? 0
        : RoundToFloor(TF2Attrib_GetValue(attribute));
}

static int WeaponBuffs_NativeDisposableSentryLevels(int client)
{
    // AP never writes this attribute onto the player or his loadout entities,
    // so direct values are exactly the ranks owned by Valve's MvM station.
    int levels = WeaponBuffs_DirectDisposableSentryLevels(client);
    for (int slot = 0; slot <= 5; slot++)
    {
        levels += WeaponBuffs_DirectDisposableSentryLevels(
            WeaponBuffs_EntityInLoadoutSlot(client, slot));
    }
    return levels;
}

static bool WeaponBuffs_HasDisposableSentryRoom(int client)
{
    int apLevels = WeaponBuffs_DisposableSentryLevels();
    if (apLevels <= 0)
    {
        return false;
    }

    // AP never supplies this attribute. What remains is exactly the upgrade
    // station's allowance, appended to the levels held by the multiworld.
    int allowed = apLevels + WeaponBuffs_NativeDisposableSentryLevels(client);
    int disposable;
    bool hasPrimary;
    int sentry = -1;
    while ((sentry = FindEntityByClassname(sentry, "obj_sentrygun")) != -1)
    {
        if (GetEntPropEnt(sentry, Prop_Send, "m_hBuilder") != client)
        {
            continue;
        }
        // Placement ghosts have a builder but are not in the player's object
        // list yet. Counting one would turn the first real sentry disposable.
        if (HasEntProp(sentry, Prop_Send, "m_bPlacing")
            && GetEntProp(sentry, Prop_Send, "m_bPlacing") != 0)
        {
            continue;
        }
        if (GetEntProp(sentry, Prop_Send, "m_bDisposableBuilding") != 0)
        {
            disposable++;
        }
        else
        {
            hasPrimary = true;
        }
    }
    return hasPrimary && disposable < allowed;
}

public MRESReturn WeaponBuffs_CanBuildPost(int client, DHookReturn result,
    DHookParam params)
{
    if (client >= 1 && client <= MaxClients
        && params.Get(1) == view_as<int>(TFObject_Sentry))
    {
        g_DisposableSentryCanBuildCalls[client]++;
        g_DisposableSentryLastCanBuildResult[client] = result.Value;
    }
    if (result.Value != CanBuild_LimitReached
        || params.Get(1) != view_as<int>(TFObject_Sentry)
        || client < 1 || client > MaxClients || !IsClientInGame(client)
        || !WeaponBuffs_CanUseRunBuffs(client)
        || !WeaponBuffs_HasDisposableSentryRoom(client))
    {
        return MRES_Ignored;
    }
    g_DisposableSentryCanBuildOverrides[client]++;
    result.Value = CanBuild_Allowed;
    return MRES_Override;
}

void WeaponBuffs_EnterUpgradeStation(int client)
{
    g_WeaponBuffAtUpgradeStation[client] = true;
    WeaponBuffs_Apply(client);
}

void WeaponBuffs_LeaveUpgradeStation(int client)
{
    g_WeaponBuffAtUpgradeStation[client] = false;
    WeaponBuffs_ApplyNextFrame(client);
}

/*
The buffs that follow the player rather than the weapon in his hands.

MvM's own movement, jump and regeneration upgrades are bought on the class and
never stop working, and a player asked for these to read the same way (gh-32).
The provider is attached to the player, so the attribute reaches him whatever
he is holding; what has to change is which weapons are counted. Levels add up
across the loadout, so a Scout who rolled the same buff on his Scattergun and
his Bat carries both.
*/

/*
What one level of an effect is worth on this weapon.

Nearly every effect is worth the same everywhere and the generated increment is
the answer. ÜberCharge on hit is the exception a player asked for (gh-32): the
saw keeps five percent a swing and a syringe gun, which lands hits far faster,
charges one. The apworld describes the pair the same way.
*/
static float WeaponBuffs_EffectIncrement(int weapon, int effect)
{
    if (effect == UberOnHitEffect && WeaponBuffs_IsSyringeGun(weapon))
    {
        return SyringeUberOnHitFraction;
    }
    return g_WeaponEffectIncrements[effect];
}

static bool WeaponBuffs_IsSyringeGun(int weapon)
{
    return weapon >= 0
        && (StrEqual(g_WeaponNames[weapon], "Blutsauger")
            || StrEqual(g_WeaponNames[weapon], "Overdose")
            || StrEqual(g_WeaponNames[weapon], "Syringe Gun"));
}

static int WeaponBuffs_LoadoutLevels(int client, int effect)
{
    int levels;
    // Every slot the game has, not the three the unlocks work in: a Spy's watch
    // and an Engineer's PDAs sit past those and can carry one of these.
    for (int slot = 0; slot <= 5; slot++)
    {
        int weapon = WeaponBuffs_ForEntity(WeaponBuffs_EntityInLoadoutSlot(client, slot));
        if (weapon >= 0)
        {
            levels += g_WeaponEffectLevels[weapon][effect];
        }
    }
    return levels;
}

public Action Timer_WeaponBuffHealthRegen(Handle timer)
{
    int resource = GetPlayerResourceEntity();
    if (resource <= MaxClients || !IsValidEntity(resource)
        || !HasEntProp(resource, Prop_Send, "m_iMaxHealth"))
    {
        return Plugin_Continue;
    }

    for (int client = 1; client <= MaxClients; client++)
    {
        if (!WeaponBuffs_CanUseRunBuffs(client) || !IsPlayerAlive(client))
        {
            continue;
        }
        int levels = WeaponBuffs_LoadoutLevels(client, ActiveHealthRegenEffect);
        if (levels <= 0)
        {
            continue;
        }

        WeaponBuffs_HealClient(client, RoundToFloor(
            g_WeaponEffectIncrements[ActiveHealthRegenEffect] * float(levels)));
    }
    return Plugin_Continue;
}

// Heals up to the player's real maximum and shows the heal popup, the way the
// game's own heal on hit does. Returns what landed.
static int WeaponBuffs_HealClient(int client, int amount)
{
    int resource = GetPlayerResourceEntity();
    if (resource <= MaxClients || !IsValidEntity(resource)
        || !HasEntProp(resource, Prop_Send, "m_iMaxHealth"))
    {
        return 0;
    }
    // CTFPlayerResource asks CTFPlayer::GetMaxHealth for this value. That
    // includes class health and every active weapon, MvM and Archipelago
    // max-health modifier; m_iMaxHealth on the entity itself does not.
    int maximum = GetEntProp(resource, Prop_Send, "m_iMaxHealth", 4, client);
    int health = GetClientHealth(client);
    int missing = maximum - health;
    int healed = amount < missing ? amount : missing;
    if (healed <= 0)
    {
        return 0;
    }
    SetEntityHealth(client, health + healed);
    Event event = CreateEvent("player_healonhit");
    if (event != null)
    {
        int active = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
        event.SetInt("amount", healed);
        event.SetInt("entindex", client);
        event.SetInt("weapon_def_index", active > MaxClients && IsValidEntity(active)
            ? GetEntProp(active, Prop_Send, "m_iItemDefinitionIndex") : -1);
        event.Fire();
    }
    return healed;
}

static bool WeaponBuffs_IsEnemyRobot(int client)
{
    return client > 0 && client <= MaxClients
        && IsClientInGame(client)
        && IsFakeClient(client)
        && !IsClientSourceTV(client)
        && GetClientTeam(client) == TeamBlue;
}

static bool WeaponBuffs_IsDefenderBot(int client)
{
    return MvM_IsDefender(client)
        && IsFakeClient(client)
        && !IsClientSourceTV(client);
}

// The catalog is run-wide, so every gameplay path must pass through this
// ownership policy before reading it. Otherwise a robot carrying a matching
// weapon silently receives the human player's Archipelago effects.
static bool WeaponBuffs_CanUseRunBuffs(int client)
{
    return MvM_IsPlayer(client)
        || (g_WeaponBuffMirrorRobots != null
            && g_WeaponBuffMirrorRobots.BoolValue
            && WeaponBuffs_IsEnemyRobot(client))
        || (g_WeaponBuffDefenderBots != null
            && g_WeaponBuffDefenderBots.BoolValue
            && WeaponBuffs_IsDefenderBot(client));
}

public void WeaponBuffs_MirrorRobotsChanged(ConVar convar,
    const char[] oldValue, const char[] newValue)
{
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!WeaponBuffs_IsEnemyRobot(client))
        {
            continue;
        }
        if (convar.BoolValue && IsPlayerAlive(client))
        {
            WeaponBuffs_Apply(client);
        }
        else
        {
            WeaponBuffs_Remove(client);
        }
    }
}

public void WeaponBuffs_DefenderBotsChanged(ConVar convar,
    const char[] oldValue, const char[] newValue)
{
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!WeaponBuffs_IsDefenderBot(client))
        {
            continue;
        }
        if (convar.BoolValue && IsPlayerAlive(client))
        {
            WeaponBuffs_Apply(client);
        }
        else
        {
            WeaponBuffs_Remove(client);
        }
    }
}

static void WeaponBuffs_InitProjectileCalls()
{
    GameData config = LoadGameConfigFile("tf2_archipelago");
    if (config == null)
    {
        LogError("Native projectile-count backends disabled: tf2_archipelago gamedata is missing.");
        return;
    }

    StartPrepSDKCall(SDKCall_Static);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "CTFProjectile_Arrow::Create"))
    {
        PrepSDKCall_AddParameter(SDKType_Vector, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_QAngle, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_Float, SDKPass_Plain);
        PrepSDKCall_AddParameter(SDKType_Float, SDKPass_Plain);
        PrepSDKCall_AddParameter(SDKType_PlainOldData, SDKPass_Plain);
        PrepSDKCall_AddParameter(SDKType_CBaseEntity, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_CBaseEntity, SDKPass_Pointer);
        PrepSDKCall_SetReturnInfo(SDKType_CBaseEntity, SDKPass_Pointer);
        g_WeaponBuffArrowCreate = EndPrepSDKCall();
    }

    StartPrepSDKCall(SDKCall_Static);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "CTFProjectile_Flare::Create"))
    {
        PrepSDKCall_AddParameter(SDKType_CBaseEntity, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_Vector, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_QAngle, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_CBaseEntity, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_CBaseEntity, SDKPass_Pointer);
        PrepSDKCall_SetReturnInfo(SDKType_CBaseEntity, SDKPass_Pointer);
        g_WeaponBuffFlareCreate = EndPrepSDKCall();
    }

    StartPrepSDKCall(SDKCall_Entity);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "CTFBaseProjectile::GetDamage"))
    {
        PrepSDKCall_SetReturnInfo(SDKType_Float, SDKPass_Plain);
        g_WeaponBuffProjectileGetDamage = EndPrepSDKCall();
    }

    StartPrepSDKCall(SDKCall_Entity);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "CTFBaseProjectile::SetDamage"))
    {
        PrepSDKCall_AddParameter(SDKType_Float, SDKPass_Plain);
        g_WeaponBuffProjectileSetDamage = EndPrepSDKCall();
    }

    StartPrepSDKCall(SDKCall_Entity);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Virtual, "CBaseProjectile::SetLauncher"))
    {
        PrepSDKCall_AddParameter(SDKType_CBaseEntity, SDKPass_Pointer);
        g_WeaponBuffProjectileSetLauncher = EndPrepSDKCall();
    }

    StartPrepSDKCall(SDKCall_Entity);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "CTFFlareGun::AddFlare"))
    {
        PrepSDKCall_AddParameter(SDKType_CBaseEntity, SDKPass_Pointer);
        g_WeaponBuffFlareAdd = EndPrepSDKCall();
    }
    delete config;

    if (g_WeaponBuffFlareCreate == null || g_WeaponBuffProjectileGetDamage == null
        || g_WeaponBuffProjectileSetDamage == null)
    {
        LogError("Projectile-count flare fanout disabled: this TF2 build does not match the native flare gamedata.");
    }
    if (g_WeaponBuffArrowCreate == null || g_WeaponBuffProjectileSetLauncher == null)
    {
        LogError("Projectile-count arrow fanout disabled: this TF2 build does not match the native arrow gamedata.");
    }
}

void WeaponBuffs_HookClient(int client)
{
    g_WeaponBuffProviderRetryAt[client] = 0.0;
    g_DisposableSentryLastSwitchArgumentRef[client] = INVALID_ENT_REFERENCE;
    g_DisposableSentryLastSwitchActiveRef[client] = INVALID_ENT_REFERENCE;
    g_DisposableSentryCanBuildCalls[client] = 0;
    g_DisposableSentryCanBuildOverrides[client] = 0;
    g_DisposableSentryLastCanBuildResult[client] = -1;
    g_WeaponBuffLastButtons[client] = 0;
    g_WeaponBuffShotWeaponRef[client] = 0;
    g_WeaponBuffProjectileDestructionReadyAt[client] = 0.0;
    g_WeaponBuffLastSubstanceWeapon[client] = -1;
    g_WeaponBuffLastSubstanceTime[client] = 0.0;
    SDKUnhook(client, SDKHook_OnTakeDamage, WeaponBuffs_OnTakeDamage);
    SDKHook(client, SDKHook_OnTakeDamage, WeaponBuffs_OnTakeDamage);
    SDKUnhook(client, SDKHook_OnTakeDamagePost, WeaponBuffs_OnTakeDamagePost);
    SDKHook(client, SDKHook_OnTakeDamagePost, WeaponBuffs_OnTakeDamagePost);
    SDKUnhook(client, SDKHook_WeaponSwitchPost, WeaponBuffs_WeaponSwitchPost);
    SDKHook(client, SDKHook_WeaponSwitchPost, WeaponBuffs_WeaponSwitchPost);
    SDKUnhook(client, SDKHook_PostThinkPost, WeaponBuffs_PostThinkPost);
    SDKHook(client, SDKHook_PostThinkPost, WeaponBuffs_PostThinkPost);
}

static void WeaponBuffs_HookTank(int tank)
{
    SDKUnhook(tank, SDKHook_OnTakeDamagePost, WeaponBuffs_OnTankTakeDamagePost);
    SDKHook(tank, SDKHook_OnTakeDamagePost, WeaponBuffs_OnTankTakeDamagePost);
}

public void WeaponBuffs_WeaponSwitchPost(int client, int weapon)
{
    // WeaponSwitchPost supplies the new entity before m_hActiveWeapon is
    // guaranteed to expose it. Applying immediately can therefore leave the
    // PDA value active after the player has taken out the toolbox. The next
    // frame observes the settled entity, just like station transactions do.
    g_DisposableSentryLastSwitchArgumentRef[client] =
        weapon > MaxClients && IsValidEntity(weapon)
        ? EntIndexToEntRef(weapon) : INVALID_ENT_REFERENCE;
    int active = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
    g_DisposableSentryLastSwitchActiveRef[client] =
        active > MaxClients && IsValidEntity(active)
        ? EntIndexToEntRef(active) : INVALID_ENT_REFERENCE;
    WeaponBuffs_ApplyNextFrame(client);
}

static bool WeaponBuffs_IsProjectile(const char[] classname)
{
    return StrContains(classname, "tf_projectile_", false) == 0
        && !StrEqual(classname, "tf_projectile_sentryrocket", false);
}

static bool WeaponBuffs_IsSubstanceProjectile(const char[] classname)
{
    return StrContains(classname, "tf_projectile_jar", false) == 0;
}

public void OnEntityCreated(int entity, const char[] classname)
{
    if (entity <= MaxClients || entity >= sizeof(g_WeaponBuffNativeExtra))
    {
        return;
    }
    g_WeaponBuffNativeExtra[entity] = g_WeaponBuffFiringExtra;
    g_WeaponBuffNativeExtraBorn[entity] = GetGameTime();
    g_WeaponBuffArrowExtra[entity] = false;
    g_WeaponBuffArrowTouched[entity] = false;
    g_WeaponBuffProjectileDestructionHits[entity] = 0;
    if (StrEqual(classname, "tank_boss"))
    {
        WeaponBuffs_HookTank(entity);
    }
    if (WeaponBuffs_IsProjectile(classname) && g_WeaponBuffProjectileDebug != null
        && g_WeaponBuffProjectileDebug.BoolValue)
    {
        WeaponBuffs_DebugLog("[AP projectile] created entity=%d class=%s extra=%d",
            entity, classname, g_WeaponBuffFiringExtra);
    }
    if (g_WeaponBuffFiringExtra && WeaponBuffs_IsProjectile(classname))
    {
        g_WeaponBuffLastExtra = entity;
    }
    if (WeaponBuffs_IsProjectile(classname))
    {
        if (WeaponBuffs_IsSubstanceProjectile(classname))
        {
            SDKHook(entity, SDKHook_SpawnPost, WeaponBuffs_SubstanceProjectileSpawned);
        }
        if (!g_WeaponBuffFiringExtra)
        {
            SDKHook(entity, SDKHook_SpawnPost, WeaponBuffs_ProjectileSpawned);
        }
    }
}

public void WeaponBuffs_SubstanceProjectileSpawned(int entity)
{
    SDKUnhook(entity, SDKHook_SpawnPost, WeaponBuffs_SubstanceProjectileSpawned);
    if (entity <= MaxClients || !IsValidEntity(entity))
    {
        return;
    }

    // FireJar sets m_hLauncher after DispatchSpawn, but the thrower's active
    // weapon is still the jar while SpawnPost runs. Resolve through either
    // shape so even an immediately shattered jar has its reward pool recorded
    // before PlayerJarated is emitted.
    int owner = WeaponBuffs_EntityOwner(entity);
    if (!WeaponBuffs_CanUseRunBuffs(owner))
    {
        return;
    }
    int launcher = GetEntPropEnt(owner, Prop_Send, "m_hActiveWeapon");
    int weapon = WeaponBuffs_ForEntity(launcher);
    if (weapon < 0)
    {
        return;
    }
    g_WeaponBuffLastSubstanceWeapon[owner] = weapon;
    g_WeaponBuffLastSubstanceTime[owner] = GetGameTime();
    char classname[64];
    GetEntityClassname(entity, classname, sizeof(classname));
    if (StrEqual(classname, "tf_projectile_jar_gas", false))
    {
        g_WeaponBuffLastGasThrow[owner] = GetGameTime();
    }
}

// The game sends no PlayerJarated for gas and credits a gassed robot's burn
// death to whatever lit it, so the Gas Passer's kills are read off the
// condition: an enemy that turns gassed was gassed by whoever threw gas last.
public Action Timer_WeaponBuffGasWatch(Handle timer)
{
    float now = GetGameTime();
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client) || !IsPlayerAlive(client))
        {
            g_WeaponBuffGassed[client] = false;
            continue;
        }
        bool gassed = TF2_IsPlayerInCondition(client, TFCond_Gas);
        if (gassed && !g_WeaponBuffGassed[client])
        {
            int thrower = WeaponBuffs_RecentGasThrower(client, now);
            if (thrower > 0)
            {
                g_WeaponBuffGasFrom[client] = thrower;
                g_WeaponBuffGasTime[client] = now;
            }
        }
        g_WeaponBuffGassed[client] = gassed;
    }
    return Plugin_Continue;
}

static int WeaponBuffs_RecentGasThrower(int victim, float now)
{
    int thrower;
    float latest = now - 5.0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (client == victim || g_WeaponBuffLastGasThrow[client] <= latest
            || !IsClientInGame(client) || !WeaponBuffs_CanUseRunBuffs(client)
            || GetClientTeam(client) == GetClientTeam(victim))
        {
            continue;
        }
        thrower = client;
        latest = g_WeaponBuffLastGasThrow[client];
    }
    return thrower;
}

public Action WeaponBuffs_PlayerJarated(UserMsg msgId, BfRead message,
    const int[] players, int playersNum, bool reliable, bool init)
{
    int attacker = message.ReadByte();
    int victim = message.ReadByte();
    if (attacker < 1 || attacker > MaxClients || victim < 1 || victim > MaxClients
        || !IsClientInGame(attacker) || !IsClientInGame(victim)
        || !WeaponBuffs_CanUseRunBuffs(attacker)
        || GetClientTeam(attacker) == GetClientTeam(victim)
        || GetGameTime() - g_WeaponBuffLastSubstanceTime[attacker] > 60.0)
    {
        return Plugin_Continue;
    }
    int weapon = g_WeaponBuffLastSubstanceWeapon[attacker];
    if (weapon < 0 || weapon >= WeaponCount)
    {
        return Plugin_Continue;
    }

    // Defer condition changes until Valve has finished broadcasting its own
    // splash message and returned from JarExplode.
    DataPack pack = new DataPack();
    pack.WriteCell(GetClientUserId(attacker));
    pack.WriteCell(GetClientUserId(victim));
    pack.WriteCell(weapon);
    RequestFrame(WeaponBuffs_ApplySubstances, pack);
    return Plugin_Continue;
}

public void WeaponBuffs_ApplySubstances(any rawPack)
{
    DataPack pack = view_as<DataPack>(rawPack);
    pack.Reset();
    int attacker = GetClientOfUserId(pack.ReadCell());
    int victim = GetClientOfUserId(pack.ReadCell());
    int weapon = pack.ReadCell();
    delete pack;
    if (attacker < 1 || attacker > MaxClients || victim < 1 || victim > MaxClients
        || !IsClientInGame(attacker) || !IsClientInGame(victim) || !IsPlayerAlive(victim)
        || !WeaponBuffs_CanUseRunBuffs(attacker)
        || GetClientTeam(attacker) == GetClientTeam(victim)
        || weapon < 0 || weapon >= WeaponCount)
    {
        return;
    }

    WeaponBuffs_ApplyHitEffects(victim, attacker, weapon);
    WeaponBuffs_ApplyJarHitRewards(attacker, weapon);
}

// The game reads heal on hit and speed on hit off a weapon attack, and a jar
// splash is not one: with the attributes on, a soaked robot gave the thrower
// nothing. What every other weapon gets from the attribute, a jar gets here,
// once per soaked enemy.
static void WeaponBuffs_ApplyJarHitRewards(int attacker, int weapon)
{
    if (!IsPlayerAlive(attacker))
    {
        return;
    }
    int heal = g_WeaponEffectLevels[weapon][HealOnHitEffect];
    int speed = g_WeaponEffectLevels[weapon][SpeedOnHitEffect];
    int healed;
    if (heal > 0)
    {
        healed = WeaponBuffs_HealClient(attacker,
            RoundToFloor(g_WeaponEffectIncrements[HealOnHitEffect] * float(heal)));
    }
    if (speed > 0)
    {
        TF2_AddCondition(attacker, TFCond_SpeedBuffAlly, SpeedOnHitSeconds, attacker);
    }
    if (g_WeaponBuffProjectileDebug.BoolValue && (heal > 0 || speed > 0))
    {
        WeaponBuffs_DebugLog("[AP jar hit] weapon=%s attacker=%d healed=%d speed=%d",
            g_WeaponNames[weapon], attacker, healed, speed);
    }
}

// The on-kill attributes are read off the weapon that killed, through its
// owner, and the owner's provider carries the active weapon's buffs. A robot
// that burns down after the Pyro put the Gas Passer away was credited to the
// flamethrower. When the weapon that killed is not the one held, its own
// on-kill buffs are paid here; when it is, the game already paid them.
public void WeaponBuffs_PlayerDeath(Event event, const char[] name, bool dontBroadcast)
{
    int attacker = GetClientOfUserId(event.GetInt("attacker"));
    int victim = GetClientOfUserId(event.GetInt("userid"));
    if (attacker < 1 || victim < 1 || attacker == victim
        || !IsClientInGame(attacker) || !IsClientInGame(victim)
        || !IsPlayerAlive(attacker) || !WeaponBuffs_CanUseRunBuffs(attacker)
        || GetClientTeam(attacker) == GetClientTeam(victim))
    {
        return;
    }
    // A stomp carries no weapon, and the game reads no on-kill attribute for
    // it, so the boots or the pack in the secondary slot are paid here.
    char eventWeapon[64];
    char logWeapon[64];
    event.GetString("weapon", eventWeapon, sizeof(eventWeapon));
    event.GetString("weapon_logclassname", logWeapon, sizeof(logWeapon));
    bool mantreads = StrEqual(eventWeapon, "mantreads") || StrEqual(logWeapon, "mantreads");
    bool thermal = StrEqual(eventWeapon, "rocketpack_stomp") || StrEqual(logWeapon, "rocketpack_stomp");
    int weapon;
    if (mantreads || thermal)
    {
        int entity = WeaponBuffs_EntityInLoadoutSlot(attacker, Slot_Secondary);
        weapon = WeaponBuffs_ForEntity(entity);
        if (weapon >= 0 && ((mantreads && StrEqual(g_WeaponNames[weapon], "Mantreads"))
            || (thermal && StrEqual(g_WeaponNames[weapon], "Thermal Thruster"))))
        {
            WeaponBuffs_ApplyStompKillEffects(attacker, entity, weapon);
        }
        return;
    }
    int definition = event.GetInt("weapon_def_index");
    weapon = WeaponBuffs_ForDefinition(definition);
    int active = WeaponBuffs_ForEntity(GetEntPropEnt(attacker, Prop_Send, "m_hActiveWeapon"));
    int custom = event.GetInt("customkill");
    if (g_WeaponBuffProjectileDebug.BoolValue && weapon >= 0)
    {
        WeaponBuffs_DebugLog("[AP kill] weapon=%s definition=%d active=%d attacker=%d victim=%d custom=%d",
            g_WeaponNames[weapon], definition, active, attacker, victim, custom);
    }
    if (weapon >= 0 && active != weapon)
    {
        WeaponBuffs_ApplyKillRewards(attacker, weapon);
    }
    if (custom == TF_CUSTOM_BURNING)
    {
        WeaponBuffs_CreditGasBurn(attacker, victim, weapon);
    }
}

// A burn death of an enemy this attacker gassed is the Gas Passer's kill too,
// whichever weapon lit the gas. Its own on-kill buffs are paid here unless
// the game already credited the Gas Passer itself.
static void WeaponBuffs_CreditGasBurn(int attacker, int victim, int killedWith)
{
    int gasPasser = WeaponBuffs_ForDefinition(GasPasserDefinition);
    if (gasPasser < 0 || killedWith == gasPasser || g_WeaponBuffGasFrom[victim] != attacker
        || GetGameTime() - g_WeaponBuffGasTime[victim] > GasCreditSeconds)
    {
        return;
    }
    g_WeaponBuffGasFrom[victim] = 0;
    if (g_WeaponBuffProjectileDebug.BoolValue)
    {
        WeaponBuffs_DebugLog("[AP kill] gas burn attacker=%d victim=%d", attacker, victim);
    }
    WeaponBuffs_ApplyKillRewards(attacker, gasPasser);
}

static void WeaponBuffs_ApplyKillRewards(int attacker, int weapon)
{
    int heal = g_WeaponEffectLevels[weapon][HealOnKillEffect];
    int crits = g_WeaponEffectLevels[weapon][CritsOnKillEffect];
    int minicrits = g_WeaponEffectLevels[weapon][MinicritsOnKillEffect];
    int speed = g_WeaponEffectLevels[weapon][SpeedOnKillEffect];
    int healed;
    if (heal > 0)
    {
        healed = WeaponBuffs_HealClient(attacker,
            RoundToFloor(g_WeaponEffectIncrements[HealOnKillEffect] * float(heal)));
    }
    if (crits > 0)
    {
        TF2_AddCondition(attacker, TFCond_CritOnKill,
            g_WeaponEffectIncrements[CritsOnKillEffect] * float(crits) + CritsOnKillExtraSeconds, attacker);
    }
    if (minicrits > 0)
    {
        TF2_AddCondition(attacker, TFCond_MiniCritOnKill,
            g_WeaponEffectIncrements[MinicritsOnKillEffect] * float(minicrits), attacker);
    }
    if (speed > 0)
    {
        TF2_AddCondition(attacker, TFCond_SpeedBuffAlly, SpeedOnKillSeconds, attacker);
    }
    if (g_WeaponBuffProjectileDebug.BoolValue && (heal > 0 || crits > 0 || minicrits > 0 || speed > 0))
    {
        WeaponBuffs_DebugLog("[AP kill] weapon=%s attacker=%d healed=%d crits=%d minicrits=%d speed=%d",
            g_WeaponNames[weapon], attacker, healed, crits, minicrits, speed);
    }
}

// Item attributes implement these effects for only some of the weapons they
// can be awarded to. Apply the reward from the resolved catalogue entry on a
// confirmed enemy hit so bullets, projectiles and jars all use the same rules.
static bool WeaponBuffs_IsManualHitEffect(int effect)
{
    return effect == BleedEffect
        || effect == IgniteEffect
        || effect == MadMilkEffect
        || effect == GasolineEffect
        || effect == MarkForDeathEffect
        || effect == JarateEffect;
}

static void WeaponBuffs_ApplyHitEffects(int victim, int attacker, int weapon)
{
    int bleed = g_WeaponEffectLevels[weapon][BleedEffect];
    int ignite = g_WeaponEffectLevels[weapon][IgniteEffect];
    int milk = g_WeaponEffectLevels[weapon][MadMilkEffect];
    int gas = g_WeaponEffectLevels[weapon][GasolineEffect];
    int marked = g_WeaponEffectLevels[weapon][MarkForDeathEffect];
    int jarate = g_WeaponEffectLevels[weapon][JarateEffect];
    // One stack per enemy. TF2_MakeBleed adds a stack on every call, and a
    // syringe gun landed one per syringe on the same target.
    if (bleed > 0 && !TF2_IsPlayerInCondition(victim, TFCond_Bleeding))
    {
        TF2_MakeBleed(victim, attacker, 3.0 * float(bleed));
    }
    if (ignite > 0)
    {
        TF2_IgnitePlayer(victim, attacker);
    }
    if (milk > 0)
    {
        TF2_AddCondition(victim, TFCond_Milked, 10.0, attacker);
    }
    if (gas > 0)
    {
        TF2_AddCondition(victim, TFCond_Gas, 10.0, attacker);
    }
    if (marked > 0)
    {
        TF2_AddCondition(victim, TFCond_MarkedForDeath, 10.0, attacker);
    }
    if (jarate > 0)
    {
        TF2_AddCondition(victim, TFCond_Jarated, 10.0 + 2.0 * float(jarate), attacker);
    }
    if (g_WeaponBuffProjectileDebug.BoolValue
        && (bleed > 0 || ignite > 0 || milk > 0 || gas > 0 || marked > 0 || jarate > 0))
    {
        WeaponBuffs_DebugLog("[AP substance] weapon=%s attacker=%d victim=%d bleed=%d ignite=%d milk=%d gas=%d marked=%d jarate=%d",
            g_WeaponNames[weapon], attacker, victim, bleed, ignite, milk, gas, marked, jarate);
    }
}

// Energy weapons do not use a player ammo reserve. Return the amount of meter
// that one normal shot consumes so one ammo-on-hit level still means one shot.
// Direct flamethrower contact and afterburn both use TF_CUSTOM_BURNING. Valve
// identifies direct contact with the weapon as the inflictor; an afterburn tick
// uses the attacking player instead. Keep the former as a hit and reject the
// latter, along with all periodic bleed damage.
static bool WeaponBuffs_IsDamageOverTime(int attacker, int inflictor, int damagecustom)
{
    return damagecustom == TF_CUSTOM_BLEEDING
        || (damagecustom == TF_CUSTOM_BURNING && inflictor == attacker);
}

static float WeaponBuffs_EnergyShotCost(int entity)
{
    char classname[64];
    GetEntityClassname(entity, classname, sizeof(classname));
    if (StrEqual(classname, "tf_weapon_particle_cannon")
        || StrEqual(classname, "tf_weapon_raygun")
        || StrEqual(classname, "tf_weapon_drg_pomson"))
    {
        return 5.0;
    }
    return 0.0;
}

// TF2 expresses an energy weapon's capacity as a number of shots. The MvM
// station and our private provider both contribute through this hook, so the
// cap follows either source of clip-size upgrades instead of assuming 20.
static float WeaponBuffs_EnergyMaxCharge(int entity, float shotCost)
{
    int baseShots = RoundToFloor(EnergyWeaponMaxCharge / shotCost);
    float upgradedShots = TF2Attrib_HookValueFloat(float(baseShots),
        EnergyClipSizeAttributeClass, entity);
    return float(RoundToFloor(upgradedShots)) * shotCost;
}

// A shield bash is credited to the shield wearable. Like a melee hit, it
// supplies the Demoman's primary reserve rather than the attacking entity.
static bool WeaponBuffs_SuppliesPrimaryAmmo(int entity)
{
    if (WeaponBuffs_IsMelee(entity))
    {
        return true;
    }
    char classname[64];
    GetEntityClassname(entity, classname, sizeof(classname));
    if (StrEqual(classname, "tf_wearable_demoshield"))
    {
        return true;
    }
    int weapon = WeaponBuffs_ForEntity(entity);
    return weapon >= 0
        && StrEqual(g_WeaponNames[weapon], "Mantreads");
}

static float WeaponBuffs_RocketPackLaunchCost()
{
    ConVar cost = FindConVar("tf_rocketpack_cost");
    return cost != null && cost.FloatValue > 0.0 ? cost.FloatValue : 50.0;
}

// The engine always requires 50 meter points before it permits a launch.
// Treating the portion above that threshold as the spendable pool lets every
// clip-size level add exactly one usable launch to the stock two.
static float WeaponBuffs_ThermalThrusterEffectiveCost(int weapon)
{
    float stockCost = WeaponBuffs_RocketPackLaunchCost();
    int stockCharges = RoundToFloor(
        (ItemChargeMeterMaximum - RocketPackReadyCharge) / stockCost) + 1;
    int levels = g_WeaponEffectLevels[weapon][ClipSizeEffect];
    if (levels <= 0)
    {
        return stockCost;
    }
    return (ItemChargeMeterMaximum - RocketPackReadyCharge)
        / float(stockCharges + levels - 1);
}

// The Thermal Thruster has an ammo-looking field, but its usable launches live
// in the player's secondary item meter. Restore one effective launch per level,
// including when clip-size has increased the number of launches in the meter.
static void WeaponBuffs_AddThermalThrusterCharge(int attacker, int weapon, int amount)
{
    if (!HasEntProp(attacker, Prop_Send, "m_flItemChargeMeter"))
    {
        return;
    }
    float charge = GetEntPropFloat(attacker, Prop_Send,
        "m_flItemChargeMeter", Slot_Secondary)
        + WeaponBuffs_ThermalThrusterEffectiveCost(weapon) * float(amount);
    if (charge > ItemChargeMeterMaximum)
    {
        charge = ItemChargeMeterMaximum;
    }
    SetEntPropFloat(attacker, Prop_Send, "m_flItemChargeMeter",
        charge, Slot_Secondary);
}

static bool WeaponBuffs_IsStompWeapon(int weapon)
{
    return weapon >= 0
        && (StrEqual(g_WeaponNames[weapon], "Mantreads")
            || StrEqual(g_WeaponNames[weapon], "Thermal Thruster"));
}

static int WeaponBuffs_ClassBaseHealth(int client)
{
    switch (TF2_GetPlayerClass(client))
    {
        case TFClass_Scout, TFClass_Engineer, TFClass_Sniper, TFClass_Spy:
        {
            return 125;
        }
        case TFClass_Soldier:
        {
            return 200;
        }
        case TFClass_Pyro, TFClass_DemoMan:
        {
            return 175;
        }
        case TFClass_Heavy:
        {
            return 300;
        }
        case TFClass_Medic:
        {
            return 150;
        }
    }
    return 0;
}

// Stomp damage is credited to the player rather than to the equipped wearable,
// so TF2 never consults on-kill attributes attached to Mantreads or the Thermal
// Thruster. Reproduce their supported effects from the confirmed lethal stomp.
static void WeaponBuffs_ApplyStompKillEffects(int attacker, int entity, int weapon)
{
    int levels = g_WeaponEffectLevels[weapon][BaseHealthOnKillEffect];
    int baseHealth = WeaponBuffs_ClassBaseHealth(attacker);
    if (levels > 0 && baseHealth > 0)
    {
        int resource = GetPlayerResourceEntity();
        // m_iMaxHealth belongs to tf_player_manager, not the player entity.
        // Class base is a safe fallback during the brief map transitions where
        // the resource entity is unavailable.
        int maximum = baseHealth;
        if (resource > MaxClients && IsValidEntity(resource)
            && HasEntProp(resource, Prop_Send, "m_iMaxHealth"))
        {
            maximum = GetEntProp(resource, Prop_Send, "m_iMaxHealth", 4, attacker);
        }
        int health = GetClientHealth(attacker);
        int amount = RoundToCeil(float(baseHealth)
            * g_WeaponEffectIncrements[BaseHealthOnKillEffect]
            * float(levels) / 100.0);
        int overhealCap = RoundToFloor(float(maximum) * 1.5);
        int healed = amount < overhealCap - health ? amount : overhealCap - health;
        if (healed > 0)
        {
            SetEntityHealth(attacker, health + healed);
            Event event = CreateEvent("player_healonhit");
            if (event != null)
            {
                event.SetInt("amount", healed);
                event.SetInt("entindex", attacker);
                event.SetInt("weapon_def_index",
                    GetEntProp(entity, Prop_Send, "m_iItemDefinitionIndex"));
                event.Fire();
            }
        }
    }

    WeaponBuffs_ApplyKillRewards(attacker, weapon);
}

// TF2 always requires 50 meter points to begin a launch, even if the amount
// consumed afterward is changed. Refund part of the stock cost after each
// launch so 100 meter points contain exactly one additional usable launch per
// AP clip-size level: 2 stock, then 3, 4, and so on.
public void WeaponBuffs_RocketPackLaunch(Event event, const char[] name, bool dontBroadcast)
{
    int client = GetClientOfUserId(event.GetInt("userid"));
    if (!WeaponBuffs_CanUseRunBuffs(client) || !IsPlayerAlive(client)
        || !HasEntProp(client, Prop_Send, "m_flItemChargeMeter"))
    {
        return;
    }
    int entity = WeaponBuffs_EntityInLoadoutSlot(client, Slot_Secondary);
    int weapon = WeaponBuffs_ForEntity(entity);
    if (weapon < 0 || !StrEqual(g_WeaponNames[weapon], "Thermal Thruster"))
    {
        return;
    }
    int levels = g_WeaponEffectLevels[weapon][ClipSizeEffect];
    if (levels <= 0)
    {
        return;
    }

    float launchCost = WeaponBuffs_RocketPackLaunchCost();
    float refund = launchCost - WeaponBuffs_ThermalThrusterEffectiveCost(weapon);
    if (refund <= 0.0)
    {
        return;
    }

    float charge = GetEntPropFloat(client, Prop_Send,
        "m_flItemChargeMeter", Slot_Secondary) + refund;
    if (charge > ItemChargeMeterMaximum)
    {
        charge = ItemChargeMeterMaximum;
    }
    SetEntPropFloat(client, Prop_Send, "m_flItemChargeMeter",
        charge, Slot_Secondary);
}

// TF2's add_onhit_addammo attribute queues ammo on a hit, but only pays that
// queue when a clipless gun fires. Give the reserve ammo here so the effect has
// the same meaning for clipped and clipless weapons. A melee hit has no useful
// ammo type of its own, so it supplies the attacker's primary weapon instead.
// Shield bashes follow that same rule.
static void WeaponBuffs_AddAmmoOnHit(int attacker, int entity, int weapon)
{
    int amount = g_WeaponEffectLevels[weapon][AmmoOnHitEffect];
    if (amount <= 0 || entity <= MaxClients || !IsValidEntity(entity))
    {
        return;
    }

    if (StrEqual(g_WeaponNames[weapon], "Thermal Thruster"))
    {
        WeaponBuffs_AddThermalThrusterCharge(attacker, weapon, amount);
        return;
    }

    int ammoWeapon = entity;
    if (WeaponBuffs_SuppliesPrimaryAmmo(entity))
    {
        ammoWeapon = GetPlayerWeaponSlot(attacker, TFWeaponSlot_Primary);
    }
    if (ammoWeapon <= MaxClients || !IsValidEntity(ammoWeapon)
        || !HasEntProp(ammoWeapon, Prop_Send, "m_iPrimaryAmmoType"))
    {
        return;
    }

    float energyPerShot = WeaponBuffs_EnergyShotCost(ammoWeapon);
    if (energyPerShot > 0.0 && HasEntProp(ammoWeapon, Prop_Send, "m_flEnergy"))
    {
        float energy = GetEntPropFloat(ammoWeapon, Prop_Send, "m_flEnergy")
            + energyPerShot * float(amount);
        float maximum = WeaponBuffs_EnergyMaxCharge(ammoWeapon, energyPerShot);
        if (energy > maximum)
        {
            energy = maximum;
        }
        SetEntPropFloat(ammoWeapon, Prop_Send, "m_flEnergy", energy);
        return;
    }

    int ammoType = GetEntProp(ammoWeapon, Prop_Send, "m_iPrimaryAmmoType");
    if (ammoType >= 0)
    {
        GivePlayerAmmo(attacker, amount, ammoType, true);
    }
}

public void OnEntityDestroyed(int entity)
{
    if (entity > MaxClients && entity < sizeof(g_WeaponBuffNativeExtra))
    {
        int mimic = EntRefToEntIndex(g_WeaponBuffStickyMimic[entity]);
        g_WeaponBuffStickyMimic[entity] = 0;
        if (mimic > MaxClients && IsValidEntity(mimic))
        {
            RequestFrame(WeaponBuffs_RemoveEntityFrame, EntIndexToEntRef(mimic));
        }
        if (g_WeaponBuffNativeExtra[entity] && g_WeaponBuffProjectileDebug.BoolValue)
        {
            WeaponBuffs_DebugLog("[AP projectile] extra entity=%d removed after %.3f seconds",
                entity, GetGameTime() - g_WeaponBuffNativeExtraBorn[entity]);
        }
        g_WeaponBuffNativeExtra[entity] = false;
        g_WeaponBuffArrowExtra[entity] = false;
        g_WeaponBuffArrowTouched[entity] = false;
        g_WeaponBuffProjectileDestructionHits[entity] = 0;
    }
}

static bool WeaponBuffs_CanDestroyProjectiles(int weapon)
{
    static const char names[][] = {
        "Ambassador", "Baby Face's Blaster", "Back Scatter", "Bazaar Bargain", "Brass Beast",
        "Classic", "Cleaner's Carbine", "Diamondback", "Enforcer", "Family Business",
        "Force-a-Nature", "Frontier Justice", "Hitman's Heatmaker", "Huo-Long Heater",
        "L'Étranger", "Machina", "Minigun", "Natascha", "Panic Attack", "Pistol",
        "Pomson 6000", "Pretty Boy's Pocket Pistol", "Reserve Shooter", "Revolver",
        "Righteous Bison", "Scattergun", "Short Circuit", "Shortstop", "Shotgun", "SMG",
        "Sniper Rifle", "Soda Popper", "Sydney Sleeper", "Tomislav", "Widowmaker", "Winger"
    };
    if (weapon < 0)
    {
        return false;
    }
    for (int index = 0; index < sizeof(names); index++)
    {
        if (StrEqual(g_WeaponNames[weapon], names[index]))
        {
            return true;
        }
    }
    return false;
}

static bool WeaponBuffs_UsesNativeProjectileDestruction(int weapon)
{
    return weapon >= 0 && (StrEqual(g_WeaponNames[weapon], "Brass Beast")
        || StrEqual(g_WeaponNames[weapon], "Huo-Long Heater")
        || StrEqual(g_WeaponNames[weapon], "Minigun")
        || StrEqual(g_WeaponNames[weapon], "Natascha")
        || StrEqual(g_WeaponNames[weapon], "Tomislav"));
}

static bool WeaponBuffs_IsDestroyableProjectile(int entity)
{
    if (entity <= MaxClients || !IsValidEntity(entity))
    {
        return false;
    }
    char classname[64];
    GetEntityClassname(entity, classname, sizeof(classname));
    // The upgrade's weapon-side eligibility is deliberately narrow, but its
    // target should be every ordinary TF2 projectile: robot arrows, baseballs,
    // flares and energy shots are just as threatening as rockets and pipes.
    return StrContains(classname, "tf_projectile_", false) == 0;
}

static float WeaponBuffs_DistanceToShot(const float point[3], const float start[3],
    const float finish[3])
{
    float line[3], offset[3];
    SubtractVectors(finish, start, line);
    SubtractVectors(point, start, offset);
    float lengthSquared = GetVectorDotProduct(line, line);
    float fraction = lengthSquared > 0.0
        ? GetVectorDotProduct(offset, line) / lengthSquared : 0.0;
    if (fraction < 0.0) fraction = 0.0;
    if (fraction > 1.0) fraction = 1.0;
    float nearest[3];
    nearest[0] = start[0] + line[0] * fraction;
    nearest[1] = start[1] + line[1] * fraction;
    nearest[2] = start[2] + line[2] * fraction;
    return GetVectorDistance(point, nearest);
}

public bool WeaponBuffs_ProjectileVisibilityFilter(int entity, int contentsMask, any client)
{
    if (entity == client)
    {
        return false;
    }
    // The invisible attribute provider and the player's equipped entities sit
    // at their owner. They must not make a trace from that player's eye start
    // solid, but normal map props and other entities still block visibility.
    if (entity > MaxClients && IsValidEntity(entity)
        && HasEntProp(entity, Prop_Send, "m_hOwnerEntity")
        && GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity") == client)
    {
        return false;
    }
    return true;
}

static bool WeaponBuffs_ProjectileVisible(int client, const float start[3], const float point[3])
{
    // A projectile commonly flies close to a floor or wall. Tracing into its
    // exact origin can finish inside that brush and call a clear shot blocked.
    // Stop sixteen units in front of it, along the shooter's line of sight.
    float direction[3], finish[3];
    SubtractVectors(start, point, direction);
    NormalizeVector(direction, direction);
    ScaleVector(direction, 48.0);
    AddVectors(point, direction, finish);
    Handle trace = TR_TraceRayFilterEx(start, finish, MASK_SOLID, RayType_EndPoint,
        WeaponBuffs_ProjectileVisibilityFilter, client);
    float fraction = TR_GetFraction(trace);
    bool visible = !TR_DidHit(trace) || fraction >= 0.99;
    int blocker = visible ? -1 : TR_GetEntityIndex(trace);
    delete trace;
    if (!visible && g_WeaponBuffProjectileDebug.BoolValue)
    {
        WeaponBuffs_DebugLog("[AP destroy] candidate blocked at trace fraction %.3f by entity %d",
            fraction, blocker);
    }
    return visible;
}

// CTFBaseRocket::Destroy(false, true) sends this same message before removing
// a rocket. Clients with a breakable model render the projectile tumbling
// apart; projectile types without model gibs still get sparks and the impact
// sound below.
static void WeaponBuffs_ShowProjectileDud(int projectile, float point[3])
{
    int model = HasEntProp(projectile, Prop_Send, "m_nModelIndex")
        ? GetEntProp(projectile, Prop_Send, "m_nModelIndex") : 0;
    if (model <= 0)
    {
        return;
    }
    float angles[3];
    GetEntPropVector(projectile, Prop_Data, "m_angAbsRotation", angles);
    Handle message = StartMessageAll("BreakModelRocketDud", USERMSG_RELIABLE);
    if (message == null)
    {
        return;
    }
    BfWriteShort(message, model);
    BfWriteVecCoord(message, point);
    BfWriteAngles(message, angles);
    EndMessage();
}

// Valve's minigun scans a short cylinder in front of the player on every shot.
// Reproduce that geometry for the bullet and energy guns whose weapon classes
// never read attack_projectiles themselves.
static void WeaponBuffs_AttackEnemyProjectiles(int client, int levels)
{
    float now = GetGameTime();
    if (now < g_WeaponBuffProjectileDestructionReadyAt[client])
    {
        if (g_WeaponBuffProjectileDebug.BoolValue)
        {
            WeaponBuffs_DebugLog("[AP destroy] shot by %N ignored for %.3f second cooldown",
                client, g_WeaponBuffProjectileDestructionReadyAt[client] - now);
        }
        return;
    }
    float start[3], aim[3], direction[3], finish[3];
    GetClientEyePosition(client, start);
    GetClientEyeAngles(client, aim);
    GetAngleVectors(aim, direction, NULL_VECTOR, NULL_VECTOR);
    ScaleVector(direction, ProjectileDestructionRange);
    AddVectors(start, direction, finish);

    int candidates;
    int inRange;
    int blocked;
    int intercepted;
    float nearest = -1.0;
    int chosen = -1;
    float chosenDistance = -1.0;
    float chosenPoint[3];
    for (int projectile = MaxClients + 1; projectile < sizeof(g_WeaponBuffProjectileDestructionHits); projectile++)
    {
        if (!WeaponBuffs_IsDestroyableProjectile(projectile)
            || (GetEntityFlags(projectile) & FL_ONGROUND) != 0
            || (HasEntProp(projectile, Prop_Send, "m_iTeamNum")
                && GetEntProp(projectile, Prop_Send, "m_iTeamNum") == GetClientTeam(client)))
        {
            continue;
        }
        float point[3];
        GetEntPropVector(projectile, Prop_Data, "m_vecAbsOrigin", point);
        float distance = WeaponBuffs_DistanceToShot(point, start, finish);
        candidates++;
        if (nearest < 0.0 || distance < nearest)
        {
            nearest = distance;
        }
        if (distance > ProjectileDestructionRadius)
        {
            continue;
        }
        inRange++;
        if (!WeaponBuffs_ProjectileVisible(client, start, point))
        {
            blocked++;
            continue;
        }
        if (chosen < 0 || distance < chosenDistance)
        {
            chosen = projectile;
            chosenDistance = distance;
            chosenPoint[0] = point[0];
            chosenPoint[1] = point[1];
            chosenPoint[2] = point[2];
        }
    }
    if (chosen >= 0)
    {
        // Slow and burst-fire guns cannot practically land Valve's two minigun
        // hits on one projectile. One confirmed shot therefore intercepts one
        // projectile, then the upgrade level controls when another can occur.
        g_WeaponBuffProjectileDestructionHits[chosen]++;
        float sparkDirection[3];
        SubtractVectors(start, chosenPoint, sparkDirection);
        NormalizeVector(sparkDirection, sparkDirection);
        WeaponBuffs_ShowProjectileDud(chosen, chosenPoint);
        TE_SetupSparks(chosenPoint, sparkDirection, 2, 1);
        TE_SendToAll();
        EmitGameSoundToAll("Halloween.HeadlessBossAxeHitWorld", chosen);
        if (g_WeaponBuffProjectileDebug.BoolValue)
        {
            char classname[64];
            GetEntityClassname(chosen, classname, sizeof(classname));
            WeaponBuffs_DebugLog("[AP destroy] %N intercepted entity=%d class=%s distance=%.1f level=%d cooldown=%.2f",
                client, chosen, classname, chosenDistance, levels,
                WeaponBuffs_ProjectileDestructionCooldown(levels));
        }
        intercepted = 1;
        g_WeaponBuffProjectileDestructionReadyAt[client] = now
            + WeaponBuffs_ProjectileDestructionCooldown(levels);
        RemoveEntity(chosen);
    }
    if (g_WeaponBuffProjectileDebug.BoolValue)
    {
        WeaponBuffs_DebugLog("[AP destroy] %N swept range=%.0f radius=%.0f candidates=%d in-range=%d blocked=%d intercepted=%d nearest=%.1f",
            client, ProjectileDestructionRange, ProjectileDestructionRadius,
            candidates, inRange, blocked, intercepted, nearest);
    }
}

static int WeaponBuffs_PrimaryAmmo(int client, int entity)
{
    if (!HasEntProp(entity, Prop_Send, "m_iPrimaryAmmoType"))
    {
        return -1;
    }
    int ammoType = GetEntProp(entity, Prop_Send, "m_iPrimaryAmmoType");
    return ammoType >= 0 ? GetEntProp(client, Prop_Send, "m_iAmmo", 4, ammoType) : -1;
}

public void WeaponBuffs_PostThinkPost(int client)
{
    if (!IsPlayerAlive(client) || !WeaponBuffs_CanUseRunBuffs(client))
    {
        g_WeaponBuffShotWeaponRef[client] = 0;
        return;
    }
    int entity = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
    int catalog = WeaponBuffs_ForEntity(entity);
    if (catalog < 0)
    {
        g_WeaponBuffShotWeaponRef[client] = 0;
        return;
    }

    int reference = EntIndexToEntRef(entity);
    int clip = HasEntProp(entity, Prop_Send, "m_iClip1")
        ? GetEntProp(entity, Prop_Send, "m_iClip1") : -1;
    int ammo = WeaponBuffs_PrimaryAmmo(client, entity);
    float energy = HasEntProp(entity, Prop_Send, "m_flEnergy")
        ? GetEntPropFloat(entity, Prop_Send, "m_flEnergy") : -1.0;
    float nextAttack = HasEntProp(entity, Prop_Send, "m_flNextPrimaryAttack")
        ? GetEntPropFloat(entity, Prop_Send, "m_flNextPrimaryAttack") : 0.0;

    if (g_WeaponBuffShotWeaponRef[client] == reference)
    {
        bool spentAmmo = (clip >= 0 && clip < g_WeaponBuffShotClip[client])
            || (ammo >= 0 && ammo < g_WeaponBuffShotAmmo[client])
            || (energy >= 0.0 && energy + 0.001 < g_WeaponBuffShotEnergy[client]);
        bool hadAmmo = clip >= 0 ? g_WeaponBuffShotClip[client] > 0
            : (energy >= 0.0 ? g_WeaponBuffShotEnergy[client] > 0.0
            : g_WeaponBuffShotAmmo[client] > 0);
        bool fired = spentAmmo || (hadAmmo
            && nextAttack > g_WeaponBuffShotNextAttack[client] + 0.001);
        int levels = g_WeaponEffectLevels[catalog][DestroyProjectilesEffect];
        if (fired && levels > 0 && WeaponBuffs_CanDestroyProjectiles(catalog)
            && !WeaponBuffs_UsesNativeProjectileDestruction(catalog))
        {
            if (g_WeaponBuffProjectileDebug.BoolValue)
            {
                WeaponBuffs_DebugLog("[AP destroy] confirmed shot by %N weapon=%s level=%d clip=%d ammo=%d energy=%.1f next=%.3f",
                    client, g_WeaponNames[catalog], levels, clip, ammo, energy, nextAttack);
            }
            WeaponBuffs_AttackEnemyProjectiles(client, levels);
        }
    }
    g_WeaponBuffShotWeaponRef[client] = reference;
    g_WeaponBuffShotClip[client] = clip;
    g_WeaponBuffShotAmmo[client] = ammo;
    g_WeaponBuffShotEnergy[client] = energy;
    g_WeaponBuffShotNextAttack[client] = nextAttack;
}

static void WeaponBuffs_RemoveEntityFrame(any reference)
{
    int entity = EntRefToEntIndex(reference);
    if (entity > MaxClients && IsValidEntity(entity))
    {
        RemoveEntity(entity);
    }
}

public Action OnPlayerRunCmd(int client, int &buttons, int &impulse, float velocity[3],
    float angles[3], int &weapon)
{
    bool pressed = (buttons & IN_ATTACK2) != 0
        && (g_WeaponBuffLastButtons[client] & IN_ATTACK2) == 0;
    g_WeaponBuffLastButtons[client] = buttons;
    if (!pressed || !WeaponBuffs_CanUseRunBuffs(client))
    {
        return Plugin_Continue;
    }
    for (int projectile = MaxClients + 1; projectile < sizeof(g_WeaponBuffStickyMimic); projectile++)
    {
        int mimic = EntRefToEntIndex(g_WeaponBuffStickyMimic[projectile]);
        if (mimic <= MaxClients || !IsValidEntity(mimic))
        {
            continue;
        }
        int owner = IsValidEntity(projectile) ? WeaponBuffs_EntityOwner(projectile) : -1;
        if (owner == client)
        {
            AcceptEntityInput(mimic, "DetonateStickies");
        }
    }
    return Plugin_Continue;
}

public void WeaponBuffs_ProjectileSpawned(int entity)
{
    SDKUnhook(entity, SDKHook_SpawnPost, WeaponBuffs_ProjectileSpawned);
    if (g_WeaponBuffProjectileDebug.BoolValue && IsValidEntity(entity))
    {
        char classname[64];
        GetEntityClassname(entity, classname, sizeof(classname));
        WeaponBuffs_DebugLog("[AP projectile] spawn-post entity=%d class=%s", entity, classname);
    }
    RequestFrame(WeaponBuffs_FireExtraProjectiles, EntIndexToEntRef(entity));
}

static void WeaponBuffs_LogProjectile(const char[] label, int entity)
{
    if (!g_WeaponBuffProjectileDebug.BoolValue || entity <= MaxClients || !IsValidEntity(entity))
    {
        return;
    }
    int owner = HasEntProp(entity, Prop_Send, "m_hOwnerEntity")
        ? GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity") : -1;
    int launcher = HasEntProp(entity, Prop_Send, "m_hLauncher")
        ? GetEntPropEnt(entity, Prop_Send, "m_hLauncher") : -1;
    int originalLauncher = HasEntProp(entity, Prop_Send, "m_hOriginalLauncher")
        ? GetEntPropEnt(entity, Prop_Send, "m_hOriginalLauncher") : -1;
    int projectileType = HasEntProp(entity, Prop_Send, "m_iProjectileType")
        ? GetEntProp(entity, Prop_Send, "m_iProjectileType") : -1;
    int team = GetEntProp(entity, Prop_Send, "m_iTeamNum");
    int collision = GetEntProp(entity, Prop_Send, "m_CollisionGroup");
    float damage = HasEntProp(entity, Prop_Data, "m_flDamage")
        ? GetEntPropFloat(entity, Prop_Data, "m_flDamage") : -1.0;
    float origin[3], velocity[3];
    GetEntPropVector(entity, Prop_Data, "m_vecAbsOrigin", origin);
    GetEntPropVector(entity, Prop_Data, "m_vecAbsVelocity", velocity);
    WeaponBuffs_DebugLog("[AP projectile] %s entity=%d owner=%d launcher=%d original=%d type=%d team=%d collision=%d damage=%.1f origin=%.1f %.1f %.1f velocity=%.1f %.1f %.1f",
        label, entity, owner, launcher, originalLauncher, projectileType, team, collision,
        damage, origin[0], origin[1], origin[2], velocity[0], velocity[1], velocity[2]);
}

static int WeaponBuffs_MimicType(int source, const char[] classname)
{
    if (StrEqual(classname, "tf_projectile_rocket", false)) return 0;
    if (StrEqual(classname, "tf_projectile_pipe", false))
    {
        // Both names can represent the same CTFGrenadePipebombProjectile.
        // m_iType=1/2 is a sticky; zero is a timed grenade.
        if (HasEntProp(source, Prop_Send, "m_iType")
            && GetEntProp(source, Prop_Send, "m_iType") != 0)
        {
            return 3;
        }
        return 1;
    }
    if (StrEqual(classname, "tf_projectile_pipe_remote", false)) return 3;
    return -1;
}

static bool WeaponBuffs_IsArrow(const char[] classname)
{
    return StrEqual(classname, "tf_projectile_arrow", false)
        || StrEqual(classname, "tf_projectile_healing_bolt", false);
}

static void WeaponBuffs_CopySendInt(int source, int extra, const char[] property)
{
    if (HasEntProp(source, Prop_Send, property) && HasEntProp(extra, Prop_Send, property))
    {
        SetEntProp(extra, Prop_Send, property, GetEntProp(source, Prop_Send, property));
    }
}

static void WeaponBuffs_CopySendEntity(int source, int extra, const char[] property)
{
    if (HasEntProp(source, Prop_Send, property) && HasEntProp(extra, Prop_Send, property))
    {
        SetEntPropEnt(extra, Prop_Send, property, GetEntPropEnt(source, Prop_Send, property));
    }
}

static void WeaponBuffs_CopyDataFloat(int source, int extra, const char[] property)
{
    if (HasEntProp(source, Prop_Data, property) && HasEntProp(extra, Prop_Data, property))
    {
        SetEntPropFloat(extra, Prop_Data, property, GetEntPropFloat(source, Prop_Data, property));
    }
}

static int WeaponBuffs_CloneProjectile(int source, const char[] classname, int owner,
    int launcher, const float origin[3], const float angles[3], const float velocity[3])
{
    g_WeaponBuffFiringExtra = true;
    int extra = CreateEntityByName(classname);
    g_WeaponBuffFiringExtra = false;
    if (extra == -1)
    {
        LogError("Could not create %s for projectile-count.", classname);
        return -1;
    }

    // Projectile type selects the correct model and behavior for arrows,
    // healing bolts and the festive variants, and must be set before Spawn.
    WeaponBuffs_CopySendInt(source, extra, "m_iProjectileType");
    bool isFlare = StrEqual(classname, "tf_projectile_flare", false);
    if (isFlare && HasEntProp(extra, Prop_Send, "m_hOwnerEntity"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hOwnerEntity", -1);
    }
    if (!isFlare && HasEntProp(extra, Prop_Send, "m_hOwnerEntity"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hOwnerEntity", owner);
    }
    SetEntProp(extra, Prop_Send, "m_iTeamNum", GetClientTeam(owner));
    // Flare Spawn() reads both launcher handles to choose projectile speed,
    // heat-seeking behavior and other weapon-specific behavior.
    if (HasEntProp(extra, Prop_Send, "m_hLauncher"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hLauncher", launcher);
    }
    if (HasEntProp(extra, Prop_Send, "m_hOriginalLauncher"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hOriginalLauncher", launcher);
    }
    // CTFProjectile_Flare::Spawn traces from the owner's eyes to its current
    // origin and removes itself if that path crosses world geometry. Creating
    // at 0,0,0 and moving afterward therefore fizzles every cloned flare.
    TeleportEntity(extra, origin, angles, NULL_VECTOR);
    DispatchSpawn(extra);
    // Flare Spawn() performs a one-time line trace to its owner and fizzles
    // when that trace sees world geometry. A clone made one frame after the
    // shot can be beyond a corner even though the original spawned safely.
    // Assigning its owner immediately after Spawn preserves attribution while
    // avoiding a second, no-longer-relevant muzzle safety trace.
    if (isFlare && HasEntProp(extra, Prop_Send, "m_hOwnerEntity"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hOwnerEntity", owner);
    }

    WeaponBuffs_CopySendEntity(source, extra, "m_hLauncher");
    WeaponBuffs_CopySendEntity(source, extra, "m_hOriginalLauncher");
    WeaponBuffs_CopySendInt(source, extra, "m_bCritical");
    WeaponBuffs_CopySendInt(source, extra, "m_bArrowAlight");
    WeaponBuffs_CopySendInt(source, extra, "m_bChargedShot");
    WeaponBuffs_CopySendInt(source, extra, "m_CollisionGroup");
    WeaponBuffs_CopyDataFloat(source, extra, "m_flDamage");
    WeaponBuffs_CopyDataFloat(source, extra, "m_flDamageRadius");
    WeaponBuffs_CopyDataFloat(source, extra, "m_flDetonateTime");
    WeaponBuffs_CopyDataFloat(source, extra, "m_flCreationTime");
    WeaponBuffs_CopyDataFloat(source, extra, "m_flInitTime");
    if (HasEntProp(extra, Prop_Send, "m_hLauncher"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hLauncher", launcher);
    }
    if (HasEntProp(extra, Prop_Send, "m_hOriginalLauncher"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hOriginalLauncher", launcher);
    }
    if (HasEntProp(extra, Prop_Send, "m_vInitialVelocity"))
    {
        SetEntPropVector(extra, Prop_Send, "m_vInitialVelocity", velocity);
    }
    SetEntityMoveType(extra, GetEntityMoveType(source));
    SetEntityGravity(extra, GetEntityGravity(source));
    TeleportEntity(extra, origin, angles, velocity);
    return extra;
}

static int WeaponBuffs_FireMimicProjectile(int mimicType, int owner,
    int launcher, float speed, float damage, bool critical, const float origin[3],
    const float angles[3], const float velocity[3])
{
    int mimic = CreateEntityByName("tf_point_weapon_mimic");
    if (mimic == -1)
    {
        LogError("Could not create tf_point_weapon_mimic for projectile-count.");
        return -1;
    }
    char rawType[4];
    IntToString(mimicType, rawType, sizeof(rawType));
    DispatchKeyValue(mimic, "WeaponType", rawType);
    DispatchKeyValueFloat(mimic, "SpeedMin", speed);
    DispatchKeyValueFloat(mimic, "SpeedMax", speed);
    DispatchKeyValueFloat(mimic, "Damage", damage);
    DispatchKeyValueFloat(mimic, "SplashRadius", 146.0);
    DispatchKeyValue(mimic, "Crits", critical ? "1" : "0");
    DispatchSpawn(mimic);
    TeleportEntity(mimic, origin, angles, NULL_VECTOR);

    g_WeaponBuffLastExtra = -1;
    g_WeaponBuffFiringExtra = true;
    AcceptEntityInput(mimic, "FireOnce");
    g_WeaponBuffFiringExtra = false;
    int extra = g_WeaponBuffLastExtra;
    if (extra <= MaxClients || !IsValidEntity(extra))
    {
        RemoveEntity(mimic);
        return -1;
    }
    if (mimicType == 3)
    {
        // The mimic owns the private list required to detonate the sticky.
        // Keep it until the projectile dies and forward secondary fire to it.
        g_WeaponBuffStickyMimic[extra] = EntIndexToEntRef(mimic);
    }
    else
    {
        RemoveEntity(mimic);
    }
    SetEntPropEnt(extra, Prop_Send, "m_hOwnerEntity", owner);
    if (HasEntProp(extra, Prop_Send, "m_hLauncher"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hLauncher", launcher);
    }
    if (HasEntProp(extra, Prop_Send, "m_hOriginalLauncher"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hOriginalLauncher", launcher);
    }
    SetEntProp(extra, Prop_Send, "m_iTeamNum", GetClientTeam(owner));
    if (HasEntProp(extra, Prop_Send, "m_vInitialVelocity"))
    {
        SetEntPropVector(extra, Prop_Send, "m_vInitialVelocity", velocity);
    }
    TeleportEntity(extra, NULL_VECTOR, angles, velocity);
    return extra;
}

static int WeaponBuffs_FireNativeFlare(int owner, int launcher, float damage,
    bool critical, const float origin[3], const float angles[3], const float velocity[3])
{
    if (g_WeaponBuffFlareCreate == null || g_WeaponBuffProjectileSetDamage == null)
    {
        return -1;
    }

    g_WeaponBuffLastExtra = -1;
    g_WeaponBuffFiringExtra = true;
    int extra = SDKCall(g_WeaponBuffFlareCreate, launcher, origin, angles, owner, owner);
    g_WeaponBuffFiringExtra = false;
    if (extra <= MaxClients || !IsValidEntity(extra))
    {
        return -1;
    }

    SDKCall(g_WeaponBuffProjectileSetDamage, extra, damage);
    if (HasEntProp(extra, Prop_Send, "m_bCritical"))
    {
        SetEntProp(extra, Prop_Send, "m_bCritical", critical);
    }
    // The native factory computes the correct family speed. Replace only its
    // direction with our symmetric fanout while retaining the original shot's
    // attribute-adjusted speed and vertical component.
    TeleportEntity(extra, NULL_VECTOR, angles, velocity);

    // FireFlare normally does this after calling the factory. Without it the
    // Detonator's secondary attack cannot see the extra flares.
    int definition = HasEntProp(launcher, Prop_Send, "m_iItemDefinitionIndex")
        ? GetEntProp(launcher, Prop_Send, "m_iItemDefinitionIndex") : -1;
    if ((definition == 351 || definition == 1081) && g_WeaponBuffFlareAdd != null)
    {
        SDKCall(g_WeaponBuffFlareAdd, launcher, extra);
    }
    return extra;
}

static int WeaponBuffs_FireNativeArrow(int source, int owner, int launcher, float speed,
    float damage, bool critical, const float origin[3], const float angles[3],
    const float velocity[3])
{
    if (g_WeaponBuffArrowCreate == null || g_WeaponBuffProjectileSetDamage == null
        || g_WeaponBuffProjectileSetLauncher == null
        || !HasEntProp(source, Prop_Send, "m_iProjectileType"))
    {
        return -1;
    }

    int projectileType = GetEntProp(source, Prop_Send, "m_iProjectileType");
    float gravity = GetEntityGravity(source);
    g_WeaponBuffLastExtra = -1;
    g_WeaponBuffFiringExtra = true;
    int extra = SDKCall(g_WeaponBuffArrowCreate, origin, angles, speed, gravity,
        projectileType, owner, owner);
    g_WeaponBuffFiringExtra = false;
    if (extra <= MaxClients || !IsValidEntity(extra))
    {
        return -1;
    }

    // HealingBolt::GetDamage scales m_flDamage by flight time. Preserve the
    // unscaled backing value so the extra is not distance-scaled twice.
    float baseDamage = HasEntProp(source, Prop_Data, "m_flDamage")
        ? GetEntPropFloat(source, Prop_Data, "m_flDamage") : damage;
    SDKCall(g_WeaponBuffProjectileSetDamage, extra, baseDamage);

    // Calling SetLauncher matters: its native ResetCollideWithTeammates path
    // immediately enables teammate collision for healing bolts. Assigning the
    // launcher netprop alone skips that private state. Unlike FireArrow, this
    // setup call does not replay the weapon's firing sound for every extra.
    SDKCall(g_WeaponBuffProjectileSetLauncher, extra, launcher);
    WeaponBuffs_CopySendInt(source, extra, "m_bCritical");
    WeaponBuffs_CopySendInt(source, extra, "m_bArrowAlight");
    WeaponBuffs_CopySendInt(source, extra, "m_bChargedShot");
    WeaponBuffs_CopySendInt(source, extra, "m_CollisionGroup");
    WeaponBuffs_CopyArrowPenetration(source, extra);
    if (!HasEntProp(source, Prop_Send, "m_bCritical")
        && HasEntProp(extra, Prop_Send, "m_bCritical"))
    {
        SetEntProp(extra, Prop_Send, "m_bCritical", critical);
    }
    if (HasEntProp(extra, Prop_Send, "m_vInitialVelocity"))
    {
        SetEntPropVector(extra, Prop_Send, "m_vInitialVelocity", velocity);
    }
    TeleportEntity(extra, origin, angles, velocity);

    g_WeaponBuffArrowExtra[extra] = true;
    g_WeaponBuffArrowTouched[extra] = false;
    g_WeaponBuffArrowOrigin[extra] = origin;
    SDKHook(extra, SDKHook_StartTouch, WeaponBuffs_ArrowStartTouch);
    CreateTimer(0.15, WeaponBuffs_ArrowFlightCheck, EntIndexToEntRef(extra),
        TIMER_FLAG_NO_MAPCHANGE);
    return extra;
}

static void WeaponBuffs_CopyArrowPenetration(int source, int extra)
{
    int criticalOffset = GetEntSendPropOffs(source, "m_bCritical");
    if (criticalOffset < 0 || !HasEntProp(source, Prop_Data, "m_usSolidFlags")
        || !HasEntProp(extra, Prop_Data, "m_usSolidFlags"))
    {
        return;
    }

    int penetrateOffset = criticalOffset + 1;
    int penetrate = GetEntData(source, penetrateOffset, 1);
    if (penetrate != 0 && penetrate != 1)
    {
        LogError("Could not copy arrow penetration: unexpected value %d at offset %d.",
            penetrate, penetrateOffset);
        return;
    }
    SetEntData(extra, penetrateOffset, penetrate, 1, true);
    if (penetrate != 0)
    {
        SetEntProp(extra, Prop_Data, "m_usSolidFlags",
            GetEntProp(source, Prop_Data, "m_usSolidFlags"));
    }
}

public Action WeaponBuffs_ArrowStartTouch(int entity, int other)
{
    if (!g_WeaponBuffArrowExtra[entity] || g_WeaponBuffArrowTouched[entity])
    {
        return Plugin_Continue;
    }
    g_WeaponBuffArrowTouched[entity] = true;
    SDKUnhook(entity, SDKHook_StartTouch, WeaponBuffs_ArrowStartTouch);
    if (g_WeaponBuffProjectileDebug.BoolValue)
    {
        char classname[64] = "worldspawn";
        if (other > 0 && IsValidEntity(other))
        {
            GetEntityClassname(other, classname, sizeof(classname));
        }
        WeaponBuffs_DebugLog("[AP projectile] arrow-extra entity=%d first-touch=%d class=%s age=%.3f",
            entity, other, classname, GetGameTime() - g_WeaponBuffNativeExtraBorn[entity]);
    }
    return Plugin_Continue;
}

public Action WeaponBuffs_ArrowFlightCheck(Handle timer, any reference)
{
    int entity = EntRefToEntIndex(reference);
    if (entity <= MaxClients || !IsValidEntity(entity) || !g_WeaponBuffArrowExtra[entity]
        || !g_WeaponBuffProjectileDebug.BoolValue)
    {
        return Plugin_Stop;
    }
    float origin[3], velocity[3];
    GetEntPropVector(entity, Prop_Data, "m_vecAbsOrigin", origin);
    GetEntPropVector(entity, Prop_Data, "m_vecAbsVelocity", velocity);
    float distance = GetVectorDistance(origin, g_WeaponBuffArrowOrigin[entity]);
    WeaponBuffs_DebugLog("[AP projectile] arrow-extra entity=%d flight age=%.3f moved=%.1f speed=%.1f touched=%d",
        entity, GetGameTime() - g_WeaponBuffNativeExtraBorn[entity], distance,
        GetVectorLength(velocity), g_WeaponBuffArrowTouched[entity]);
    return Plugin_Stop;
}

static void WeaponBuffs_ScheduleNativeFlare(int owner, int launcher, float damage,
    bool critical, const float origin[3], const float angles[3], const float velocity[3],
    float delay)
{
    DataPack pack = new DataPack();
    pack.WriteCell(GetClientUserId(owner));
    pack.WriteCell(EntIndexToEntRef(launcher));
    pack.WriteFloat(damage);
    pack.WriteCell(critical);
    for (int axis = 0; axis < 3; axis++)
    {
        pack.WriteFloat(origin[axis]);
        pack.WriteFloat(angles[axis]);
        pack.WriteFloat(velocity[axis]);
    }
    CreateTimer(delay, WeaponBuffs_FireScheduledFlare, pack,
        TIMER_DATA_HNDL_CLOSE | TIMER_FLAG_NO_MAPCHANGE);
}

public Action WeaponBuffs_FireScheduledFlare(Handle timer, DataPack pack)
{
    pack.Reset();
    int owner = GetClientOfUserId(pack.ReadCell());
    int launcher = EntRefToEntIndex(pack.ReadCell());
    float damage = pack.ReadFloat();
    bool critical = pack.ReadCell() != 0;
    float origin[3], angles[3], velocity[3];
    for (int axis = 0; axis < 3; axis++)
    {
        origin[axis] = pack.ReadFloat();
        angles[axis] = pack.ReadFloat();
        velocity[axis] = pack.ReadFloat();
    }
    if (!WeaponBuffs_CanUseRunBuffs(owner) || launcher <= MaxClients || !IsValidEntity(launcher))
    {
        return Plugin_Stop;
    }
    int extra = WeaponBuffs_FireNativeFlare(owner, launcher, damage, critical,
        origin, angles, velocity);
    if (extra <= MaxClients || !IsValidEntity(extra))
    {
        if (g_WeaponBuffProjectileDebug.BoolValue)
        {
            WeaponBuffs_DebugLog("[AP projectile] scheduled native flare backend failed (entity %d)",
                extra);
        }
        return Plugin_Stop;
    }
    WeaponBuffs_LogProjectile("native-flare-extra", extra);
    return Plugin_Stop;
}

static void WeaponBuffs_FlareMuzzleOrigin(int owner, const float shotAngles[3],
    float origin[3])
{
    float ahead[3], right[3], up[3];
    GetClientEyePosition(owner, origin);
    GetAngleVectors(shotAngles, ahead, right, up);
    float vertical = (GetEntityFlags(owner) & FL_DUCKING) != 0 ? 8.0 : -3.0;
    for (int axis = 0; axis < 3; axis++)
    {
        // Mirrors CTFWeaponBaseGun::FireFlare's (23.5, 12, -3/8)
        // GetProjectileFireSetup offset. In particular, do not create extras
        // at the original projectile's one-frame-old position: coincident
        // flares touch each other and enter their 0.1-second ImpactThink.
        origin[axis] += ahead[axis] * 23.5
            + right[axis] * 12.0 + up[axis] * vertical;
    }
}

static int WeaponBuffs_EntityOwner(int entity)
{
    if (entity > 0 && entity <= MaxClients)
    {
        return entity;
    }
    if (entity > MaxClients && IsValidEntity(entity)
        && HasEntProp(entity, Prop_Send, "m_hOwnerEntity"))
    {
        return GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity");
    }
    return -1;
}

// TF2 projectile families do not agree about m_hOwnerEntity: depending on the
// weapon, it can be the player or the launcher. Normalize both shapes here.
static void WeaponBuffs_ResolveProjectile(int source, int &owner, int &launcher)
{
    owner = WeaponBuffs_EntityOwner(source);
    launcher = HasEntProp(source, Prop_Send, "m_hLauncher")
        ? GetEntPropEnt(source, Prop_Send, "m_hLauncher") : -1;
    if ((launcher <= MaxClients || !IsValidEntity(launcher))
        && HasEntProp(source, Prop_Send, "m_hOriginalLauncher"))
    {
        launcher = GetEntPropEnt(source, Prop_Send, "m_hOriginalLauncher");
    }

    // Flares and several grenade-derived projectiles expose the weapon as the
    // owner rather than through m_hLauncher.
    if (WeaponBuffs_ForEntity(owner) >= 0)
    {
        launcher = owner;
        owner = WeaponBuffs_EntityOwner(launcher);
    }
    if (!WeaponBuffs_CanUseRunBuffs(owner) && launcher > MaxClients && IsValidEntity(launcher))
    {
        int launcherOwner = WeaponBuffs_EntityOwner(launcher);
        if (WeaponBuffs_CanUseRunBuffs(launcherOwner))
        {
            owner = launcherOwner;
        }
    }
    if ((launcher <= MaxClients || !IsValidEntity(launcher))
        && WeaponBuffs_CanUseRunBuffs(owner))
    {
        launcher = GetEntPropEnt(owner, Prop_Send, "m_hActiveWeapon");
    }
}

static void WeaponBuffs_FireExtraProjectiles(any reference)
{
    int source = EntRefToEntIndex(reference);
    if (source <= MaxClients || !IsValidEntity(source))
    {
        return;
    }
    int owner, launcher;
    WeaponBuffs_ResolveProjectile(source, owner, launcher);
    if (!WeaponBuffs_CanUseRunBuffs(owner)
        || launcher <= MaxClients || !IsValidEntity(launcher))
    {
        if (g_WeaponBuffProjectileDebug.BoolValue)
        {
            char rejectedClass[64];
            GetEntityClassname(source, rejectedClass, sizeof(rejectedClass));
            WeaponBuffs_DebugLog("[AP projectile] rejected entity=%d class=%s owner=%d launcher=%d (unresolved owner/launcher)",
                source, rejectedClass, owner, launcher);
        }
        return;
    }
    int weapon = WeaponBuffs_ForEntity(launcher);
    if (weapon < 0)
    {
        if (g_WeaponBuffProjectileDebug.BoolValue)
        {
            char rejectedClass[64], launcherClass[64];
            GetEntityClassname(source, rejectedClass, sizeof(rejectedClass));
            GetEntityClassname(launcher, launcherClass, sizeof(launcherClass));
            WeaponBuffs_DebugLog("[AP projectile] rejected entity=%d class=%s owner=%d launcher=%d class=%s (launcher not catalogued)",
                source, rejectedClass, owner, launcher, launcherClass);
        }
        return;
    }
    int copies = g_WeaponEffectLevels[weapon][ProjectileCountEffect];
    if (copies <= 0)
    {
        if (g_WeaponBuffProjectileDebug.BoolValue)
        {
            char rejectedClass[64];
            GetEntityClassname(source, rejectedClass, sizeof(rejectedClass));
            WeaponBuffs_DebugLog("[AP projectile] ignored entity=%d class=%s owner=%d launcher=%d catalog=%d projectile-count=0",
                source, rejectedClass, owner, launcher, weapon);
        }
        return;
    }

    WeaponBuffs_LogProjectile("original", source);
    float origin[3], sourceVelocity[3];
    GetEntPropVector(source, Prop_Data, "m_vecAbsOrigin", origin);
    GetEntPropVector(source, Prop_Data, "m_vecAbsVelocity", sourceVelocity);
    float absSpeedSquared = sourceVelocity[0] * sourceVelocity[0]
        + sourceVelocity[1] * sourceVelocity[1]
        + sourceVelocity[2] * sourceVelocity[2];
    // Pipebombs, stickies and flares derive from the grenade projectile base.
    // During this deferred callback their physics velocity can read as zero,
    // while this networked launch vector still contains the real shot speed.
    if (absSpeedSquared < 1.0 && HasEntProp(source, Prop_Send, "m_vInitialVelocity"))
    {
        GetEntPropVector(source, Prop_Send, "m_vInitialVelocity", sourceVelocity);
    }
    float speed = SquareRoot(sourceVelocity[0] * sourceVelocity[0]
        + sourceVelocity[1] * sourceVelocity[1]
        + sourceVelocity[2] * sourceVelocity[2]);
    float damage;
    if (g_WeaponBuffProjectileGetDamage != null)
    {
        damage = view_as<float>(SDKCall(g_WeaponBuffProjectileGetDamage, source));
    }
    else
    {
        damage = HasEntProp(source, Prop_Data, "m_flDamage")
            ? GetEntPropFloat(source, Prop_Data, "m_flDamage") : 90.0;
    }
    bool critical = HasEntProp(source, Prop_Send, "m_bCritical")
        && GetEntProp(source, Prop_Send, "m_bCritical") != 0;
    char classname[64];
    GetEntityClassname(source, classname, sizeof(classname));
    int mimicType = WeaponBuffs_MimicType(source, classname);
    bool isFlare = StrEqual(classname, "tf_projectile_flare", false);
    bool isArrow = WeaponBuffs_IsArrow(classname);
    float flareOrigin[3];
    if (isFlare)
    {
        float sourceAngles[3];
        GetVectorAngles(sourceVelocity, sourceAngles);
        WeaponBuffs_FlareMuzzleOrigin(owner, sourceAngles, flareOrigin);
    }

    for (int copy = 0; copy < copies; copy++)
    {
        float velocity[3], angles[3];
        velocity = sourceVelocity;
        float degrees = 3.0 * float(copy / 2 + 1);
        if (copy % 2 != 0) degrees = -degrees;
        float radians = DegToRad(degrees);
        float x = velocity[0];
        velocity[0] = x * Cosine(radians) - velocity[1] * Sine(radians);
        velocity[1] = x * Sine(radians) + velocity[1] * Cosine(radians);
        GetVectorAngles(velocity, angles);

        int extra;
        if (isFlare)
        {
            // A handled Touch leaves coincident Source projectiles touching
            // forever. Give each flare at least one physics step to clear the
            // shared muzzle, and start it ahead along its own fanout ray. The
            // forward offset also clears the player's collision hull and the
            // original flare; increasing it per copy keeps scheduled extras
            // from occupying one another's path.
            float extraOrigin[3], direction[3];
            extraOrigin = flareOrigin;
            NormalizeVector(velocity, direction);
            float clearance = 64.0 + float(copy) * 12.0;
            for (int axis = 0; axis < 3; axis++)
            {
                extraOrigin[axis] += direction[axis] * clearance;
            }
            WeaponBuffs_ScheduleNativeFlare(owner, launcher, damage, critical,
                extraOrigin, angles, velocity, 0.02 * float(copy + 1));
            continue;
        }
        else if (isArrow)
        {
            // Native InitArrow records the real player scorer and excludes the
            // shooter from the arrow's private hit list. A small longitudinal
            // stagger keeps the fanout entities from beginning coincident.
            float extraOrigin[3], direction[3];
            extraOrigin = origin;
            NormalizeVector(velocity, direction);
            float clearance = 8.0 + float(copy) * 2.0;
            for (int axis = 0; axis < 3; axis++)
            {
                extraOrigin[axis] += direction[axis] * clearance;
            }
            extra = WeaponBuffs_FireNativeArrow(source, owner, launcher, speed,
                damage, critical, extraOrigin, angles, velocity);
        }
        else if (mimicType >= 0)
        {
            extra = WeaponBuffs_FireMimicProjectile(mimicType, owner, launcher,
                speed, damage, critical, origin, angles, velocity);
        }
        else
        {
            extra = WeaponBuffs_CloneProjectile(source, classname, owner, launcher,
                origin, angles, velocity);
        }
        if (extra <= MaxClients || !IsValidEntity(extra))
        {
            if (g_WeaponBuffProjectileDebug.BoolValue)
            {
                WeaponBuffs_DebugLog("[AP projectile] backend failed for %s (entity %d)", classname, extra);
            }
            continue;
        }
        WeaponBuffs_LogProjectile(isFlare ? "native-flare-extra"
            : (isArrow ? "native-arrow-extra"
                : (mimicType >= 0 ? "mimic-extra" : "clone-extra")), extra);
    }
}

// The weapon a hit came from: the launcher of the projectile that landed,
// else what the attacker is holding.
static int WeaponBuffs_WeaponOfHit(int attacker, int inflictor, int weapon, int damagecustom)
{
    // Mantreads and Thermal Thruster landings both use BOOTS_STOMP. The damage
    // info is allowed to carry no weapon, so recover the class's secondary and
    // accept it only when it is one of those two actual stomp sources.
    if (damagecustom == TF_CUSTOM_BOOTS_STOMP)
    {
        int stomp = WeaponBuffs_EntityInLoadoutSlot(attacker, Slot_Secondary);
        int catalog = WeaponBuffs_ForEntity(stomp);
        if (catalog >= 0
            && (StrEqual(g_WeaponNames[catalog], "Mantreads")
                || StrEqual(g_WeaponNames[catalog], "Thermal Thruster")))
        {
            return stomp;
        }
    }
    if ((weapon <= MaxClients || !IsValidEntity(weapon)) && inflictor > MaxClients
        && IsValidEntity(inflictor) && HasEntProp(inflictor, Prop_Send, "m_hLauncher"))
    {
        weapon = GetEntPropEnt(inflictor, Prop_Send, "m_hLauncher");
    }
    if ((weapon <= MaxClients || !IsValidEntity(weapon)) && IsPlayerAlive(attacker))
    {
        weapon = GetEntPropEnt(attacker, Prop_Send, "m_hActiveWeapon");
    }
    return weapon;
}

// Scale self-inflicted blast damage for rocket-jump protection. Do not turn
// no-self-blast into a zero-damage SDKHook event here: TF2 stops processing a
// zero-damage event before it applies the explosion's push. The launcher
// multiplier installed below lets TF2's damage rules zero health damage after
// the explosion and its force have already been established.
public Action WeaponBuffs_OnTakeDamage(int victim, int &attacker, int &inflictor,
    float &damage, int &damagetype, int &weapon, float damageForce[3],
    float damagePosition[3], int damagecustom)
{
    if (damagecustom == TF_CUSTOM_BOOTS_STOMP
        && victim >= 1 && victim <= MaxClients && attacker >= 1 && attacker <= MaxClients
        && victim != attacker && IsClientInGame(victim) && IsClientInGame(attacker)
        && GetClientTeam(victim) != GetClientTeam(attacker)
        && WeaponBuffs_CanUseRunBuffs(attacker))
    {
        int stomp = WeaponBuffs_WeaponOfHit(attacker, inflictor, weapon, damagecustom);
        int catalog = WeaponBuffs_ForEntity(stomp);
        int levels = WeaponBuffs_IsStompWeapon(catalog)
            ? g_WeaponEffectLevels[catalog][DamageEffect] : 0;
        if (levels > 0)
        {
            damage *= (StompBaseFallDamageMultiplier
                + StompFallDamagePerLevel * float(levels))
                / StompBaseFallDamageMultiplier;
            return Plugin_Changed;
        }
    }
    if (victim != attacker || victim < 1 || victim > MaxClients || !(damagetype & DMG_BLAST)
        || !IsClientInGame(victim) || !WeaponBuffs_CanUseRunBuffs(victim))
    {
        return Plugin_Continue;
    }
    weapon = WeaponBuffs_WeaponOfHit(attacker, inflictor, weapon, damagecustom);
    int catalog = WeaponBuffs_ForEntity(weapon);
    if (catalog < 0)
    {
        return Plugin_Continue;
    }
    int levels = g_WeaponEffectLevels[catalog][RocketJumpProtectionEffect];
    if (levels <= 0)
    {
        return Plugin_Continue;
    }
    float kept = 1.0 - RocketJumpProtectionPerLevel * float(levels);
    if (kept < 0.0)
    {
        kept = 0.0;
    }
    damage *= kept;
    return Plugin_Changed;
}

public void WeaponBuffs_OnTakeDamagePost(int victim, int attacker, int inflictor,
    float damage, int damagetype, int weapon, const float damageForce[3],
    const float damagePosition[3], int damagecustom)
{
    if (inflictor > MaxClients && inflictor < sizeof(g_WeaponBuffArrowExtra)
        && IsValidEntity(inflictor) && g_WeaponBuffArrowExtra[inflictor]
        && g_WeaponBuffProjectileDebug.BoolValue)
    {
        WeaponBuffs_DebugLog("[AP projectile] arrow-extra entity=%d damaged victim=%d attacker=%d weapon=%d amount=%.1f custom=%d",
            inflictor, victim, attacker, weapon, damage, damagecustom);
    }
    if (victim < 1 || victim > MaxClients || attacker < 1 || attacker > MaxClients
        || victim == attacker || !IsClientInGame(victim) || !IsClientInGame(attacker)
        || !IsPlayerAlive(victim) || !WeaponBuffs_CanUseRunBuffs(attacker)
        || GetClientTeam(victim) == GetClientTeam(attacker)
        || WeaponBuffs_IsDamageOverTime(attacker, inflictor, damagecustom))
    {
        return;
    }
    // PlayerJarated carries every victim in a jar splash through the deferred
    // path above. A jar that also reports impact damage must not apply twice.
    if (inflictor > MaxClients && IsValidEntity(inflictor))
    {
        char inflictorClass[64];
        GetEntityClassname(inflictor, inflictorClass, sizeof(inflictorClass));
        if (WeaponBuffs_IsSubstanceProjectile(inflictorClass))
        {
            return;
        }
    }
    weapon = WeaponBuffs_WeaponOfHit(attacker, inflictor, weapon, damagecustom);
    int catalog = WeaponBuffs_ForEntity(weapon);
    if (catalog >= 0)
    {
        WeaponBuffs_ApplyHitEffects(victim, attacker, catalog);
        if (damage > 0.0)
        {
            WeaponBuffs_AddAmmoOnHit(attacker, weapon, catalog);
        }
    }
}

public void WeaponBuffs_OnTankTakeDamagePost(int tank, int attacker, int inflictor,
    float damage, int damagetype, int weapon, const float damageForce[3],
    const float damagePosition[3], int damagecustom)
{
    if (damage <= 0.0 || attacker < 1 || attacker > MaxClients
        || !IsClientInGame(attacker) || !WeaponBuffs_CanUseRunBuffs(attacker)
        || GetClientTeam(attacker) == GetEntProp(tank, Prop_Send, "m_iTeamNum")
        || WeaponBuffs_IsDamageOverTime(attacker, inflictor, damagecustom))
    {
        return;
    }

    weapon = WeaponBuffs_WeaponOfHit(attacker, inflictor, weapon, damagecustom);
    int catalog = WeaponBuffs_ForEntity(weapon);
    if (catalog >= 0)
    {
        WeaponBuffs_AddAmmoOnHit(attacker, weapon, catalog);
    }
}

public Action Command_ProjectileDebug(int client, int args)
{
    if (args >= 1)
    {
        char wanted[16];
        GetCmdArg(1, wanted, sizeof(wanted));
        g_WeaponBuffProjectileDebug.BoolValue = StrEqual(wanted, "on", false)
            || StrEqual(wanted, "1");
    }
    ReplyToCommand(client, "[AP] Projectile debug %s; backend=mimic, native flare/arrow, plus family clone.",
        g_WeaponBuffProjectileDebug.BoolValue ? "ON" : "OFF");
    ReplyToCommand(client, "[AP] Native calls: arrow=%d flare=%d get-damage=%d set-damage=%d add-detonator=%d.",
        g_WeaponBuffArrowCreate != null && g_WeaponBuffProjectileSetLauncher != null,
        g_WeaponBuffFlareCreate != null, g_WeaponBuffProjectileGetDamage != null,
        g_WeaponBuffProjectileSetDamage != null, g_WeaponBuffFlareAdd != null);
    if (args == 0 && g_WeaponBuffProjectileLineHeld > 0)
    {
        int first = (g_WeaponBuffProjectileLineNext - g_WeaponBuffProjectileLineHeld
            + ProjectileDebugLineCount) % ProjectileDebugLineCount;
        for (int offset = 0; offset < g_WeaponBuffProjectileLineHeld; offset++)
        {
            int line = (first + offset) % ProjectileDebugLineCount;
            ReplyToCommand(client, "%s", g_WeaponBuffProjectileLines[line]);
        }
    }
    if (MvM_IsPlayer(client) && IsPlayerAlive(client))
    {
        int entity = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
        int weapon = WeaponBuffs_ForEntity(entity);
        char classname[64];
        GetEntityClassname(entity, classname, sizeof(classname));
        int levels = weapon >= 0 ? g_WeaponEffectLevels[weapon][ProjectileCountEffect] : 0;
        ReplyToCommand(client, "[AP] Active entity=%d class=%s catalog=%d projectile-count=%d.",
            entity, classname, weapon, levels);
    }
    return Plugin_Handled;
}

void WeaponBuffs_Clear()
{
    // A bridge resync replaces the canonical levels. Remove the contribution
    // represented by the old levels before the new unlock set is installed.
    for (int client = 1; client <= MaxClients; client++)
    {
        WeaponBuffs_Remove(client);
    }
    for (int weapon = 0; weapon < WeaponCount; weapon++)
    {
        for (int effect = 0; effect < WeaponEffectCount; effect++)
        {
            g_WeaponEffectLevels[weapon][effect] = 0;
        }
    }
}

bool WeaponBuffs_Grant(const char[] key)
{
    for (int index = 0; index < sizeof(g_WeaponBuffKeys); index++)
    {
        if (StrEqual(g_WeaponBuffKeys[index], key))
        {
            int weapon = g_WeaponBuffWeapon[index];
            int effect = g_WeaponBuffEffect[index];
            if (g_WeaponEffectModes[effect] == BuffMode_Toggle)
            {
                g_WeaponEffectLevels[weapon][effect] = 1;
            }
            else
            {
                g_WeaponEffectLevels[weapon][effect]++;
            }
            return true;
        }
    }
    LogError("The bridge granted weapon buff %s. This plugin does not know it.", key);
    return false;
}

static int WeaponBuffs_ForEntity(int entity)
{
    if (entity <= MaxClients || !IsValidEntity(entity)
        || !HasEntProp(entity, Prop_Send, "m_iItemDefinitionIndex"))
    {
        return -1;
    }
    return WeaponBuffs_ForDefinition(GetEntProp(entity, Prop_Send, "m_iItemDefinitionIndex"));
}

// GetPlayerWeaponSlot cannot see items implemented as wearables: shields,
// boots, backpacks, the Razorback, and Darwin's Danger Shield. Resolve those
// through the same class-aware loadout mapping used by slot enforcement.
static int WeaponBuffs_EntityInLoadoutSlot(int client, int slot)
{
    int entity = GetPlayerWeaponSlot(client, slot);
    if (entity > MaxClients && IsValidEntity(entity))
    {
        return entity;
    }

    TFClassType class = TF2_GetPlayerClass(client);
    entity = -1;
    while ((entity = FindEntityByClassname(entity, "tf_wearable*")) != -1)
    {
        if (GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity") != client
            || !HasEntProp(entity, Prop_Send, "m_iItemDefinitionIndex"))
        {
            continue;
        }
        int definition = GetEntProp(entity, Prop_Send, "m_iItemDefinitionIndex");
        if (Unlocks_WearableSlot(class, definition) == slot)
        {
            return entity;
        }
    }
    return -1;
}

static int WeaponBuffs_Provider(int client, int weaponEntity)
{
    int provider = EntRefToEntIndex(g_WeaponBuffProviderRef[client]);
    if (provider > MaxClients && IsValidEntity(provider)
        && HasEntProp(provider, Prop_Send, "m_hOwnerEntity")
        && GetEntPropEnt(provider, Prop_Send, "m_hOwnerEntity") == client)
    {
        return provider;
    }
    if (provider > MaxClients && IsValidEntity(provider))
    {
        RemoveEntity(provider);
    }
    g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
    if (GetGameTime() < g_WeaponBuffProviderRetryAt[client])
    {
        return -1;
    }

    provider = CreateEntityByName("tf_wearable");
    if (provider <= MaxClients || !IsValidEntity(provider))
    {
        LogError("Could not create the private Archipelago attribute provider for %N.", client);
        g_WeaponBuffProviderRetryAt[client] = GetGameTime() + WeaponBuffProviderRetrySeconds;
        return -1;
    }
    // A real all-class cosmetic gives the economy entity a valid item view;
    // RENDER_NONE keeps this implementation detail out of the world. Runtime
    // attributes live only on this cosmetic, never on the player's weapon or
    // player ledgers that the MvM station owns.
    SetEntProp(provider, Prop_Send, "m_iItemDefinitionIndex", 940);
    SetEntProp(provider, Prop_Send, "m_bInitialized", 1);
    SetEntProp(provider, Prop_Send, "m_iEntityQuality", 6);
    SetEntProp(provider, Prop_Send, "m_iEntityLevel", 1);
    SetEntProp(provider, Prop_Send, "m_iTeamNum", GetClientTeam(client));
    SetEntityRenderMode(provider, RENDER_NONE);
    DispatchKeyValue(provider, "targetname", WeaponBuffProviderTargetName);
    DispatchSpawn(provider);
    if (!HasEntProp(provider, Prop_Send, "m_ProviderType"))
    {
        LogError("The private Archipelago attribute provider has no m_ProviderType.");
        RemoveEntity(provider);
        g_WeaponBuffProviderRetryAt[client] = GetGameTime() + WeaponBuffProviderRetrySeconds;
        return -1;
    }
    // CEconEntity initializes as PROVIDER_WEAPON. TF2 deliberately refuses
    // to apply one weapon-type provider while a different weapon is the hook
    // initiator, which made every AP weapon effect a no-op. This entity is an
    // out-of-band modifier, so it must be PROVIDER_GENERIC (zero).
    SetEntProp(provider, Prop_Send, "m_ProviderType", 0);
    g_WeaponBuffProviderRef[client] = EntIndexToEntRef(provider);
    SetEntPropEnt(provider, Prop_Send, "m_hOwnerEntity", client);
    // CEconEntity::ReapplyProvision follows m_hOwnerEntity and asks its
    // attribute manager to ProvideTo that owner. This deliberately does not
    // add the entity to CTFPlayer::m_hMyWearables: TF2Utils' wearable-equip
    // virtual offset is stale on current Windows TF2, and no visible/equipped
    // cosmetic is needed for an attribute provider.
    /* The whole provider rests on this input being accepted. A server with no
     * VScript VM refuses it, ReapplyProvision never runs, and every weapon buff
     * in the run silently does nothing. That was reported as the provider not
     * providing, which is the symptom two steps downstream: say which it was,
     * because the fix for one is a server setting and the fix for the other is
     * here. */
    SetVariantString("self.ReapplyProvision()");
    if (!AcceptEntityInput(provider, "RunScriptCode"))
    {
        LogError("This server refused RunScriptCode, so no weapon buff can be applied to %N. The provider is attached through VScript and this server has none.", client);
        RemoveEntity(provider);
        g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
        g_WeaponBuffProviderRetryAt[client] = GetGameTime() + WeaponBuffProviderRetrySeconds;
        return -1;
    }
    if (GetEntPropEnt(provider, Prop_Send, "m_hOwnerEntity") != client)
    {
        LogError("Could not attach the private Archipelago attribute provider to %N.", client);
        RemoveEntity(provider);
        g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
        g_WeaponBuffProviderRetryAt[client] = GetGameTime() + WeaponBuffProviderRetrySeconds;
        return -1;
    }

    // Prove the provider is in the active weapon's aggregate chain. The probe
    // exists only inside this callback and is removed before real AP effects
    // are installed; failure is explicit instead of another silent no-op.
    TF2Attrib_ClearCache(weaponEntity);
    float damageBefore = TF2Attrib_HookValueFloat(1.0, "mult_dmg", weaponEntity);
    TF2Attrib_SetByName(provider, "damage bonus", 1.01);
    TF2Attrib_ClearCache(weaponEntity);
    float damageAfter = TF2Attrib_HookValueFloat(1.0, "mult_dmg", weaponEntity);
    TF2Attrib_RemoveByName(provider, "damage bonus");
    TF2Attrib_ClearCache(weaponEntity);
    if (FloatAbs(damageAfter - damageBefore) < 0.001)
    {
        /* What was in hand, and both readings. The line used to say only that
         * the probe failed, which left apw-l3v with a player whose buffs never
         * applied and nothing in the bundle to say what was different about the
         * moment it was tried. */
        char held[64];
        if (!GetEntityClassname(weaponEntity, held, sizeof(held)))
        {
            strcopy(held, sizeof(held), "an unnamed entity");
        }
        int definition = -1;
        if (HasEntProp(weaponEntity, Prop_Send, "m_iItemDefinitionIndex"))
        {
            definition = GetEntProp(weaponEntity, Prop_Send, "m_iItemDefinitionIndex");
        }
        LogError("The private Archipelago provider attached to %N but did not provide attributes, so no weapon buff can apply. Held %s (item %d), which read %.4f before the probe and %.4f after.",
            client, held, definition, damageBefore, damageAfter);
        RemoveEntity(provider);
        g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
        g_WeaponBuffProviderRetryAt[client] = GetGameTime() + WeaponBuffProviderRetrySeconds;
        return -1;
    }
    g_WeaponBuffProviderRetryAt[client] = 0.0;
    return provider;
}

// Attribute 181 (no self blast dmg) changes rockets and grenades to the
// Jumper explosion particle as well as suppressing damage. Attribute 207 is
// the independent self-blast damage multiplier: zeroing it in TF2's native
// damage rules retains the ordinary explosion, blast-jump force and
// conditions. Keep it on every loadout weapon so a projectile still has the
// attribute after its owner switches.
static void WeaponBuffs_SyncNativeSelfBlast(int client, bool removeAll)
{
    for (int slot = 0; slot <= 5; slot++)
    {
        int entity = GetPlayerWeaponSlot(client, slot);
        if (entity <= MaxClients || !IsValidEntity(entity))
        {
            continue;
        }
        TF2Attrib_RemoveByName(entity, SelfBlastDamageAttribute);
        int weapon = WeaponBuffs_ForEntity(entity);
        if (!removeAll && weapon >= 0
            && g_WeaponEffectLevels[weapon][NoSelfBlastEffect] > 0)
        {
            if (!TF2Attrib_SetByName(entity, SelfBlastDamageAttribute, 0.0))
            {
                LogError("Could not apply native no-self-blast effect to %s.",
                    g_WeaponNames[weapon]);
            }
        }
        TF2Attrib_ClearCache(entity);
    }
}

static void WeaponBuffs_Remove(int client)
{
    int provider = EntRefToEntIndex(g_WeaponBuffProviderRef[client]);
    if (provider > MaxClients && IsValidEntity(provider))
    {
        TF2Attrib_RemoveAll(provider);
    }
    if (client > 0 && client <= MaxClients && IsClientInGame(client))
    {
        WeaponBuffs_SyncNativeSelfBlast(client, true);
        int entity = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
        if (entity > MaxClients && IsValidEntity(entity))
        {
            TF2Attrib_ClearCache(entity);
        }
    }
}

void WeaponBuffs_Disconnect(int client)
{
    if (client > 0 && client <= MaxClients && IsClientInGame(client))
    {
        WeaponBuffs_SyncNativeSelfBlast(client, true);
    }
    int provider = EntRefToEntIndex(g_WeaponBuffProviderRef[client]);
    g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
    g_WeaponBuffProviderRetryAt[client] = 0.0;
    g_WeaponBuffAtUpgradeStation[client] = false;
    if (provider > MaxClients && IsValidEntity(provider))
    {
        RemoveEntity(provider);
    }
}

// A weapon in its owner's melee slot. An entity with no owner is not capped.
static bool WeaponBuffs_IsMelee(int entity)
{
    int owner = GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity");
    if (owner <= 0 || owner > MaxClients || !IsClientInGame(owner))
    {
        return false;
    }
    return GetPlayerWeaponSlot(owner, TFWeaponSlot_Melee) == entity;
}

static void Frame_ApplyWeaponBuffs(any userid)
{
    int client = GetClientOfUserId(userid);
    if (client > 0)
    {
        WeaponBuffs_Apply(client);
    }
}

// Let the station command finish rebuilding Valve's own values before
// recalculating the independent provider's additive percentage ratios.
void WeaponBuffs_ApplyNextFrame(int client)
{
    if (client > 0 && IsClientInGame(client))
    {
        RequestFrame(Frame_ApplyWeaponBuffs, GetClientUserId(client));
    }
}

void WeaponBuffs_Apply(int client)
{
    if (!g_HaveUnlocks || !WeaponBuffs_CanUseRunBuffs(client) || !IsPlayerAlive(client))
    {
        return;
    }
    int entity = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
    int weapon = WeaponBuffs_ForEntity(entity);
    WeaponBuffs_SyncNativeSelfBlast(client, false);
    // Reloads, respawns and special movement weapons can briefly leave the
    // player with no valid active weapon. A provider cannot be verified
    // against that state, and repeatedly creating/removing one feeds another
    // WeaponSwitchPost event back into this function.
    if (weapon < 0)
    {
        return;
    }
    int provider = WeaponBuffs_Provider(client, entity);
    if (provider <= MaxClients || !IsValidEntity(provider))
    {
        return;
    }

    // Rebuild from an empty private list. The aggregate queried below is now
    // purely Valve/static/MvM state, so refunds cannot consume AP levels and
    // repeated purchases cannot fold AP values back into the station ledger.
    TF2Attrib_RemoveAll(provider);
    // The active weapon caches its final hook result, including attributes
    // reached through its owner. Provider changes clear the provider/player
    // caches but do not walk back into the weapon, so both sides of this
    // rebuild explicitly invalidate the weapon cache.
    TF2Attrib_ClearCache(entity);
    for (int effect = 0; effect < WeaponEffectCount; effect++)
    {
        // These follow the player, not the weapon in his hands, so the pass
        // after this loop reads them off every slot of the loadout instead.
        // Health regeneration is the timer's besides, so it can heal to the
        // player's real, modified maximum rather than the schema attribute's
        // negative-damage heal, which emits a popup and changes no health.
        if (WeaponBuffs_IsPassiveEffect(effect))
        {
            continue;
        }
        // The shared confirmed-enemy-hit path applies these. Installing their
        // schema attributes on the player-owned provider also lets TF2 apply
        // them to self-inflicted blast damage (a Rocket Jumper can bleed and
        // ignite its owner), before our victim/attacker guards ever run.
        if (WeaponBuffs_IsManualHitEffect(effect))
        {
            continue;
        }
        // Only CTFMinigun reads attack_projectiles. The confirmed-shot sweep
        // above implements it for every other eligible bullet/energy gun.
        if (effect == DestroyProjectilesEffect
            && !WeaponBuffs_UsesNativeProjectileDestruction(weapon))
        {
            continue;
        }
        // TF2 only pays this attribute's queued reward when a clipless gun
        // fires. The confirmed-hit path above handles every weapon explicitly.
        if (effect == AmmoOnHitEffect)
        {
            continue;
        }
        // This is a Construction-PDA effect, not an active-weapon effect. The
        // separate prediction pass below installs it even though opening the
        // build menu commonly leaves the wrench as m_hActiveWeapon.
        if (effect == DisposableSentryEffect)
        {
            continue;
        }
        // This one uses a different launcher attribute below so TF2 retains
        // both normal explosion visuals and blast-jump movement.
        if (effect == NoSelfBlastEffect)
        {
            continue;
        }
        // Stomp damage and Thermal launch capacity are not ordinary weapon
        // attributes. Their event/damage paths above implement these effects.
        if ((WeaponBuffs_IsStompWeapon(weapon)
                && (effect == DamageEffect || WeaponBuffs_IsOnKillEffect(effect)))
            || (StrEqual(g_WeaponNames[weapon], "Thermal Thruster") && effect == ClipSizeEffect))
        {
            continue;
        }
        int levels = g_WeaponEffectLevels[weapon][effect];
        if (levels <= 0)
        {
            continue;
        }
        float value;
        bool energyClipSize = effect == ClipSizeEffect
            && WeaponBuffs_EnergyShotCost(entity) > 0.0;
        if (g_WeaponEffectModes[effect] == BuffMode_Toggle)
        {
            value = 1.0;
        }
        else if (g_WeaponEffectModes[effect] == BuffMode_Add)
        {
            value = WeaponBuffs_EffectIncrement(weapon, effect) * float(levels);
        }
        else
        {
            float base;
            if (energyClipSize)
            {
                base = TF2Attrib_HookValueFloat(1.0,
                    EnergyClipSizeAttributeClass, entity);
            }
            else
            {
                base = TF2Attrib_HookValueFloat(1.0,
                    g_WeaponEffectAttributeClasses[effect], entity);
            }
            if (FloatAbs(base) < 0.0001)
            {
                base = 1.0;
            }
            float combined = base
                + g_WeaponEffectIncrements[effect] * float(levels);
            // Inverted percentages such as fire/reload speed must never reach
            // zero: the engine can then generate attacks and snapshots faster
            // than the client channel can carry them. A melee swing past +50%
            // is faster than its own animation and lands nothing at all.
            float minimum = effect == FireRateEffect ? 0.2 : 0.05;
            if (effect == FireRateEffect && WeaponBuffs_IsMelee(entity))
            {
                minimum = MeleeFireRateFloor;
            }
            if (combined < minimum)
            {
                combined = minimum;
            }
            value = combined / base;
        }
        bool applied;
        if (energyClipSize)
        {
            applied = TF2Attrib_SetByName(provider, EnergyClipSizeAttribute, value);
        }
        else
        {
            applied = TF2Attrib_SetByName(provider, g_WeaponEffectAttributes[effect], value);
        }
        if (!applied)
        {
            LogError("Could not apply private effect %s to %s.",
                g_WeaponEffectAttributes[effect], g_WeaponNames[weapon]);
        }
    }
    WeaponBuffs_ApplyDisposableSentryPrediction(client, provider);
    WeaponBuffs_ApplyPassives(client, provider);
    TF2Attrib_ClearCache(entity);
}

static void WeaponBuffs_ApplyDisposableSentryPrediction(int client, int provider)
{
    // TF2's client refuses to send a build command unless predicted CanBuild
    // sees this attribute. Use the same private provider as every other AP
    // buff, leaving the item lists owned by the MvM station untouched. Both
    // PDAs share one reward pool; placement consumes one toolbox-side slot.
    int levels = WeaponBuffs_DisposableSentryLevels();
    if (g_WeaponBuffAtUpgradeStation[client] || levels <= 0)
    {
        return;
    }
    int active = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
    if (active <= MaxClients || !IsValidEntity(active)
        || !HasEntProp(active, Prop_Send, "m_iItemDefinitionIndex"))
    {
        return;
    }
    int definition = GetEntProp(active, Prop_Send, "m_iItemDefinitionIndex");
    bool pda = definition == 25 || definition == 737 || definition == 15025;
    bool toolbox = definition == 28 || definition == 15028;
    if (!pda && !toolbox)
    {
        return;
    }
    if (toolbox)
    {
        levels++;
    }
    if (!TF2Attrib_SetByName(provider, "engy disposable sentries", float(levels)))
    {
        LogError("Could not provide disposable-sentry level %d to item %d for %N.",
            levels, definition, client);
    }
}

/*
The passive buffs, from every slot of the loadout rather than the held weapon.

These are player attributes: the base to scale from is the player's own hooked
value, not a weapon's, because no weapon contributes to it. Health regeneration
is not here at all, since the timer pays it directly.
*/
static void WeaponBuffs_ApplyPassives(int client, int provider)
{
    static const int passives[2] = { MoveSpeedEffect, JumpHeightEffect };
    for (int index = 0; index < sizeof(passives); index++)
    {
        int effect = passives[index];
        int levels = WeaponBuffs_LoadoutLevels(client, effect);
        if (levels <= 0)
        {
            continue;
        }
        float base = TF2Attrib_HookValueFloat(1.0,
            g_WeaponEffectAttributeClasses[effect], client);
        if (FloatAbs(base) < 0.0001)
        {
            base = 1.0;
        }
        float combined = base + g_WeaponEffectIncrements[effect] * float(levels);
        if (combined < 0.05)
        {
            combined = 0.05;
        }
        if (!TF2Attrib_SetByName(provider, g_WeaponEffectAttributes[effect],
            combined / base))
        {
            LogError("Could not apply passive effect %s to %N.",
                g_WeaponEffectAttributes[effect], client);
        }
    }
}

public int Menu_WeaponBuffs(Menu menu, MenuAction action, int client, int selection)
{
    if (action == MenuAction_End)
    {
        delete menu;
    }
    return 0;
}

// One line per buff on the loadout, and how many lines a loadout can carry.
#define BuffLineMax 160
#define BuffLinesMax (6 * WeaponEffectCount)

static void WeaponBuffs_DescribeEffect(int weapon, int effect,
    char[] description, int maxlength)
{
    if (effect == DamageEffect && WeaponBuffs_IsStompWeapon(weapon))
    {
        strcopy(description, maxlength, "+5x fall-damage multiplier");
    }
    else if (effect == ClipSizeEffect
        && StrEqual(g_WeaponNames[weapon], "Thermal Thruster"))
    {
        strcopy(description, maxlength, "+1 launch charge");
    }
    else
    {
        strcopy(description, maxlength, g_WeaponEffectDescriptions[effect]);
    }
}

/* The listing, which the menu and the station announcement both draw from:
 * one line per buffed effect on the weapons the player is holding now.
 */
static int WeaponBuffs_Lines(int client, char lines[BuffLinesMax][BuffLineMax])
{
    int listed;
    bool seen[WeaponCount];
    for (int slot = 0; slot <= 5; slot++)
    {
        int weapon = WeaponBuffs_ForEntity(WeaponBuffs_EntityInLoadoutSlot(client, slot));
        if (weapon < 0 || seen[weapon])
        {
            continue;
        }
        // Functional families can occupy two loadout slots at once. The
        // Construction PDA and its toolbox are one reward pool and one line.
        seen[weapon] = true;
        for (int effect = 0; effect < WeaponEffectCount; effect++)
        {
            int levels = g_WeaponEffectLevels[weapon][effect];
            if (levels == 0)
            {
                continue;
            }
            char description[96];
            WeaponBuffs_DescribeEffect(weapon, effect, description, sizeof(description));
            if (levels > 1)
            {
                FormatEx(lines[listed], BuffLineMax, "%s: %s x%d",
                    g_WeaponNames[weapon], description, levels);
            }
            else
            {
                FormatEx(lines[listed], BuffLineMax, "%s: %s",
                    g_WeaponNames[weapon], description);
            }
            listed++;
        }
    }
    return listed;
}

/* False when the loadout has nothing to list, and nothing was drawn. A menu
 * holds the number keys, which are the weapon slots in MvM, so an empty one
 * costs the player more than the space it takes.
 */
bool WeaponBuffs_ShowMenu(int client)
{
    if (!MvM_IsPlayer(client))
    {
        return false;
    }
    char lines[BuffLinesMax][BuffLineMax];
    int listed = WeaponBuffs_Lines(client, lines);
    if (listed == 0)
    {
        return false;
    }
    Menu menu = new Menu(Menu_WeaponBuffs);
    menu.SetTitle("Archipelago weapon buffs");
    for (int line = 0; line < listed; line++)
    {
        menu.AddItem("", lines[line]);
    }
    menu.ExitButton = true;
    menu.Display(client, 20);
    return true;
}

// How long the station stays quiet after it has listed a player's buffs once.
#define BuffAnnounceQuietSeconds 30.0

static float g_BuffsAnnouncedAt[MAXPLAYERS + 1];

/* The station says what the loadout carries, in chat, where it takes no keys.
 *
 * A menu holds the number keys and those are the weapon slots, so the window
 * that used to open here cost the player twenty seconds of switching weapons
 * by scroll. The list is the same one !ap buffs draws, and the window is still
 * there for whoever asks for it. A loadout with buffs that lists nothing is the
 * fault behind apw-z7o, so that case says which of the three checks failed.
 */
void WeaponBuffs_AnnounceAtStation(int client)
{
    if (!MvM_IsPlayer(client))
    {
        return;
    }
    if (GetGameTime() < g_BuffsAnnouncedAt[client] + BuffAnnounceQuietSeconds)
    {
        return;
    }
    char lines[BuffLinesMax][BuffLineMax];
    int listed = WeaponBuffs_Lines(client, lines);
    if (listed == 0)
    {
        WeaponBuffs_DebugEmptyListing(client);
        return;
    }
    g_BuffsAnnouncedAt[client] = GetGameTime();
    AP_PrintToClient(client, "Buffs on this loadout (!ap buffs for the list):");
    for (int line = 0; line < listed; line++)
    {
        AP_PrintToClient(client, "  %s", lines[line]);
    }
}

/* Names the three things an empty listing can mean, once per station visit,
 * so the next report says which one it was rather than "it did not open".
 */
static void WeaponBuffs_DebugEmptyListing(int client)
{
    int granted;
    for (int weapon = 0; weapon < WeaponCount; weapon++)
    {
        for (int effect = 0; effect < WeaponEffectCount; effect++)
        {
            granted += g_WeaponEffectLevels[weapon][effect];
        }
    }
    char held[64];
    for (int slot = 0; slot <= 5; slot++)
    {
        int entity = WeaponBuffs_EntityInLoadoutSlot(client, slot);
        int definition = entity > MaxClients && IsValidEntity(entity)
            && HasEntProp(entity, Prop_Send, "m_iItemDefinitionIndex")
            ? GetEntProp(entity, Prop_Send, "m_iItemDefinitionIndex") : -1;
        Format(held, sizeof(held), "%s%s%d", held, slot == 0 ? "" : " ", definition);
    }
    AP_Debug("%N opened the station with nothing to list: %d buff level(s) granted in total, holding %s.",
        client, granted, held);
}

static void WeaponBuffs_DebugDisposableEntity(int client, const char[] label,
    int entity)
{
    if (entity <= MaxClients || !IsValidEntity(entity))
    {
        ReplyToCommand(client, "[AP debug] %s: none", label);
        return;
    }
    char classname[48];
    GetEntityClassname(entity, classname, sizeof(classname));
    int definition = HasEntProp(entity, Prop_Send, "m_iItemDefinitionIndex")
        ? GetEntProp(entity, Prop_Send, "m_iItemDefinitionIndex") : -1;
    int weapon = WeaponBuffs_ForEntity(entity);
    int levels = weapon >= 0
        ? g_WeaponEffectLevels[weapon][DisposableSentryEffect] : -1;
    float hooked = TF2Attrib_HookValueFloat(0.0,
        "engy_disposable_sentries", entity);
    ReplyToCommand(client,
        "[AP debug] %s: ent=%d class=%s def=%d canonical=%d AP=%d direct=%d hooked=%.1f",
        label, entity, classname, definition, weapon, levels,
        WeaponBuffs_DirectDisposableSentryLevels(entity), hooked);
}

public Action Command_DisposableSentryDebug(int client, int args)
{
    if (client < 1 || client > MaxClients || !IsClientInGame(client))
    {
        ReplyToCommand(client, "[AP debug] Run sm_ap_buff_debug from the player console.");
        return Plugin_Handled;
    }
    int active = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
    WeaponBuffs_DebugDisposableEntity(client, "active", active);
    WeaponBuffs_DebugDisposableEntity(client, "slot 3 / Construction PDA",
        WeaponBuffs_EntityInLoadoutSlot(client, 3));
    WeaponBuffs_DebugDisposableEntity(client, "slot 5 / PDA toolbox",
        WeaponBuffs_EntityInLoadoutSlot(client, 5));
    WeaponBuffs_DebugDisposableEntity(client, "last switch argument",
        EntRefToEntIndex(g_DisposableSentryLastSwitchArgumentRef[client]));
    WeaponBuffs_DebugDisposableEntity(client, "active during switch callback",
        EntRefToEntIndex(g_DisposableSentryLastSwitchActiveRef[client]));

    int provider = EntRefToEntIndex(g_WeaponBuffProviderRef[client]);
    int providerDirect = provider > MaxClients && IsValidEntity(provider)
        ? WeaponBuffs_DirectDisposableSentryLevels(provider) : -1;
    int primary;
    int disposable;
    int sentry = -1;
    while ((sentry = FindEntityByClassname(sentry, "obj_sentrygun")) != -1)
    {
        if (GetEntPropEnt(sentry, Prop_Send, "m_hBuilder") != client
            || (HasEntProp(sentry, Prop_Send, "m_bPlacing")
                && GetEntProp(sentry, Prop_Send, "m_bPlacing") != 0))
        {
            continue;
        }
        if (GetEntProp(sentry, Prop_Send, "m_bDisposableBuilding") != 0)
        {
            disposable++;
        }
        else
        {
            primary++;
        }
    }
    ReplyToCommand(client,
        "[AP debug] shared AP=%d native=%d provider=%d provider-direct=%d private-providers=%d station=%d sentries=%d primary/%d disposable",
        WeaponBuffs_DisposableSentryLevels(),
        WeaponBuffs_NativeDisposableSentryLevels(client), provider,
        providerDirect, WeaponBuffs_PrivateProviderCount(),
        g_WeaponBuffAtUpgradeStation[client], primary, disposable);
    ReplyToCommand(client,
        "[AP debug] CanBuild sentry calls=%d overrides=%d last-result=%d",
        g_DisposableSentryCanBuildCalls[client],
        g_DisposableSentryCanBuildOverrides[client],
        g_DisposableSentryLastCanBuildResult[client]);
    return Plugin_Handled;
}

public Action Command_WeaponBuffs(int client, int args)
{
    if (!WeaponBuffs_ShowMenu(client))
    {
        ReplyToCommand(client, "[AP] No buffs for this loadout yet.");
    }
    return Plugin_Handled;
}

static int WeaponBuffs_AddTestEffects(int weapon, const char[] wanted, int levels)
{
    int granted;
    for (int effect = 0; effect < WeaponEffectCount; effect++)
    {
        bool selected = StrEqual(wanted, "all", false)
            || StrEqual(wanted, g_WeaponEffectKeys[effect], false)
            || StringToInt(wanted) == effect + 1;
        if (!selected)
        {
            continue;
        }
        g_WeaponEffectLevels[weapon][effect] = g_WeaponEffectModes[effect] == BuffMode_Toggle
            ? 1 : g_WeaponEffectLevels[weapon][effect] + levels;
        granted++;
    }
    return granted;
}

static int WeaponBuffs_ParseLoadoutSlot(const char[] wanted)
{
    if (StrEqual(wanted, "1") || StrEqual(wanted, "primary", false))
    {
        return Slot_Primary;
    }
    if (StrEqual(wanted, "2") || StrEqual(wanted, "secondary", false))
    {
        return Slot_Secondary;
    }
    if (StrEqual(wanted, "3") || StrEqual(wanted, "melee", false))
    {
        return Slot_Melee;
    }
    return -1;
}

// sm_ap_buff_test <effect number|effect key|all> [levels]
public Action Command_TestWeaponBuff(int client, int args)
{
    if (!MvM_IsPlayer(client) || !IsPlayerAlive(client))
    {
        ReplyToCommand(client, "[AP] Join RED and spawn before testing weapon buffs.");
        return Plugin_Handled;
    }
    if (args < 1)
    {
        ReplyToCommand(client, "[AP] Usage: sm_ap_buff_test <1-%d|effect-key|all> [levels]", WeaponEffectCount);
        return Plugin_Handled;
    }
    int entity = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
    int weapon = WeaponBuffs_ForEntity(entity);
    if (weapon < 0)
    {
        ReplyToCommand(client, "[AP] The active weapon is not in the catalog.");
        return Plugin_Handled;
    }
    char wanted[64];
    GetCmdArg(1, wanted, sizeof(wanted));
    int levels = 1;
    if (args >= 2)
    {
        char rawLevels[12];
        GetCmdArg(2, rawLevels, sizeof(rawLevels));
        levels = StringToInt(rawLevels);
        if (levels < 1) levels = 1;
    }
    int granted = WeaponBuffs_AddTestEffects(weapon, wanted, levels);
    if (granted == 0)
    {
        ReplyToCommand(client, "[AP] Unknown effect %s. Use 1-%d or an effect key.", wanted, WeaponEffectCount);
        return Plugin_Handled;
    }
    WeaponBuffs_Apply(client);
    ReplyToCommand(client, "[AP] Applied %d test effect(s) to %s.",
        granted, g_WeaponNames[weapon]);
    WeaponBuffs_ShowMenu(client);
    return Plugin_Handled;
}

// sm_ap_buff_give <target> <effect number|effect key|all> [levels]
public Action Command_GiveWeaponBuff(int client, int args)
{
    if (args < 2)
    {
        ReplyToCommand(client,
            "[AP] Usage: sm_ap_buff_give <target> <1-%d|effect-key|all> [levels]",
            WeaponEffectCount);
        return Plugin_Handled;
    }

    char pattern[MAX_TARGET_LENGTH];
    GetCmdArg(1, pattern, sizeof(pattern));
    int target = FindTarget(client, pattern);
    if (target < 1)
    {
        return Plugin_Handled;
    }
    if (!WeaponBuffs_CanUseRunBuffs(target) || !IsPlayerAlive(target))
    {
        ReplyToCommand(client, "[AP] %N must be alive on RED before receiving a test buff.", target);
        return Plugin_Handled;
    }

    int entity = GetEntPropEnt(target, Prop_Send, "m_hActiveWeapon");
    int weapon = WeaponBuffs_ForEntity(entity);
    if (weapon < 0)
    {
        ReplyToCommand(client, "[AP] %N's active weapon is not in the catalog.", target);
        return Plugin_Handled;
    }

    char wanted[64];
    GetCmdArg(2, wanted, sizeof(wanted));
    int levels = 1;
    if (args >= 3)
    {
        char rawLevels[12];
        GetCmdArg(3, rawLevels, sizeof(rawLevels));
        levels = StringToInt(rawLevels);
        if (levels < 1) levels = 1;
    }

    int granted = WeaponBuffs_AddTestEffects(weapon, wanted, levels);
    if (granted == 0)
    {
        ReplyToCommand(client, "[AP] Unknown effect %s. Use 1-%d or an effect key.",
            wanted, WeaponEffectCount);
        return Plugin_Handled;
    }

    WeaponBuffs_Apply(target);
    ReplyToCommand(client, "[AP] Applied %d test effect(s) to %N's %s.",
        granted, target, g_WeaponNames[weapon]);
    if (client != target)
    {
        PrintToChat(target, "[AP] An admin applied %d test effect(s) to your %s.",
            granted, g_WeaponNames[weapon]);
    }
    WeaponBuffs_ShowMenu(target);
    return Plugin_Handled;
}

// sm_ap_buff_slot <target> <1|2|3|primary|secondary|melee> <effect> [levels]
public Action Command_GiveSlotWeaponBuff(int client, int args)
{
    if (args < 3)
    {
        ReplyToCommand(client,
            "[AP] Usage: sm_ap_buff_slot <target> <1|2|3|primary|secondary|melee> <1-%d|effect-key|all> [levels]",
            WeaponEffectCount);
        return Plugin_Handled;
    }

    char pattern[MAX_TARGET_LENGTH];
    GetCmdArg(1, pattern, sizeof(pattern));
    int target = FindTarget(client, pattern);
    if (target < 1)
    {
        return Plugin_Handled;
    }
    if (!WeaponBuffs_CanUseRunBuffs(target) || !IsPlayerAlive(target))
    {
        ReplyToCommand(client, "[AP] %N must be alive on RED before receiving a test buff.", target);
        return Plugin_Handled;
    }

    char rawSlot[16];
    GetCmdArg(2, rawSlot, sizeof(rawSlot));
    int slot = WeaponBuffs_ParseLoadoutSlot(rawSlot);
    if (slot < 0)
    {
        ReplyToCommand(client, "[AP] Unknown slot %s. Use 1/primary, 2/secondary, or 3/melee.", rawSlot);
        return Plugin_Handled;
    }

    int entity = WeaponBuffs_EntityInLoadoutSlot(target, slot);
    int weapon = WeaponBuffs_ForEntity(entity);
    if (weapon < 0)
    {
        ReplyToCommand(client, "[AP] %N has no catalog item equipped in slot %d (%s).",
            target, slot + 1, g_SlotKeys[slot]);
        return Plugin_Handled;
    }

    char wanted[64];
    GetCmdArg(3, wanted, sizeof(wanted));
    int levels = 1;
    if (args >= 4)
    {
        char rawLevels[12];
        GetCmdArg(4, rawLevels, sizeof(rawLevels));
        levels = StringToInt(rawLevels);
        if (levels < 1) levels = 1;
    }

    int granted = WeaponBuffs_AddTestEffects(weapon, wanted, levels);
    if (granted == 0)
    {
        ReplyToCommand(client, "[AP] Unknown effect %s. Use 1-%d or an effect key.",
            wanted, WeaponEffectCount);
        return Plugin_Handled;
    }

    WeaponBuffs_Apply(target);
    ReplyToCommand(client, "[AP] Applied %d test effect(s) to %N's slot %d %s.",
        granted, target, slot + 1, g_WeaponNames[weapon]);
    if (client != target)
    {
        PrintToChat(target, "[AP] An admin applied %d test effect(s) to your %s.",
            granted, g_WeaponNames[weapon]);
    }
    WeaponBuffs_ShowMenu(target);
    return Plugin_Handled;
}
