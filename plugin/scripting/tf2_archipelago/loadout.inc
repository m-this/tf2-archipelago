/**
 * !ap loadout: any weapon a class can hold, whether or not the player owns it.
 *
 * The weapons offered are the buff catalogue's, one per functional family, so
 * the count beside each is the run's buffs on it. Which class holds a weapon,
 * and where, is asked of the live item schema. The stock weapons are left out:
 * everybody has them, and Reset at the end of a list hands the slot back to
 * the player's own. A choice lasts while the player stays connected and is put
 * in their hands on every inventory application, in a slot the run has
 * unlocked.
 *
 * The game keeps MvM upgrades against the item they were bought for, and hands
 * them back when it hands that item out. A weapon given here is not one it
 * handed out, so ReapplyItemUpgrades is called for it; without that call a
 * chosen weapon would lose its upgrades on every resupply, so no call means no
 * menu.
 */

#define LoadoutItemsMax 64

static int g_LoadoutCount[view_as<int>(TFClass_Engineer) + 1][LoadoutPositionCount];
static int g_LoadoutWeapon[view_as<int>(TFClass_Engineer) + 1][LoadoutPositionCount][LoadoutItemsMax];
static int g_LoadoutDefinition[view_as<int>(TFClass_Engineer) + 1][LoadoutPositionCount][LoadoutItemsMax];
static bool g_LoadoutCatalogued;

// The buff weapon chosen per class and position, -1 for the player's own.
static int g_LoadoutChoice[MAXPLAYERS + 1][view_as<int>(TFClass_Engineer) + 1][LoadoutPositionCount];

static Handle g_LoadoutReapplyUpgrades;
static ConVar g_LoadoutForBots;

static char g_LoadoutSlotNames[][] = { "Primary", "Secondary", "Melee", "Watch" };

static char g_LoadoutClassNames[view_as<int>(TFClass_Engineer) + 1][] = {
    "", "Scout", "Sniper", "Soldier", "Demoman", "Medic", "Heavy", "Pyro", "Spy", "Engineer",
};

void Loadout_Init()
{
    g_LoadoutForBots = CreateConVar("tf2ap_loadout_for_bots", "0",
        "Let bots on RED use !ap loadout, to test it on a server with no player", FCVAR_NONE, true, 0.0, true, 1.0);
    for (int client = 0; client <= MaxClients; client++)
    {
        Loadout_Forget(client);
    }
    GameData data = LoadGameConfigFile("tf2_archipelago");
    if (data == null)
    {
        LogError("!ap loadout is off: gamedata tf2_archipelago did not load.");
        return;
    }
    StartPrepSDKCall(SDKCall_Player);
    if (PrepSDKCall_SetFromConf(data, SDKConf_Signature, "CTFPlayer::ReapplyItemUpgrades"))
    {
        PrepSDKCall_AddParameter(SDKType_PlainOldData, SDKPass_Plain);
        g_LoadoutReapplyUpgrades = EndPrepSDKCall();
    }
    delete data;
    if (g_LoadoutReapplyUpgrades == null)
    {
        LogError("!ap loadout is off: CTFPlayer::ReapplyItemUpgrades was not found.");
    }
}

void Loadout_Forget(int client)
{
    for (int class = 0; class < sizeof(g_LoadoutChoice[]); class++)
    {
        for (int position = 0; position < LoadoutPositionCount; position++)
        {
            g_LoadoutChoice[client][class][position] = -1;
        }
    }
}

// The schema is the server's for as long as it runs, so this is read once.
static void Loadout_Catalogue()
{
    if (g_LoadoutCatalogued)
    {
        return;
    }
    g_LoadoutCatalogued = true;
    bool baseSet[WeaponCount];
    for (int row = 0; row < sizeof(g_WeaponByDefinition); row++)
    {
        if (TF2Econ_IsItemInBaseSet(g_WeaponByDefinition[row][0]))
        {
            baseSet[g_WeaponByDefinition[row][1]] = true;
        }
    }
    for (int row = 0; row < sizeof(g_WeaponByDefinition); row++)
    {
        int definition = g_WeaponByDefinition[row][0];
        if (baseSet[g_WeaponByDefinition[row][1]] || !TF2Econ_IsValidItemDefinition(definition))
        {
            continue;
        }
        for (int class = view_as<int>(TFClass_Scout); class <= view_as<int>(TFClass_Engineer); class++)
        {
            int position = TF2Econ_GetItemLoadoutSlot(definition, view_as<TFClassType>(class));
            if (position >= 0 && position < LoadoutPositionCount
                && Loadout_UnlockSlot(class, position) >= 0)
            {
                Loadout_Add(class, position, g_WeaponByDefinition[row][1], definition);
            }
        }
    }
}

// The first definition of a weapon that the class can hold is the one given.
static void Loadout_Add(int class, int position, int weapon, int definition)
{
    int count = g_LoadoutCount[class][position];
    for (int index = 0; index < count; index++)
    {
        if (g_LoadoutWeapon[class][position][index] == weapon)
        {
            return;
        }
    }
    if (count == LoadoutItemsMax)
    {
        LogError("!ap loadout offers %d %s weapons in position %d and leaves out %s.",
            LoadoutItemsMax, g_LoadoutClassNames[class], position, g_WeaponNames[weapon]);
        return;
    }
    g_LoadoutWeapon[class][position][count] = weapon;
    g_LoadoutDefinition[class][position][count] = definition;
    g_LoadoutCount[class][position] = count + 1;
}

static int Loadout_Definition(int class, int position, int weapon)
{
    for (int index = 0; index < g_LoadoutCount[class][position]; index++)
    {
        if (g_LoadoutWeapon[class][position][index] == weapon)
        {
            return g_LoadoutDefinition[class][position][index];
        }
    }
    return -1;
}

static bool Loadout_Eligible(int client)
{
    return MvM_IsPlayer(client)
        || (g_LoadoutForBots.BoolValue && MvM_IsDefender(client) && IsFakeClient(client));
}

static int Loadout_BuffCount(int weapon)
{
    int count = 0;
    for (int effect = 0; effect < WeaponEffectCount; effect++)
    {
        count += g_WeaponEffectLevels[weapon][effect];
    }
    return count;
}

// The schema position this class holds in an unlock slot, or -1.
static int Loadout_Position(int class, int slot)
{
    for (int position = 0; position < LoadoutPositionCount; position++)
    {
        if (Loadout_UnlockSlot(class, position) == slot)
        {
            return position;
        }
    }
    return -1;
}

static bool Loadout_SlotOpen(TFClassType class, int position)
{
    int slot = Loadout_UnlockSlot(view_as<int>(class), position);
    return slot == LoadoutSlotAlwaysOpen || Unlocks_SlotAvailable(class, slot);
}

/* The client's items in a schema position: the weapons, and the wearables that
 * stand in for one (boots, shields, the Razorback). Cosmetics sit in positions
 * of their own and never match. Returns how many were written.
 */
static int Loadout_ItemsAt(int client, TFClassType class, int position, int[] items, int max)
{
    int found = 0;
    int weapons = GetEntPropArraySize(client, Prop_Send, "m_hMyWeapons");
    for (int index = 0; index < weapons && found < max; index++)
    {
        int weapon = GetEntPropEnt(client, Prop_Send, "m_hMyWeapons", index);
        if (weapon > MaxClients && Loadout_ItemPosition(weapon, class) == position)
        {
            items[found++] = weapon;
        }
    }
    int wearable = -1;
    while (found < max && (wearable = FindEntityByClassname(wearable, "tf_wearable*")) != -1)
    {
        if (GetEntPropEnt(wearable, Prop_Send, "m_hOwnerEntity") == client
            && Loadout_ItemPosition(wearable, class) == position)
        {
            items[found++] = wearable;
        }
    }
    return found;
}

static int Loadout_ItemPosition(int entity, TFClassType class)
{
    if (!HasEntProp(entity, Prop_Send, "m_iItemDefinitionIndex"))
    {
        return -1;
    }
    return TF2Econ_GetItemLoadoutSlot(GetEntProp(entity, Prop_Send, "m_iItemDefinitionIndex"), class);
}

/* Every inventory application. Runs ahead of the slot locks, and gives nothing
 * in a slot they would take away again.
 */
void Loadout_Apply(int client)
{
    if (g_LoadoutReapplyUpgrades == null || !Loadout_Eligible(client) || !IsPlayerAlive(client))
    {
        return;
    }
    TFClassType class = TF2_GetPlayerClass(client);
    if (class == TFClass_Unknown)
    {
        return;
    }
    Loadout_Catalogue();
    for (int position = 0; position < LoadoutPositionCount; position++)
    {
        int weapon = g_LoadoutChoice[client][class][position];
        if (weapon < 0 || !Loadout_SlotOpen(class, position))
        {
            continue;
        }
        int definition = Loadout_Definition(view_as<int>(class), position, weapon);
        if (definition < 0)
        {
            continue;
        }
        int items[4];
        int count = Loadout_ItemsAt(client, class, position, items, sizeof(items));
        if (count == 1 && WeaponBuffs_ForDefinition(
                GetEntProp(items[0], Prop_Send, "m_iItemDefinitionIndex")) == weapon)
        {
            continue;
        }
        bool wasActive = false;
        int active = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
        for (int index = 0; index < count; index++)
        {
            wasActive = wasActive || items[index] == active;
            Loadout_Remove(client, items[index]);
        }
        Loadout_Give(client, class, definition, wasActive);
    }
    // Boots or a shield in place of the held weapon leave nothing in hand.
    if (GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon") == -1)
    {
        Loadout_HoldAny(client);
    }
}

static void Loadout_HoldAny(int client)
{
    int weapons = GetEntPropArraySize(client, Prop_Send, "m_hMyWeapons");
    for (int index = 0; index < weapons; index++)
    {
        int weapon = GetEntPropEnt(client, Prop_Send, "m_hMyWeapons", index);
        char classname[64];
        if (weapon > MaxClients && GetEntityClassname(weapon, classname, sizeof(classname)))
        {
            FakeClientCommand(client, "use %s", classname);
            return;
        }
    }
}

static void Loadout_Remove(int client, int item)
{
    if (TF2Util_IsEntityWearable(item))
    {
        TF2_RemoveWearable(client, item);
        return;
    }
    TF2_RemoveWeaponSlot(client, TF2Util_GetWeaponSlot(item));
}

static void Loadout_Give(int client, TFClassType class, int definition, bool hold)
{
    char classname[64];
    if (!TF2Econ_GetItemClassName(definition, classname, sizeof(classname)))
    {
        LogError("!ap loadout cannot give item %d: the schema has no class name for it.", definition);
        return;
    }
    TF2Econ_TranslateWeaponEntForClass(classname, sizeof(classname), class);
    int item = CreateEntityByName(classname);
    if (item == -1)
    {
        LogError("!ap loadout cannot create %s for item %d.", classname, definition);
        return;
    }
    SetEntProp(item, Prop_Send, "m_iItemDefinitionIndex", definition);
    SetEntProp(item, Prop_Send, "m_bInitialized", 1);
    SetEntData(item, FindSendPropInfo("CEconEntity", "m_iEntityQuality"), 6);
    SetEntData(item, FindSendPropInfo("CEconEntity", "m_iEntityLevel"), 1);
    DispatchSpawn(item);
    if (TF2Util_IsEntityWearable(item))
    {
        TF2Util_EquipPlayerWearable(client, item);
    }
    else
    {
        EquipPlayerWeapon(client, item);
    }
    SDKCall(g_LoadoutReapplyUpgrades, client,
        GetEntityAddress(item) + view_as<Address>(FindSendPropInfo("CEconEntity", "m_Item")));

    if (HasEntProp(item, Prop_Send, "m_iPrimaryAmmoType"))
    {
        int ammo = GetEntProp(item, Prop_Send, "m_iPrimaryAmmoType");
        if (ammo > 0)
        {
            GivePlayerAmmo(client, 1000, ammo, true);
        }
    }
    if (hold && !TF2Util_IsEntityWearable(item))
    {
        FakeClientCommand(client, "use %s", classname);
    }
}

/* The menu. A slot, then a weapon for it, for the class the player is now. */
void Loadout_Open(int client)
{
    if (!Loadout_Eligible(client))
    {
        AP_PrintToClient(client, "Join RED to pick a loadout.");
        return;
    }
    if (g_LoadoutReapplyUpgrades == null)
    {
        AP_PrintToClient(client, "This server cannot keep MvM upgrades on a chosen weapon, so !ap loadout is off.");
        return;
    }
    TFClassType class = TF2_GetPlayerClass(client);
    if (class == TFClass_Unknown)
    {
        AP_PrintToClient(client, "Pick a class first.");
        return;
    }
    Loadout_Catalogue();

    Menu menu = new Menu(Loadout_SlotChosen);
    menu.SetTitle("%s loadout", g_LoadoutClassNames[class]);
    for (int slot = 0; slot < sizeof(g_LoadoutSlotNames); slot++)
    {
        int position = Loadout_Position(view_as<int>(class), slot);
        if (position < 0)
        {
            continue;
        }
        char key[8];
        IntToString(position, key, sizeof(key));
        char label[96];
        if (!Loadout_SlotOpen(class, position))
        {
            Format(label, sizeof(label), "%s: locked", g_LoadoutSlotNames[slot]);
            menu.AddItem(key, label, ITEMDRAW_DISABLED);
            continue;
        }
        int weapon = g_LoadoutChoice[client][class][position];
        Format(label, sizeof(label), "%s: %s", g_LoadoutSlotNames[slot],
            weapon < 0 ? "your own" : g_WeaponNames[weapon]);
        menu.AddItem(key, label);
    }
    menu.ExitButton = true;
    menu.Display(client, MENU_TIME_FOREVER);
}

public int Loadout_SlotChosen(Menu menu, MenuAction action, int client, int item)
{
    if (action == MenuAction_End)
    {
        delete menu;
        return 0;
    }
    if (action != MenuAction_Select)
    {
        return 0;
    }
    char key[8];
    menu.GetItem(item, key, sizeof(key));
    Loadout_OpenSlot(client, StringToInt(key));
    return 0;
}

static void Loadout_OpenSlot(int client, int position)
{
    TFClassType class = TF2_GetPlayerClass(client);
    int classIndex = view_as<int>(class);
    int count = g_LoadoutCount[classIndex][position];
    int buffs[LoadoutItemsMax];
    int order[LoadoutItemsMax];
    for (int index = 0; index < count; index++)
    {
        buffs[index] = Loadout_BuffCount(g_LoadoutWeapon[classIndex][position][index]);
    }
    Loadout_OrderByBuffs(buffs, count, order);

    Menu menu = new Menu(Loadout_WeaponChosen);
    menu.SetTitle("%s %s", g_LoadoutClassNames[classIndex],
        g_LoadoutSlotNames[Loadout_UnlockSlot(classIndex, position)]);

    // The class rides on every item, so a class change while the menu is open
    // cannot file the choice under the wrong class.
    char key[24];
    for (int rank = 0; rank < count; rank++)
    {
        int index = order[rank];
        int weapon = g_LoadoutWeapon[classIndex][position][index];
        char label[96];
        if (buffs[index] == 0)
        {
            strcopy(label, sizeof(label), g_WeaponNames[weapon]);
        }
        else
        {
            Format(label, sizeof(label), "%s (%d buff%s)", g_WeaponNames[weapon],
                buffs[index], buffs[index] == 1 ? "" : "s");
        }
        Format(key, sizeof(key), "%d:%d:%d", classIndex, position, weapon);
        menu.AddItem(key, label);
    }
    Format(key, sizeof(key), "%d:%d:-1", classIndex, position);
    menu.AddItem(key, "Reset: your own");
    menu.ExitBackButton = true;
    menu.Display(client, MENU_TIME_FOREVER);
}

public int Loadout_WeaponChosen(Menu menu, MenuAction action, int client, int item)
{
    if (action == MenuAction_End)
    {
        delete menu;
        return 0;
    }
    if (action == MenuAction_Cancel && item == MenuCancel_ExitBack)
    {
        Loadout_Open(client);
        return 0;
    }
    if (action != MenuAction_Select)
    {
        return 0;
    }
    char key[24];
    menu.GetItem(item, key, sizeof(key));
    char parts[3][8];
    if (ExplodeString(key, ":", parts, sizeof(parts), sizeof(parts[])) != 3)
    {
        return 0;
    }
    Loadout_Choose(client, StringToInt(parts[0]), StringToInt(parts[1]), StringToInt(parts[2]));
    Loadout_Open(client);
    return 0;
}

static void Loadout_Choose(int client, int class, int position, int weapon)
{
    g_LoadoutChoice[client][class][position] = weapon;

    char name[64];
    strcopy(name, sizeof(name), weapon < 0 ? "your own" : g_WeaponNames[weapon]);
    if (view_as<int>(TF2_GetPlayerClass(client)) == class && IsPlayerAlive(client)
        && Loadout_CanChangeNow(client))
    {
        TF2_RegeneratePlayer(client);
        AP_PrintToClient(client, "%s %s: %s.", g_LoadoutClassNames[class],
            g_LoadoutSlotNames[Loadout_UnlockSlot(class, position)], name);
        return;
    }
    AP_PrintToClient(client, "%s %s: %s, from your next resupply.", g_LoadoutClassNames[class],
        g_LoadoutSlotNames[Loadout_UnlockSlot(class, position)], name);
}

// Where the game itself lets a loadout change take effect at once.
static bool Loadout_CanChangeNow(int client)
{
    if (GameRules_GetRoundState() == RoundState_BetweenRounds)
    {
        return true;
    }
    float origin[3];
    GetClientAbsOrigin(client, origin);
    return TF2Util_IsPointInRespawnRoom(origin, client, true);
}

public Action Command_Loadout(int client, int args)
{
    if (client > 0)
    {
        Loadout_Open(client);
    }
    return Plugin_Handled;
}

// sm_ap_loadout_set <target> <primary|secondary|melee|watch> <weapon|own>
public Action Command_LoadoutSet(int client, int args)
{
    if (args < 3)
    {
        ReplyToCommand(client, "[AP] Usage: sm_ap_loadout_set <target> <primary|secondary|melee|watch> <weapon|own>");
        return Plugin_Handled;
    }
    char pattern[MAX_TARGET_LENGTH];
    GetCmdArg(1, pattern, sizeof(pattern));
    int target = FindTarget(client, pattern);
    if (target < 1)
    {
        return Plugin_Handled;
    }
    if (!Loadout_Eligible(target) || g_LoadoutReapplyUpgrades == null)
    {
        ReplyToCommand(client, "[AP] %N cannot use !ap loadout.", target);
        return Plugin_Handled;
    }
    int class = view_as<int>(TF2_GetPlayerClass(target));
    char rawSlot[16];
    GetCmdArg(2, rawSlot, sizeof(rawSlot));
    int position = -1;
    for (int slot = 0; slot < sizeof(g_LoadoutSlotNames); slot++)
    {
        if (StrEqual(rawSlot, g_LoadoutSlotNames[slot], false))
        {
            position = Loadout_Position(class, slot);
        }
    }
    if (position < 0)
    {
        ReplyToCommand(client, "[AP] %N has no %s slot.", target, rawSlot);
        return Plugin_Handled;
    }
    char wanted[64];
    GetCmdArg(3, wanted, sizeof(wanted));
    Loadout_Catalogue();
    int weapon = -1;
    if (!StrEqual(wanted, "own", false))
    {
        for (int index = 0; index < g_LoadoutCount[class][position]; index++)
        {
            if (StrEqual(g_WeaponNames[g_LoadoutWeapon[class][position][index]], wanted, false))
            {
                weapon = g_LoadoutWeapon[class][position][index];
            }
        }
        if (weapon < 0)
        {
            ReplyToCommand(client, "[AP] The %s %s list has no %s.", g_LoadoutClassNames[class], rawSlot, wanted);
            return Plugin_Handled;
        }
    }
    Loadout_Choose(target, class, position, weapon);
    ReplyToCommand(client, "[AP] %N's %s is %s.", target, rawSlot, weapon < 0 ? "their own" : g_WeaponNames[weapon]);
    return Plugin_Handled;
}
