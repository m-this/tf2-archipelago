/**
 * The unlock set, and what it does to players.
 *
 * None of this is authoritative. The bridge owns the set; the plugin holds a
 * copy for as long as the map lasts and asks again after every reload or map
 * change. Nothing here is ever written back.
 */

enum
{
    Slot_Primary = 0,
    Slot_Secondary,
    Slot_Melee,
    Slot_Count,
};

// g_HaveUnlocks stays false until the bridge has answered once. Enforcement is
// a no-op until then: a server where nobody can hold a weapon because the
// bridge hiccuped is worse than a wave played with too much kit.
bool g_HaveUnlocks;

bool g_ClassUnlocked[view_as<int>(TFClass_Engineer) + 1];
bool g_SlotUnlocked[Slot_Count];
StringMap g_UnlockedMissions;

// The class keys the bridge speaks, indexed by TFClassType. Verified against
// tf2.inc: the game's own order is not the order the class menu shows.
char g_ClassKeys[view_as<int>(TFClass_Engineer) + 1][] = {
    "",
    "scout",
    "sniper",
    "soldier",
    "demoman",
    "medic",
    "heavy",
    "pyro",
    "spy",
    "engineer",
};

char g_SlotKeys[Slot_Count][] = { "primary", "secondary", "melee" };

void Unlocks_Init()
{
    g_UnlockedMissions = new StringMap();
}

void Unlocks_Clear()
{
    g_HaveUnlocks = false;
    for (int class = 0; class < sizeof(g_ClassUnlocked); class++)
    {
        g_ClassUnlocked[class] = false;
    }
    for (int slot = 0; slot < Slot_Count; slot++)
    {
        g_SlotUnlocked[slot] = false;
    }
    g_UnlockedMissions.Clear();
}

TFClassType Unlocks_ClassFromKey(const char[] key)
{
    for (int class = 1; class < sizeof(g_ClassKeys); class++)
    {
        if (StrEqual(g_ClassKeys[class], key, false))
        {
            return view_as<TFClassType>(class);
        }
    }
    return TFClass_Unknown;
}

int Unlocks_SlotFromKey(const char[] key)
{
    for (int slot = 0; slot < Slot_Count; slot++)
    {
        if (StrEqual(g_SlotKeys[slot], key, false))
        {
            return slot;
        }
    }
    return -1;
}

void Unlocks_GrantClass(const char[] key)
{
    TFClassType class = Unlocks_ClassFromKey(key);
    if (class == TFClass_Unknown)
    {
        LogError("bridge granted class %s, which this plugin does not know", key);
        return;
    }
    g_ClassUnlocked[view_as<int>(class)] = true;
}

void Unlocks_GrantSlot(const char[] key)
{
    int slot = Unlocks_SlotFromKey(key);
    if (slot == -1)
    {
        LogError("bridge granted weapon slot %s, which this plugin does not know", key);
        return;
    }
    g_SlotUnlocked[slot] = true;
}

void Unlocks_GrantMission(const char[] popFile)
{
    g_UnlockedMissions.SetValue(popFile, true);
}

bool Unlocks_HasMission(const char[] popFile)
{
    bool held;
    return g_UnlockedMissions.GetValue(popFile, held);
}

/**
 * Say something when the server is running a mission the run has not unlocked.
 *
 * The plugin does not refuse to play it: the map is the operator's choice, and
 * a plugin that fights the server's map rotation is a plugin that gets
 * unloaded. The checks are reported either way, so the only failure worth
 * preventing is nobody noticing the run has gone off its own logic.
 */
void Unlocks_WarnIfMissionLocked()
{
    if (!g_HaveUnlocks || !MvM_IsActive())
    {
        return;
    }
    char popFile[64];
    if (!MvM_PopFile(popFile, sizeof(popFile)) || Unlocks_HasMission(popFile))
    {
        return;
    }
    AP_Error("%s has not been unlocked by this run; its checks will still count", popFile);
}

/**
 * Apply the whole set to everyone currently on the server. Called after the
 * bridge answers, because a grant that only takes effect next spawn is a grant
 * the player thinks did not arrive.
 */
void Unlocks_ApplyToAll()
{
    if (!g_HaveUnlocks || !MvM_IsActive())
    {
        return;
    }
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client))
        {
            continue;
        }
        Unlocks_EnforceClass(client);
        Unlocks_EnforceSlots(client);
    }
}

/**
 * Move a player off a class they are not allowed to be. Classes are only ever
 * unlocked, never taken away, so this fires for someone who was already
 * playing when the plugin loaded rather than as a punishment.
 */
void Unlocks_EnforceClass(int client)
{
    if (!g_HaveUnlocks || !IsClientInGame(client))
    {
        return;
    }
    TFClassType current = TF2_GetPlayerClass(client);
    if (current == TFClass_Unknown || g_ClassUnlocked[view_as<int>(current)])
    {
        return;
    }
    TFClassType allowed = Unlocks_FirstAllowedClass();
    if (allowed == TFClass_Unknown)
    {
        return;
    }
    TF2_SetPlayerClass(client, allowed);
    AP_PrintToClient(client, "%s is locked, you are now %s.",
        g_ClassKeys[view_as<int>(current)], g_ClassKeys[view_as<int>(allowed)]);
    if (IsPlayerAlive(client))
    {
        TF2_RespawnPlayer(client);
    }
}

TFClassType Unlocks_FirstAllowedClass()
{
    for (int class = 1; class < sizeof(g_ClassUnlocked); class++)
    {
        if (g_ClassUnlocked[class])
        {
            return view_as<TFClassType>(class);
        }
    }
    return TFClass_Unknown;
}

/**
 * Take away the weapons in slots the run has not unlocked. Done on every
 * inventory application, which covers spawning, touching a resupply locker and
 * the loadout changes the upgrade station makes.
 */
void Unlocks_EnforceSlots(int client)
{
    if (!g_HaveUnlocks || !IsClientInGame(client) || !IsPlayerAlive(client))
    {
        return;
    }
    for (int slot = 0; slot < Slot_Count; slot++)
    {
        if (!g_SlotUnlocked[slot])
        {
            TF2_RemoveWeaponSlot(client, slot);
        }
    }
}
