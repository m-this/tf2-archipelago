/**
 * Team Fortress 2 Mann vs Machine, Archipelago integration.
 *
 * Reports objectives to the bridge and applies what it grants. Holds no
 * authoritative state and knows nothing about Archipelago (ADR 0002).
 *
 * The MvM events and entity properties it reads are UNVERIFIED against a live
 * server, so every hook is optional and every failure is announced: the first
 * real session says which of them exist. `sm_ap_status` prints the picture.
 */

#pragma semicolon 1
#pragma newdecls required

#include <sourcemod>
#include <sdktools>
#include <tf2>
#include <tf2_stocks>
#include <ripext>

#include "tf2_archipelago/log.inc"
#include "tf2_archipelago/mvm.inc"
#include "tf2_archipelago/unlocks.inc"
#include "tf2_archipelago/deathlink.inc"
#include "tf2_archipelago/bridge.inc"

#define PLUGIN_VERSION "1.1.0"

// Only used when the wave events turn out not to exist.
#define WavePollInterval 1.0

// Long enough after a map change for the mission to be loadable. tf_mvm_popfile
// reloads the mission in place, so it cannot be run while the map still is.
#define PopFileDelay 2.0

// Keeps the welcome out of the map load, where it would scroll past unread.
#define WelcomeDelay 8.0

public Plugin myinfo =
{
    name = "TF2 MvM Archipelago",
    author = "mathis",
    description = "Reports Mann vs Machine objectives to an Archipelago bridge and applies its grants.",
    version = PLUGIN_VERSION,
    url = "https://github.com/m-this/tf2-archipelago",
};

// Zero when no wave is running, or when the plugin loaded mid-mission.
int g_CurrentWave;
int g_MaxWaves;

bool g_HaveBeginWave;
bool g_HaveWaveComplete;
bool g_HaveMissionComplete;
bool g_HaveWaveFailed;

int g_PolledWave;

// Both mission detectors fire on purpose; the bridge dedups, chat should say it once.
bool g_MissionReported;

// The mission asked for while the map was still changing. tf_mvm_popfile acts
// on the map that is loaded, so a switch across maps has to wait for the new
// one before it can name the mission.
char g_PendingPopFile[64];

public void OnPluginStart()
{
    Log_Init();
    MvM_Init();
    Unlocks_Init();
    Bridge_Init();

    g_HaveBeginWave = HookEventEx("mvm_begin_wave", Event_BeginWave);
    g_HaveWaveComplete = HookEventEx("mvm_wave_complete", Event_WaveComplete);
    g_HaveMissionComplete = HookEventEx("mvm_mission_complete", Event_MissionComplete);
    g_HaveWaveFailed = HookEventEx("mvm_wave_failed", Event_WaveFailed);
    HookEvent("post_inventory_application", Event_InventoryApplied);
    HookEvent("player_spawn", Event_PlayerSpawn);

    AddCommandListener(Command_JoinClass, "joinclass");
    AddCommandListener(Command_Say, "say");
    AddCommandListener(Command_Say, "say_team");
    RegAdminCmd("sm_ap_status", Command_Status, ADMFLAG_GENERIC,
        "Show the state of the Archipelago integration");
    RegAdminCmd("sm_ap_report", Command_Report, ADMFLAG_ROOT,
        "Report an objective by hand: sm_ap_report <wave_cleared|mission_cleared|death> [wave]");
    RegAdminCmd("sm_ap_resync", Command_Resync, ADMFLAG_GENERIC,
        "Ask the bridge for the unlock set again");
    RegAdminCmd("sm_ap_mission", Command_Mission, ADMFLAG_CHANGEMAP,
        "List the run's missions, or switch to one: sm_ap_mission [number|popfile]");

    AutoExecConfig(true, "tf2_archipelago");

    if (!g_HaveWaveComplete)
    {
        AP_Error("This server has no mvm_wave_complete event. The plugin reads the wave counter instead.");
        // No TIMER_FLAG_NO_MAPCHANGE: it dies at the first map change; this must outlive every map.
        CreateTimer(WavePollInterval, Timer_PollWave, _, TIMER_REPEAT);
    }
    if (!g_HaveBeginWave)
    {
        AP_Error("This server has no mvm_begin_wave event. The plugin reads wave numbers from the game.");
    }
    if (!g_HaveWaveFailed)
    {
        AP_Error("This server has no mvm_wave_failed event. A lost wave cannot reach Death Link.");
    }

    // No grant poll here. It starts once the unlock set has landed: polling
    // before that means polling from sequence zero, and every effect the run
    // has ever received arrives a second time.
    Bridge_CheckVersion();
    Bridge_FetchUnlocks();
    Bridge_FetchMissions();
    Bridge_PollMessages();
    Bridge_PollDeaths();
}

public void OnClientPutInServer(int client)
{
    // Client indexes are reused, so the previous occupant's cooldown is not
    // this player's.
    Bridge_ClearCooldown(client);
    if (IsFakeClient(client))
    {
        return;
    }
    CreateTimer(WelcomeDelay, Timer_Welcome, GetClientUserId(client));
}

public Action Timer_Welcome(Handle timer, any userid)
{
    int client = GetClientOfUserId(userid);
    if (client <= 0 || !IsClientInGame(client) || !MvM_IsActive())
    {
        return Plugin_Stop;
    }

    char popFile[64];
    MvM_PopFile(popFile, sizeof(popFile));

    AP_PrintToClient(client, "This server runs an Archipelago randomizer.");
    AP_PrintToClient(client, "The run locks the classes and the weapon slots until it finds them. All players share the unlocks.");
    AP_PrintToClient(client, "Mission: %s. Each wave you clear is a check.", popFile);
    AP_PrintToClient(client, "Unlocked classes: %s", Status_ClassList());
    AP_PrintToClient(client, "Unlocked slots: %s", Status_SlotList());
    if (g_DeathLinkOn)
    {
        AP_PrintToClient(client, "Death Link is on: a lost wave kills every linked player, and their deaths wipe this team.");
    }
    AP_PrintToClient(client, "Type \x07FFD700!ap\x01 to speak to the multiworld. Examples: \x07FFD700!ap hint Class: Scout\x01 and \x07FFD700!ap missing\x01.");
    return Plugin_Stop;
}

// !ap forwards Archipelago server commands, replacing the separate client a player would need.
public Action Command_Say(int client, const char[] command, int argc)
{
    if (client <= 0)
    {
        return Plugin_Continue;
    }
    char message[256];
    GetCmdArgString(message, sizeof(message));
    StripQuotes(message);
    TrimString(message);

    if (StrEqual(message, "!ap", false))
    {
        AP_PrintToClient(client, "!ap <command> sends a command to the multiworld: hint, missing, status, checked.");
        AP_PrintToClient(client, "!apchat <text> speaks to the other players in the multiworld.");
        if (CheckCommandAccess(client, "sm_ap_mission", ADMFLAG_CHANGEMAP))
        {
            AP_PrintToClient(client, "!mission lists the run's missions. !mission <number> switches to one.");
        }
        return Plugin_Handled;
    }
    if (strncmp(message, "!ap ", 4, false) == 0)
    {
        char text[256];
        strcopy(text, sizeof(text), message[4]);
        TrimString(text);
        if (text[0] != '!')
        {
            Format(text, sizeof(text), "!%s", text);
        }
        Bridge_Say(client, text);
        return Plugin_Handled;
    }
    if (StrEqual(message, "!mission", false) || StrEqual(message, "!missions", false))
    {
        if (!MissionSwitchAllowed(client))
        {
            return Plugin_Handled;
        }
        ListMissions(client);
        return Plugin_Handled;
    }
    if (strncmp(message, "!mission ", 9, false) == 0)
    {
        if (!MissionSwitchAllowed(client))
        {
            return Plugin_Handled;
        }
        char choice[64];
        strcopy(choice, sizeof(choice), message[9]);
        TrimString(choice);
        SwitchMission(client, choice);
        return Plugin_Handled;
    }
    if (strncmp(message, "!apchat ", 8, false) == 0)
    {
        char text[256];
        char nickname[MAX_NAME_LENGTH];
        GetClientName(client, nickname, sizeof(nickname));
        FormatEx(text, sizeof(text), "%s: %s", nickname, message[8]);
        Bridge_Say(client, text);
        return Plugin_Handled;
    }
    return Plugin_Continue;
}

public void OnMapStart()
{
    g_CurrentWave = 0;
    g_MaxWaves = 0;
    g_PolledWave = 0;
    g_MissionReported = false;

    // The plugin's copy of the unlock set went with the map; ask before anyone spawns.
    Bridge_FetchUnlocks();
    // The seed does not change under a running server, but which of its
    // missions are unlocked does.
    Bridge_FetchMissions();
}

public void OnConfigsExecuted()
{
    if (!MvM_IsActive())
    {
        AP_Debug("This map is not Mann vs Machine. The plugin does nothing here.");
        return;
    }
    if (g_PendingPopFile[0] != '\0')
    {
        CreateTimer(PopFileDelay, Timer_ApplyPendingPopFile);
    }
}

// tf_mvm_popfile reloads the mission on the map that is already loaded, so the
// map change goes first and this lands after it.
public Action Timer_ApplyPendingPopFile(Handle timer)
{
    if (g_PendingPopFile[0] == '\0')
    {
        return Plugin_Stop;
    }
    AP_Debug("The plugin names the mission %s now that the map is up.", g_PendingPopFile);
    ServerCommand("tf_mvm_popfile %s", g_PendingPopFile);
    g_PendingPopFile[0] = '\0';
    return Plugin_Stop;
}

// The only source of the wave number: mvm_wave_complete does not carry one.
public void Event_BeginWave(Event event, const char[] name, bool dontBroadcast)
{
    g_CurrentWave = event.GetInt("wave_index") + 1;
    g_MaxWaves = event.GetInt("max_waves");
    g_PolledWave = g_CurrentWave;
    if (g_CurrentWave == 1)
    {
        g_MissionReported = false;
    }

    int fromGame = MvM_WaveFromGame();
    if (fromGame > 0 && fromGame != g_CurrentWave)
    {
        AP_Debug("Wave %d from the event, wave %d from the game.", g_CurrentWave, fromGame);
    }
    AP_Debug("Wave %d of %d started.", g_CurrentWave, g_MaxWaves);
}

public void Event_WaveComplete(Event event, const char[] name, bool dontBroadcast)
{
    ReportWaveCleared(g_CurrentWave > 0 ? g_CurrentWave : MvM_WaveFromGame());
}

public void Event_MissionComplete(Event event, const char[] name, bool dontBroadcast)
{
    ReportMissionCleared();
}

public void Event_WaveFailed(Event event, const char[] name, bool dontBroadcast)
{
    ReportWaveFailed(g_CurrentWave > 0 ? g_CurrentWave : MvM_WaveFromGame());
}

static void ReportWaveFailed(int wave)
{
    if (!MvM_IsActive())
    {
        return;
    }
    char popFile[64];
    MvM_PopFile(popFile, sizeof(popFile));
    AP_Debug("Wave %d of %s lost.", wave, popFile);
    DeathLink_WaveFailed(popFile, wave);
}

static void ReportWaveCleared(int wave)
{
    if (!MvM_IsActive())
    {
        return;
    }
    char popFile[64];
    if (!MvM_PopFile(popFile, sizeof(popFile)))
    {
        AP_Error("The team cleared a wave, but the mission has no name. The plugin did not report the check.");
        return;
    }
    if (wave < 1)
    {
        AP_Error("The team cleared a wave on %s, but its number is unknown. The plugin did not report the check.", popFile);
        return;
    }

    int maxWaves = g_MaxWaves > 0 ? g_MaxWaves : MvM_MaxWavesFromGame();

    AP_Announce("Wave %d cleared.", wave);
    Bridge_ReportObjective("wave_cleared", popFile, wave, maxWaves);

    if (maxWaves > 0 && wave >= maxWaves)
    {
        ReportMissionCleared();
    }
    g_CurrentWave = 0;
}

static void ReportMissionCleared()
{
    if (!MvM_IsActive())
    {
        return;
    }
    char popFile[64];
    if (!MvM_PopFile(popFile, sizeof(popFile)))
    {
        AP_Error("The team cleared the mission, but it has no name. The plugin did not report the check.");
        return;
    }
    if (g_MissionReported)
    {
        AP_Debug("The plugin already reported the mission clear for %s.", popFile);
        return;
    }
    g_MissionReported = true;
    AP_Announce("Mission cleared: %s", popFile);
    Bridge_ReportObjective("mission_cleared", popFile, 0,
        g_MaxWaves > 0 ? g_MaxWaves : MvM_MaxWavesFromGame());
}

// Fallback for a missing mvm_wave_complete: a rising wave counter means a wave was beaten.
public Action Timer_PollWave(Handle timer)
{
    if (!MvM_IsActive())
    {
        return Plugin_Continue;
    }
    int wave = MvM_WaveFromGame();
    if (wave > g_PolledWave && g_PolledWave > 0)
    {
        ReportWaveCleared(g_PolledWave);
    }
    g_PolledWave = wave;
    return Plugin_Continue;
}

// Both events fire for robots too; MvM_IsPlayer is what leaves their loadout alone.
public void Event_InventoryApplied(Event event, const char[] name, bool dontBroadcast)
{
    int client = GetClientOfUserId(event.GetInt("userid"));
    if (MvM_IsPlayer(client))
    {
        Unlocks_EnforceSlots(client);
    }
}

public void Event_PlayerSpawn(Event event, const char[] name, bool dontBroadcast)
{
    int client = GetClientOfUserId(event.GetInt("userid"));
    if (MvM_IsPlayer(client))
    {
        Unlocks_EnforceClass(client);
    }
}

// The class menu issues joinclass, so this is the one place to refuse a locked class.
public Action Command_JoinClass(int client, const char[] command, int argc)
{
    if (!Unlocks_Enforceable() || argc < 1 || !MvM_IsActive() || !MvM_IsPlayer(client))
    {
        return Plugin_Continue;
    }
    char requested[24];
    GetCmdArg(1, requested, sizeof(requested));

    TFClassType class = Unlocks_ClassFromKey(requested);
    if (class == TFClass_Unknown || g_ClassUnlocked[view_as<int>(class)])
    {
        return Plugin_Continue;
    }
    AP_PrintToClient(client, "The run did not unlock %s yet.", requested);
    return Plugin_Handled;
}

// The map rotation belongs to the operator, so the switcher does too. Anyone
// else asking is told no rather than left wondering why nothing happened.
static bool MissionSwitchAllowed(int client)
{
    if (CheckCommandAccess(client, "sm_ap_mission", ADMFLAG_CHANGEMAP))
    {
        return true;
    }
    AP_PrintToClient(client, "Only an admin changes the mission.");
    return false;
}

static void ListMissions(int client)
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
        AP_PrintToClient(client, "%d. %s (%s), %d waves%s%s",
            index + 1, mission.name, mission.popFile, mission.waves,
            mission.unlocked ? "" : " [locked]",
            StrEqual(mission.popFile, current) ? " [playing]" : "");
    }
}

// Takes a number from the list or a pop file name. The map comes from the
// bridge: a pop file name does not reliably carry the map it runs on.
static void SwitchMission(int client, const char[] choice)
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
        for (int index = 0; index < count && !found; index++)
        {
            Mission candidate;
            if (Bridge_MissionAt(index, candidate) && StrEqual(candidate.popFile, choice, false))
            {
                mission = candidate;
                found = true;
            }
        }
    }
    if (!found)
    {
        AP_PrintToClient(client, "No mission %s in this run. Type !mission for the list.", choice);
        return;
    }

    LogMessage("mission switched to %s (%s) on %s", mission.name, mission.popFile, mission.map);

    // tf_mvm_popfile is a command, not a variable, and it acts on the map that
    // is loaded: it reloads the mission in place. So the map only changes when
    // it has to, and naming the mission waits for the new map when it does.
    char current[64];
    GetCurrentMap(current, sizeof(current));
    if (StrEqual(current, mission.map, false))
    {
        AP_Announce("Mission: %s. Reloading.", mission.name);
        ServerCommand("tf_mvm_popfile %s", mission.popFile);
        return;
    }
    AP_Announce("Mission: %s. The map changes now.", mission.name);
    strcopy(g_PendingPopFile, sizeof(g_PendingPopFile), mission.popFile);
    ServerCommand("changelevel %s", mission.map);
}

public Action Command_Mission(int client, int argc)
{
    if (argc < 1)
    {
        ListMissions(client);
        return Plugin_Handled;
    }
    char choice[64];
    GetCmdArg(1, choice, sizeof(choice));
    SwitchMission(client, choice);
    return Plugin_Handled;
}

public Action Command_Status(int client, int argc)
{
    char popFile[64], lastError[192];
    MvM_PopFile(popFile, sizeof(popFile));
    Bridge_LastError(lastError, sizeof(lastError));

    ReplyToCommand(client, "[AP] version %s, mvm %s, mission %s, wave %d of %d",
        PLUGIN_VERSION, MvM_IsActive() ? "yes" : "no", popFile,
        g_CurrentWave, g_MaxWaves > 0 ? g_MaxWaves : MvM_MaxWavesFromGame());
    ReplyToCommand(client, "[AP] events: begin_wave %s, wave_complete %s, mission_complete %s, wave_failed %s",
        g_HaveBeginWave ? "yes" : "no",
        g_HaveWaveComplete ? "yes" : "no",
        g_HaveMissionComplete ? "yes" : "no",
        g_HaveWaveFailed ? "yes" : "no");
    ReplyToCommand(client, "[AP] death link %s", g_DeathLinkOn ? "on" : "off");
    ReplyToCommand(client, "[AP] unlocks %s at sequence %d, %d objective(s) waiting to be sent",
        g_HaveUnlocks ? "held" : "NOT FETCHED", Bridge_Sequence(), Bridge_PendingCount());
    ReplyToCommand(client, "[AP] classes: %s", Status_ClassList());
    ReplyToCommand(client, "[AP] slots: %s", Status_SlotList());
    if (lastError[0] != '\0')
    {
        ReplyToCommand(client, "[AP] Last bridge error: %s", lastError);
    }
    return Plugin_Handled;
}

static char[] Status_ClassList()
{
    char list[192];
    for (int class = 1; class < sizeof(g_ClassUnlocked); class++)
    {
        if (g_ClassUnlocked[class])
        {
            Format(list, sizeof(list), "%s%s%s", list, list[0] == '\0' ? "" : ", ", g_ClassKeys[class]);
        }
    }
    if (list[0] == '\0')
    {
        strcopy(list, sizeof(list), "none");
    }
    return list;
}

static char[] Status_SlotList()
{
    char list[96];
    for (int slot = 0; slot < Slot_Count; slot++)
    {
        if (g_SlotUnlocked[slot])
        {
            Format(list, sizeof(list), "%s%s%s", list, list[0] == '\0' ? "" : ", ", g_SlotKeys[slot]);
        }
    }
    if (list[0] == '\0')
    {
        strcopy(list, sizeof(list), "none");
    }
    return list;
}

// Tests the wiring without playing a wave, and sends a check the game failed to fire an event for.
public Action Command_Report(int client, int argc)
{
    if (argc < 1)
    {
        ReplyToCommand(client, "[AP] Usage: sm_ap_report <wave_cleared|mission_cleared|death> [wave]");
        return Plugin_Handled;
    }
    char kind[24];
    GetCmdArg(1, kind, sizeof(kind));

    if (StrEqual(kind, "mission_cleared"))
    {
        ReportMissionCleared();
        return Plugin_Handled;
    }
    if (StrEqual(kind, "death"))
    {
        ReportWaveFailed(g_CurrentWave);
        return Plugin_Handled;
    }
    if (!StrEqual(kind, "wave_cleared"))
    {
        ReplyToCommand(client, "[AP] Unknown objective kind: %s", kind);
        return Plugin_Handled;
    }

    int wave = g_CurrentWave;
    if (argc >= 2)
    {
        char argument[8];
        GetCmdArg(2, argument, sizeof(argument));
        wave = StringToInt(argument);
    }
    ReportWaveCleared(wave);
    return Plugin_Handled;
}

public Action Command_Resync(int client, int argc)
{
    Bridge_FetchUnlocks();
    ReplyToCommand(client, "[AP] The plugin asked the bridge for the unlock set.");
    return Plugin_Handled;
}
