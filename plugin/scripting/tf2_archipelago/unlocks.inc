/**
 * The unlock set, and what it does to players.
 *
 * The bridge owns the set. The plugin holds a copy for as long as the map
 * lasts, asks again after every reload or map change, and never writes back.
 */

enum
{
    Slot_Primary = 0,
    Slot_Secondary,
    Slot_Melee,
    Slot_Count,
};

// False until the bridge answers once; enforcement is a no-op until then, too much kit beats none.
bool g_HaveUnlocks;
bool g_UnlocksTestOverride;

bool g_ClassUnlocked[view_as<int>(TFClass_Engineer) + 1];
bool g_SlotUnlocked[Slot_Count];
StringMap g_UnlockedMissions;

// Indexed by TFClassType, whose order is not the class menu's order. Checked against tf2.inc.
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

// The order the progressive weapon slot opens a class's loadout in: the n-th
// slot held opens the n-th entry here. Indexed by TFClassType, like the arrays
// above, and holding slot numbers rather than keys.
//
// Six classes take the item's own order. Three cannot: a Medic without the
// Medigun, an Engineer without the Wrench and a Spy without the Knife play no
// part in a wave. gamedata/classes.go holds the same table and says why, and a
// test there compares the two.
int g_SlotOrder[view_as<int>(TFClass_Engineer) + 1][Slot_Count] = {
    { Slot_Primary, Slot_Secondary, Slot_Melee },   // unknown
    { Slot_Primary, Slot_Secondary, Slot_Melee },   // scout
    { Slot_Primary, Slot_Secondary, Slot_Melee },   // sniper
    { Slot_Primary, Slot_Secondary, Slot_Melee },   // soldier
    { Slot_Primary, Slot_Secondary, Slot_Melee },   // demoman
    { Slot_Secondary, Slot_Primary, Slot_Melee },   // medic
    { Slot_Primary, Slot_Secondary, Slot_Melee },   // heavy
    { Slot_Primary, Slot_Secondary, Slot_Melee },   // pyro
    { Slot_Melee, Slot_Secondary, Slot_Primary },   // spy
    { Slot_Melee, Slot_Primary, Slot_Secondary },   // engineer
};

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
    WeaponBuffs_Clear();
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
        LogError("The bridge granted the class %s. This plugin does not know it.", key);
        return;
    }
    g_ClassUnlocked[view_as<int>(class)] = true;
}

void Unlocks_GrantSlot(const char[] key)
{
    int slot = Unlocks_SlotFromKey(key);
    if (slot == -1)
    {
        LogError("The bridge granted the weapon slot %s. This plugin does not know it.", key);
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

// Classes and slots are enforced apart. A set with neither cannot be right,
// the generator guarantees one of each, so an empty one is read as "the bridge
// has not told us yet" rather than "take everything away". Holding them apart
// is what stops a run with classes but no slot yet from enforcing nothing,
// which is what a play-test saw: every class pickable for the first four waves.
bool Unlocks_ClassesEnforced()
{
    if (!g_HaveUnlocks || g_UnlocksTestOverride)
    {
        return false;
    }
    for (int class = 1; class < sizeof(g_ClassUnlocked); class++)
    {
        if (g_ClassUnlocked[class])
        {
            return true;
        }
    }
    return false;
}

bool Unlocks_SlotsEnforced()
{
    if (!g_HaveUnlocks || g_UnlocksTestOverride)
    {
        return false;
    }
    for (int slot = 0; slot < Slot_Count; slot++)
    {
        if (g_SlotUnlocked[slot])
        {
            return true;
        }
    }
    return false;
}

public Action Command_UnlockOverride(int client, int args)
{
    if (args < 1)
    {
        ReplyToCommand(client, "[AP] Class/slot override is %s. Usage: sm_ap_unlock_override <on|off>",
            g_UnlocksTestOverride ? "ON" : "OFF");
        return Plugin_Handled;
    }
    char wanted[16];
    GetCmdArg(1, wanted, sizeof(wanted));
    bool enabled = StrEqual(wanted, "on", false) || StrEqual(wanted, "1");
    bool disabled = StrEqual(wanted, "off", false) || StrEqual(wanted, "0");
    if (!enabled && !disabled)
    {
        ReplyToCommand(client, "[AP] Usage: sm_ap_unlock_override <on|off>");
        return Plugin_Handled;
    }
    g_UnlocksTestOverride = enabled;
    if (enabled)
    {
        for (int target = 1; target <= MaxClients; target++)
        {
            if (MvM_IsPlayer(target) && IsPlayerAlive(target))
            {
                Unlocks_RestoreFullLoadout(target);
            }
        }
    }
    else
    {
        Unlocks_ApplyToAll();
    }
    ReplyToCommand(client, "[AP] Class/slot override %s.", enabled ? "enabled" : "disabled");
    PrintToChatAll("[AP] Class and weapon-slot override %s for every player.",
        enabled ? "enabled" : "disabled");
    return Plugin_Handled;
}

static void Unlocks_RestoreFullLoadout(int client)
{
    if (!MvM_IsPlayer(client) || !IsPlayerAlive(client))
    {
        return;
    }
    // A between-wave respawn reconstructs weapons removed with
    // TF2_RemoveWeaponSlot more reliably than inventory regeneration alone.
    if (GameRules_GetRoundState() == RoundState_BetweenRounds)
    {
        TF2_RespawnPlayer(client);
    }
    else
    {
        TF2_RegeneratePlayer(client);
    }
}

static void Frame_UnlockOverrideLoadout(any userid)
{
    int client = GetClientOfUserId(userid);
    if (g_UnlocksTestOverride && MvM_IsPlayer(client) && IsPlayerAlive(client))
    {
        TF2_RegeneratePlayer(client);
    }
}

void Unlocks_RestoreOverrideNextFrame(int client)
{
    if (g_UnlocksTestOverride && client > 0)
    {
        RequestFrame(Frame_UnlockOverrideLoadout, GetClientUserId(client));
    }
}

// Runs as soon as the bridge answers: a grant that waits for the next spawn looks like no grant.
void Unlocks_ApplyToAll()
{
    if (!g_HaveUnlocks || !MvM_IsActive())
    {
        return;
    }
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!MvM_IsPlayer(client))
        {
            continue;
        }
        Unlocks_EnforceClass(client);
        Unlocks_EnforceSlots(client);
        WeaponBuffs_Apply(client);
    }
}

// Between waves the player is respawned on the spot, which is what the game
// itself does for a class change there. Mid-wave it waits for the next spawn:
// a forced respawn then hands back the life the death cost.
void Unlocks_EnforceClass(int client)
{
    if (!Unlocks_ClassesEnforced() || !MvM_IsPlayer(client))
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
        // Nothing to move them to; leave them alone rather than as nothing.
        return;
    }
    TF2_SetPlayerClass(client, allowed, false, true);
    if (GameRules_GetRoundState() == RoundState_BetweenRounds && IsPlayerAlive(client))
    {
        // Not from inside the spawn event that brought us here.
        RequestFrame(Frame_Respawn, GetClientUserId(client));
        AP_PrintToClient(client, "The run did not unlock %s. You play %s.",
            g_ClassKeys[view_as<int>(current)], g_ClassKeys[view_as<int>(allowed)]);
        return;
    }
    AP_PrintToClient(client, "The run did not unlock %s. You spawn as %s next time.",
        g_ClassKeys[view_as<int>(current)], g_ClassKeys[view_as<int>(allowed)]);
}

static void Frame_Respawn(any userid)
{
    int client = GetClientOfUserId(userid);
    if (MvM_IsPlayer(client) && GameRules_GetRoundState() == RoundState_BetweenRounds)
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

// How many slots the run holds. The item is progressive, so the count is all
// that carries: the bridge grants the keys in the item's own order, and which
// slot each one opens is the player's class to decide, not the bridge's.
static int Unlocks_SlotsHeld()
{
    int held = 0;
    for (int slot = 0; slot < Slot_Count; slot++)
    {
        if (g_SlotUnlocked[slot])
        {
            held++;
        }
    }
    return held;
}

static bool Unlocks_SlotOpen(TFClassType class, int slot, int held)
{
    for (int position = 0; position < held; position++)
    {
        if (g_SlotOrder[view_as<int>(class)][position] == slot)
        {
            return true;
        }
    }
    return false;
}

// Every inventory application: spawn, resupply locker, and the upgrade station's loadout changes.
void Unlocks_EnforceSlots(int client)
{
    if (!Unlocks_SlotsEnforced() || !MvM_IsPlayer(client) || !IsPlayerAlive(client))
    {
        return;
    }
    TFClassType class = TF2_GetPlayerClass(client);
    int held = Unlocks_SlotsHeld();
    bool removedActive = false;
    int active = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");

    for (int slot = 0; slot < Slot_Count; slot++)
    {
        if (Unlocks_SlotOpen(class, slot, held))
        {
            continue;
        }
        if (active != -1 && GetPlayerWeaponSlot(client, slot) == active)
        {
            removedActive = true;
        }
        TF2_RemoveWeaponSlot(client, slot);
    }
    if (removedActive)
    {
        SwitchToAnyWeapon(client, class, held);
    }
}

// Removing the held weapon leaves no viewmodel and no attack, and the player cannot switch back.
static void SwitchToAnyWeapon(int client, TFClassType class, int held)
{
    for (int position = 0; position < held; position++)
    {
        int weapon = GetPlayerWeaponSlot(client, g_SlotOrder[view_as<int>(class)][position]);
        if (weapon == -1)
        {
            continue;
        }
        char classname[64];
        if (GetEntityClassname(weapon, classname, sizeof(classname)))
        {
            FakeClientCommand(client, "use %s", classname);
            return;
        }
    }
    AP_PrintToClient(client, "The run locked every weapon slot. You hold nothing yet.");
}
