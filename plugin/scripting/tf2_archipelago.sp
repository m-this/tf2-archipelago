/**
 * Team Fortress 2 Mann vs Machine, Archipelago integration.
 *
 * This plugin sees the game and nothing else. It reports objectives to the
 * bridge and applies what the bridge grants; it holds no authoritative state
 * and knows nothing about Archipelago. See docs/adr/0002.
 *
 * The MvM events and entity properties it reads are UNVERIFIED against a live
 * server. Every hook is optional and every failure is announced, so the first
 * real session says which of them exist rather than the plugin quietly
 * reporting nothing. `sm_ap_status` prints the whole picture.
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
#include "tf2_archipelago/bridge.inc"

#define PLUGIN_VERSION "0.1.0"

// WavePollInterval is only used when the wave events turn out not to exist.
#define WavePollInterval 1.0

// WelcomeDelay keeps the welcome out of the map load, where it would scroll
// past before the player can read it.
#define WelcomeDelay 8.0

public Plugin myinfo =
{
    name = "TF2 MvM Archipelago",
    author = "mathis",
    description = "Reports Mann vs Machine objectives to an Archipelago bridge and applies its grants.",
    version = PLUGIN_VERSION,
    url = "https://git-ssh.croque.top/mathis/tf2-archipelago",
};

// The wave in progress, from mvm_begin_wave. Zero when no wave is running or
// when the plugin loaded mid-mission.
int g_CurrentWave;
int g_MaxWaves;

// Which of the game's events actually exist here. Reported by sm_ap, because
// the answer decides how much of this plugin works.
bool g_HaveBeginWave;
bool g_HaveWaveComplete;
bool g_HaveMissionComplete;

// The last wave count seen by the fallback poller, which only runs when the
// wave events are missing.
int g_PolledWave;

public void OnPluginStart()
{
    Log_Init();
    Unlocks_Init();
    Bridge_Init();

    g_HaveBeginWave = HookEventEx("mvm_begin_wave", Event_BeginWave);
    g_HaveWaveComplete = HookEventEx("mvm_wave_complete", Event_WaveComplete);
    g_HaveMissionComplete = HookEventEx("mvm_mission_complete", Event_MissionComplete);
    HookEvent("post_inventory_application", Event_InventoryApplied);
    HookEvent("player_spawn", Event_PlayerSpawn);

    AddCommandListener(Command_JoinClass, "joinclass");
    AddCommandListener(Command_Say, "say");
    AddCommandListener(Command_Say, "say_team");
    RegAdminCmd("sm_ap_status", Command_Status, ADMFLAG_GENERIC,
        "Print the Archipelago integration's state");
    RegAdminCmd("sm_ap_report", Command_Report, ADMFLAG_ROOT,
        "Report an objective by hand: sm_ap_report <wave_cleared|mission_cleared> [wave]");
    RegAdminCmd("sm_ap_resync", Command_Resync, ADMFLAG_GENERIC,
        "Fetch the unlock set from the bridge again");

    AutoExecConfig(true, "tf2_archipelago");

    if (!g_HaveWaveComplete)
    {
        AP_Error("mvm_wave_complete does not exist on this server, watching the wave counter instead");
        CreateTimer(WavePollInterval, Timer_PollWave, _, TIMER_REPEAT | TIMER_FLAG_NO_MAPCHANGE);
    }
    if (!g_HaveBeginWave)
    {
        AP_Error("mvm_begin_wave does not exist on this server, wave numbers come from the game state");
    }

    Bridge_FetchUnlocks();
    Bridge_PollGrants();
    Bridge_PollMessages();
}

/**
 * Tell an arriving player what they have walked into.
 *
 * There is no client mod and the web MOTD is a panel most people close without
 * reading, so this is chat. Delayed a few seconds: a message printed while the
 * map is still loading is a message nobody sees.
 */
public void OnClientPutInServer(int client)
{
    if (IsFakeClient(client))
    {
        return;
    }
    CreateTimer(WelcomeDelay, Timer_Welcome, GetClientUserId(client));
}

public Action Timer_Welcome(Handle timer, any userid)
{
    int client = GetClientOfUserId(userid);
    if (client <= 0 || !IsClientInGame(client))
    {
        return Plugin_Stop;
    }

    char popFile[64];
    MvM_PopFile(popFile, sizeof(popFile));

    AP_PrintToClient(client, "This server is running an Archipelago randomizer.");
    AP_PrintToClient(client, "Classes and weapon slots are locked until the run finds them, and everyone shares the same unlocks.");
    AP_PrintToClient(client, "Mission: %s. Clearing a wave is a check.", popFile);
    AP_PrintToClient(client, "Unlocked classes: %s", Status_ClassList());
    AP_PrintToClient(client, "Unlocked slots: %s", Status_SlotList());
    AP_PrintToClient(client, "Talk to the multiworld with \x07FFD700!ap\x01, for example \x07FFD700!ap hint Scout\x01 or \x07FFD700!ap missing\x01.");
    return Plugin_Stop;
}

/**
 * Chat passing through to the multiworld.
 *
 *   !ap                one line of help
 *   !ap <command>      an Archipelago server command, ! added if it is missing
 *   !apchat <text>     plain talk to the other players in the multiworld
 *
 * Archipelago's own commands are what a player would otherwise need a separate
 * client for: !hint, !missing, !status, !release, !collect.
 */
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
        AP_PrintToClient(client, "!ap <command> talks to the multiworld: try hint, missing, status, release.");
        AP_PrintToClient(client, "!apchat <text> says something to the other players in it.");
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

    // The unlock set is the bridge's, and this plugin has just forgotten its
    // copy. Ask before anyone spawns.
    Bridge_FetchUnlocks();
}

public void OnConfigsExecuted()
{
    if (!MvM_IsActive())
    {
        AP_Debug("this map is not Mann vs Machine, the plugin is inert here");
    }
}

/**
 * Wave started. This is where the wave number comes from: mvm_wave_complete
 * does not carry one.
 */
public void Event_BeginWave(Event event, const char[] name, bool dontBroadcast)
{
    g_CurrentWave = event.GetInt("wave_index") + 1;
    g_MaxWaves = event.GetInt("max_waves");
    g_PolledWave = g_CurrentWave;

    int fromGame = MvM_WaveFromGame();
    if (fromGame > 0 && fromGame != g_CurrentWave)
    {
        AP_Debug("wave %d from the event, %d from the game state", g_CurrentWave, fromGame);
    }
    AP_Debug("wave %d of %d started", g_CurrentWave, g_MaxWaves);
}

public void Event_WaveComplete(Event event, const char[] name, bool dontBroadcast)
{
    ReportWaveCleared(g_CurrentWave > 0 ? g_CurrentWave : MvM_WaveFromGame());
}

public void Event_MissionComplete(Event event, const char[] name, bool dontBroadcast)
{
    ReportMissionCleared();
}

/**
 * Report a wave, and the mission with it when that was the last one. Reporting
 * the mission from here as well as from mvm_mission_complete is deliberate:
 * both are idempotent at the bridge, and between them one will fire.
 */
static void ReportWaveCleared(int wave)
{
    if (!MvM_IsActive())
    {
        return;
    }
    char popFile[64];
    if (!MvM_PopFile(popFile, sizeof(popFile)))
    {
        AP_Error("a wave was cleared but the mission could not be identified, check not reported");
        return;
    }
    if (wave < 1)
    {
        AP_Error("a wave was cleared on %s but its number is unknown, check not reported", popFile);
        return;
    }

    AP_Announce("Wave %d cleared.", wave);
    Bridge_ReportObjective("wave_cleared", popFile, wave);

    int maxWaves = g_MaxWaves > 0 ? g_MaxWaves : MvM_MaxWavesFromGame();
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
        AP_Error("the mission was cleared but could not be identified, check not reported");
        return;
    }
    AP_Announce("Mission cleared: %s", popFile);
    Bridge_ReportObjective("mission_cleared", popFile, 0);
}

/**
 * The fallback wave detector, running only when mvm_wave_complete is missing.
 * The wave counter going up means the previous wave was beaten; it going back
 * to one means a new mission.
 */
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

public void Event_InventoryApplied(Event event, const char[] name, bool dontBroadcast)
{
    int client = GetClientOfUserId(event.GetInt("userid"));
    if (client > 0)
    {
        Unlocks_EnforceSlots(client);
    }
}

public void Event_PlayerSpawn(Event event, const char[] name, bool dontBroadcast)
{
    int client = GetClientOfUserId(event.GetInt("userid"));
    if (client > 0)
    {
        Unlocks_EnforceClass(client);
    }
}

/**
 * Refuse a locked class at the class menu. The menu issues joinclass, so this
 * is the one place to catch it.
 */
public Action Command_JoinClass(int client, const char[] command, int argc)
{
    if (!g_HaveUnlocks || client <= 0 || argc < 1 || !MvM_IsActive())
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
    AP_PrintToClient(client, "%s is not unlocked yet.", requested);
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
    ReplyToCommand(client, "[AP] events: begin_wave %s, wave_complete %s, mission_complete %s",
        g_HaveBeginWave ? "yes" : "no",
        g_HaveWaveComplete ? "yes" : "no",
        g_HaveMissionComplete ? "yes" : "no");
    ReplyToCommand(client, "[AP] unlocks %s at sequence %d, %d objective(s) waiting to be sent",
        g_HaveUnlocks ? "held" : "NOT FETCHED", Bridge_Sequence(), Bridge_PendingCount());
    ReplyToCommand(client, "[AP] classes: %s", Status_ClassList());
    ReplyToCommand(client, "[AP] slots: %s", Status_SlotList());
    if (lastError[0] != '\0')
    {
        ReplyToCommand(client, "[AP] last bridge error: %s", lastError);
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

/**
 * Report an objective by hand. This is how the wiring gets tested without
 * playing a wave, and how a check gets sent when the game did not fire the
 * event it should have.
 */
public Action Command_Report(int client, int argc)
{
    if (argc < 1)
    {
        ReplyToCommand(client, "[AP] usage: sm_ap_report <wave_cleared|mission_cleared> [wave]");
        return Plugin_Handled;
    }
    char kind[24];
    GetCmdArg(1, kind, sizeof(kind));

    if (StrEqual(kind, "mission_cleared"))
    {
        ReportMissionCleared();
        return Plugin_Handled;
    }
    if (!StrEqual(kind, "wave_cleared"))
    {
        ReplyToCommand(client, "[AP] unknown objective kind %s", kind);
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
    ReplyToCommand(client, "[AP] asked the bridge for the unlock set");
    return Plugin_Handled;
}
