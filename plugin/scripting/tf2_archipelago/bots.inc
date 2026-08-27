/**
 * The defender bots, as this plugin sees them.
 *
 * The bots belong to OfficerSpy's mod. What lives here is what a run needs
 * from them and the mod does not do: a bot steps aside for a human, and what
 * the bots buy can be read in the chat, since the game no longer lets a
 * player inspect a teammate's upgrades.
 */

ConVar g_CvarBotUpgradesChat;
ConVar g_CvarBotsWaitForPlayers;
ConVar g_CvarBotsBackfill;

// Whether the hold is on, so it is said once rather than twice a second.
bool g_Holding;

// How often the ready hold runs. The mod sets a bot ready from several places
// and the hold takes it away again, so it is a poll rather than a gate: one
// gate per call site is one call site away from being missed.
#define BotReadyInterval 0.5

// How often the team is topped up between waves. Slower than the ready hold:
// this one asks the mod for bots, and a seat that is empty for three seconds
// costs nothing.
#define BotFillInterval 3.0

// How long the mod is given to add its lineup before the seats are counted.
// Its own add is one call inside the wave-begin handler, so this only has to
// outlast the bots landing.
#define BotTrimDelay 3.0

// The upgrade names, in the order scripts/items/mvm_upgrades.txt lists the
// entries the game actually loads. That order is the index MVM_Upgrade carries:
// checked against CMannVsMachineUpgradeManager with sm_dump_upgrades in
// tf2-mvm-bots, 63 entries, identical from 0 to 62.
//
// Identical only once the commented-out entries are dropped. The file has 64
// attribute lines and "heal rate bonus" is one of them, commented out, so
// counting every line named everything from index 19 on after the upgrade
// before it.
ArrayList g_UpgradeNames;

// When the next fill may ask, so that one ask is not made five times over.
float g_NextFill;

/* How long an ask is given to become a player before another one is made.
 *
 * A bot asked for is not a bot on RED: sm_addbots connects one, which takes a
 * couple of seconds, and nothing on the team changes until it lands. The count
 * below reads the team, so without this the next poll asked for the same empty
 * seats again. The poll is the ready hold's, twice a second, and a round held
 * by an engineer who could not finish his nest gave it minutes to do it in: a
 * six man RED came up ten, and the server ran out of slots to make robots with.
 *
 * Five seconds is longer than a bot takes to connect and shorter than anybody
 * waits for a seat to be filled.
 */
#define BotFillGrace 5.0

// Wide enough for any line of mvm_upgrades.txt.
#define UpgradeLineMax 256

// What tf_mvm_defenders_team_size says when the game will not.
#define DefenderTeamSizeDefault 6

void Bots_Init()
{
    g_CvarBotUpgradesChat = CreateConVar("tf2ap_bot_upgrades_chat", "0",
        "Write what the defender bots buy at the upgrade station to the chat.", _, true, 0.0, true, 1.0);
    g_CvarBotsWaitForPlayers = CreateConVar("tf2ap_bots_wait_for_players", "1",
        "Hold the bots unready until every player on RED is ready.", _, true, 0.0, true, 1.0);
    g_CvarBotsBackfill = CreateConVar("tf2ap_bots_backfill", "1",
        "Put a bot back on RED when a player leaves between waves.", _, true, 0.0, true, 1.0);
    g_UpgradeNames = new ArrayList(ByteCountToCells(64));

    // No TIMER_FLAG_NO_MAPCHANGE on either: they must outlive every map change.
    CreateTimer(BotReadyInterval, Timer_HoldBotsReady, _, TIMER_REPEAT);
    CreateTimer(BotFillInterval, Timer_FillTeam, _, TIMER_REPEAT);
}

void Bots_OnMapStart()
{
    Bots_LoadUpgradeNames();

    // The clock this is measured against restarts with the map, so a deadline
    // left over from a long one would hold the fill off on the next.
    g_NextFill = 0.0;
}

// Every "attribute" line the file means, in file order. Read through the game's
// own file system, because the file lives inside a VPK rather than on disk.
static void Bots_LoadUpgradeNames()
{
    g_UpgradeNames.Clear();
    File file = OpenFile("scripts/items/mvm_upgrades.txt", "r", true);
    if (file == null)
    {
        AP_Debug("The plugin cannot read mvm_upgrades.txt. Bot purchases show as numbers.");
        return;
    }
    char line[UpgradeLineMax];
    while (file.ReadLine(line, sizeof(line)))
    {
        StripComment(line);
        char key[32], value[64];
        if (!SplitQuoted(line, key, sizeof(key), value, sizeof(value)))
        {
            continue;
        }
        if (StrEqual(key, "attribute", false))
        {
            g_UpgradeNames.PushString(value);
        }
    }
    delete file;
    AP_Debug("The plugin read %d upgrade names.", g_UpgradeNames.Length);
}

// One KeyValues line with its comment cut off. A // outside quotes starts one,
// and the file uses both kinds: a whole entry commented out line by line, and a
// note after a value.
static void StripComment(char[] line)
{
    bool quoted = false;
    for (int index = 0; line[index] != '\0'; index++)
    {
        if (line[index] == '"')
        {
            quoted = !quoted;
        }
        else if (!quoted && line[index] == '/' && line[index + 1] == '/')
        {
            line[index] = '\0';
            return;
        }
    }
}

// "key"   "value" from one KeyValues line. False for anything else.
static bool SplitQuoted(const char[] line, char[] key, int keyMax, char[] value, int valueMax)
{
    int quotes[4];
    int found = 0;
    for (int index = 0; line[index] != '\0' && found < 4; index++)
    {
        if (line[index] == '"')
        {
            quotes[found++] = index;
        }
    }
    if (found < 4)
    {
        return false;
    }
    int keyLength = quotes[1] - quotes[0];
    int valueLength = quotes[3] - quotes[2];
    strcopy(key, keyLength < keyMax ? keyLength : keyMax, line[quotes[0] + 1]);
    strcopy(value, valueLength < valueMax ? valueLength : valueMax, line[quotes[2] + 1]);
    return true;
}

/* Every purchase and every sale at the upgrade station.
 *
 * The game handles it; this only says what it was. For the mod's bots that is
 * the chat, because the game no longer lets a player inspect a teammate's
 * upgrades. For players it is the log and nothing else, because a player can
 * see their own station: what it is there for is the question nobody can
 * answer afterwards, which is whether a sale the station appeared to accept
 * ever reached the game at all.
 */
public void OnClientCommandKeyValues_Post(int client, KeyValues kv)
{
    if (client <= 0 || client > MaxClients)
    {
        return;
    }
    char name[32];
    kv.GetSectionName(name, sizeof(name));

    if (!StrEqual(name, "MVM_Upgrade"))
    {
        // The money side wants the refund as much as this wants the purchase.
        MvM_OnCommandKeyValues(client, name, 0);
        return;
    }
    if (!kv.JumpToKey("upgrade"))
    {
        MvM_OnCommandKeyValues(client, name, 0);
        return;
    }
    int slot = kv.GetNum("itemslot", -1);
    int index = kv.GetNum("upgrade", -1);
    int count = kv.GetNum("count", 1);
    kv.GoBack();

    /* The count is read before the money side is told, because its sign is the
     * only thing that separates a sale from a refund. The minus button on the
     * station sends this same command with a negative count. */
    MvM_OnCommandKeyValues(client, name, count);

    // The mod asks for zero charges when a bot's canteen is already full. The
    // game does nothing with it, so neither does anything here.
    if (count == 0)
    {
        return;
    }

    char upgrade[64];
    UpgradeLabel(index, upgrade, sizeof(upgrade));

    /* The credits held afterwards, for a bot as readily as for a player.
     *
     * A play-test showed one medic asking for the same upgrade twenty-five
     * times in one mission, ten of them inside thirty seconds. The line said
     * what was asked for and not what it cost, so nothing in the log said
     * whether the game was charging for those or refusing them at their
     * ceiling, and those are different bugs.
     *
     * The same reasoning as the sale a player makes: what a station shows and
     * what the game does are two things, and only the second moves money.
     */
    int credits = HasEntProp(client, Prop_Send, "m_nCurrency")
        ? GetEntProp(client, Prop_Send, "m_nCurrency") : -1;

    if (!IsFakeClient(client))
    {
        AP_Debug("%N %s %s (%s) x%d, and holds %d credits.",
            client, count < 0 ? "sold" : "bought", upgrade, SlotLabel(slot), count < 0 ? -count : count,
            credits);
        return;
    }
    // The log takes every bot purchase whatever the chat is set to: the chat is
    // one line per tier and unreadable during a wave, and the log is what a
    // debug bundle carries.
    AP_Debug("%N %s %s (%s) x%d, and holds %d credits.",
        client, count < 0 ? "sold" : "bought", upgrade, SlotLabel(slot), count < 0 ? -count : count,
        credits);
    if (!g_CvarBotUpgradesChat.BoolValue)
    {
        return;
    }
    // The count is what the chat used to leave out. The mod buys several tiers
    // in one command, so one line stood for up to four purchases and the credits
    // it spent looked like they went nowhere.
    AP_Announce("%N %s %s (%s) x%d.",
        client, count < 0 ? "sold" : "bought", upgrade, SlotLabel(slot), count < 0 ? -count : count);
}

/* The name of an upgrade, and the number behind it either way.
 *
 * The name comes from the order mvm_upgrades.txt lists its attributes, and that
 * order has been checked against the game's own upgrade manager. The number
 * stays in the line anyway: the next update to the file is free to add an entry
 * or comment one out, and a name that does not match the upgrade a player can
 * see on the weapon is then one glance away from saying so.
 */
static void UpgradeLabel(int index, char[] buffer, int maxlength)
{
    if (index < 0 || index >= g_UpgradeNames.Length)
    {
        FormatEx(buffer, maxlength, "upgrade %d", index);
        return;
    }
    char name[64];
    g_UpgradeNames.GetString(index, name, sizeof(name));
    FormatEx(buffer, maxlength, "%s [%d]", name, index);
}

// The loadout slot MVM_Upgrade carries, which is not the weapon slot: the
// canteen is slot 9, and slot 4 is the Engineer's buildings and the Spy's
// sapper. Everything above three used to read as a canteen, which is how an
// Engineer buying sentry damage filled the chat with canteens.
static char[] SlotLabel(int slot)
{
    char label[16];
    switch (slot)
    {
        case -1: strcopy(label, sizeof(label), "player");
        case 0: strcopy(label, sizeof(label), "primary");
        case 1: strcopy(label, sizeof(label), "secondary");
        case 2: strcopy(label, sizeof(label), "melee");
        case 3: strcopy(label, sizeof(label), "utility");
        case 4: strcopy(label, sizeof(label), "building");
        case 5: strcopy(label, sizeof(label), "pda");
        case 6: strcopy(label, sizeof(label), "watch");
        case 9: strcopy(label, sizeof(label), "canteen");
        default: FormatEx(label, sizeof(label), "slot %d", slot);
    }
    return label;
}

// A human arrived and RED is full of bots. One bot leaves so the human has a
// seat: the mod fills the team but never empties it, and the game refuses a
// player a full team. The mod tops the team back up when a wave begins.
void Bots_MakeRoom()
{
    if (!MvM_IsActive())
    {
        return;
    }
    int size = DefenderTeamSize();
    int onRed = 0;
    int bot = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client) || GetClientTeam(client) != TeamRed)
        {
            continue;
        }
        onRed++;
        if (IsFakeClient(client))
        {
            bot = client;
        }
    }
    if (onRed < size || bot == 0)
    {
        return;
    }
    AP_Debug("RED is full at %d. A bot leaves for the player who arrived.", onRed);
    KickClient(bot, "Making room for a player");
}

// How many seats RED holds. The game's own number, and six when it will not say.
static int DefenderTeamSize()
{
    ConVar teamSize = FindConVar("tf_mvm_defenders_team_size");
    if (teamSize != null && teamSize.IntValue > 0)
    {
        return teamSize.IntValue;
    }
    return DefenderTeamSizeDefault;
}

// UNVERIFIED m_bPlayerReady on the game rules, which is what the F4 sets and
// what the mod itself reads. Absent, nothing can be held, so nothing is.
static bool IsReady(int client)
{
    if (FindSendPropInfo("CTFGameRulesProxy", "m_bPlayerReady") == -1)
    {
        return true;
    }
    return GameRules_GetProp("m_bPlayerReady", 1, client) != 0;
}

/**
 * The bots do not start the wave. The players do.
 *
 * A bot readies up as soon as it has done its own job, which on a team of one
 * player and five bots means the wave begins while that player is still in the
 * upgrade station. So a bot that is ready is made unready again for as long as
 * any player on RED is not.
 *
 * Nobody on RED is the same answer for a stronger reason: an empty server
 * would otherwise play the run by itself, wave after wave, and check off
 * locations nobody was there for.
 *
 * A poll and not a command listener. The mod lets its own bots through that
 * listener on purpose, and it sets them ready from four places; taking the
 * ready away wherever it came from is one rule instead of four.
 */
public Action Timer_HoldBotsReady(Handle timer)
{
    if (!g_CvarBotsWaitForPlayers.BoolValue || !MvM_IsActive())
    {
        return Plugin_Continue;
    }
    // A wave that is already running has nothing left to hold.
    if (GameRules_GetRoundState() != RoundState_BetweenRounds)
    {
        return Plugin_Continue;
    }
    if (Bots_PlayersReady())
    {
        if (g_Holding)
        {
            g_Holding = false;
            AP_Debug("Every player on RED is ready. The bots are let go.");
        }
        return Plugin_Continue;
    }
    // Edge-triggered: this runs twice a second, and a line each time would be
    // the only thing in the log by the end of the evening.
    if (!g_Holding)
    {
        g_Holding = true;
        AP_Debug("The bots are held unready: %d player(s) on RED, not all ready.", Bots_PlayerCount());
    }
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client) || !IsFakeClient(client)
            || GetClientTeam(client) != TeamRed || !IsReady(client))
        {
            continue;
        }
        // The same command the F4 sends, so the game does its own bookkeeping.
        FakeClientCommand(client, "tournament_player_readystate 0");
    }
    return Plugin_Continue;
}

// How many players are on RED, bots excluded.
static int Bots_PlayerCount()
{
    int players = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (MvM_IsPlayer(client))
        {
            players++;
        }
    }
    return players;
}

// Whether every player on RED is ready. False for an empty RED: no player is
// not the same as every player.
static bool Bots_PlayersReady()
{
    int players = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!MvM_IsPlayer(client))
        {
            continue;
        }
        players++;
        if (!IsReady(client))
        {
            return false;
        }
    }
    return players > 0;
}

// Whether the client leaving was a player. Read in _Post, where the client is
// already gone and nothing can be asked about it any more.
bool g_Leaving[MAXPLAYERS + 1];

void Bots_OnClientLeaving(int client)
{
    g_Leaving[client] = IsClientInGame(client) && !IsFakeClient(client);
}

void Bots_OnClientLeft(int client)
{
    if (!g_Leaving[client])
    {
        return;
    }
    g_Leaving[client] = false;
    Bots_Fill();
}

/**
 * Fill RED, between waves.
 *
 * The mod fills RED from its own mvm_begin_wave handler and tops it up every
 * second while the wave runs. Between waves it does neither, which leaves two
 * holes this closes.
 *
 * A player who leaves during the upgrade station leaves a wave balanced for six
 * to five bodies. And, before the first wave of all, the bots were not there at
 * all: they arrived at the moment the wave did, having never seen the upgrade
 * station, so wave one was played by six bots with stock weapons, no upgrades
 * and no sentry. The engineer had nothing built because he had not existed
 * long enough to build it.
 *
 * Only between waves, for the same reason both times: during a wave the mod is
 * already doing it, and asking twice puts two bots in one seat.
 *
 * Safe to do early only because of the two rules above: a bot that turns up
 * before the players is held unready until they are ready, and never readies at
 * all with nobody on RED. Without those, filling the team early would mean the
 * bots starting wave one by themselves while the first player was still
 * choosing a class.
 */
static void Bots_Fill()
{
    if (!g_CvarBotsBackfill.BoolValue || !MvM_IsActive())
    {
        return;
    }
    if (GameRules_GetRoundState() != RoundState_BetweenRounds)
    {
        return;
    }
    // Mode 0 leaves the bots to an admin, and the mod being absent leaves them
    // to nobody. Neither wants this plugin adding any.
    ConVar mode = FindConVar("sm_redbots_manager_mode");
    if (mode == null || mode.IntValue == 0)
    {
        return;
    }
    // One ask can add more than it asked for, so the surplus goes on this poll
    // rather than three seconds after the next wave begins.
    Bots_TrimSurplus();

    /* A seat is taken by whoever is on RED, and held for every player who is
     * connected and not on it yet.
     *
     * Without the second half this fights Bots_MakeRoom: that frees a seat the
     * moment a player connects, and a fill three seconds later put a bot back
     * in it before the player had finished choosing a class. The game then
     * refuses them RED for being full, which is the door this plugin exists to
     * hold open.
     *
     * A player who sits in spectator all evening holds a seat by this rule.
     * That is the right way round: the mod's own top-up fills it once the wave
     * starts, and until then the seat is theirs to take.
     */
    int taken = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client) || IsClientSourceTV(client))
        {
            continue;
        }
        if (GetClientTeam(client) == TeamRed || !IsFakeClient(client))
        {
            taken++;
        }
    }
    int missing = DefenderTeamSize() - taken;
    if (missing <= 0)
    {
        return;
    }
    // The last ask has not had time to arrive, and asking again would double it
    if (GetGameTime() < g_NextFill)
    {
        return;
    }
    AP_Debug("RED holds %d of %d between waves. %d bot(s) take the empty seats.",
        taken, DefenderTeamSize(), missing);
    ServerCommand("sm_addbots %d", missing);
    g_NextFill = GetGameTime() + BotFillGrace;

    /* And count the seats back once the mod has had its turn
     *
     * The trim above only ever saw the previous poll's surplus. An ask that
     * added six where five were wanted survived until the next poll, and the
     * poll returns early once the wave starts, so it survived the whole wave
     * in a seat that belongs to a player. */
    CreateTimer(BotTrimDelay, Timer_TrimBots);
}

/* RED came up over its team size, so the extra bots leave.
 *
 * Two things overfill it. The mod adds its whole chosen lineup when a wave
 * begins, without counting who is already standing there. This plugin fills the
 * team before that, so a six man RED came up at ten or twelve.
 *
 * And one ask can add more than it asked for. sm_addbots fills the seats the
 * mod's named team wants. When that team does not name every seat, the mod
 * falls back to the lineup mode and adds the whole count again. Three named
 * seats and an ask for six is nine bots.
 *
 * The seats are counted back after the mod has had its turn rather than the
 * fill being taken out again, because either side adding without counting ends
 * the same way, and this is the one rule that covers both. It is also where
 * Bots_MakeRoom already lives: whose seat it is, is this plugin's question.
 *
 * Bots only, newest first. A human is never in the surplus.
 */
void Bots_TrimSurplus()
{
    if (!MvM_IsActive())
    {
        return;
    }
    /* The same seat arithmetic the fill uses, or the two fight
     *
     * Bots_Fill counts a connected player as holding a seat wherever he is
     * standing, so it will not add a bot for one. The trim counted RED alone,
     * so it left RED full of bots while a player who had just reconnected sat
     * in spectator and the game refused him the team. That is the second half
     * of what was reported as "what is with the 7 players". */
    int size = DefenderTeamSize() - Bots_SeatsHeldOffRed();
    if (size < 0)
    {
        size = 0;
    }
    int onRed = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (IsClientInGame(client) && GetClientTeam(client) == TeamRed && !IsClientSourceTV(client))
        {
            onRed++;
        }
    }
    // Highest index last in, near enough: the game hands out the lowest free
    // slot, so the bots that arrived last hold the highest numbers.
    for (int client = MaxClients; client >= 1 && onRed > size; client--)
    {
        if (!IsClientInGame(client) || !IsFakeClient(client) || IsClientSourceTV(client)
            || GetClientTeam(client) != TeamRed)
        {
            continue;
        }
        AP_Debug("RED holds %d of %d. %N leaves.", onRed, size, client);
        KickClient(client, "RED is full");
        onRed--;
    }
}

/* Seats belonging to players who are connected but not on RED yet.
 *
 * Someone choosing a class, someone the game has just put in spectator, someone
 * who reconnected mid-mission. Bots_Fill already holds a seat for each of them,
 * so this is the same count and the trim can hold the same ones.
 */
static int Bots_SeatsHeldOffRed()
{
    int held = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client) || IsClientSourceTV(client) || IsFakeClient(client))
        {
            continue;
        }
        if (GetClientTeam(client) != TeamRed)
        {
            held++;
        }
    }
    return held;
}

/* Whether anybody is here to play with.
 *
 * Any human on the server, not only one on RED: a player who has connected and
 * is still looking at the class menu is the moment the team should be forming,
 * which is the whole point of filling it early.
 */
static bool Bots_AnyPlayerHere()
{
    for (int client = 1; client <= MaxClients; client++)
    {
        if (IsClientInGame(client) && !IsFakeClient(client) && !IsClientSourceTV(client))
        {
            return true;
        }
    }
    return false;
}

/* The team forms while the players are still in the upgrade station.
 *
 * Runs on the same poll as the ready hold rather than on a player joining: a
 * bot that is kicked to make room for that player, a mission that changes, and
 * a wave that ends are all moments the team is short, and one poll covers them
 * without a hook for each.
 */
public Action Timer_TrimBots(Handle timer)
{
    Bots_TrimSurplus();
    Bots_LogLineup();
    return Plugin_Stop;
}

/* Who is on RED, and what they are playing.
 *
 * The one line a debug bundle could not answer. A blacklisted Spy was drawn
 * into a lineup for a whole evening and the logs held nothing to prove it: the
 * player had to send a screenshot of the team panel, because bot class is
 * printed nowhere. Reading a bundle should not need a photograph.
 *
 * Written when the wave starts and the team has settled, which is after the
 * trim above: RED before that is whatever the mod and this plugin have each
 * added without seeing the other.
 */
void Bots_LogLineup()
{
    char line[512];
    int count = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client) || IsClientSourceTV(client) || GetClientTeam(client) != TeamRed)
        {
            continue;
        }
        TFClassType class = TF2_GetPlayerClass(client);
        char key[16];
        if (class > TFClass_Unknown && view_as<int>(class) < sizeof(g_ClassKeys))
        {
            strcopy(key, sizeof(key), g_ClassKeys[view_as<int>(class)]);
        }
        else
        {
            strcopy(key, sizeof(key), "none");
        }
        Format(line, sizeof(line), "%s%s%N (%s%s)", line, count == 0 ? "" : ", ",
            client, key, IsFakeClient(client) ? " bot" : " player");
        count++;
    }
    if (count == 0)
    {
        AP_Debug("RED is empty at the wave start.");
        return;
    }
    AP_Debug("RED holds %d at the wave start: %s", count, line);
}

public Action Timer_FillTeam(Handle timer)
{
    if (!g_CvarBotsBackfill.BoolValue || !Bots_AnyPlayerHere())
    {
        return Plugin_Continue;
    }
    Bots_Fill();
    return Plugin_Continue;
}
