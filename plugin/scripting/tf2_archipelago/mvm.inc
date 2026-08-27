/**
 * Reading the game.
 *
 * The entity properties and event fields here are UNVERIFIED: what the
 * community documents, checked against nothing but a compiler.
 *
 * So nothing is read without asking whether it exists. A missing property is a
 * warning and a fallback, never a native error: SourceMod turns a bad property
 * name into a runtime error on every call, burying the console mid-wave.
 */

// Players are on RED in Mann vs Machine; BLU is the robots, and they are bots.
#define TeamRed 2
#define TeamBlue 3

// Keeps each missing-property complaint to one line rather than one per frame.
StringMap g_WarnedProps;

// The bundles handed to each player that the game's own record never saw.
// Read in full where it is written back, below.
static int g_BundleCredits[MAXPLAYERS + 1];

void MvM_Init()
{
    g_WarnedProps = new StringMap();
}

static bool HasProp(int entity, const char[] property)
{
    if (entity > 0 && HasEntProp(entity, Prop_Send, property))
    {
        return true;
    }
    bool warned;
    if (!g_WarnedProps.GetValue(property, warned))
    {
        g_WarnedProps.SetValue(property, true);
        AP_Error("This server has no %s. The plugin works without it.", property);
    }
    return false;
}

static int FindObjectiveResource()
{
    return FindEntityByClassname(-1, "tf_objective_resource");
}

// The game rules property is the real answer; the mvm_ map prefix is the fallback if it is absent.
stock bool MvM_IsActive()
{
    if (FindSendPropInfo("CTFGameRulesProxy", "m_bPlayingMannVsMachine") != -1)
    {
        return GameRules_GetProp("m_bPlayingMannVsMachine") != 0;
    }
    char map[PLATFORM_MAX_PATH];
    GetCurrentMap(map, sizeof(map));
    return strncmp(map, "mvm_", 4, false) == 0;
}

/* Everybody holding the bomb line, human or defender bot.
 *
 * Separate from MvM_IsPlayer on purpose, and the two are not interchangeable.
 * MvM_IsPlayer answers "is this somebody the run belongs to", which is what
 * unlock enforcement and the ready gate want, and its !IsFakeClient is what
 * leaves a defender bot's loadout alone. This one answers "is this somebody who
 * fights for RED", which is what being paid depends on.
 *
 * Widening MvM_IsPlayer instead would have paid the bots and stripped their
 * weapons in the same edit.
 */
stock bool MvM_IsDefender(int client)
{
    return client > 0 && client <= MaxClients
        && IsClientInGame(client)
        && GetClientTeam(client) == TeamRed;
}

// Humans on RED only: the robots are bots on BLU, and stripping one would take the wave apart.
stock bool MvM_IsPlayer(int client)
{
    return client > 0 && client <= MaxClients
        && IsClientInGame(client)
        && !IsFakeClient(client)
        && GetClientTeam(client) == TeamRed;
}

// UNVERIFIED m_bIsMiniBoss on CTFPlayer, which is what the game calls a giant.
// The defender bot mod reads the same property to pick its targets, so it is
// the community's answer rather than only ours.
//
// Robots are fake clients on BLU. A human is never a giant, and neither is an
// allied bot on RED, so the team is checked before the property is.
stock bool MvM_IsGiant(int client)
{
    if (client <= 0 || client > MaxClients || !IsClientInGame(client)
        || !IsFakeClient(client) || GetClientTeam(client) != TeamBlue)
    {
        return false;
    }
    if (!HasProp(client, "m_bIsMiniBoss"))
    {
        return false;
    }
    if (GetEntProp(client, Prop_Send, "m_bIsMiniBoss") == 0)
    {
        return false;
    }
    return !MvM_IsSentryBuster(client);
}

/* A sentry buster carries m_bIsMiniBoss and is not a giant.
 *
 * It usually dies by blowing itself up, so without this the mission's giant
 * check is spent by a robot nobody killed. Worse quietly than loudly:
 * g_GiantReported latches for the rest of the mission, so the real giant that
 * dies later reports nothing and no line anywhere says it was missed.
 *
 * The model is the test the defender bot mod already uses, in its own
 * source/redbots3/util.sp. Same string, and it needs no gamedata.
 *
 * A buster killed by the team is excluded too, rather than only one that
 * detonates. It is one rule instead of two, and it matches what a player means
 * by killing a giant.
 */
stock bool MvM_IsSentryBuster(int client)
{
    char model[PLATFORM_MAX_PATH];
    if (!GetClientModel(client, model, sizeof(model)))
    {
        return false;
    }
    return StrEqual(model, "models/bots/demo/bot_sentry_buster.mdl");
}

// UNVERIFIED m_iszMvMPopfileName: scripts/population/mvm_decoy.pop, trimmed to mvm_decoy.
stock bool MvM_PopFile(char[] buffer, int maxlength)
{
    buffer[0] = '\0';

    int resource = FindObjectiveResource();
    if (resource != -1 && HasProp(resource, "m_iszMvMPopfileName"))
    {
        char raw[PLATFORM_MAX_PATH];
        if (GetEntPropString(resource, Prop_Send, "m_iszMvMPopfileName", raw, sizeof(raw)) > 0)
        {
            TrimPopFileName(raw, sizeof(raw));
            strcopy(buffer, maxlength, raw);
        }
    }
    if (buffer[0] == '\0')
    {
        GetCurrentMap(buffer, maxlength);
    }
    return buffer[0] != '\0';
}

static void TrimPopFileName(char[] name, int maxlength)
{
    int lastSlash = FindCharInString(name, '/', true);
    if (lastSlash == -1)
    {
        lastSlash = FindCharInString(name, '\\', true);
    }
    if (lastSlash != -1)
    {
        strcopy(name, maxlength, name[lastSlash + 1]);
    }
    int extension = FindCharInString(name, '.', true);
    if (extension != -1)
    {
        name[extension] = '\0';
    }
}

// UNVERIFIED m_nMannVsMachineWaveCount: the cross-check, mvm_begin_wave is the primary source.
stock int MvM_WaveFromGame()
{
    int resource = FindObjectiveResource();
    if (resource == -1 || !HasProp(resource, "m_nMannVsMachineWaveCount"))
    {
        return 0;
    }
    return GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveCount");
}

// UNVERIFIED m_nMannVsMachineMaxWaveCount. Zero when the game will not say.
stock int MvM_MaxWavesFromGame()
{
    int resource = FindObjectiveResource();
    if (resource == -1 || !HasProp(resource, "m_nMannVsMachineMaxWaveCount"))
    {
        return 0;
    }
    return GetEntProp(resource, Prop_Send, "m_nMannVsMachineMaxWaveCount");
}

// Whether credits paid now would survive. A wave that is lost takes the team
// back to where the wave began, and money paid into a wave goes back with it,
// which is what a play-test saw a Cash Bundle do. Between waves is where the
// upgrade station is anyway, so nothing is lost by waiting for it.
//
// Nobody on RED is the other unpayable moment: a bundle that arrives with an
// empty server used to pay nobody and was gone.
stock bool MvM_CanPayCredits()
{
    if (!MvM_IsActive() || GameRules_GetRoundState() != RoundState_BetweenRounds)
    {
        return false;
    }
    for (int client = 1; client <= MaxClients; client++)
    {
        // Alive, because the money arrives as a pack to walk over and a dead
        // player walks nowhere. Between waves everybody is alive; the check is
        // for the moment after a wave is lost and before the respawn.
        if (MvM_IsPlayer(client) && IsPlayerAlive(client))
        {
            return true;
        }
    }
    return false;
}

/**
 * Pay a bundle into the run.
 *
 * Credits are per player, so a bundle pays each of them fully. Returns how many
 * players it paid, zero for none: the caller holds the grant back rather than
 * acknowledging a payment nobody got.
 *
 * The money goes in as a currency pack the player picks up, which is the path
 * the game itself uses when a robot drops cash. Writing m_nCurrency is the
 * obvious way and it is the wrong one: it puts a number on the player that the
 * game's own bookkeeping never saw, and two things read that bookkeeping rather
 * than the number.
 *
 * The refund button is one. It does not hand back what was spent, it restores
 * the balance the game recorded at wave start: 1200 on screen, 400 recorded,
 * press refund and 800 of bundles are gone. Losing a wave is the other, and it
 * restores the same record, which is where the negative balances came from.
 *
 * A pack is collected by walking over it, so it is dropped at the player's own
 * feet and only between waves, where they are standing at the upgrade station
 * anyway. m_bDistributed stays off: that flag means the team has already been
 * paid, and a pack carrying it pays nobody.
 */
stock int MvM_GrantCredits(int amount)
{
    if (amount <= 0 || !MvM_IsActive())
    {
        return 0;
    }
    int paid = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        // Defenders, not players. Five of the six RED seats are usually bots,
        // and a bundle that pays only the humans leaves the team that has to
        // hold the bomb unable to buy anything with it.
        if (!MvM_IsDefender(client) || !IsPlayerAlive(client))
        {
            continue;
        }
        if (MvM_DropCredits(client, amount))
        {
            paid++;
        }
    }
    if (paid > 0)
    {
        AP_Announce("The run received %d credits for %d defender(s).", amount, paid);
    }
    return paid;
}

/* One pack, at one player's feet, or the property write that always worked.
 *
 * The pack is the right shape and it has not worked yet. m_nAmount is not a
 * send property on this server and not a data property either: the first
 * spelling threw eighty times in ten minutes and left a dead pack behind each
 * time, and the second says so once a bundle and pays nobody.
 *
 * So the property write is back, and the pack is tried first only while
 * something says which name to use. Writing m_nCurrency is what has always
 * paid, and it is what the refund and the wave-loss restore do not know about:
 * that is TODO item 1, still open, and a bundle nobody receives is worse than
 * a bundle the refund forgets.
 */
static bool MvM_DropCredits(int client, int amount)
{
    if (MvM_PackCredits(client, amount))
    {
        return true;
    }
    if (!HasProp(client, "m_nCurrency"))
    {
        return false;
    }
    int held = GetEntProp(client, Prop_Send, "m_nCurrency");
    SetEntProp(client, Prop_Send, "m_nCurrency", held + amount);
    g_BundleCredits[client] += amount;
    return true;
}

/* What the game does not know it paid.
 *
 * A bundle written straight onto m_nCurrency is money the game's own record of
 * the wave never saw, and the refund button does not hand back what was spent:
 * it restores that record. So the bundles vanish, which is the bug a play-test
 * found by spending 1200 and refunding into 400.
 *
 * The ledger is per client and it only grows, because the record the game
 * restores never contains a single credit of ours. Correct balance after any
 * restore is therefore the game's number plus everything we ever added. It is
 * cleared on a map change, where the game clears its own side too.
 */
// Everything a bundle ever added to this client, which is what the game's own
// record of the wave is missing. Read across the plugin boundary by the defender
// bot mod, so a bot that respawns is not reset to a number with no bundles in it.
stock int MvM_BundleCredits(int client)
{
    return (client >= 0 && client <= MAXPLAYERS) ? g_BundleCredits[client] : 0;
}

stock void MvM_ForgetBundles(int client)
{
    g_BundleCredits[client] = 0;
}

stock void MvM_OnMapStart()
{
    for (int client = 0; client <= MaxClients; client++)
    {
        g_BundleCredits[client] = 0;
    }
}

/* Every command the upgrade station sends, so a refund can be put right.
 *
 * The name is the one thing here that is a guess. MVM_Respec is what the
 * community documents the refund button as sending; anything else the station
 * sends is written to the log once, which is how the next bundle says the
 * real name if this one is wrong.
 */
stock void MvM_OnCommandKeyValues(int client, const char[] name)
{
    if (StrEqual(name, "MvM_UpgradesBegin", false))
    {
        WeaponBuffs_ShowMenu(client);
        return;
    }
    if (StrEqual(name, "MVM_Respec", false))
    {
        RequestFrame(Frame_RestoreBundles, GetClientUserId(client));
        WeaponBuffs_ApplyNextFrame(client);
        return;
    }
    if (StrEqual(name, "MVM_Upgrade", false))
    {
        WeaponBuffs_ApplyNextFrame(client);
        return;
    }
    bool seen;
    if (!g_WarnedProps.GetValue(name, seen))
    {
        g_WarnedProps.SetValue(name, true);
        AP_Debug("The station sent %s, which nothing here handles.", name);
    }
}

/* A lost wave restores the same record the refund does, and to everyone.
 *
 * The balance can come back negative there: the game restores what it recorded
 * at wave start, and the bundles that were spent on top of it were never in
 * that number, so the subtraction runs past zero. Same ledger, same frame's
 * delay, and a balance that would still be negative is put at zero rather than
 * left as a number no upgrade station can spend.
 */
stock void MvM_RestoreBundlesAll()
{
    for (int client = 1; client <= MaxClients; client++)
    {
        // Whoever was paid is whoever is owed, so this walks defenders too.
        if (MvM_IsDefender(client))
        {
            RequestFrame(Frame_RestoreBundles, GetClientUserId(client));
        }
    }
}

// A frame later, because the game writes the restored balance during the
// command and a number put back before that is the number it overwrites.
static void Frame_RestoreBundles(any userid)
{
    int client = GetClientOfUserId(userid);
    if (client <= 0 || !IsClientInGame(client))
    {
        return;
    }
    if (!HasProp(client, "m_nCurrency"))
    {
        return;
    }
    int held = GetEntProp(client, Prop_Send, "m_nCurrency");
    int owed = held + g_BundleCredits[client];
    if (owed < 0)
    {
        owed = 0;
    }
    if (owed == held)
    {
        return;
    }
    SetEntProp(client, Prop_Send, "m_nCurrency", owed);
    AP_Debug("%N came back to %d credits, and %d of bundles put that at %d.",
        client, held, g_BundleCredits[client], owed);
}

// The names a currency pack's amount might go by, newest guess first. Checked
// rather than assumed, because two of them have already been wrong.
static const char PackAmountProps[][] = {"m_nAmount", "m_nCurrencyAmount", "m_iAmount"};

/* MvM_PackCredits hands the money over as a pack the player walks into, which
 * is how the game itself pays. False when this server's packs cannot be told
 * what they are worth, which is every server so far.
 *
 * Said once per map rather than once per bundle. A line that repeats every few
 * seconds is a line nobody reads, and it drowns the log a bundle carries.
 */
static bool g_PackAmountChecked;
static int g_PackAmountProp = -1;

static bool MvM_PackCredits(int client, int amount)
{
    int pack = CreateEntityByName("item_currencypack_custom");
    if (pack == -1)
    {
        return false;
    }
    if (!g_PackAmountChecked)
    {
        g_PackAmountChecked = true;
        for (int index = 0; index < sizeof(PackAmountProps); index++)
        {
            if (HasEntProp(pack, Prop_Data, PackAmountProps[index])
                || HasEntProp(pack, Prop_Send, PackAmountProps[index]))
            {
                g_PackAmountProp = index;
                break;
            }
        }
        if (g_PackAmountProp == -1)
        {
            AP_Debug("A currency pack here has none of the amount properties. Bundles pay the player directly.");
        }
        else
        {
            AP_Debug("A currency pack here carries %s.", PackAmountProps[g_PackAmountProp]);
        }
    }
    if (g_PackAmountProp == -1)
    {
        RemoveEntity(pack);
        return false;
    }

    bool sent = HasEntProp(pack, Prop_Send, PackAmountProps[g_PackAmountProp]);
    SetEntProp(pack, sent ? Prop_Send : Prop_Data, PackAmountProps[g_PackAmountProp], amount);
    DispatchSpawn(pack);
    if (!IsValidEntity(pack))
    {
        // Nothing to remove: a spawn that failed took the entity with it.
        return false;
    }

    // At the feet and a little above them, so it lands rather than falling
    // through whatever the player is standing on.
    float origin[3];
    GetClientAbsOrigin(client, origin);
    origin[2] += 8.0;
    TeleportEntity(pack, origin, NULL_VECTOR, NULL_VECTOR);
    AP_Debug("A pack worth %d credits is at %N's feet.", amount, client);
    return true;
}

// Everyone on RED, bots included. The kill is the whole effect: no wave-failed
// event is fired, and an undefended hatch is what loses the wave.
stock int MvM_KillTeam()
{
    int killed = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (IsClientInGame(client) && GetClientTeam(client) == TeamRed && IsPlayerAlive(client))
        {
            ForcePlayerSuicide(client);
            killed++;
        }
    }
    return killed;
}
