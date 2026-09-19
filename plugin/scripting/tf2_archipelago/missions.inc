/**
 * Which mission the server plays.
 *
 * The run decides, not the map cycle. A mission is playable once the run holds
 * its ticket; the server starts on one, refuses to sit on one it does not
 * hold, and chains from a cleared mission to the next one the run unlocked.
 * The bridge lists the missions and says which are cleared; the unlock set
 * says which are unlocked, and it is the live copy that counts here because
 * a ticket can land in the middle of a map.
 */

// The mission the server starts on, as a popfile name. Empty leaves the loaded
// map's own mission, and the guard below still moves off it if the run does
// not hold it.
ConVar g_CvarStartMission;

// How long after a mission clear the next one loads. Long enough for the
// victory sequence, short enough to beat the game's own cycle: a live server
// followed mapcycle.txt to cp_dustbowl fifty-two seconds after a mission
// clear. Zero leaves the game to cycle on its own, and the guard catches up at
// the next map start.
ConVar g_CvarNextMissionDelay;

// Consulted once: the start mission is where the evening begins, not where
// every map change goes back to.
bool g_StartMissionApplied;

// The mission asked for while the map was still changing. tf_mvm_popfile acts
// on the map that is loaded, so a switch across maps has to wait for the new
// one before it can name the mission.
char g_PendingPopFile[64];

// What the plugin chose after a mission clear, kept until it is playing. The
// game may change the level first; the map start then loads this rather than
// whatever the game's cycle picked.
char g_PlannedPopFile[64];

// Long enough after a map change for the mission to be loadable. tf_mvm_popfile
// reloads the mission in place, so it cannot be run while the map still is.
#define PopFileDelay 2.0

/* The mission the plugin last asked the game for, and until when it is still
 * waiting for it.
 *
 * A switch is not instant. A map change is a load, and the mission the map
 * comes up with is its own default until tf_mvm_popfile lands two seconds
 * later. The unlock set and the mission list, meanwhile, are re-fetched at
 * every map start and answer off loopback in milliseconds.
 *
 * So the guard below used to run while the switch was still in the air, read
 * the map's default mission as the mission the run is on, find it is not in the
 * run, and move the server to the first unlocked mission of the list: the
 * switch that was asked for was undone by the plugin that was making it, every
 * time, which is why only tf2ap_start_mission appeared to work. That one is
 * written into server.cfg and is already loaded before any of this runs.
 *
 * GetEngineTime and not GetGameTime: game time restarts at zero with the map,
 * and this deadline has to survive exactly that.
 */
char g_SwitchingPopFile[64];
float g_SwitchingUntil;

// Popfiles this server tried to read and could not, by name. Session-lived on
// purpose: installing the mod the file needs means a restart anyway.
StringMap g_RefusedPopFiles;

// How long to wait for a switch before believing it will not happen. A map
// change plus the popfile that follows it is a few seconds; a minute is the
// point at which something else went wrong and the guard should have the
// server back.
#define SwitchGrace 60.0

void Missions_Init()
{
    g_RefusedPopFiles = new StringMap();
    Missions_HookPopulationParse();
    g_CvarStartMission = CreateConVar("tf2ap_start_mission", "",
        "The mission the server starts on, as a popfile name. Blank keeps the map's own.");
    g_CvarNextMissionDelay = CreateConVar("tf2ap_next_mission_delay", "30",
        "Seconds between a mission clear and the next mission loading. 0 leaves it to the game.",
        _, true, 0.0, true, 300.0);
}

void Missions_OnConfigsExecuted()
{
    if (g_PendingPopFile[0] != '\0')
    {
        CreateTimer(PopFileDelay, Timer_ApplyPendingPopFile);
    }
}

/* Ask TF2 for a mission using the map-relative name its mission picker uses.
 *
 * The run and bridge keep the complete popfile name because it is the stable
 * mission id. The game command accepts that name too, except SigMod's mission
 * vote guard mistakes every command in the first ten seconds of a map for a
 * vote. Its allowed list contains `int_name`, not `mvm_map_int_name`, so the
 * complete name is rejected during exactly the two-second window in which a
 * cross-map switch lands.
 *
 * A relative name is TF2's normal spelling for this command and works with or
 * without SigMod. The complete id remains untouched everywhere else.
 */
static void Missions_ApplyPopFile(const char[] popFile, const char[] map)
{
    char commandName[64];
    if (StrEqual(popFile, CaliginousCaperID, false))
    {
        strcopy(commandName, sizeof(commandName), "ap_caliginous_caper");
    }
    else if (StrEqual(popFile, map, false))
    {
        strcopy(commandName, sizeof(commandName), "normal");
    }
    else
    {
        char prefix[64];
        FormatEx(prefix, sizeof(prefix), "%s_", map);
        int prefixLength = strlen(prefix);
        if (StrContains(popFile, prefix, false) == 0)
        {
            strcopy(commandName, sizeof(commandName), popFile[prefixLength]);
        }
        else
        {
            strcopy(commandName, sizeof(commandName), popFile);
        }
    }
    ServerCommand("tf_mvm_popfile %s", commandName);
}

void Missions_ReloadCurrentForModifiers()
{
    char popFile[64];
    char map[64];
    MvM_PopFile(popFile, sizeof(popFile));
    GetCurrentMap(map, sizeof(map));
    Missions_ApplyPopFile(popFile, map);
}

// tf_mvm_popfile reloads the mission on the map that is already loaded, so the
// map change goes first and this lands after it.
static Action Timer_ApplyPendingPopFile(Handle timer)
{
    if (g_PendingPopFile[0] == '\0')
    {
        return Plugin_Stop;
    }
    AP_Debug("The plugin names the mission %s now that the map is up.", g_PendingPopFile);
    char currentMap[64];
    GetCurrentMap(currentMap, sizeof(currentMap));
    Mission pending;
    if (Missions_Find(g_PendingPopFile, pending))
    {
        MissionModifiers_Apply(pending.popFile, pending.modifierFlags);
    }
    Missions_ApplyPopFile(g_PendingPopFile, currentMap);
    g_PendingPopFile[0] = '\0';
    return Plugin_Stop;
}

static bool Missions_Find(const char[] popFile, Mission mission)
{
    for (int index = 0; index < Bridge_MissionCount(); index++)
    {
        if (Bridge_MissionAt(index, mission) && StrEqual(mission.popFile, popFile, false))
        {
            return true;
        }
    }
    return false;
}

// Unlocked, by the live set: a ticket granted since the list was fetched counts.
static bool Missions_Playable(Mission mission)
{
    return mission.unlocked || Unlocks_HasMission(mission.popFile);
}

// The next mission after `current`, in the order the seed drew them: the first
// unlocked one that is not cleared, else the first unlocked one to play again.
// `fresh` says which of the two it was.
static bool Missions_Next(const char[] current, Mission next, bool &fresh)
{
    // A replay of another mission beats a replay of this one, which beats nothing.
    Mission replay;
    int replayRank = 0;
    for (int index = 0; index < Bridge_MissionCount(); index++)
    {
        Mission candidate;
        if (!Bridge_MissionAt(index, candidate) || !Missions_Playable(candidate))
        {
            continue;
        }
        // A mission whose file this server does not have would load the map and
        // never start a wave, so it is not a mission to move anybody to.
        if (!Missions_PopFileInstalled(candidate.popFile)
            || Missions_PopFileRefused(candidate.popFile))
        {
            continue;
        }
        bool isCurrent = StrEqual(candidate.popFile, current, false);
        if (!candidate.cleared && !isCurrent)
        {
            next = candidate;
            fresh = true;
            return true;
        }
        int rank = isCurrent ? 1 : 2;
        if (rank > replayRank)
        {
            replay = candidate;
            replayRank = rank;
        }
    }
    if (replayRank == 0)
    {
        return false;
    }
    next = replay;
    fresh = false;
    return true;
}

// Runs whenever the plugin learns something about the run: the mission list
// or the unlock set. Both have to be in hand, or a run that unlocked
// everything looks like one that unlocked nothing.
void Missions_OnListUpdated()
{
    if (Bridge_MissionCount() == 0 || !g_HaveUnlocks)
    {
        return;
    }
    char current[64];
    MvM_PopFile(current, sizeof(current));
    bool playingMvM = MvM_IsActive();

    // Nothing here may second-guess a switch that has not landed yet.
    if (Missions_SwitchInFlight(current))
    {
        return;
    }

    if (g_PlannedPopFile[0] != '\0')
    {
        if (StrEqual(g_PlannedPopFile, current, false))
        {
            g_PlannedPopFile[0] = '\0';
        }
        else if (Missions_LoadNamed(g_PlannedPopFile, "The plugin loads the mission it planned"))
        {
            return;
        }
    }

    if (!g_StartMissionApplied)
    {
        g_StartMissionApplied = true;
        char wanted[64];
        g_CvarStartMission.GetString(wanted, sizeof(wanted));
        if (wanted[0] != '\0' && !StrEqual(wanted, current, false)
            && Missions_LoadNamed(wanted, "The server starts on the mission it was set to"))
        {
            return;
        }
    }

    Mission loaded;
    bool inRun = playingMvM && Missions_Find(current, loaded);
    if (inRun && Missions_Playable(loaded))
    {
        MissionModifiers_Apply(loaded.popFile, loaded.modifierFlags);
        if (MissionModifiers_ShouldReloadBotSurgeMission())
        {
            AP_Debug("Bot Surge reloads the initial mission to update its support waves.");
            Missions_ReloadCurrentForModifiers();
        }
        return;
    }
    char why[96];
    if (inRun)
    {
        FormatEx(why, sizeof(why), "The run did not unlock %s", loaded.name);
    }
    else if (playingMvM)
    {
        FormatEx(why, sizeof(why), "%s is not part of the run", current);
    }
    else
    {
        strcopy(why, sizeof(why), "This map is not part of the run");
    }
    Mission next;
    bool fresh;
    if (!Missions_Next(current, next, fresh))
    {
        AP_Error("%s, and the run holds no playable mission yet.", why);
        return;
    }
    AP_Announce("%s. The server moves to %s.", why, next.name);
    Missions_Load(next);
}

// Whether the plugin is still waiting for the mission it asked for.
//
// Arriving clears it. So does the grace running out, because a switch that
// never lands must not leave the guard switched off for the evening: past it
// the server goes back to being moved off a mission the run does not hold.
static bool Missions_SwitchInFlight(const char[] current)
{
    if (g_SwitchingPopFile[0] == '\0')
    {
        return false;
    }
    if (StrEqual(g_SwitchingPopFile, current, false))
    {
        AP_Debug("The mission %s is loaded.", g_SwitchingPopFile);
        g_SwitchingPopFile[0] = '\0';
        return false;
    }
    if (GetEngineTime() > g_SwitchingUntil)
    {
        AP_Error("The mission %s never loaded. The server is on %s.", g_SwitchingPopFile, current);
        g_SwitchingPopFile[0] = '\0';
        return false;
    }
    AP_Debug("The plugin is still waiting for %s, and the game says %s.", g_SwitchingPopFile, current);
    return true;
}

// Loads a mission of the run by popfile name if the run holds it and unlocked
// it. Says why not otherwise. `why` is the debug line's reason.
static bool Missions_LoadNamed(const char[] popFile, const char[] why)
{
    Mission mission;
    if (!Missions_Find(popFile, mission))
    {
        AP_Error("The mission %s is not in this run.", popFile);
        return false;
    }
    if (!Missions_Playable(mission))
    {
        AP_Error("The run did not unlock %s. The server stays where it is.", mission.name);
        return false;
    }
    AP_Debug("%s: %s.", why, mission.name);
    Missions_Load(mission);
    return true;
}

// The team cleared the mission that is loaded. Chain to the next one, after a
// pause the scoreboard fits in.
void Missions_OnMissionCleared(const char[] popFile)
{
    float delay = g_CvarNextMissionDelay.FloatValue;
    if (delay <= 0.0)
    {
        return;
    }
    Mission next;
    bool fresh;
    if (!Missions_Next(popFile, next, fresh))
    {
        return;
    }
    if (fresh)
    {
        AP_Announce("Next: %s on %s, in %d seconds.", next.name, next.map, RoundFloat(delay));
    }
    else
    {
        AP_Announce("Every unlocked mission is cleared. %s loads again in %d seconds, until a ticket opens another.",
            next.name, RoundFloat(delay));
    }
    strcopy(g_PlannedPopFile, sizeof(g_PlannedPopFile), next.popFile);
    // Survives the map change on purpose: the game may cycle first, and the
    // plan has to outlive that.
    CreateTimer(delay, Timer_LoadPlanned);
}

static Action Timer_LoadPlanned(Handle timer)
{
    if (g_PlannedPopFile[0] == '\0')
    {
        return Plugin_Stop;
    }
    char current[64];
    MvM_PopFile(current, sizeof(current));
    if (StrEqual(g_PlannedPopFile, current, false) && MvM_IsActive())
    {
        g_PlannedPopFile[0] = '\0';
        return Plugin_Stop;
    }
    Mission mission;
    if (Missions_Find(g_PlannedPopFile, mission))
    {
        Missions_Load(mission);
    }
    return Plugin_Stop;
}

// tf_mvm_popfile is a command, not a variable, and it acts on the map that is
// loaded: it reloads the mission in place. So the map only changes when it has
// to, and naming the mission waits for the new map when it does.
/* Whether this server has the mission's file at all.
 *
 * TF2 ignores tf_mvm_popfile naming a file it cannot find, and keeps the map's
 * own mission instead: the map loads, no wave ever starts, and the team sits on
 * wave 0 with nothing saying why. Reported twice by Doctor Pyro, once with a
 * bundle, and the log said only that the plugin was "still waiting for
 * mvm_rottenburg_rev_int_breaking_walls" while "the game says mvm_rottenburg".
 *
 * A community mission is the way this happens: the seed draws one and the pack
 * carrying its file was never installed on the server playing it.
 */
static bool Missions_PopFileInstalled(const char[] popFile)
{
    char path[PLATFORM_MAX_PATH];
    if (StrEqual(popFile, CaliginousCaperID, false))
    {
        Format(path, sizeof(path), "scripts/population/%s.pop", CaliginousCaperRuntimePop);
    }
    else
    {
        Format(path, sizeof(path), "scripts/population/%s.pop", popFile);
    }
    return FileExists(path, true);
}

/* A file the server has and the game will not read.
 *
 * CPopulationManager::Parse returns false when it meets a populator it does not
 * know, and the whole file is dropped: the map stays up with no waves in it and
 * the team sits on wave 0. Doctor Pyro reported it on Breaking Walls, whose file
 * opens with ItemBlacklist, a SigMod populator that stock TF2 rejects. The only
 * thing the plugin said was that a wave jump "never took", which is the symptom.
 *
 * The engine prints `Invalid populator '<name>'` immediately before it gives up,
 * so the console already names what it refused; what was missing was anything
 * tying that to the mission, and any reason not to sit there.
 */
static void Missions_HookPopulationParse()
{
    GameData data = LoadGameConfigFile("tf2_archipelago");
    if (data == null)
    {
        return;
    }
    DynamicDetour parse = DynamicDetour.FromConf(data, "CPopulationManager::Parse");
    if (parse == null || !parse.Enable(Hook_Post, Missions_PopulationParsed))
    {
        // Debug, not an error: the signature is Linux-only, so every Windows
        // server would carry this line on every map for a report it was never
        // going to get. Those servers keep the old silent wave 0 until the
        // Windows signature exists.
        AP_Debug("A refused mission file will not be reported: CPopulationManager::Parse could not be hooked.");
    }
    delete parse;
    delete data;
}

public MRESReturn Missions_PopulationParsed(Address populationManager, DHookReturn result)
{
    if (result.Value)
    {
        return MRES_Ignored;
    }
    char popFile[64];
    if (!MvM_PopFile(popFile, sizeof(popFile)) || popFile[0] == '\0')
    {
        return MRES_Ignored;
    }
    // Not here: this runs inside the parse that is failing, and the answer to
    // it is a changelevel. Next frame, once the game has finished giving up.
    if (g_RefusedPopFiles != null)
    {
        int seen;
        if (g_RefusedPopFiles.GetValue(popFile, seen))
        {
            return MRES_Ignored;
        }
        g_RefusedPopFiles.SetValue(popFile, 1);
        DataPack pack = new DataPack();
        pack.WriteString(popFile);
        RequestFrame(Missions_ReportRefused, pack);
    }
    return MRES_Ignored;
}

public void Missions_ReportRefused(DataPack pack)
{
    char popFile[64];
    pack.Reset();
    pack.ReadString(popFile, sizeof(popFile));
    delete pack;
    Missions_NoteRefused(popFile);
}

/* Say it once per file, and keep it out of the pool for the rest of the session.
 *
 * Once, because the game re-parses on every wave restart and the same failure
 * would otherwise scroll past a dozen times; the hook records it before asking
 * for this. Out of the pool, because a mission this server cannot read is not
 * one to move anybody to next either.
 */
static void Missions_NoteRefused(const char[] popFile)
{
    char name[64];
    Mission mission;
    if (Missions_Find(popFile, mission))
    {
        strcopy(name, sizeof(name), mission.name);
    }
    else
    {
        strcopy(name, sizeof(name), popFile);
    }
    // Several short lines rather than one long one: AP_Error cuts at
    // ChatLineMax, and the first draft of this lost its own last sentence.
    AP_Error("%s cannot be played here: the server refused %s.pop, so this map has no waves.",
        name, popFile);
    AP_Error("The console line above names the populator it did not know.");
    if (FindConVar("sig_pop_popmgr_extensions") == null)
    {
        AP_Error("SigMod is not loaded here, and that is where those populators come from.");
        AP_Error("Turn SigMod on in the launcher, or leave %s out of the run.", name);
    }
    Missions_LeaveRefused(popFile);
}

/* Move off the mission that was refused, if the run has somewhere else to go.
 *
 * Reported rather than silent is the point of this, but leaving the team on a
 * map with no waves is still leaving them stuck, and the next mission is chosen
 * the same way a clear chooses one.
 */
static void Missions_LeaveRefused(const char[] popFile)
{
    Mission next;
    bool fresh;
    if (!Missions_Next(popFile, next, fresh))
    {
        AP_Error("Nothing else in the run is playable, so the server stays here.");
        AP_Error("Pick one with sm_ap_mission once there is one.");
        return;
    }
    AP_Announce("Moving on to %s.", next.name);
    Missions_Load(next);
}

static bool Missions_PopFileRefused(const char[] popFile)
{
    int seen;
    return g_RefusedPopFiles != null && g_RefusedPopFiles.GetValue(popFile, seen);
}

static void Missions_Load(Mission mission)
{
    if (Missions_PopFileRefused(mission.popFile))
    {
        AP_Error("%s cannot be played here: this server already refused its population file once.", mission.name);
        return;
    }
    if (!Missions_PopFileInstalled(mission.popFile))
    {
        AP_Error("%s cannot be played here: %s.pop is not on this server, so the game would keep the map's own mission and never start a wave. Install the community pack that carries it, or leave the mission out of the run.",
            mission.name, mission.popFile);
        return;
    }

    LogMessage("mission switched to %s (%s) on %s", mission.name, mission.popFile, mission.map);

    // Before either branch: both are asking for a mission that is not loaded
    // yet, and the guard has to know that for as long as it takes.
    strcopy(g_SwitchingPopFile, sizeof(g_SwitchingPopFile), mission.popFile);
    g_SwitchingUntil = GetEngineTime() + SwitchGrace;

    MissionModifiers_Apply(mission.popFile, mission.modifierFlags);

    char current[64];
    GetCurrentMap(current, sizeof(current));
    if (StrEqual(current, mission.map, false) && MvM_IsActive())
    {
        AP_Announce("Mission: %s. Reloading.", mission.name);
        Missions_ApplyPopFile(mission.popFile, mission.map);
        return;
    }
    AP_Announce("Mission: %s. The map changes now.", mission.name);
    strcopy(g_PendingPopFile, sizeof(g_PendingPopFile), mission.popFile);
    ServerCommand("changelevel %s", mission.map);
}

void Missions_List(int client)
{
    int count = Bridge_MissionCount();
    if (count == 0)
    {
        AP_PrintToClient(client, "The plugin has no mission list yet. The bridge has not answered.");
        return;
    }

    char current[64];
    MvM_PopFile(current, sizeof(current));

    AP_PrintToClient(client, "The run holds %d mission(s). Switch with \x07FFD700!mission <number>\x01.", count);
    for (int index = 0; index < count; index++)
    {
        Mission mission;
        if (!Bridge_MissionAt(index, mission))
        {
            continue;
        }
        AP_PrintToClient(client, "%d. %s (%s), %d waves%s%s%s",
            index + 1, mission.name, mission.popFile, mission.waves,
            Missions_PopFileRefused(mission.popFile) ? " [refused by this server]"
                : (Missions_PopFileInstalled(mission.popFile) ? (Missions_Playable(mission) ? "" : " [locked]") : " [not installed]"),
            mission.cleared ? " [cleared]" : "",
            StrEqual(mission.popFile, current) ? " [playing]" : "");
    }
}

// Takes a number from the list or a pop file name, and refuses a mission the
// run has not unlocked: playing one out of turn is what the tickets exist to
// stop.
void Missions_Switch(int client, const char[] choice)
{
    int count = Bridge_MissionCount();
    if (count == 0)
    {
        AP_PrintToClient(client, "The plugin has no mission list yet. The bridge has not answered.");
        return;
    }

    Mission mission;
    bool found = false;
    int number = StringToInt(choice);
    if (number >= 1 && number <= count)
    {
        found = Bridge_MissionAt(number - 1, mission);
    }
    else
    {
        found = Missions_Find(choice, mission);
    }
    if (!found)
    {
        AP_PrintToClient(client, "No mission %s in this run. Type !mission for the list.", choice);
        return;
    }
    if (!Missions_Playable(mission))
    {
        AP_PrintToClient(client, "The run did not unlock %s yet. Its ticket is somewhere in the multiworld.", mission.name);
        return;
    }
    g_PlannedPopFile[0] = '\0';
    Missions_Load(mission);
}

// One line for the status: how many the run holds, unlocked and cleared.
void Missions_Summary(char[] buffer, int maxlength)
{
    int count = Bridge_MissionCount();
    if (count == 0)
    {
        strcopy(buffer, maxlength, "no mission list from the bridge yet");
        return;
    }
    int unlocked = 0, cleared = 0;
    for (int index = 0; index < count; index++)
    {
        Mission mission;
        if (!Bridge_MissionAt(index, mission))
        {
            continue;
        }
        if (Missions_Playable(mission))
        {
            unlocked++;
        }
        if (mission.cleared)
        {
            cleared++;
        }
    }
    FormatEx(buffer, maxlength, "%d unlocked of %d, %d cleared", unlocked, count, cleared);
}

/* Where the team was when the server last stopped.
 *
 * The bridge moves this record forward all evening and serves it on every
 * mission list, so it is a restore only at the first answer after a load. Read
 * again at the third map change it would drag the team off the mission they
 * are playing.
 */
char g_ResumePopFile[64];
int g_ResumeWave;
bool g_ResumeConsulted;
int g_ResumeTicks;

/* The wave the jump asked for, and how long to keep watching for it.
 *
 * The jump used to be told and never checked. It said "Restored: Quarry at
 * wave 3 of 6" and a player answered "im in the first wave again", and nothing
 * in the log could tell the two apart. Now the announcement waits until the
 * game agrees, and says so when it never does.
 */
int g_ResumeTarget;
int g_ResumeSettleTicks;
#define ResumeSettleTick 1.0
#define ResumeSettleTicksMax 30

// How long to keep waiting for the mission to come up before giving the
// restore up. A changelevel plus the popfile behind it is a few seconds; past
// two minutes something else went wrong and the team is better left alone.
#define ResumeTick 1.0
#define ResumeTicksMax 120

/* What the bridge says the team had cleared, from the mission list answer.
 *
 * Acted on once per plugin load. The mission is loaded here, before the guard
 * above gets a look at the list: that way g_SwitchingPopFile is already set and
 * the guard leaves the restore alone instead of moving the server elsewhere
 * first.
 */
void Missions_NoteResume(const char[] popFile, int wave)
{
    if (g_ResumeConsulted || popFile[0] == '\0' || wave <= 0)
    {
        return;
    }
    g_ResumeConsulted = true;

    Mission mission;
    if (!Missions_Find(popFile, mission) || !Missions_Playable(mission))
    {
        AP_Error("The run cannot play %s any more, so there is nothing to go back to.", popFile);
        return;
    }
    if (wave >= mission.waves)
    {
        AP_Debug("The record has %s cleared to wave %d of %d, which leaves no wave to go back to.",
            popFile, wave, mission.waves);
        return;
    }

    strcopy(g_ResumePopFile, sizeof(g_ResumePopFile), popFile);
    g_ResumeWave = wave;
    CreateTimer(ResumeTick, Timer_Resume, _, TIMER_REPEAT);

    char current[64];
    MvM_PopFile(current, sizeof(current));
    if (MvM_IsActive() && StrEqual(current, popFile, false))
    {
        return;
    }
    // The start mission is where the evening begins, and this is not a fresh
    // evening: the record wins over it.
    g_StartMissionApplied = true;
    g_PlannedPopFile[0] = '\0';
    AP_Announce("The server came back. %s loads again, at the wave the team had reached.", mission.name);
    Missions_Load(mission);
}

/* Load a mission and start it at a wave.
 *
 * The wave is the player's, clamped to one the mission has: a number past the
 * end would jump nowhere and a number below one is the start. Wave one needs
 * no jump at all, which is the ordinary load.
 *
 * Everything else is the load-time restore's machinery, reused: the same timer
 * waits for the mission to come up and the same watch checks the jump landed.
 */
void Missions_ResumeMission(int client, const char[] choice, int wave)
{
    Mission mission;
    if (!Missions_Find(choice, mission))
    {
        AP_PrintToClient(client, "No mission %s in this run. Type !mission for the list.", choice);
        return;
    }
    if (!Missions_Playable(mission))
    {
        AP_PrintToClient(client, "The run did not unlock %s yet. Its ticket is somewhere in the multiworld.", mission.name);
        return;
    }
    if (wave < 1)
    {
        wave = 1;
    }
    if (mission.waves > 0 && wave > mission.waves)
    {
        AP_PrintToClient(client, "%s has %d waves, so wave %d is not one of them.", mission.name, mission.waves, wave);
        return;
    }

    strcopy(g_ResumePopFile, sizeof(g_ResumePopFile), mission.popFile);
    // The machinery below counts in waves cleared, and the player picked the
    // wave they want to play: the wave before it is the one they are past.
    g_ResumeWave = wave - 1;
    g_ResumeTicks = 0;
    // A player asking is not the load-time restore, so the once-only guard
    // that stops the bridge's record dragging the team around does not apply.
    g_ResumeConsulted = true;
    CreateTimer(ResumeTick, Timer_Resume, _, TIMER_REPEAT);

    g_PlannedPopFile[0] = '\0';
    Missions_Load(mission);
}

/* The restore, one look per second until the mission is up.
 *
 * Not driven from Missions_OnListUpdated: that returns early while a switch is
 * in flight, and nothing fetches the list again after the changelevel, so at
 * the moment the mission finally lands there would be nobody left to jump.
 */
static Action Timer_Resume(Handle timer)
{
    if (g_ResumePopFile[0] == '\0')
    {
        return Plugin_Stop;
    }
    if (++g_ResumeTicks > ResumeTicksMax)
    {
        AP_Error("The mission %s never came up. The team stays where they are.", g_ResumePopFile);
        g_ResumePopFile[0] = '\0';
        return Plugin_Stop;
    }

    // MvM_PopFile falls back to the map name when the game will not say, and a
    // mission whose popfile is its map name would then look loaded while the
    // map was still coming up. The pending name is the map change not finished.
    if (g_PendingPopFile[0] != '\0')
    {
        return Plugin_Continue;
    }
    char current[64];
    MvM_PopFile(current, sizeof(current));
    if (!MvM_IsActive() || !StrEqual(current, g_ResumePopFile, false))
    {
        return Plugin_Continue;
    }

    Mission mission;
    if (Missions_Find(g_ResumePopFile, mission))
    {
        Missions_Resume(mission, g_ResumeWave);
    }
    g_ResumePopFile[0] = '\0';
    return Plugin_Stop;
}

// The wave after the last one cleared, and never one the team has not earned:
// a record naming a wave beyond the mission, or one the game is already at or
// past, is a way to skip content rather than a way back to it.
static void Missions_Resume(Mission mission, int cleared)
{
    int target = cleared + 1;
    if (target > mission.waves)
    {
        AP_Debug("The record has %s cleared to wave %d of %d. Nothing to go back to.",
            mission.popFile, cleared, mission.waves);
        return;
    }
    int playing = MvM_WaveFromGame();
    if (playing >= target)
    {
        AP_Debug("%s is already on wave %d, at or past the recorded %d.",
            mission.popFile, playing, target);
        return;
    }
    AP_Debug("Putting %s back to wave %d of %d, from a record of %d cleared.",
        mission.popFile, target, mission.waves, cleared);
    Missions_JumpToWave(target);

    // The jump is asked for, not confirmed. What follows watches until the
    // game agrees, pays the money back once it does, and says so either way.
    g_ResumeTarget = target;
    g_ResumeSettleTicks = 0;
    CreateTimer(ResumeSettleTick, Timer_ResumeSettled, _, TIMER_REPEAT);
}

/* Did the jump land, and pay the team back once it has.
 *
 * A wave jump is a round restart, so nothing is true immediately: the wave
 * number moves a moment later, and there is no live defender to hand money to
 * until the respawn. Both are waited for here rather than assumed, and the
 * announcement is held until the wave is real so the team is never told they
 * were restored to somewhere they are not.
 *
 * The money is the game's own: the jump is asked for full currency, which is
 * what the team would hold having won every wave before this one perfectly.
 */
static Action Timer_ResumeSettled(Handle timer)
{
    if (g_ResumeTarget <= 0)
    {
        return Plugin_Stop;
    }
    if (++g_ResumeSettleTicks > ResumeSettleTicksMax)
    {
        AP_Error("The wave jump to %d never took: the game is on wave %d. The team keeps what they have.",
            g_ResumeTarget, MvM_WaveFromGame());
        g_ResumeTarget = 0;
        return Plugin_Stop;
    }
    if (!MvM_IsActive() || MvM_WaveFromGame() < g_ResumeTarget)
    {
        return Plugin_Continue;
    }

    AP_Announce("Restored: wave %d, with the money for the waves before it.", g_ResumeTarget);
    g_ResumeTarget = 0;
    return Plugin_Stop;
}

// tf_mvm_jump_to_wave is a cheat, and a server left with cheats on is worse
// than a lost mission. All three go through the command queue so they run in
// that order: putting the variable back here and now would land before the
// jump does. The previous value goes back rather than a hardcoded zero,
// because somebody may have been playing with cheats on.
static void Missions_JumpToWave(int wave)
{
    ConVar cheats = FindConVar("sv_cheats");
    if (cheats == null)
    {
        AP_Error("This server has no sv_cheats, so the wave cannot be jumped to.");
        return;
    }
    char previous[8];
    cheats.GetString(previous, sizeof(previous));
    ServerCommand("sv_cheats 1");
    // The second argument is the share of the skipped waves' money to hand
    // over, and 1 is all of it: the team comes back with what perfect play
    // would have left them holding, which is the only honest answer when the
    // waves they are skipping are ones they had already won.
    ServerCommand("tf_mvm_jump_to_wave %d 1", wave);
    ServerCommand("sv_cheats %s", previous);
}
