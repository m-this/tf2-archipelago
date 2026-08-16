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
#include "tf2_archipelago/bridge.inc"

#define PLUGIN_VERSION "0.1.0"

// Only used when the wave events turn out not to exist.
#define WavePollInterval 1.0

// Keeps the welcome out of the map load, where it would scroll past unread.
#define WelcomeDelay 8.0

public Plugin myinfo =
{
    name = "TF2 MvM Archipelago",
    author = "mathis",
    description = "Reports Mann vs Machine objectives to an Archipelago bridge and applies its grants.",
    version = PLUGIN_VERSION,
    url = "https://git-ssh.croque.top/mathis/tf2-archipelago",
};

// Zero when no wave is running, or when the plugin loaded mid-mission.
int g_CurrentWave;
int g_MaxWaves;

bool g_HaveBeginWave;
bool g_HaveWaveComplete;
bool g_HaveMissionComplete;

int g_PolledWave;

// Both mission detectors fire on purpose; the bridge dedups, chat should say it once.
bool g_MissionReported;

public void OnPluginStart()
{
    Log_Init();
    MvM_Init();
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
        "Show the state of the Archipelago integration");
    RegAdminCmd("sm_ap_report", Command_Report, ADMFLAG_ROOT,
        "Report an objective by hand: sm_ap_report <wave_cleared|mission_cleared> [wave]");
    RegAdminCmd("sm_ap_resync", Command_Resync, ADMFLAG_GENERIC,
        "Ask the bridge for the unlock set again");

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

    Bridge_FetchUnlocks();
    Bridge_PollGrants();
    Bridge_PollMessages();
}

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
    AP_PrintToClient(client, "Type \x07FFD700!ap\x01 to speak to the multiworld. Examples: \x07FFD700!ap hint Scout\x01 and \x07FFD700!ap missing\x01.");
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
        AP_PrintToClient(client, "!ap <command> sends a command to the multiworld: hint, missing, status, release.");
        AP_PrintToClient(client, "!apchat <text> speaks to the other players in the multiworld.");
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
    g_MissionReported = false;

    // The plugin's copy of the unlock set went with the map; ask before anyone spawns.
    Bridge_FetchUnlocks();
}

public void OnConfigsExecuted()
{
    if (!MvM_IsActive())
    {
        AP_Debug("This map is not Mann vs Machine. The plugin does nothing here.");
    }
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
    Bridge_ReportObjective("mission_cleared", popFile, 0);
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
        ReplyToCommand(client, "[AP] Usage: sm_ap_report <wave_cleared|mission_cleared> [wave]");
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
