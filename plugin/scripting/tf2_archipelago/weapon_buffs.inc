/** Every positive weapon/effect permutation received from the multiworld. */

#define ProjectileCountEffect 5
#define FireRateEffect 1
// The multiplier a melee weapon's fire rate never goes below: +50% and no more.
#define MeleeFireRateFloor 0.5
#define BleedEffect 8
#define IgniteEffect 9
#define MadMilkEffect 14
#define GasolineEffect 15
#define MarkForDeathEffect 16
#define JarateEffect 60
#define ActiveHealthRegenEffect 66

#define BuffMode_Add 1
#define BuffMode_Percentage 2
#define BuffMode_Toggle 3

int g_WeaponEffectLevels[WeaponCount][WeaponEffectCount];
int g_WeaponBuffProviderRef[MAXPLAYERS + 1];
ConVar g_WeaponBuffProjectileDebug;
ConVar g_WeaponBuffMirrorRobots;
bool g_WeaponBuffFiringExtra;
int g_WeaponBuffLastExtra;
Handle g_WeaponBuffFlareCreate;
Handle g_WeaponBuffArrowCreate;
Handle g_WeaponBuffProjectileGetDamage;
Handle g_WeaponBuffProjectileSetDamage;
Handle g_WeaponBuffFlareAdd;
bool g_WeaponBuffNativeExtra[2049];
float g_WeaponBuffNativeExtraBorn[2049];
bool g_WeaponBuffArrowExtra[2049];
bool g_WeaponBuffArrowTouched[2049];
float g_WeaponBuffArrowOrigin[2049][3];
int g_WeaponBuffStickyMimic[2049];
int g_WeaponBuffLastButtons[MAXPLAYERS + 1];
int g_WeaponBuffLastSubstanceWeapon[MAXPLAYERS + 1];
float g_WeaponBuffLastSubstanceTime[MAXPLAYERS + 1];
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
    g_WeaponBuffProjectileDebug = CreateConVar("tf2ap_projectile_debug", "0",
        "Log extra-projectile creation and lifetime", FCVAR_NONE, true, 0.0, true, 1.0);
    CreateTimer(1.0, Timer_WeaponBuffHealthRegen, _, TIMER_REPEAT);
    g_WeaponBuffMirrorRobots = CreateConVar("tf2ap_mirror_buffs_to_robots", "0",
        "Mirror Archipelago weapon buffs to enemy BLU robots", FCVAR_NONE, true, 0.0, true, 1.0);
    g_WeaponBuffMirrorRobots.AddChangeHook(WeaponBuffs_MirrorRobotsChanged);
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
        int active = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
        int weapon = WeaponBuffs_ForEntity(active);
        if (weapon < 0)
        {
            continue;
        }
        int levels = g_WeaponEffectLevels[weapon][ActiveHealthRegenEffect];
        if (levels <= 0)
        {
            continue;
        }

        // CTFPlayerResource asks CTFPlayer::GetMaxHealth for this value. That
        // includes class health and every active weapon, MvM and Archipelago
        // max-health modifier; m_iMaxHealth on the entity itself does not.
        int maximum = GetEntProp(resource, Prop_Send, "m_iMaxHealth", 4, client);
        int health = GetClientHealth(client);
        int amount = RoundToFloor(
            g_WeaponEffectIncrements[ActiveHealthRegenEffect] * float(levels));
        int missing = maximum - health;
        int healed = amount < missing ? amount : missing;
        if (healed <= 0)
        {
            continue;
        }

        SetEntityHealth(client, health + healed);
        Event event = CreateEvent("player_healonhit");
        if (event != null)
        {
            event.SetInt("amount", healed);
            event.SetInt("entindex", client);
            event.SetInt("weapon_def_index",
                GetEntProp(active, Prop_Send, "m_iItemDefinitionIndex"));
            event.Fire();
        }
    }
    return Plugin_Continue;
}

static bool WeaponBuffs_IsEnemyRobot(int client)
{
    return client > 0 && client <= MaxClients
        && IsClientInGame(client)
        && IsFakeClient(client)
        && !IsClientSourceTV(client)
        && GetClientTeam(client) == TeamBlue;
}

// The catalog is run-wide, so every gameplay path must pass through this
// ownership policy before reading it. Otherwise a robot carrying a matching
// weapon silently receives the human player's Archipelago effects.
static bool WeaponBuffs_CanUseRunBuffs(int client)
{
    return MvM_IsPlayer(client)
        || (g_WeaponBuffMirrorRobots != null
            && g_WeaponBuffMirrorRobots.BoolValue
            && WeaponBuffs_IsEnemyRobot(client));
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
    if (g_WeaponBuffArrowCreate == null)
    {
        LogError("Projectile-count arrow fanout disabled: this TF2 build does not match the native arrow gamedata.");
    }
}

void WeaponBuffs_HookClient(int client)
{
    g_WeaponBuffLastButtons[client] = 0;
    g_WeaponBuffLastSubstanceWeapon[client] = -1;
    g_WeaponBuffLastSubstanceTime[client] = 0.0;
    SDKUnhook(client, SDKHook_OnTakeDamagePost, WeaponBuffs_OnTakeDamagePost);
    SDKHook(client, SDKHook_OnTakeDamagePost, WeaponBuffs_OnTakeDamagePost);
    SDKUnhook(client, SDKHook_WeaponSwitchPost, WeaponBuffs_WeaponSwitchPost);
    SDKHook(client, SDKHook_WeaponSwitchPost, WeaponBuffs_WeaponSwitchPost);
}

public void WeaponBuffs_WeaponSwitchPost(int client, int weapon)
{
    WeaponBuffs_Apply(client);
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
}

// Item attributes implement these effects for only some of the weapons they
// can be awarded to. Apply the reward from the resolved catalogue entry on a
// confirmed enemy hit so bullets, projectiles and jars all use the same rules.
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
    }
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

    // Create()/InitArrow() supplies the private scorer, hit list, touch
    // callback and launch time that cannot be reproduced with netprops.
    SDKCall(g_WeaponBuffProjectileSetDamage, extra, damage);
    if (HasEntProp(extra, Prop_Send, "m_hLauncher"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hLauncher", launcher);
    }
    if (HasEntProp(extra, Prop_Send, "m_hOriginalLauncher"))
    {
        SetEntPropEnt(extra, Prop_Send, "m_hOriginalLauncher", launcher);
    }
    WeaponBuffs_CopySendInt(source, extra, "m_bCritical");
    WeaponBuffs_CopySendInt(source, extra, "m_bArrowAlight");
    WeaponBuffs_CopySendInt(source, extra, "m_bChargedShot");
    WeaponBuffs_CopySendInt(source, extra, "m_CollisionGroup");
    if (!HasEntProp(source, Prop_Send, "m_bCritical")
        && HasEntProp(extra, Prop_Send, "m_bCritical"))
    {
        SetEntProp(extra, Prop_Send, "m_bCritical", critical);
    }
    if (HasEntProp(extra, Prop_Send, "m_vInitialVelocity"))
    {
        SetEntPropVector(extra, Prop_Send, "m_vInitialVelocity", velocity);
    }
    TeleportEntity(extra, NULL_VECTOR, angles, velocity);

    g_WeaponBuffArrowExtra[extra] = true;
    g_WeaponBuffArrowTouched[extra] = false;
    g_WeaponBuffArrowOrigin[extra] = origin;
    SDKHook(extra, SDKHook_StartTouch, WeaponBuffs_ArrowStartTouch);
    CreateTimer(0.15, WeaponBuffs_ArrowFlightCheck, EntIndexToEntRef(extra),
        TIMER_FLAG_NO_MAPCHANGE);
    return extra;
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
        || damagecustom == TF_CUSTOM_BURNING || damagecustom == TF_CUSTOM_BLEEDING)
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
    if ((weapon <= MaxClients || !IsValidEntity(weapon)) && inflictor > MaxClients
        && IsValidEntity(inflictor) && HasEntProp(inflictor, Prop_Send, "m_hLauncher"))
    {
        weapon = GetEntPropEnt(inflictor, Prop_Send, "m_hLauncher");
    }
    if ((weapon <= MaxClients || !IsValidEntity(weapon)) && IsPlayerAlive(attacker))
    {
        weapon = GetEntPropEnt(attacker, Prop_Send, "m_hActiveWeapon");
    }
    int catalog = WeaponBuffs_ForEntity(weapon);
    if (catalog >= 0)
    {
        WeaponBuffs_ApplyHitEffects(victim, attacker, catalog);
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
        g_WeaponBuffArrowCreate != null, g_WeaponBuffFlareCreate != null, g_WeaponBuffProjectileGetDamage != null,
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

static int WeaponBuffs_ForDefinition(int definition)
{
    for (int index = 0; index < sizeof(g_WeaponByDefinition); index++)
    {
        if (g_WeaponByDefinition[index][0] == definition)
        {
            return g_WeaponByDefinition[index][1];
        }
    }
    return -1;
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

    provider = CreateEntityByName("tf_wearable");
    if (provider <= MaxClients || !IsValidEntity(provider))
    {
        LogError("Could not create the private Archipelago attribute provider for %N.", client);
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
    DispatchSpawn(provider);
    if (!HasEntProp(provider, Prop_Send, "m_ProviderType"))
    {
        LogError("The private Archipelago attribute provider has no m_ProviderType.");
        RemoveEntity(provider);
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
    SetVariantString("self.ReapplyProvision()");
    AcceptEntityInput(provider, "RunScriptCode");
    if (GetEntPropEnt(provider, Prop_Send, "m_hOwnerEntity") != client)
    {
        LogError("Could not attach the private Archipelago attribute provider to %N.", client);
        RemoveEntity(provider);
        g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
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
        LogError("The private Archipelago provider attached to %N but did not provide attributes.", client);
        RemoveEntity(provider);
        g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
        return -1;
    }
    return provider;
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
        int entity = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
        if (entity > MaxClients && IsValidEntity(entity))
        {
            TF2Attrib_ClearCache(entity);
        }
    }
}

void WeaponBuffs_Disconnect(int client)
{
    int provider = EntRefToEntIndex(g_WeaponBuffProviderRef[client]);
    g_WeaponBuffProviderRef[client] = INVALID_ENT_REFERENCE;
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
    if (weapon < 0)
    {
        return;
    }
    for (int effect = 0; effect < WeaponEffectCount; effect++)
    {
        // Current TF2 emits the heal popup for this schema attribute but its
        // negative-damage heal changes no health. The timer above owns this
        // effect so it can heal to the player's real, modified maximum.
        if (effect == ActiveHealthRegenEffect)
        {
            continue;
        }
        int levels = g_WeaponEffectLevels[weapon][effect];
        if (levels <= 0)
        {
            continue;
        }
        float value;
        if (g_WeaponEffectModes[effect] == BuffMode_Toggle)
        {
            value = 1.0;
        }
        else if (g_WeaponEffectModes[effect] == BuffMode_Add)
        {
            value = g_WeaponEffectIncrements[effect] * float(levels);
        }
        else
        {
            float base = TF2Attrib_HookValueFloat(1.0,
                g_WeaponEffectAttributeClasses[effect], entity);
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
        if (!TF2Attrib_SetByName(provider, g_WeaponEffectAttributes[effect], value))
        {
            LogError("Could not apply private effect %s to %s.",
                g_WeaponEffectAttributes[effect], g_WeaponNames[weapon]);
        }
    }
    TF2Attrib_ClearCache(entity);
}

public int Menu_WeaponBuffs(Menu menu, MenuAction action, int client, int selection)
{
    if (action == MenuAction_End)
    {
        delete menu;
    }
    return 0;
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
    Menu menu = new Menu(Menu_WeaponBuffs);
    menu.SetTitle("Archipelago weapon buffs");
    int listed;
    for (int slot = 0; slot <= 5; slot++)
    {
        int weapon = WeaponBuffs_ForEntity(GetPlayerWeaponSlot(client, slot));
        if (weapon < 0)
        {
            continue;
        }
        for (int effect = 0; effect < WeaponEffectCount; effect++)
        {
            int levels = g_WeaponEffectLevels[weapon][effect];
            if (levels == 0)
            {
                continue;
            }
            char line[160];
            if (levels > 1)
            {
                FormatEx(line, sizeof(line), "%s: %s x%d",
                    g_WeaponNames[weapon], g_WeaponEffectDescriptions[effect], levels);
            }
            else
            {
                FormatEx(line, sizeof(line), "%s: %s",
                    g_WeaponNames[weapon], g_WeaponEffectDescriptions[effect]);
            }
            menu.AddItem(g_WeaponEffectKeys[effect], line);
            listed++;
        }
    }
    if (listed == 0)
    {
        delete menu;
        return false;
    }
    menu.ExitButton = true;
    menu.Display(client, 20);
    return true;
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
    int target = FindTarget(client, pattern, true);
    if (target < 1)
    {
        return Plugin_Handled;
    }
    if (!MvM_IsPlayer(target) || !IsPlayerAlive(target))
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
