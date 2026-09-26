/** RED ammo limits and dispenser timing for Make it Count. */
bool g_ChamberReady;
Handle g_ChamberMaxAmmo;
float g_ChamberNextSupply[MAXPLAYERS + 1];
int g_ChamberBonusDamagePercent = 150;

// Whether the GetMaxAmmo hook has run since the map started. SigMod
// reconfigures its own detour on that function at every mission load, and a
// reconfigure used to delete this hook; the first call on each map is logged
// so a server's log shows the hook survived it.
bool g_ChamberHookSeen;

void Chamber_OnMapStart()
{
    g_ChamberHookSeen = false;
}

bool Chamber_Enabled()
{
    return g_ChamberReady && (g_ActiveMissionModifiers & MissionModifier_OneInTheChamber) != 0;
}

bool Chamber_Red(int client)
{
    return client >= 1 && client <= MaxClients && IsClientInGame(client)
        && GetClientTeam(client) == TeamRed;
}

void Chamber_Init()
{
    GameData data = LoadGameConfigFile("tf2_archipelago");
    if (data == null) return;
    bool ready = Chamber_Detour(data, "CTFPlayer::GetMaxAmmo", Hook_Post, Chamber_MaxAmmo);
    ready = Chamber_Detour(data, "CObjectDispenser::DispenseAmmo", Hook_Pre, Chamber_DispensePre) && ready;
    // CRobotDispenser::DispenseAmmo is `return false`, so a hook that can only
    // hold a supply back has nothing to do there. On Windows it is also one
    // stub MSVC shares with every other function that returns false.
    StartPrepSDKCall(SDKCall_Player);
    if (PrepSDKCall_SetFromConf(data, SDKConf_Signature, "CTFPlayer::GetMaxAmmo"))
    {
        PrepSDKCall_AddParameter(SDKType_PlainOldData, SDKPass_Plain);
        PrepSDKCall_AddParameter(SDKType_PlainOldData, SDKPass_Plain);
        PrepSDKCall_SetReturnInfo(SDKType_PlainOldData, SDKPass_Plain);
        g_ChamberMaxAmmo = EndPrepSDKCall();
    }
    g_ChamberReady = ready && g_ChamberMaxAmmo != null;
    if (!g_ChamberReady) AP_Error("Make it Count disabled: required ammo hooks unavailable.");
    delete data;
    int entity = -1;
    while ((entity = FindEntityByClassname(entity, "tank_boss")) != -1) Chamber_DamageHook(entity);
}

bool Chamber_Detour(GameData data, const char[] name, HookMode mode, DHookCallback callback)
{
    DynamicDetour hook = DynamicDetour.FromConf(data, name);
    bool ok = hook != null && hook.Enable(mode, callback);
    if (!ok) LogError("Make it Count: cannot hook %s", name);
    delete hook;
    return ok;
}

public MRESReturn Chamber_MaxAmmo(int client, DHookReturn result, DHookParam args)
{
    if (!g_ChamberHookSeen)
    {
        g_ChamberHookSeen = true;
        char map[64];
        GetCurrentMap(map, sizeof(map));
        LogMessage("Make it Count: the GetMaxAmmo hook runs on %s", map);
    }
    // Ammo types 1/2 are primary/secondary, 3 is Engineer metal, and
    // remaining positive types include throwable ammo.
    if (!Chamber_Enabled() || !Chamber_Red(client) || args.Get(1) <= 0)
        return MRES_Ignored;
    result.Value = Chamber_ReserveLimit(result.Value, args.Get(1));
    return MRES_Override;
}

public MRESReturn Chamber_DispensePre(int dispenser, DHookReturn result, DHookParam args)
{
    int client = args.Get(1);
    if (!Chamber_Enabled() || !Chamber_Red(client)) return MRES_Ignored;
    float now = GetGameTime();
    if (!Chamber_ShouldSupply(now, g_ChamberNextSupply[client]))
    {
        result.Value = false;
        return MRES_Supercede;
    }
    // Count even a pulse that finds full ammo. Spending ammo while standing
    // beside a dispenser must not trigger an immediate refill.
    g_ChamberNextSupply[client] = now + 4.0;
    return MRES_Ignored;
}

public Action Command_ChamberDamage(int client, int argc)
{
    if (argc > 0)
    {
        char value[24];
        GetCmdArg(1, value, sizeof(value));
        if (StrEqual(value, "reset", false)) g_ChamberBonusDamagePercent = 150;
        else
        {
            int percent;
            if (StringToIntEx(value, percent) != strlen(value) || percent < 0 || percent > 5000)
            {
                ReplyToCommand(client, "[AP] Usage: sm_ap_count_damage [0-5000|reset] (extra damage percent)");
                return Plugin_Handled;
            }
            g_ChamberBonusDamagePercent = percent;
        }
    }
    ReplyToCommand(client, "[AP] Make it Count: +%d%% non-melee damage (%.2fx total).",
        g_ChamberBonusDamagePercent, 1.0 + float(g_ChamberBonusDamagePercent) / 100.0);
    return Plugin_Handled;
}

void Chamber_HookClient(int client)
{
    g_ChamberNextSupply[client] = 0.0;
    SDKUnhook(client, SDKHook_PostThinkPost, Chamber_ClampClient);
    SDKHook(client, SDKHook_PostThinkPost, Chamber_ClampClient);
    Chamber_DamageHook(client);
}

public void Chamber_ClampClient(int client)
{
    if (!Chamber_Enabled() || !Chamber_Red(client) || !IsPlayerAlive(client)) return;
    int ammoCount = GetEntPropArraySize(client, Prop_Send, "m_iAmmo");
    for (int ammo = 1; ammo < ammoCount; ammo++)
    {
        int limit = SDKCall(g_ChamberMaxAmmo, client, ammo, -1);
        int held = GetEntProp(client, Prop_Send, "m_iAmmo", _, ammo);
        if (limit >= 0 && held > limit) SetEntProp(client, Prop_Send, "m_iAmmo", limit, _, ammo);
    }
}

void Chamber_DamageHook(int entity)
{
    SDKUnhook(entity, SDKHook_OnTakeDamage, Chamber_Damage);
    SDKHook(entity, SDKHook_OnTakeDamage, Chamber_Damage);
}

public Action Chamber_Damage(int victim, int &attacker, int &inflictor, float &damage,
    int &damageType, int &weapon, float force[3], float position[3], int custom)
{
    if (!Chamber_Enabled() || !Chamber_Red(attacker) || victim == attacker) return Plugin_Continue;
    bool ranged = weapon > MaxClients && IsValidEntity(weapon)
        && weapon != GetPlayerWeaponSlot(attacker, TFWeaponSlot_Melee);
    if (inflictor > MaxClients && IsValidEntity(inflictor))
    {
        char classname[64];
        GetEntityClassname(inflictor, classname, sizeof(classname));
        if (StrEqual(classname, "obj_sentrygun") || StrEqual(classname, "tf_projectile_sentryrocket")) ranged = true;
    }
    if (!ranged) return Plugin_Continue;
    damage *= 1.0 + float(g_ChamberBonusDamagePercent) / 100.0;
    return Plugin_Changed;
}

void Chamber_EntityCreated(int entity, const char[] classname)
{
    if (entity <= MaxClients || entity >= MissionModifier_MaxEntities) return;
    if (StrEqual(classname, "tank_boss") || StrContains(classname, "obj_") == 0)
        SDKHook(entity, SDKHook_SpawnPost, Chamber_BuildingSpawned);
}

public void Chamber_BuildingSpawned(int entity)
{
    Chamber_DamageHook(entity);
}

void Chamber_Reset()
{
    for (int client = 1; client <= MaxClients; client++) g_ChamberNextSupply[client] = 0.0;
}

void Chamber_Shutdown()
{
    delete g_ChamberMaxAmmo;
    g_ChamberMaxAmmo = null;
}
