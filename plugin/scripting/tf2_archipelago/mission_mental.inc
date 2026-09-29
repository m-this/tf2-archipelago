/** Mental uses stock all-class stealth and RemoveInvisibility for decloaking. */
bool g_MentalCloak[MAXPLAYERS + 1];
bool g_MentalFade[MAXPLAYERS + 1];
float g_MentalNextCloak[MAXPLAYERS + 1];
float g_MentalRecloakFadeStart[MAXPLAYERS + 1];
float g_MentalShimmerUntil[MAXPLAYERS + 1];
bool g_MentalAllowDecloak[MAXPLAYERS + 1];
Handle g_MentalRemoveInvisibility;
bool g_MentalReady;
bool g_MentalHidden[MissionModifier_MaxEntities];
RenderMode g_MentalRender[MissionModifier_MaxEntities];
int g_MentalColor[MissionModifier_MaxEntities][4];
float g_MentalEntityShimmerUntil[MissionModifier_MaxEntities];
float g_MentalEntityRevealUntil[MissionModifier_MaxEntities];
bool g_MentalEntityDamageHooked[MissionModifier_MaxEntities];

void Mental_Init()
{
    GameData data = LoadGameConfigFile("tf2_archipelago");
    if (data == null) return;
    StartPrepSDKCall(SDKCall_Player);
    if (PrepSDKCall_SetFromConf(data, SDKConf_Signature, "CTFPlayer::RemoveInvisibility"))
        g_MentalRemoveInvisibility = EndPrepSDKCall();
    DynamicDetour hook = DynamicDetour.FromConf(data, "CTFPlayer::RemoveInvisibility");
    bool hooked = hook != null && hook.Enable(Hook_Pre, Mental_BlockAttackDecloak);
    delete hook;
    delete data;
    g_MentalReady = g_MentalRemoveInvisibility != null && hooked;
    if (!g_MentalReady)
        AP_Error("Mental robot cloak disabled: stock RemoveInvisibility hook is unavailable.");
}

public MRESReturn Mental_BlockAttackDecloak(int client)
{
    if (client >= 1 && client <= MaxClients && g_MentalCloak[client]
        && !g_MentalAllowDecloak[client]) return MRES_Supercede;
    return MRES_Ignored;
}

bool Mental_ActiveRobot(int client)
{
    return g_MentalReady
        && (g_ActiveMissionModifiers & MissionModifier_Mental) != 0
        && MvM_WaveInProgress() && MissionModifiers_IsRobot(client) && IsPlayerAlive(client);
}

void Mental_Decloak(int client, float until)
{
    if (until > g_MentalNextCloak[client]) g_MentalNextCloak[client] = until;
    g_MentalRecloakFadeStart[client] = 0.0;
    g_MentalShimmerUntil[client] = 0.0;
    if (g_MentalCloak[client] && TF2_IsPlayerInCondition(client, TFCond_Stealthed))
    {
        // This is the same function the reveal-on-hit attribute calls for a
        // Spy. The SDK call bypasses that caller's class check, not the fade.
        g_MentalAllowDecloak[client] = true;
        SDKCall(g_MentalRemoveInvisibility, client);
        g_MentalAllowDecloak[client] = false;
    }
}

public void TF2_OnConditionRemoved(int client, TFCond condition)
{
    if (condition != TFCond_Stealthed || !g_MentalCloak[client]) return;
    g_MentalCloak[client] = false;
    g_MentalRecloakFadeStart[client] = 0.0;
    g_MentalFade[client] = TF2_IsPlayerInCondition(client, TFCond_StealthedUserBuffFade);
    float until = GetGameTime() + 1.5;
    if (until > g_MentalNextCloak[client]) g_MentalNextCloak[client] = until;
}

void Mental_ForgetEntity(int entity)
{
    g_MentalHidden[entity] = false;
    g_MentalEntityDamageHooked[entity] = false;
    g_MentalEntityShimmerUntil[entity] = 0.0;
    g_MentalEntityRevealUntil[entity] = 0.0;
}

void Mental_Render(int entity, float amount)
{
    if (amount <= 0.0)
    {
        if (!g_MentalHidden[entity]) return;
        SetEntityRenderMode(entity, g_MentalRender[entity]);
        SetEntityRenderColor(entity, g_MentalColor[entity][0], g_MentalColor[entity][1],
            g_MentalColor[entity][2], g_MentalColor[entity][3]);
        g_MentalHidden[entity] = false;
        return;
    }
    if (!g_MentalHidden[entity])
    {
        g_MentalRender[entity] = GetEntityRenderMode(entity);
        GetEntityRenderColor(entity, g_MentalColor[entity][0], g_MentalColor[entity][1],
            g_MentalColor[entity][2], g_MentalColor[entity][3]);
        g_MentalHidden[entity] = true;
    }
    SetEntityRenderMode(entity, RENDER_TRANSCOLOR);
    SetEntityRenderColor(entity, g_MentalColor[entity][0], g_MentalColor[entity][1],
        g_MentalColor[entity][2], RoundToNearest(float(g_MentalColor[entity][3]) * (1.0 - amount)));
}

void Mental_Reset()
{
    for (int client = 1; client <= MaxClients; client++)
    {
        Mental_ReleaseClient(client);
    }
    for (int entity = MaxClients + 1; entity < MissionModifier_MaxEntities; entity++)
    {
        if (g_MentalHidden[entity] && IsValidEntity(entity)) Mental_Render(entity, 0.0);
        g_MentalHidden[entity] = false;
        g_MentalEntityShimmerUntil[entity] = 0.0;
        g_MentalEntityRevealUntil[entity] = 0.0;
    }
}

public Action Mental_Timer(Handle timer)
{
    float now = GetGameTime();
    bool active = (g_ActiveMissionModifiers & MissionModifier_Mental) != 0 && MvM_WaveInProgress();
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client)) { Mental_ReleaseClient(client); continue; }
        if (!active || !MissionModifiers_IsRobot(client) || !IsPlayerAlive(client))
        {
            Mental_ReleaseClient(client);
            continue;
        }
        if (!g_MentalReady) continue;
        if (g_MentalCloak[client] && now < g_MentalShimmerUntil[client])
            SetEntPropFloat(client, Prop_Send, "m_flInvisChangeCompleteTime", now + 0.15);
        else if (g_MentalCloak[client] && g_MentalShimmerUntil[client] > 0.0)
        {
            g_MentalShimmerUntil[client] = 0.0;
            SetEntPropFloat(client, Prop_Send, "m_flInvisChangeCompleteTime", now);
        }
        else if (g_MentalCloak[client] && g_MentalRecloakFadeStart[client] > 0.0)
        {
            float progress = Mental_RecloakProgress(now, g_MentalRecloakFadeStart[client]);
            // A one-time write can be lost when the client adds condition 64.
            // Update the native cloak endpoint throughout the fade so the
            // material follows the same gradual transition on both sides.
            SetEntPropFloat(client, Prop_Send, "m_flInvisChangeCompleteTime",
                now + 1.0 - progress);
            if (progress >= 1.0) g_MentalRecloakFadeStart[client] = 0.0;
        }
        if (g_MentalFade[client] && !TF2_IsPlayerInCondition(client, TFCond_StealthedUserBuffFade))
            g_MentalFade[client] = false;
        // Never overwrite another source of stealth or restart an in-progress
        // stock fade. The old timer reapplied stealth during these transitions.
        bool stealthed = TF2_IsPlayerInCondition(client, TFCond_Cloaked)
            || TF2_IsPlayerInCondition(client, TFCond_Stealthed)
            || TF2_IsPlayerInCondition(client, TFCond_StealthedUserBuffFade);
        if (!Mental_CanRecloak(now, g_MentalNextCloak[client], stealthed)) continue;
        g_MentalCloak[client] = true;
        TF2_AddCondition(client, TFCond_Stealthed);
        g_MentalRecloakFadeStart[client] = now;
        // Condition 64 normally enters instantly. The timer maintains this
        // endpoint until the cloak transition is complete.
        SetEntPropFloat(client, Prop_Send, "m_flInvisChangeCompleteTime", now + 1.0);
    }
    int tank = -1;
    while ((tank = FindEntityByClassname(tank, "tank_boss")) != -1)
    {
        // A tank with no edict is a reference, not an index: no client sees
        // it, and it indexes none of these arrays.
        if (tank < 0) continue;
        Mental_HookEntityDamage(tank);
        float tankAmount = active ? 1.0 - Mental_EntityVisibility(now,
            g_MentalEntityShimmerUntil[tank], g_MentalEntityRevealUntil[tank]) : 0.0;
        Mental_RenderTree(tank, tankAmount, 0);
        int turret = EntRefToEntIndex(g_TankSentryRef[tank]);
        if (turret > MaxClients && IsValidEntity(turret))
        {
            Mental_HookEntityDamage(turret);
            Mental_Render(turret, tankAmount);
        }
    }
    int building = -1;
    while ((building = FindEntityByClassname(building, "obj_*")) != -1)
    {
        // The tank loop already renders its mounted sentry with the tank's
        // visibility. Rendering it again here would hide a revealed tank's gun.
        if (Mental_TankForSentry(building) > MaxClients) continue;
        if (GetEntProp(building, Prop_Send, "m_iTeamNum") == TeamBlue)
        {
            Mental_HookEntityDamage(building);
            float buildingAmount = active ? 1.0 - Mental_EntityVisibility(now,
                g_MentalEntityShimmerUntil[building], g_MentalEntityRevealUntil[building]) : 0.0;
            Mental_RenderTree(building, buildingAmount, 0);
        }
        else if (g_MentalHidden[building]) Mental_RenderTree(building, 0.0, 0);
    }
    return Plugin_Continue;
}

void Mental_ReleaseClient(int client)
{
    if (client < 1 || client > MaxClients) return;
    bool cloak = g_MentalCloak[client];
    bool fade = g_MentalFade[client];
    // Clear ownership before removal so our cleanup cannot schedule a recloak.
    g_MentalCloak[client] = false;
    g_MentalFade[client] = false;
    g_MentalNextCloak[client] = 0.0;
    g_MentalRecloakFadeStart[client] = 0.0;
    g_MentalShimmerUntil[client] = 0.0;
    g_MentalAllowDecloak[client] = false;
    if (!IsClientInGame(client)) return;
    if (cloak) TF2_RemoveCondition(client, TFCond_Stealthed);
    if (fade) TF2_RemoveCondition(client, TFCond_StealthedUserBuffFade);
    if ((cloak || fade) && !TF2_IsPlayerInCondition(client, TFCond_Cloaked)
        && !TF2_IsPlayerInCondition(client, TFCond_Stealthed)
        && !TF2_IsPlayerInCondition(client, TFCond_StealthedUserBuffFade))
        SetEntPropFloat(client, Prop_Send, "m_flInvisChangeCompleteTime", GetGameTime());
}

void Mental_RenderTree(int entity, float amount, int depth)
{
    if (entity <= MaxClients || entity >= MissionModifier_MaxEntities || !IsValidEntity(entity) || depth > 8) return;
    Mental_Render(entity, amount);
    // Tanks have separate track/bomb models attached as move children.
    if (!HasEntProp(entity, Prop_Data, "m_hMoveChild")) return;
    int child = GetEntPropEnt(entity, Prop_Data, "m_hMoveChild");
    while (child > MaxClients && IsValidEntity(child))
    {
        Mental_RenderTree(child, amount, depth + 1);
        if (!HasEntProp(child, Prop_Data, "m_hMovePeer")) break;
        child = GetEntPropEnt(child, Prop_Data, "m_hMovePeer");
    }
}

public void Mental_OnTakeDamagePost(int victim, int attacker, int inflictor,
    float damage, int damagetype, int weapon, const float damageForce[3],
    const float damagePosition[3], int damagecustom)
{
    if (damage <= 0.0 || damagecustom == TF_CUSTOM_BLEEDING
        || (damagecustom == TF_CUSTOM_BURNING && inflictor == attacker)
        || !Mental_ActiveRobot(victim) || attacker < 1
        || attacker > MaxClients || !IsClientInGame(attacker)
        || GetClientTeam(attacker) != TeamRed) return;
    if (WeaponBuffs_HasRevealOnHit(attacker, inflictor, weapon, damagecustom))
    {
        Mental_RevealOnHit(victim);
        return;
    }
    float until = GetGameTime() + 0.3;
    if (until > g_MentalShimmerUntil[victim]) g_MentalShimmerUntil[victim] = until;
}

void Mental_HookEntityDamage(int entity)
{
    if (entity <= MaxClients || entity >= MissionModifier_MaxEntities
        || g_MentalEntityDamageHooked[entity]) return;
    SDKHook(entity, SDKHook_OnTakeDamagePost, Mental_OnEntityDamagePost);
    g_MentalEntityDamageHooked[entity] = true;
}

int Mental_TankForSentry(int sentry)
{
    int tank = -1;
    while ((tank = FindEntityByClassname(tank, "tank_boss")) != -1)
    {
        if (tank < 0) continue;
        if (EntRefToEntIndex(g_TankSentryRef[tank]) == sentry) return tank;
    }
    return -1;
}

public void Mental_OnEntityDamagePost(int entity, int attacker, int inflictor,
    float damage, int damagetype, int weapon, const float damageForce[3],
    const float damagePosition[3], int damagecustom)
{
    if (damage <= 0.0 || damagecustom == TF_CUSTOM_BLEEDING
        || (damagecustom == TF_CUSTOM_BURNING && inflictor == attacker)
        || (g_ActiveMissionModifiers & MissionModifier_Mental) == 0
        || !MvM_WaveInProgress() || attacker < 1 || attacker > MaxClients
        || !IsClientInGame(attacker) || GetClientTeam(attacker) != TeamRed
        || GetEntProp(entity, Prop_Send, "m_iTeamNum") != TeamBlue) return;
    int tank = Mental_TankForSentry(entity);
    if (tank > MaxClients) entity = tank;
    if (WeaponBuffs_HasRevealOnHit(attacker, inflictor, weapon, damagecustom))
        g_MentalEntityRevealUntil[entity] = GetGameTime() + 5.0;
    else g_MentalEntityShimmerUntil[entity] = GetGameTime() + 0.3;
}

// Called only by the confirmed enemy-hit path for a weapon with the buff.
void Mental_RevealOnHit(int victim)
{
    if (!Mental_ActiveRobot(victim)) return;
    bool newlyRevealed = GetGameTime() >= g_MentalNextCloak[victim];
    // Further buffed hits renew the full five seconds; ordinary damage only
    // changes the visibility of the cloak for 0.3 seconds.
    Mental_Decloak(victim, GetGameTime() + 5.0);
    if (newlyRevealed) AP_Debug("Mental: a reveal-on-hit weapon decloaked %N for five seconds.", victim);
}
