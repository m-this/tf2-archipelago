/** Collectible RED robot cards. The launcher writes identities and AP innates
 * into the same numbered seat file the defender mod already reads. */
bool g_CardSelected[MAXPLAYERS + 1];
bool g_CardRobot[MAXPLAYERS + 1];
bool g_CardGiant[MAXPLAYERS + 1];
int g_CardTier[MAXPLAYERS + 1];
int g_CardCosmetic[MAXPLAYERS + 1];
bool g_CardUnusual[MAXPLAYERS + 1];
int g_CardHat[MAXPLAYERS + 1] = {INVALID_ENT_REFERENCE, ...};
int g_CardParticle[MAXPLAYERS + 1];
bool g_CardCosmeticPending[MAXPLAYERS + 1];
int g_CardEffect[MAXPLAYERS + 1][3];
int g_CardStacks[MAXPLAYERS + 1][3];

void BotCards_Clear(int client)
{
    g_CardSelected[client] = false;
    g_CardRobot[client] = false;
    g_CardGiant[client] = false;
    g_CardTier[client] = 0;
    g_CardCosmetic[client] = 0;
    g_CardUnusual[client] = false;
    g_CardHat[client] = INVALID_ENT_REFERENCE;
    g_CardParticle[client] = 0;
    for (int i = 0; i < 3; i++)
    {
        g_CardEffect[client][i] = -1;
        g_CardStacks[client][i] = 0;
    }
}

// A Giant card cannot walk back through a spawn door once it has grown, and
// the upgrade station is usually behind one. The defender mod lets it buy
// where it stands; Bots_PublishDirective asks for that.
bool BotCards_IsGiant(int client)
{
    return BotCards_HasInnates(client) && g_CardGiant[client];
}

bool BotCards_HasInnates(int client)
{
    return client > 0 && client <= MaxClients && g_CardSelected[client];
}

int BotCards_InnateLevels(int client, int effect)
{
    if (!BotCards_HasInnates(client))
    {
        return 0;
    }
    int total;
    for (int i = 0; i < 3; i++)
    {
        if (g_CardEffect[client][i] == effect)
        {
            total += g_CardStacks[client][i];
        }
    }
    return total;
}

void BotCards_OnSpawn(int client)
{
    if (client > 0 && IsClientInGame(client) && IsFakeClient(client)
        && GetClientTeam(client) == 2)
    {
        CreateTimer(0.4, Timer_BotCardsApply, GetClientUserId(client), TIMER_FLAG_NO_MAPCHANGE);
        // The defender mod dresses bots at 0.5–1.55 seconds after spawn.
        // Dress the selected card after it, so its requested hat wins.
        if (!g_CardCosmeticPending[client])
        {
            g_CardCosmeticPending[client] = true;
            CreateTimer(1.8, Timer_BotCardsCosmetic, GetClientUserId(client), TIMER_FLAG_NO_MAPCHANGE);
        }
    }
}

public Action Timer_BotCardsApply(Handle timer, any userid)
{
    int client = GetClientOfUserId(userid);
    if (client <= 0 || !IsClientInGame(client) || !IsFakeClient(client)
        || GetClientTeam(client) != 2 || !IsPlayerAlive(client))
    {
        return Plugin_Stop;
    }
    BotCards_Clear(client);
    char path[PLATFORM_MAX_PATH];
    BuildPath(Path_SM, path, sizeof(path), "configs/defenderbots/loadout.cfg");
    KeyValues lineup = new KeyValues("loadout");
    if (!lineup.ImportFromFile(path) || !lineup.JumpToKey("seats"))
    {
        delete lineup;
        return Plugin_Stop;
    }
    char botName[128], seatKey[8], seatName[128], seatClass[32], actualClass[32], raw[48], fields[3][16];
    GetClientName(client, botName, sizeof(botName));
    BotCards_ClassKey(TF2_GetPlayerClass(client), actualClass, sizeof(actualClass));
    // Every seat the file could hold: RED is six by default and more under a
    // larger team size, and a card in seat seven is still a card.
    for (int seat = 1; seat <= MaxClients; seat++)
    {
        IntToString(seat, seatKey, sizeof(seatKey));
        if (!lineup.JumpToKey(seatKey))
        {
            continue;
        }
        lineup.GetString("name", seatName, sizeof(seatName));
        if (StrEqual(botName, seatName, false) && lineup.GetNum("card", 0) == 1)
        {
            lineup.GetString("class", seatClass, sizeof(seatClass));
            if (!StrEqual(seatClass, actualClass))
            {
                LogError("[AP cards] %N spawned as %s, but saved seat %d requests %s. Apply the saved bot composition to the running server.",
                    client, actualClass, seat, seatClass);
                lineup.GoBack();
                continue;
            }
            g_CardSelected[client] = true;
            g_CardRobot[client] = lineup.GetNum("robot", 0) == 1;
            g_CardGiant[client] = lineup.GetNum("giant", 0) == 1;
            g_CardTier[client] = lineup.GetNum("tier", 1);
            g_CardCosmetic[client] = lineup.GetNum("cosmetic", 0);
            g_CardUnusual[client] = lineup.GetNum("unusual", 0) == 1;
            // The effect belongs to the card, not this spawn. An old seat
            // file without the field gets a stable effect until regenerated.
            g_CardParticle[client] = lineup.GetNum("unusual_effect", 0);
            if (g_CardUnusual[client] && g_CardParticle[client] <= 0)
            {
                g_CardParticle[client] = 13;
            }
            for (int i = 0; i < 3; i++)
            {
                Format(raw, sizeof(raw), "innate_%d", i + 1);
                char value[48];
                lineup.GetString(raw, value, sizeof(value));
                int parts = ExplodeString(value, ",", fields, sizeof(fields), sizeof(fields[]));
                if (parts == 2 || parts == 3)
                {
                    // Three fields are the older weapon-specific file shape;
                    // keep reading it so a plugin hot reload is map-safe.
                    g_CardEffect[client][i] = StringToInt(fields[parts - 2]);
                    g_CardStacks[client][i] = StringToInt(fields[parts - 1]);
                }
            }
            lineup.GoBack();
            break;
        }
        lineup.GoBack();
    }
    delete lineup;
    if (!g_CardSelected[client])
    {
        return Plugin_Stop;
    }

    if (g_CardRobot[client])
    {
        char model[PLATFORM_MAX_PATH], className[16];
        switch (TF2_GetPlayerClass(client))
        {
            case TFClass_Scout: strcopy(className, sizeof(className), "scout");
            case TFClass_Soldier: strcopy(className, sizeof(className), "soldier");
            case TFClass_Pyro: strcopy(className, sizeof(className), "pyro");
            case TFClass_DemoMan: strcopy(className, sizeof(className), "demo");
            case TFClass_Heavy: strcopy(className, sizeof(className), "heavy");
            case TFClass_Engineer: strcopy(className, sizeof(className), "engineer");
            case TFClass_Medic: strcopy(className, sizeof(className), "medic");
            case TFClass_Sniper: strcopy(className, sizeof(className), "sniper");
            case TFClass_Spy: strcopy(className, sizeof(className), "spy");
            default: return Plugin_Stop;
        }
        Format(model, sizeof(model), "models/bots/%s/bot_%s.mdl", className, className);
        PrecacheModel(model, true);
        SetVariantString(model);
        AcceptEntityInput(client, "SetCustomModel");
        SetEntProp(client, Prop_Send, "m_bUseClassAnimations", 1);
        if (g_CardGiant[client])
        {
            // A giant-sized collision hull cannot pass some spawn doors. Keep
            // the normal hull until the bot has cleared spawn and there is
            // actually room for the giant hull at its current position.
            SetEntPropFloat(client, Prop_Send, "m_flModelScale", 1.0);
            CreateTimer(0.5, Timer_BotCardsGrowGiant, userid,
                TIMER_REPEAT | TIMER_FLAG_NO_MAPCHANGE);
        }
    }

    // A tier's listed HP is natural class HP × 1/1.5/2. Giant doubles that.
    // This is a test-mode card modifier on RED bots only.
    int natural = BotCards_BaseHealth(TF2_GetPlayerClass(client));
    int tierMaximum = RoundToNearest(float(natural) * (g_CardTier[client] == 3 ? 2.0
        : g_CardTier[client] == 2 ? 1.5 : 1.0) * (g_CardGiant[client] ? 2.0 : 1.0));
    int maximum = tierMaximum + BotCards_LoadoutHealthDelta(client);
    TF2Attrib_RemoveByName(client, "max health additive bonus");
    if (tierMaximum > natural)
    {
        TF2Attrib_SetByName(client, "max health additive bonus", float(tierMaximum - natural));
        TF2Attrib_ClearCache(client);
        if (HasEntProp(client, Prop_Data, "m_iMaxHealth"))
        {
            SetEntProp(client, Prop_Data, "m_iMaxHealth", maximum);
        }
        SetEntityHealth(client, maximum);
    }
    WeaponBuffs_ApplyNextFrame(client);
    return Plugin_Stop;
}

bool BotCards_NearSpawn(int client)
{
    float point[3], origin[3], mins[3], maxs[3];
    GetClientAbsOrigin(client, point);
    int room = -1;
    while ((room = FindEntityByClassname(room, "func_respawnroom")) != -1)
    {
        int team = GetEntProp(room, Prop_Send, "m_iTeamNum");
        if (team != 0 && team != GetClientTeam(client))
        {
            continue;
        }
        GetEntPropVector(room, Prop_Data, "m_vecOrigin", origin);
        GetEntPropVector(room, Prop_Data, "m_vecMins", mins);
        GetEntPropVector(room, Prop_Data, "m_vecMaxs", maxs);
        float distanceSquared;
        for (int axis = 0; axis < 3; axis++)
        {
            float low = origin[axis] + mins[axis];
            float high = origin[axis] + maxs[axis];
            float outside = point[axis] < low ? low - point[axis]
                : point[axis] > high ? point[axis] - high : 0.0;
            distanceSquared += outside * outside;
        }
        if (distanceSquared < 192.0 * 192.0)
        {
            return true;
        }
    }
    return false;
}

public bool BotCards_TraceIgnorePlayers(int entity, int contentsMask, any data)
{
    return entity < 1 || entity > MaxClients;
}

public Action Timer_BotCardsGrowGiant(Handle timer, any userid)
{
    int client = GetClientOfUserId(userid);
    if (client <= 0 || !IsClientInGame(client) || !IsPlayerAlive(client)
        || !g_CardSelected[client] || !g_CardGiant[client])
    {
        return Plugin_Stop;
    }
    if (BotCards_NearSpawn(client))
    {
        return Plugin_Continue;
    }
    float point[3], mins[3] = {-42.0, -42.0, 0.0};
    float maxs[3] = {42.0, 42.0, 144.0};
    GetClientAbsOrigin(client, point);
    point[2] += 2.0;
    TR_TraceHullFilter(point, point, mins, maxs, MASK_PLAYERSOLID,
        BotCards_TraceIgnorePlayers);
    if (TR_DidHit())
    {
        return Plugin_Continue;
    }
    SetEntPropFloat(client, Prop_Send, "m_flModelScale", 1.75);
    return Plugin_Stop;
}

void BotCards_ClassKey(TFClassType classType, char[] out, int length)
{
    switch (classType)
    {
        case TFClass_Scout: strcopy(out, length, "scout");
        case TFClass_Soldier: strcopy(out, length, "soldier");
        case TFClass_Pyro: strcopy(out, length, "pyro");
        case TFClass_DemoMan: strcopy(out, length, "demoman");
        case TFClass_Heavy: strcopy(out, length, "heavyweapons");
        case TFClass_Engineer: strcopy(out, length, "engineer");
        case TFClass_Medic: strcopy(out, length, "medic");
        case TFClass_Sniper: strcopy(out, length, "sniper");
        case TFClass_Spy: strcopy(out, length, "spy");
        default: strcopy(out, length, "unknown");
    }
}

public Action Timer_BotCardsCosmetic(Handle timer, any userid)
{
    int client = GetClientOfUserId(userid);
    if (client > 0) g_CardCosmeticPending[client] = false;
    if (client <= 0 || !IsClientInGame(client) || !IsPlayerAlive(client)
        || !g_CardSelected[client] || g_CardCosmetic[client] <= 0)
    {
        return Plugin_Stop;
    }
    int item = g_CardCosmetic[client];
    TFClassType playerClass = TF2_GetPlayerClass(client);
    int miscSlot = TF2Econ_TranslateLoadoutSlotNameToIndex("misc");
    if (TF2Econ_GetItemLoadoutSlot(item, playerClass) != miscSlot)
    {
        LogError("[AP cards] Cosmetic %d cannot be worn by %N's class.", item, client);
        return Plugin_Stop;
    }
    char className[32], key[80], model[PLATFORM_MAX_PATH];
    BotCards_ClassKey(playerClass, className, sizeof(className));
    if (StrEqual(className, "heavyweapons")) strcopy(className, sizeof(className), "heavy");
    Format(key, sizeof(key), "model_player_per_class/%s", className);
    if (!TF2Econ_GetItemDefinitionString(item, key, model, sizeof(model)))
    {
        if (!TF2Econ_GetItemDefinitionString(item, "model_player", model, sizeof(model)))
        {
            LogError("[AP cards] Cosmetic %d has no wearable model for %N.", item, client);
            return Plugin_Stop;
        }
    }
    if (!IsModelPrecached(model)) PrecacheModel(model);

    // Only remove cosmetics, never weapon wearables (e.g. shields) or the
    // private AP buff provider. TF2Utils enumerates actual equipped wearables.
    for (int i = TF2Util_GetPlayerWearableCount(client) - 1; i >= 0; i--)
    {
        int worn = TF2Util_GetPlayerWearable(client, i);
        if (worn <= MaxClients || !IsValidEntity(worn)
            || !HasEntProp(worn, Prop_Send, "m_iItemDefinitionIndex")) continue;
        int wornItem = GetEntProp(worn, Prop_Send, "m_iItemDefinitionIndex");
        if (TF2Econ_GetItemLoadoutSlot(wornItem, playerClass) == miscSlot)
        {
            TF2_RemoveWearable(client, worn);
        }
    }
    int hat = CreateEntityByName("tf_wearable");
    if (hat <= MaxClients || !IsValidEntity(hat))
    {
        LogError("[AP cards] Could not create cosmetic %d for %N.", item, client);
        return Plugin_Stop;
    }
    int effect = g_CardUnusual[client] ? g_CardParticle[client] : 0;
    SetEntProp(hat, Prop_Send, "m_iItemDefinitionIndex", item);
    SetEntProp(hat, Prop_Send, "m_bInitialized", 1);
    SetEntProp(hat, Prop_Send, "m_iEntityQuality", effect > 0 ? 5 : 6);
    SetEntProp(hat, Prop_Send, "m_iEntityLevel", 1);
    SetEntProp(hat, Prop_Send, "m_iTeamNum", GetClientTeam(client));
    SetEntityModel(hat, model);
    DispatchSpawn(hat);
    if (effect > 0) TF2Attrib_SetByDefIndex(hat, 134, float(effect));
    TF2Util_SetWearableAlwaysValid(hat, true);
    g_CardHat[client] = EntIndexToEntRef(hat);
    TF2Util_EquipPlayerWearable(client, hat);
    g_CardParticle[client] = effect;
    LogMessage("[AP cards] %N equipped cosmetic %d with fixed Unusual effect %d", client, item, effect);
    return Plugin_Stop;
}

// The curated Spy card equips Big Earner, whose -25 max health is still part
// of the real loadout after the tier bonus. Detect the item itself so swapping
// that melee in the loadout editor also updates the card's live max HP.
int BotCards_LoadoutHealthDelta(int client)
{
    int melee = GetPlayerWeaponSlot(client, 2);
    if (melee > MaxClients && IsValidEntity(melee)
        && HasEntProp(melee, Prop_Send, "m_iItemDefinitionIndex")
        && GetEntProp(melee, Prop_Send, "m_iItemDefinitionIndex") == 461)
    {
        return -25;
    }
    return 0;
}

int BotCards_BaseHealth(TFClassType classType)
{
    switch (classType)
    {
        case TFClass_Scout, TFClass_Engineer, TFClass_Sniper, TFClass_Spy: return 125;
        case TFClass_Soldier: return 200;
        case TFClass_DemoMan, TFClass_Pyro: return 175;
        case TFClass_Heavy: return 300;
        case TFClass_Medic: return 150;
    }
    return 125;
}

// Compose's admin has a writable community overlay but only a read-only game
// mount. It renders bot files there, then asks the game process to install
// them synchronously before any lineup commands run. No Docker socket or map
// restart is needed, and only these two fixed config filenames are copied.
bool BotCards_CopyStagedFile(const char[] name)
{
    char source[PLATFORM_MAX_PATH], target[PLATFORM_MAX_PATH], temporary[PLATFORM_MAX_PATH];
    // SourceMod treats an absolute path outside the game tree as relative
    // unless it carries the documented file:// escape hatch.
    Format(source, sizeof(source),
        "file:///opt/tf2-community-pack/tf/addons/sourcemod/configs/defenderbots/%s", name);
    if (StrEqual(name, "loadout.cfg") && !FileExists(source))
    {
        // A completely stock team has no managed loadout file.
        return true;
    }
    BuildPath(Path_SM, target, sizeof(target), "configs/defenderbots/%s", name);
    Format(temporary, sizeof(temporary), "%s.tf2ap-new", target);
    File input = OpenFile(source, "rb");
    if (input == null)
    {
        LogError("Bot card sync: cannot open staged %s", source);
        return false;
    }
    File output = OpenFile(temporary, "wb");
    if (output == null)
    {
        LogError("Bot card sync: cannot open temporary %s", temporary);
        delete input;
        return false;
    }
    int bytes[256], count;
    bool copied = true;
    while ((count = input.Read(bytes, sizeof(bytes), 1)) > 0)
    {
        if (!output.Write(bytes, count, 1))
        {
            copied = false;
            break;
        }
    }
    if (count < 0) copied = false;
    delete input;
    delete output;
    if (!copied || !RenameFile(target, temporary))
    {
        LogError("Bot card sync: cannot copy or rename %s to %s (read count %d, copied %d)",
            temporary, target, count, copied);
        DeleteFile(temporary);
        return false;
    }
    return true;
}

public Action Command_BotCardsSync(int client, int args)
{
    if (!BotCards_CopyStagedFile("bot_names.txt")
        || !BotCards_CopyStagedFile("loadout.cfg"))
    {
        ReplyToCommand(client, "[AP cards] Bot file sync failed; live team unchanged.");
        return Plugin_Handled;
    }
    ReplyToCommand(client, "[AP cards] Bot file sync OK.");
    return Plugin_Handled;
}

void BotCards_Evict(int bot)
{
    // A departing Engineer must not leave an ownerless sentry behind.
    for (int i = TF2Util_GetPlayerObjectCount(bot) - 1; i >= 0; i--)
    {
        int building = TF2Util_GetPlayerObject(bot, i);
        if (building > MaxClients && IsValidEntity(building)) RemoveEntity(building);
    }
    KickClient(bot, "AP bot card updated");
}

public Action Command_BotCardsEvict(int client, int args)
{
    if (args != 1)
    {
        ReplyToCommand(client, "Usage: sm_ap_botcards_evict <exact bot name>");
        return Plugin_Handled;
    }
    char wanted[128], current[128];
    GetCmdArg(1, wanted, sizeof(wanted));
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (!IsClientInGame(bot) || !IsFakeClient(bot) || GetClientTeam(bot) != 2) continue;
        GetClientName(bot, current, sizeof(current));
        if (!StrEqual(current, wanted, false)) continue;
        BotCards_Evict(bot);
        ReplyToCommand(client, "[AP cards] Replacing %s now.", wanted);
        return Plugin_Handled;
    }
    ReplyToCommand(client, "[AP cards] %s is not on RED; nothing to replace.", wanted);
    return Plugin_Handled;
}

public Action Command_BotCardsEvictUnselected(int client, int args)
{
    int onRed;
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (IsClientInGame(bot) && GetClientTeam(bot) == 2 && !IsClientSourceTV(bot)) onRed++;
    }
    ConVar size = FindConVar("sm_redbots_manager_defender_team_size");
    if (size == null || onRed < size.IntValue)
    {
        ReplyToCommand(client, "[AP cards] RED has an open seat; no bot evicted.");
        return Plugin_Handled;
    }
    char path[PLATFORM_MAX_PATH];
    BuildPath(Path_SM, path, sizeof(path), "configs/defenderbots/loadout.cfg");
    KeyValues lineup = new KeyValues("loadout");
    if (!lineup.ImportFromFile(path))
    {
        delete lineup;
        ReplyToCommand(client, "[AP cards] Cannot read the new lineup; no bot evicted.");
        return Plugin_Handled;
    }
    int lowest;
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (!IsClientInGame(bot) || !IsFakeClient(bot) || GetClientTeam(bot) != 2) continue;
        char botName[128];
        GetClientName(bot, botName, sizeof(botName));
        bool selected;
        lineup.Rewind();
        if (lineup.JumpToKey("seats"))
        {
            for (int seat = 1; seat <= size.IntValue; seat++)
            {
                char key[8], name[128];
                IntToString(seat, key, sizeof(key));
                if (!lineup.JumpToKey(key)) continue;
                lineup.GetString("name", name, sizeof(name));
                lineup.GoBack();
                if (name[0] != '\0' && StrEqual(name, botName, false))
                {
                    selected = true;
                    break;
                }
            }
        }
        if (!selected) lowest = bot;
    }
    if (lowest > 0)
    {
        char name[128];
        GetClientName(lowest, name, sizeof(name));
        BotCards_Evict(lowest);
        ReplyToCommand(client, "[AP cards] Freed unselected bot %s for the new card.", name);
    }
    else
    {
        ReplyToCommand(client, "[AP cards] Every bot is selected; no extra eviction.");
    }
    delete lineup;
    return Plugin_Handled;
}

// A card's priority in the new file. Name and class must both match: a card
// moved to a different class needs a new bot, not an old bot renamed in place.
int BotCards_CardPriority(KeyValues lineup, int bot, int teamSize)
{
    char current[128], actualClass[32];
    GetClientName(bot, current, sizeof(current));
    BotCards_ClassKey(TF2_GetPlayerClass(bot), actualClass, sizeof(actualClass));
    for (int seat = 1; seat <= teamSize; seat++)
    {
        char key[8], name[128], wantedClass[32];
        IntToString(seat, key, sizeof(key));
        lineup.Rewind();
        if (!lineup.JumpToKey("seats") || !lineup.JumpToKey(key)) continue;
        lineup.GetString("name", name, sizeof(name));
        lineup.GetString("class", wantedClass, sizeof(wantedClass));
        if (lineup.GetNum("card", 0) == 1 && StrEqual(name, current, false)
            && StrEqual(wantedClass, actualClass, false))
        {
            lineup.Rewind();
            return seat;
        }
    }
    lineup.Rewind();
    return 0;
}

// Reloading the file rebinds surviving bots, but does not itself make a bot
// below the human-adjusted priority cutoff yield to a newly higher card.
public Action Command_BotCardsReconcile(int client, int args)
{
    ConVar size = FindConVar("sm_redbots_manager_defender_team_size");
    if (size == null)
    {
        ReplyToCommand(client, "[AP cards] Defender manager is unavailable; cards unchanged.");
        return Plugin_Handled;
    }
    char path[PLATFORM_MAX_PATH];
    BuildPath(Path_SM, path, sizeof(path), "configs/defenderbots/loadout.cfg");
    KeyValues lineup = new KeyValues("loadout");
    if (!lineup.ImportFromFile(path))
    {
        delete lineup;
        ReplyToCommand(client, "[AP cards] Cannot read the new lineup; cards unchanged.");
        return Plugin_Handled;
    }
    int humans, bots;
    for (int player = 1; player <= MaxClients; player++)
    {
        if (!IsClientInGame(player) || GetClientTeam(player) != 2) continue;
        if (IsFakeClient(player)) bots++;
        else humans++;
    }
    int capacity = size.IntValue - humans;
    if (capacity < 0) capacity = 0;
    int desired;
    for (int seat = 1; seat <= capacity; seat++)
    {
        char key[8], name[128];
        IntToString(seat, key, sizeof(key));
        lineup.Rewind();
        if (!lineup.JumpToKey("seats") || !lineup.JumpToKey(key)) continue;
        lineup.GetString("name", name, sizeof(name));
        if (lineup.GetNum("card", 0) == 1 && name[0] != '\0') desired++;
    }
    int present;
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (!IsClientInGame(bot) || !IsFakeClient(bot) || GetClientTeam(bot) != 2) continue;
        int rank = BotCards_CardPriority(lineup, bot, size.IntValue);
        if (rank > 0 && rank <= capacity) present++;
    }
    int toEvict = desired - present - (capacity - bots);
    int evicted;
    bool chosen[MAXPLAYERS + 1];
    while (toEvict > 0)
    {
        int victim, lowest = -1;
        for (int bot = 1; bot <= MaxClients; bot++)
        {
            if (chosen[bot] || !IsClientInGame(bot) || !IsFakeClient(bot)
                || GetClientTeam(bot) != 2) continue;
            int rank = BotCards_CardPriority(lineup, bot, size.IntValue);
            if (rank > 0 && rank <= capacity) continue;
            if (rank == 0) rank = size.IntValue + 1;
            if (rank > lowest)
            {
                victim = bot;
                lowest = rank;
            }
        }
        if (victim == 0) break;
        chosen[victim] = true;
        BotCards_Evict(victim);
        evicted++;
        toEvict--;
    }
    delete lineup;
    ReplyToCommand(client, "[AP cards] Priority reconciled: %d bot(s) replaced for %d live card seat(s).",
        evicted, desired);
    return Plugin_Handled;
}

public Action Command_BotCards(int client, int args)
{
    int cards;
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (!IsClientInGame(bot) || !g_CardSelected[bot])
        {
            continue;
        }
        int innates;
        for (int i = 0; i < 3; i++)
        {
            if (g_CardStacks[bot][i] > 0)
            {
                innates++;
            }
        }
        int maximum = HasEntProp(bot, Prop_Data, "m_iMaxHealth")
            ? GetEntProp(bot, Prop_Data, "m_iMaxHealth") : GetClientHealth(bot);
        int hat = EntRefToEntIndex(g_CardHat[bot]);
        bool worn;
        if (hat > MaxClients && IsValidEntity(hat))
        {
            for (int index = 0; index < TF2Util_GetPlayerWearableCount(bot); index++)
            {
                if (TF2Util_GetPlayerWearable(bot, index) == hat) worn = true;
            }
        }
        char className[32];
        BotCards_ClassKey(TF2_GetPlayerClass(bot), className, sizeof(className));
        ReplyToCommand(client, "[AP cards] %N: %s, tier %d, %s, cosmetic %d (%s, effect %d), %d innate(s), %d active, %d max HP, scale %.2f",
            bot, className, g_CardTier[bot], g_CardGiant[bot] ? "Robot Giant" : g_CardRobot[bot] ? "Robot" : "Human",
            g_CardCosmetic[bot], worn ? "worn" : "missing", g_CardParticle[bot],
            innates, WeaponBuffs_CardInnatesOnActive(bot), maximum,
            GetEntPropFloat(bot, Prop_Send, "m_flModelScale"));
        cards++;
    }
    if (cards == 0)
    {
        ReplyToCommand(client, "[AP cards] No selected RED bot cards are spawned.");
    }
    return Plugin_Handled;
}
