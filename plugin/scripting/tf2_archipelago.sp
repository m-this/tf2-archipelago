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
#include <sdkhooks>
#include <sdktools>
#include <tf2>
#include <tf2_stocks>
#include <ripext>
#include <tf2attributes>

#include "tf2_archipelago/log.inc"
#include "tf2_archipelago/mvm.inc"
#include "tf2_archipelago/unlocks.inc"
#include "tf2_archipelago/weapon_buffs_data.inc"
#include "tf2_archipelago/weapon_buffs.inc"
#include "tf2_archipelago/deathlink.inc"
#include "tf2_archipelago/traps.inc"
#include "tf2_archipelago/bridge.inc"
#include "tf2_archipelago/missions.inc"
#include "tf2_archipelago/bots.inc"
#include "tf2_archipelago/botswitch.inc"

#define PLUGIN_VERSION "1.11.0"

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
    url = "https://github.com/m-this/tf2-archipelago",
};

// Zero when no wave is running, or when the plugin loaded mid-mission.
int g_CurrentWave;
int g_MaxWaves;

bool g_HaveBeginWave;
bool g_HaveWaveComplete;
bool g_HaveMissionComplete;
bool g_HaveWaveFailed;
bool g_HaveTankDestroyed;

int g_PolledWave;

// Both mission detectors fire on purpose; the bridge dedups, chat should say it once.
bool g_MissionReported;

// The mission's tank check is already reported. One per mission, cleared with
// the mission the way g_MissionReported is.
bool g_TankReported;

// The same for the giant check. It also keeps player_death cheap: a wave is
// hundreds of robot deaths, and once this is set the handler is one bool read.
bool g_GiantReported;

/* What the defender bot mod needs from this plugin, and nothing more.
 *
 * A Cash Bundle is written straight onto m_nCurrency, so the game's own record
 * of the wave never sees it. The bot mod sets a bot's currency from that record
 * whenever one joins, which silently throws every bundle the bot was paid away.
 * It cannot add them back without being told the number, and this is the number.
 *
 * A native rather than a convar or a file: it is read at the moment a bot spawns
 * and it has to be the current value, not one from whenever something last wrote
 * it down.
 */
public APLRes AskPluginLoad2(Handle self, bool late, char[] error, int length)
{
    CreateNative("TF2AP_GetBundleCredits", Native_GetBundleCredits);
    RegPluginLibrary("tf2_archipelago");
    return APLRes_Success;
}

static any Native_GetBundleCredits(Handle plugin, int numParams)
{
    int client = GetNativeCell(1);
    if (client < 1 || client > MaxClients)
    {
        return ThrowNativeError(SP_ERROR_NATIVE, "client %d is not a client", client);
    }
    return MvM_BundleCredits(client);
}

public void OnPluginStart()
{
    LoadTranslations("common.phrases");
    Log_Init();
    MvM_Init();
    Unlocks_Init();
    WeaponBuffs_Init();
    Bridge_Init();
    Missions_Init();
    Bots_Init();

    g_HaveBeginWave = HookEventEx("mvm_begin_wave", Event_BeginWave);
    g_HaveWaveComplete = HookEventEx("mvm_wave_complete", Event_WaveComplete);
    g_HaveMissionComplete = HookEventEx("mvm_mission_complete", Event_MissionComplete);
    g_HaveWaveFailed = HookEventEx("mvm_wave_failed", Event_WaveFailed);
    g_HaveTankDestroyed = HookEventEx("mvm_tank_destroyed_by_players", Event_TankDestroyed);
    HookEvent("post_inventory_application", Event_InventoryApplied);
    HookEvent("player_spawn", Event_PlayerSpawn);
    // The game fires no event for a giant, so the check rides on every death.
    HookEvent("player_death", Event_PlayerDeath);

    // Every way a player asks for a seat on RED. The first listener frees one
    // if the bots took them all; the second enforces the run's classes.
    AddCommandListener(Command_JoinRed, "jointeam");
    AddCommandListener(Command_JoinRed, "autoteam");
    AddCommandListener(Command_JoinRed, "joinclass");
    AddCommandListener(Command_JoinRed, "join_class");
    // The class menu sends one, the quick class picker the other.
    AddCommandListener(Command_JoinClass, "joinclass");
    AddCommandListener(Command_JoinClass, "join_class");
    AddCommandListener(Command_Say, "say");
    AddCommandListener(Command_Say, "say_team");
    RegAdminCmd("sm_ap_status", Command_Status, ADMFLAG_GENERIC,
        "Show the state of the Archipelago integration");
    RegAdminCmd("sm_ap_report", Command_Report, ADMFLAG_ROOT,
        "Report an objective by hand: sm_ap_report <wave_cleared|mission_cleared|death> [wave]");
    RegAdminCmd("sm_ap_bundle", Command_Bundle, ADMFLAG_ROOT,
        "Pay a Cash Bundle by hand, the way a grant from the room would: sm_ap_bundle [credits]");
    RegAdminCmd("sm_ap_trap", Command_Trap, ADMFLAG_ROOT,
        "Fire a trap by hand, the way a grant from the room would: sm_ap_trap <key>");
    RegAdminCmd("sm_ap_resync", Command_Resync, ADMFLAG_GENERIC,
        "Ask the bridge for the unlock set again");
    RegAdminCmd("sm_ap_mission", Command_Mission, ADMFLAG_CHANGEMAP,
        "List the run's missions, or switch to one: sm_ap_mission [number|popfile]");
    RegConsoleCmd("sm_ap_buffs", Command_WeaponBuffs,
        "Show the Archipelago buffs for your current loadout");
    RegAdminCmd("sm_ap_buff_test", Command_TestWeaponBuff, ADMFLAG_ROOT,
        "Test an active-weapon effect: sm_ap_buff_test <number|key|all> [levels]");
    RegAdminCmd("sm_ap_buff_give", Command_GiveWeaponBuff, ADMFLAG_ROOT,
        "Give a test effect to a player's active weapon: sm_ap_buff_give <target> <number|key|all> [levels]");
    RegAdminCmd("sm_ap_projectile_debug", Command_ProjectileDebug, ADMFLAG_ROOT,
        "Toggle projectile diagnostics: sm_ap_projectile_debug [on|off]");
    RegAdminCmd("sm_ap_unlock_override", Command_UnlockOverride, ADMFLAG_ROOT,
        "Temporarily allow every class and weapon slot: sm_ap_unlock_override <on|off>");

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
    if (!g_HaveTankDestroyed)
    {
        AP_Error("This server has no mvm_tank_destroyed_by_players event. Tank checks never fire.");
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
    WeaponBuffs_HookClient(client);
    // Client indexes are reused, so the previous occupant's cooldown is not
    // this player's.
    Bridge_ClearCooldown(client);
    // Client indexes are reused, and so is the bundle ledger behind them.
    MvM_ForgetBundles(client);
    if (IsFakeClient(client))
    {
        return;
    }
    Bots_MakeRoom();
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
    AP_PrintToClient(client, "Type \x07FFD700!ap status\x01 for the state of the run, \x07FFD700!ap\x01 to speak to the multiworld. Examples: \x07FFD700!ap hint Class: Scout\x01 and \x07FFD700!ap missing\x01.");
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

    if (StrEqual(message, "!ap", false) || StrEqual(message, "!ap help", false))
    {
        AP_PrintToClient(client, "!ap status shows the run: mission, wave, unlocks, and the bridge.");
        AP_PrintToClient(client, "!ap <command> sends a command to the multiworld: hint, missing, checked, players.");
        // Said plainly rather than hidden behind a test-mode flag the plugin
        // would have to be told about: in a real room the multiworld answers
        // that it has never heard of it, which is what the line already says.
        AP_PrintToClient(client, "!ap unlock mission hands over the next mission ticket, in test mode only.");
        AP_PrintToClient(client, "!ap bots changes what the bots on RED play, seat by seat.");
        AP_PrintToClient(client, "!ap buffs lists the Archipelago buffs on your loadout.");
        AP_PrintToClient(client, "!apchat <text> speaks to the other players in the multiworld.");
        AP_PrintToClient(client, "!mission lists the run's missions.%s",
            CheckCommandAccess(client, "sm_ap_mission", ADMFLAG_CHANGEMAP) ? " !mission <number> switches to one." : "");
        return Plugin_Handled;
    }
    /* Ahead of the "!ap <anything>" passthrough below, which is what used to
     * eat this: "!ap buffs" fell through to the multiworld as the chat command
     * "!buffs", so the window never opened and nothing said why.
     */
    if (StrEqual(message, "!ap buffs", false) || StrEqual(message, "!ap buff", false)
        || StrEqual(message, "!apbuffs", false))
    {
        if (!WeaponBuffs_ShowMenu(client))
        {
            AP_PrintToClient(client, "No buffs for this loadout yet.");
        }
        return Plugin_Handled;
    }
    if (StrEqual(message, "!ap bots", false) || StrEqual(message, "!apbots", false))
    {
        if (BotSwitchAllowed(client))
        {
            BotSwitch_Open(client);
        }
        return Plugin_Handled;
    }
    if (StrEqual(message, "!ap status", false) || StrEqual(message, "!apstatus", false))
    {
        PrintRunStatus(client);
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
        Missions_List(client);
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
        Missions_Switch(client, choice);
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
    g_TankReported = false;
    g_GiantReported = false;
    Bots_OnMapStart();
    MvM_OnMapStart();

    // The plugin's copy of the unlock set went with the map; ask before anyone spawns.
    Bridge_FetchUnlocks();
    // The seed does not change under a running server, but which of its
    // missions are unlocked does.
    Bridge_FetchMissions();
}

public void OnConfigsExecuted()
{
    Bots_OnConfigsExecuted();
    if (!MvM_IsActive())
    {
        AP_Debug("This map is not Mann vs Machine. The plugin waits for the run's mission list.");
        return;
    }
    Missions_OnConfigsExecuted();
}

// The only source of the wave number: mvm_wave_complete does not carry one.
public void Event_BeginWave(Event event, const char[] name, bool dontBroadcast)
{
    g_CurrentWave = event.GetInt("wave_index") + 1;
    g_MaxWaves = event.GetInt("max_waves");
    g_PolledWave = g_CurrentWave;
    // The mod adds its whole lineup at this moment whatever is already on RED,
    // and this plugin has filled the team before it. One of the two has to
    // count, and the seats belong to this one.
    CreateTimer(BotTrimDelay, Timer_TrimBots);
    if (g_CurrentWave == 1)
    {
        g_MissionReported = false;
        g_TankReported = false;
        g_GiantReported = false;
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

// A giant died, maybe. This runs on every robot death of every wave, so the
// cheapest test comes first and the whole handler stops for good once the
// mission's check is in.
public void Event_PlayerDeath(Event event, const char[] name, bool dontBroadcast)
{
    if (g_GiantReported || !MvM_IsActive())
    {
        return;
    }
    if (!MvM_IsGiant(GetClientOfUserId(event.GetInt("userid"))))
    {
        return;
    }
    char popFile[64];
    if (!MvM_PopFile(popFile, sizeof(popFile)))
    {
        AP_Error("The team killed a giant, but the mission has no name. The plugin did not report the check.");
        return;
    }
    g_GiantReported = true;
    AP_Announce("Giant killed: %s", popFile);
    Bridge_ReportObjective("giant_killed", popFile, 0,
        g_MaxWaves > 0 ? g_MaxWaves : MvM_MaxWavesFromGame());
}

// One check per mission, not one per tank: the bridge takes the check once and
// drops the rest, and reporting every tank of an eight wave mission would be
// eight requests for one location. g_TankReported keeps them off the wire.
public void Event_TankDestroyed(Event event, const char[] name, bool dontBroadcast)
{
    if (!MvM_IsActive() || g_TankReported)
    {
        return;
    }
    char popFile[64];
    if (!MvM_PopFile(popFile, sizeof(popFile)))
    {
        AP_Error("The team destroyed a tank, but the mission has no name. The plugin did not report the check.");
        return;
    }
    g_TankReported = true;
    AP_Announce("Tank destroyed: %s", popFile);
    Bridge_ReportObjective("tank_destroyed", popFile, 0,
        g_MaxWaves > 0 ? g_MaxWaves : MvM_MaxWavesFromGame());
}

// The game fires this while a mission loads, before anybody has readied up:
// a live server sent a Death Link to the whole multiworld fourteen seconds
// after a map change, with nobody playing. So a loss counts only while a wave
// this plugin saw start is running. g_CurrentWave is what says so: mvm_begin_wave
// sets it, a cleared wave and a map change clear it.
public void Event_WaveFailed(Event event, const char[] name, bool dontBroadcast)
{
    if (g_CurrentWave < 1)
    {
        AP_Debug("The game reported a lost wave with no wave running. The plugin ignores it.");
        return;
    }
    // The game restores each balance to what it recorded at wave start, which
    // never held a bundle. Without this the spent bundles come off a number
    // that never had them, and the balance goes negative.
    MvM_RestoreBundlesAll();
    ReportWaveFailed(g_CurrentWave);
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
    Missions_OnMissionCleared(popFile);
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

// Slot enforcement is player-only. Buff application has its own ownership
// policy, including the optional mirror-to-BLU-robots mode.
public void Event_InventoryApplied(Event event, const char[] name, bool dontBroadcast)
{
    int client = GetClientOfUserId(event.GetInt("userid"));
    if (MvM_IsPlayer(client))
    {
        Unlocks_EnforceSlots(client);
    }
    WeaponBuffs_Apply(client);
}

public void Event_PlayerSpawn(Event event, const char[] name, bool dontBroadcast)
{
    int client = GetClientOfUserId(event.GetInt("userid"));
    if (MvM_IsPlayer(client))
    {
        Unlocks_EnforceClass(client);
        Unlocks_RestoreOverrideNextFrame(client);
    }
}

// A bot leaving is the mod's business, or this plugin's own Bots_MakeRoom, and
// backfilling that kick would put the bot straight back into the seat the
// arriving player was making room in. The client is gone by _Post, so what it
// was has to be recorded before it goes.
public void OnClientDisconnect(int client)
{
    WeaponBuffs_Disconnect(client);
    Bots_OnClientLeaving(client);
}

// _Post, because the seat is only free once the player is gone: counting RED
// from OnClientDisconnect still counts the player who is leaving.
public void OnClientDisconnect_Post(int client)
{
    Bots_OnClientLeft(client);
}

// The class menu issues joinclass, so this is the one place to refuse a locked class.
// A player who is not on RED is asking to be. The bots fill RED to six and the
// game refuses a seventh, so a spectator coming back finds the door shut: the
// mod tops the team up but never makes room. Connecting is not the only moment
// this happens, which is what OnClientPutInServer alone assumed.
public Action Command_JoinRed(int client, const char[] command, int argc)
{
    if (client > 0 && IsClientInGame(client) && !IsFakeClient(client) && GetClientTeam(client) != TeamRed)
    {
        Bots_MakeRoom();
    }
    return Plugin_Continue;
}

public Action Command_JoinClass(int client, const char[] command, int argc)
{
    if (!Unlocks_ClassesEnforced() || argc < 1 || !MvM_IsActive() || !MvM_IsPlayer(client))
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

/* Who may retype the lineup.

The same flag the mission switch asks for. Both decide what everybody on RED
plays for the rest of the run, and a team somebody else keeps rearranging is the
same nuisance as a mission somebody else keeps loading. */
static bool BotSwitchAllowed(int client)
{
    if (CheckCommandAccess(client, "sm_ap_bots", ADMFLAG_CHANGEMAP))
    {
        return true;
    }
    AP_PrintToClient(client, "Only an admin changes the bot team.");
    return false;
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

public Action Command_Mission(int client, int argc)
{
    if (argc < 1)
    {
        Missions_List(client);
        return Plugin_Handled;
    }
    char choice[64];
    GetCmdArg(1, choice, sizeof(choice));
    Missions_Switch(client, choice);
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
    // The two moments the effects wait for. A held Cash Bundle or a held trap
    // is a question about these two, and nothing else answered it.
    ReplyToCommand(client, "[AP] wave running %s, credits payable %s",
        MvM_WaveInProgress() ? "yes" : "no", MvM_CanPayCredits() ? "yes" : "no");
    ReplyToCommand(client, "[AP] unlocks %s at sequence %d, %d objective(s) waiting to be sent",
        g_HaveUnlocks ? "held" : "NOT FETCHED", Bridge_Sequence(), Bridge_PendingCount());
    ReplyToCommand(client, "[AP] classes: %s", Status_ClassList());
    ReplyToCommand(client, "[AP] slots: %s", Status_SlotList());
    char missions[96], chat[64];
    Missions_Summary(missions, sizeof(missions));
    Bridge_ChatState(chat, sizeof(chat));
    ReplyToCommand(client, "[AP] missions: %s", missions);
    ReplyToCommand(client, "[AP] multiworld chat: %s", chat);
    if (lastError[0] != '\0')
    {
        ReplyToCommand(client, "[AP] Last bridge error: %s", lastError);
    }
    return Plugin_Handled;
}

// The run as a player sees it, in chat, without a round trip to the
// multiworld: the mission, the wave, what is unlocked, and whether the bridge
// is talking to Archipelago. The bridge's line comes back on its own.
static void PrintRunStatus(int client)
{
    char popFile[64], missions[96];
    MvM_PopFile(popFile, sizeof(popFile));
    Missions_Summary(missions, sizeof(missions));

    int maxWaves = g_MaxWaves > 0 ? g_MaxWaves : MvM_MaxWavesFromGame();
    if (!MvM_IsActive())
    {
        AP_PrintToClient(client, "This map is not Mann vs Machine.");
    }
    else if (g_CurrentWave > 0)
    {
        AP_PrintToClient(client, "Mission: %s, wave %d of %d.", popFile, g_CurrentWave, maxWaves);
    }
    else
    {
        AP_PrintToClient(client, "Mission: %s, between waves, %d waves in all.", popFile, maxWaves);
    }
    AP_PrintToClient(client, "Classes: %s. Slots: %s.", Status_ClassList(), Status_SlotList());
    AP_PrintToClient(client, "Missions: %s. Type !mission for the list.", missions);
    if (g_DeathLinkOn)
    {
        AP_PrintToClient(client, "Death Link is on.");
    }
    if (!g_HaveUnlocks)
    {
        AP_PrintToClient(client, "The plugin has no unlock set yet. Nothing is enforced.");
    }
    Bridge_FetchHealth(client);
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

/* A bundle without a room, for the same reason sm_ap_report exists
 *
 * Credits only ever reach a player through Bridge_PollGrants, which is an HTTP
 * poll to the bridge. A test-bed has no bridge and no room, so nothing on it can
 * ever be paid, and the whole of the bundle accounting is unreachable there:
 * whether bots are paid, whether a refund puts the ledger back, whether a bot
 * keeps its bundles across a reseat. None of it can be measured by playing.
 *
 * This is the same call the poll makes, so what it exercises is the real path
 * and not a second one written for testing.
 */
public Action Command_Bundle(int client, int argc)
{
    int amount = 200;
    if (argc >= 1)
    {
        char arg[16];
        GetCmdArg(1, arg, sizeof(arg));
        amount = StringToInt(arg);
    }
    if (amount <= 0)
    {
        ReplyToCommand(client, "[AP] Usage: sm_ap_bundle [credits], and credits must be above zero");
        return Plugin_Handled;
    }

    if (!MvM_CanPayCredits())
    {
        ReplyToCommand(client, "[AP] Nothing payable now: this wants Mann vs Machine, between waves, with somebody alive on RED.");
        return Plugin_Handled;
    }

    int paid = MvM_GrantCredits(amount);
    ReplyToCommand(client, "[AP] %d credits paid to %d defender(s).", amount, paid);
    return Plugin_Handled;
}

/* Fires a trap without waiting for one to arrive from the room.
 *
 * Calls Traps_Apply, the same path a grant takes, so what it exercises is the
 * real effect. It skips the hold on purpose: the hold is what makes a trap
 * wait for a wave, and a tester who wants to see the effect between waves has
 * to be able to.
 */
public Action Command_Trap(int client, int argc)
{
    if (argc < 1)
    {
        ReplyToCommand(client, "[AP] Usage: sm_ap_trap <key>, for example sm_ap_trap team_jarate");
        return Plugin_Handled;
    }
    char key[64];
    GetCmdArg(1, key, sizeof(key));

    if (!MvM_IsActive())
    {
        ReplyToCommand(client, "[AP] This map is not Mann vs Machine.");
        return Plugin_Handled;
    }
    if (!MvM_WaveInProgress())
    {
        ReplyToCommand(client, "[AP] No wave is running. A real trap would wait; this one fires anyway.");
    }
    if (!Traps_Apply(key))
    {
        ReplyToCommand(client, "[AP] No trap is named %s.", key);
    }
    return Plugin_Handled;
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
