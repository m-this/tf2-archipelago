/**
 * Isolated MvM wave smoke probe. Install only in a disposable test server.
 * It does not advance waves: only the game's mvm_wave_complete event passes a
 * test. Enemy bots and tanks are defeated 15-25 seconds after first sighting.
 */
#pragma semicolon 1
#pragma newdecls required

#include <sourcemod>
#include <sdktools>
#include <sdkhooks>
#include <tf2>
#include <tf2_stocks>

public Plugin myinfo =
{
    name = "TF2 MvM wave probe",
    author = "tf2-archipelago",
    description = "Disposable, unattended wave-completion smoke test",
    version = "0.1.0"
};

#define PROBE_TICK 0.25
#define PROBE_MAX_TANKS 64

enum ProbeState
{
    Probe_Idle,
    Probe_Armed,
    Probe_Running,
    Probe_Passed,
    Probe_Failed
};

ProbeState g_State;
int g_ExpectedWave;
int g_ObservedWave;
int g_Seed;
int g_Defender;
TFClassType g_DefenderClass = TFClass_Scout;
int g_PlayerTeam = 2;
int g_EnemyTeam = 3;
int g_SpawnSerial;
int g_BotKills;
int g_TankKills;
int g_BotSpawns;
int g_TankSpawns;
int g_KillAttempts;
int g_InitialEnemies;
// The robots left to kill, and since when that count has not moved.
int g_StallRemaining = -2;
float g_StallSince;
char g_FailureReason[32];
int g_BotUserId[MAXPLAYERS + 1];
bool g_BotKillPending[MAXPLAYERS + 1];
float g_BotDeadline[MAXPLAYERS + 1];
int g_BotHits[MAXPLAYERS + 1];
int g_TankRef[PROBE_MAX_TANKS];
float g_NpcDeadline[2049];
int g_TankIndex[PROBE_MAX_TANKS];
bool g_TankKillPending[PROBE_MAX_TANKS];
float g_TankDeadline[PROBE_MAX_TANKS];
float g_ArmedAt;
float g_StartedAt;

// The RED bots under test. Movement shorter than DEF_STILL_RADIUS is standing
// still; a jump longer than DEF_TELEPORT_JUMP within one tick is a teleport,
// since no class covers it on foot in PROBE_TICK.
#define DEF_STILL_RADIUS 72.0
#define DEF_TELEPORT_JUMP 500.0
// Idle is standing still, not attacking, with a robot or a tank this close. The
// probe kills robots before most reach the front, so a bot holding its post
// with nothing in reach is waiting, not idle.
#define DEF_IDLE_REACH 1500.0

// Off lets the RED bots die, so every respawn is another spawn exit to watch.
bool g_ProtectBots = true;
int g_DefUserId[MAXPLAYERS + 1];
float g_DefFirstAt[MAXPLAYERS + 1];
float g_DefLeftAt[MAXPLAYERS + 1];
bool g_DefAlive[MAXPLAYERS + 1];
int g_DefLives[MAXPLAYERS + 1];
float g_DefLifeAt[MAXPLAYERS + 1];
bool g_DefLifeLeft[MAXPLAYERS + 1];
float g_DefLeftMax[MAXPLAYERS + 1];
float g_DefAnchor[MAXPLAYERS + 1][3];
float g_DefAnchorAt[MAXPLAYERS + 1];
float g_DefStillMax[MAXPLAYERS + 1];
float g_DefStillAt[MAXPLAYERS + 1][3];
float g_DefIdleFrom[MAXPLAYERS + 1];
float g_DefIdleMax[MAXPLAYERS + 1];
float g_DefIdleAt[MAXPLAYERS + 1][3];
bool g_DefStillInSpawn[MAXPLAYERS + 1];
float g_DefLast[MAXPLAYERS + 1][3];
int g_DefTeleports[MAXPLAYERS + 1];
float g_DefHatchMin[MAXPLAYERS + 1];
float g_HatchCenter[3];
bool g_HasHatch;

public void OnPluginStart()
{
    RegAdminCmd("sm_waveprobe_arm", Command_Arm, ADMFLAG_ROOT,
        "Arm a wave test: sm_waveprobe_arm <wave> [seed]");
    RegAdminCmd("sm_waveprobe_status", Command_Status, ADMFLAG_ROOT,
        "Print machine-readable wave test state");
    RegAdminCmd("sm_waveprobe_debug", Command_Debug, ADMFLAG_ROOT,
        "Print a snapshot of the wave and living invaders");
    RegAdminCmd("sm_waveprobe_reset", Command_Reset, ADMFLAG_ROOT,
        "Stop the wave test, retaining its fake player client");
    RegAdminCmd("sm_waveprobe_wake", Command_Wake, ADMFLAG_ROOT,
        "Create a fake player: sm_waveprobe_wake <red|blue>");
    RegAdminCmd("sm_waveprobe_defenders", Command_Defenders, ADMFLAG_ROOT,
        "Print how each RED bot moved since the test was armed");
    RegAdminCmd("sm_waveprobe_protect_bots", Command_ProtectBots, ADMFLAG_ROOT,
        "Whether RED bots other than the fake player are spared damage: sm_waveprobe_protect_bots <0|1>");
    HookEvent("mvm_begin_wave", Event_BeginWave);
    HookEvent("mvm_wave_complete", Event_WaveComplete);
    HookEvent("mvm_wave_failed", Event_WaveFailed);
    HookEvent("player_spawn", Event_PlayerSpawn);
    HookEvent("player_death", Event_PlayerDeath);
    HookEvent("teamplay_flag_event", Event_FlagEvent);
    CreateTimer(PROBE_TICK, Timer_Probe, _, TIMER_REPEAT);
}

public void OnMapStart()
{
    ResetProbe();
}

public void OnPluginEnd()
{
    if (g_Defender > 0 && IsClientConnected(g_Defender))
    {
        KickClient(g_Defender, "wave probe stopped");
    }
}

public void OnClientDisconnect(int client)
{
    if (client == g_Defender) g_Defender = 0;
    g_BotUserId[client] = 0;
    g_BotKillPending[client] = false;
    g_BotDeadline[client] = 0.0;
}

public void OnClientPutInServer(int client)
{
    SDKHook(client, SDKHook_OnTakeDamage, DefenderDamage);
    SDKHook(client, SDKHook_SpawnPost, ClientSpawnPost);
    SDKHook(client, SDKHook_OnTakeDamageAlivePost, GiantHitPost);
}

public void GiantHitPost(int victim, int attacker, int inflictor, float damage, int damagetype, int weapon,
    const float damageForce[3], const float damagePosition[3], int damagecustom)
{
    if (!IsClientInGame(victim) || GetEntProp(victim, Prop_Send, "m_bIsMiniBoss") == 0 || GetClientHealth(victim) > 1) return;
    char cls[64];
    if (inflictor > 0 && IsValidEntity(inflictor)) GetEntityClassname(inflictor, cls, sizeof(cls));
    LogMessage("WAVEPROBE hit %N to %d by %d inflictor %s damage %.0f type %d custom %d", victim, GetClientHealth(victim),
        attacker, cls, damage, damagetype, damagecustom);
}

// MvM hands a dead robot's client, userid and all, to the next robot it
// spawns. When the probe missed the death between two ticks, the new robot
// inherited the old one's finished wear-down and was made to commit suicide
// at full health, past every IfHealthBelow threshold: accursed's Chief never
// killed the entity its wave 6 waits on. Each spawn starts the robot over.
public void ClientSpawnPost(int client)
{
    g_BotUserId[client] = 0;
    g_BotHits[client] = 0;
    g_BotDeadline[client] = 0.0;
    g_BotKillPending[client] = false;
}

static void ResetProbe()
{
    g_State = Probe_Idle;
    g_ExpectedWave = 0;
    g_ObservedWave = 0;
    g_SpawnSerial = 0;
    g_BotKills = 0;
    g_TankKills = 0;
    g_BotSpawns = 0;
    g_TankSpawns = 0;
    g_KillAttempts = 0;
    g_InitialEnemies = 0;
    g_StallRemaining = -2;
    g_StallSince = GetGameTime();
    g_FailureReason[0] = '\0';
    g_ArmedAt = 0.0;
    g_StartedAt = 0.0;
    for (int client = 1; client <= MaxClients; client++)
    {
        g_BotUserId[client] = 0;
        g_BotKillPending[client] = false;
        g_BotDeadline[client] = 0.0;
    }
    for (int i = 0; i < PROBE_MAX_TANKS; i++)
    {
        g_TankRef[i] = INVALID_ENT_REFERENCE;
        g_TankIndex[i] = -1;
        g_TankKillPending[i] = false;
        g_TankDeadline[i] = 0.0;
    }
    for (int client = 1; client <= MaxClients; client++)
    {
        g_DefUserId[client] = 0;
    }
    g_HasHatch = false;
}

public Action Command_Arm(int client, int argc)
{
    if (argc < 1)
    {
        ReplyToCommand(client, "usage: sm_waveprobe_arm <wave> [seed]");
        return Plugin_Handled;
    }
    char arg[24];
    GetCmdArg(1, arg, sizeof(arg));
    int wave = StringToInt(arg);
    if (wave < 1)
    {
        ReplyToCommand(client, "wave must be positive");
        return Plugin_Handled;
    }
    ResetProbe();
    g_ExpectedWave = wave;
    g_Seed = 1;
    if (argc >= 2)
    {
        GetCmdArg(2, arg, sizeof(arg));
        g_Seed = StringToInt(arg);
    }
    g_State = Probe_Armed;
    g_ArmedAt = GetGameTime();
    EnsureDefender();
    ReplyToCommand(client, "WAVEPROBE armed wave=%d seed=%d", wave, g_Seed);
    return Plugin_Handled;
}

public Action Command_Reset(int client, int argc)
{
    ResetProbe();
    ReplyToCommand(client, "WAVEPROBE reset");
    return Plugin_Handled;
}

public Action Command_Wake(int client, int argc)
{
    if (argc < 1)
    {
        ReplyToCommand(client, "usage: sm_waveprobe_wake <red|blue> [scout|medic]");
        return Plugin_Handled;
    }
    char team[16];
    GetCmdArg(1, team, sizeof(team));
    if (StrEqual(team, "blue", false))
    {
        g_PlayerTeam = 3;
        g_EnemyTeam = 2;
    }
    else if (StrEqual(team, "red", false))
    {
        g_PlayerTeam = 2;
        g_EnemyTeam = 3;
    }
    else
    {
        ReplyToCommand(client, "usage: sm_waveprobe_wake <red|blue> [scout|medic]");
        return Plugin_Handled;
    }
    g_DefenderClass = TFClass_Scout;
    if (argc >= 2)
    {
        char playerClass[16];
        GetCmdArg(2, playerClass, sizeof(playerClass));
        if (StrEqual(playerClass, "medic", false))
        {
            g_DefenderClass = TFClass_Medic;
        }
        else if (!StrEqual(playerClass, "scout", false))
        {
            ReplyToCommand(client, "usage: sm_waveprobe_wake <red|blue> [scout|medic]");
            return Plugin_Handled;
        }
    }
    EnsureDefender();
    ReplyToCommand(client, "WAVEPROBE defender=%d playerteam=%d enemyteam=%d class=%d",
        g_Defender, g_PlayerTeam, g_EnemyTeam, view_as<int>(g_DefenderClass));
    return Plugin_Handled;
}

static void EnsureDefender()
{
    if (g_Defender > 0 && IsClientConnected(g_Defender))
    {
        if (!IsClientInGame(g_Defender)) return;
        if (GetClientTeam(g_Defender) != g_PlayerTeam)
            ChangeClientTeam(g_Defender, g_PlayerTeam);
        if (TF2_GetPlayerClass(g_Defender) != g_DefenderClass)
        {
            TF2_SetPlayerClass(g_Defender, g_DefenderClass);
            TF2_RespawnPlayer(g_Defender);
        }
        return;
    }
    g_Defender = CreateFakeClient("Wave Probe Player");
    if (g_Defender == 0)
    {
        g_State = Probe_Failed;
        strcopy(g_FailureReason, sizeof(g_FailureReason), "defender_create");
        LogError("WAVEPROBE cannot create a fake player client");
    }
}

public Action DefenderDamage(int victim, int &attacker, int &inflictor,
    float &damage, int &damageType)
{
    // The probe measures invader wave progression. Protecting RED also keeps
    // authored escort NPCs such as Remedic's Chief Medic alive while we kill
    // BLU robots, so a survival objective does not invalidate that measure.
    if ((g_State == Probe_Armed || g_State == Probe_Running)
        && IsClientInGame(victim) && GetClientTeam(victim) == g_PlayerTeam
        && (g_ProtectBots || victim == g_Defender))
    {
        return Plugin_Handled;
    }
    return Plugin_Continue;
}

static void StateName(ProbeState state, char[] buffer, int length)
{
    switch (state)
    {
        case Probe_Armed: strcopy(buffer, length, "armed");
        case Probe_Running: strcopy(buffer, length, "running");
        case Probe_Passed: strcopy(buffer, length, "passed");
        case Probe_Failed: strcopy(buffer, length, "failed");
        default: strcopy(buffer, length, "idle");
    }
}

public Action Command_Status(int client, int argc)
{
    char map[64];
    char state[16];
    char pop[PLATFORM_MAX_PATH];
    int maxWaves = 0;
    int gameWave = 0;
    int redClients = 0;
    int blueClients = 0;
    int alive = 0;
    int remaining = -1;
    GetCurrentMap(map, sizeof(map));
    StateName(g_State, state, sizeof(state));
    strcopy(pop, sizeof(pop), "unknown");
    int resource = FindEntityByClassname(-1, "tf_objective_resource");
    if (resource != -1)
    {
        if (HasEntProp(resource, Prop_Send, "m_iszMvMPopfileName"))
        {
            GetEntPropString(resource, Prop_Send, "m_iszMvMPopfileName", pop, sizeof(pop));
        }
        if (HasEntProp(resource, Prop_Send, "m_nMannVsMachineMaxWaveCount"))
        {
            maxWaves = GetEntProp(resource, Prop_Send, "m_nMannVsMachineMaxWaveCount");
        }
        if (HasEntProp(resource, Prop_Send, "m_nMannVsMachineWaveCount"))
        {
            gameWave = GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveCount");
        }
        if (HasEntProp(resource, Prop_Send, "m_nMannVsMachineWaveEnemyCount"))
        {
            remaining = GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveEnemyCount");
            if (g_State == Probe_Running && remaining > g_InitialEnemies)
            {
                g_InitialEnemies = remaining;
            }
        }
    }
    ReplaceString(pop, sizeof(pop), "scripts/population/", "");
    ReplaceString(pop, sizeof(pop), ".pop", "");
    for (int player = 1; player <= MaxClients; player++)
    {
        if (!IsClientInGame(player))
        {
            continue;
        }
        if (GetClientTeam(player) == 2) redClients++;
        if (GetClientTeam(player) == 3) blueClients++;
        if (GetClientTeam(player) == g_EnemyTeam && IsPlayerAlive(player)) alive++;
    }
    int tank = -1;
    while ((tank = FindEntityByClassname(tank, "tank_boss")) != -1) alive++;
    float progress = -1.0;
    if (g_InitialEnemies > 0 && remaining >= 0)
    {
        progress = 100.0 * float(g_InitialEnemies - remaining) / float(g_InitialEnemies);
        if (progress < 0.0) progress = 0.0;
        if (progress > 100.0) progress = 100.0;
    }
    ReplyToCommand(client,
        "WAVEPROBE state=%s reason=%s map=%s pop=%s max=%d gamewave=%d expected=%d observed=%d botspawns=%d tankspawns=%d bots=%d tanks=%d attempts=%d alive=%d remaining=%d initial=%d progress=%.1f red=%d blue=%d defender=%d defteam=%d defclass=%d playerteam=%d enemyteam=%d elapsed=%.1f",
        state, g_FailureReason[0] == '\0' ? "none" : g_FailureReason,
        map, pop, maxWaves, gameWave, g_ExpectedWave,
        g_ObservedWave, g_BotSpawns, g_TankSpawns, g_BotKills, g_TankKills,
        g_KillAttempts, alive, remaining, g_InitialEnemies, progress,
        redClients, blueClients,
        g_Defender, g_Defender > 0 && IsClientInGame(g_Defender) ? GetClientTeam(g_Defender) : 0,
        g_Defender > 0 && IsClientInGame(g_Defender) ? view_as<int>(TF2_GetPlayerClass(g_Defender)) : 0,
        g_PlayerTeam, g_EnemyTeam,
        GetGameTime() - (g_StartedAt > 0.0 ? g_StartedAt : g_ArmedAt));
    return Plugin_Handled;
}

public Action Command_Debug(int client, int argc)
{
    int resource = FindEntityByClassname(-1, "tf_objective_resource");
    int wave = resource == -1 ? -1 : GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveCount");
    int remaining = resource == -1 ? -1 : GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveEnemyCount");
    ConVar timescale = FindConVar("host_timescale");
    ReplyToCommand(client,
        "WAVEPROBE_DEBUG state=%d wave=%d expected=%d game=%.1f engine=%.1f timescale=%.1f remaining=%d spawns=%d kills=%d attempts=%d defender=%d",
        view_as<int>(g_State), wave, g_ExpectedWave, GetGameTime(), GetEngineTime(),
        timescale == null ? -1.0 : timescale.FloatValue, remaining,
        g_BotSpawns + g_TankSpawns, g_BotKills + g_TankKills, g_KillAttempts, g_Defender);
    int listed;
    int alive;
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (!IsClientInGame(bot) || GetClientTeam(bot) != g_EnemyTeam || !IsPlayerAlive(bot))
        {
            continue;
        }
        alive++;
        if (listed++ >= 12) continue;
        char name[64];
        float origin[3];
        GetClientName(bot, name, sizeof(name));
        GetClientAbsOrigin(bot, origin);
        ReplyToCommand(client,
            "WAVEPROBE_BOT client=%d userid=%d class=%d hp=%d timer=%d deadline=%.1f origin=%.0f,%.0f,%.0f name=%s",
            bot, GetClientUserId(bot), view_as<int>(TF2_GetPlayerClass(bot)), GetClientHealth(bot),
            IsScriptedTimerBot(bot), g_BotDeadline[bot] - GetGameTime(),
            origin[0], origin[1], origin[2], name);
    }
    ReplyToCommand(client, "WAVEPROBE_DEBUG_END alive=%d listed=%d", alive, listed < 12 ? listed : 12);
    return Plugin_Handled;
}

public void Event_BeginWave(Event event, const char[] name, bool dontBroadcast)
{
    if (g_State != Probe_Armed)
    {
        return;
    }
    g_ObservedWave = event.GetInt("wave_index") + 1;
    if (g_ObservedWave != g_ExpectedWave)
    {
        g_State = Probe_Failed;
        strcopy(g_FailureReason, sizeof(g_FailureReason), "wrong_wave");
        LogError("WAVEPROBE wrong wave: expected %d, started %d",
            g_ExpectedWave, g_ObservedWave);
        return;
    }
    if (g_Defender == 0 || !IsClientInGame(g_Defender)
        || GetClientTeam(g_Defender) != g_PlayerTeam)
    {
        g_State = Probe_Failed;
        strcopy(g_FailureReason, sizeof(g_FailureReason), "defender_missing");
        LogError("WAVEPROBE fake player is not on expected team %d", g_PlayerTeam);
        return;
    }
    g_State = Probe_Running;
    g_StartedAt = GetGameTime();
    int resource = FindEntityByClassname(-1, "tf_objective_resource");
    if (resource != -1 && HasEntProp(resource, Prop_Send, "m_nMannVsMachineWaveEnemyCount"))
    {
        g_InitialEnemies = GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveEnemyCount");
    }
    LogMessage("WAVEPROBE started wave=%d seed=%d", g_ObservedWave, g_Seed);
}

public void Event_WaveComplete(Event event, const char[] name, bool dontBroadcast)
{
    if (g_State != Probe_Running)
    {
        return;
    }
    g_State = Probe_Passed;
    LogMessage("WAVEPROBE passed wave=%d seconds=%.1f bots=%d tanks=%d",
        g_ObservedWave, GetGameTime() - g_StartedAt, g_BotKills, g_TankKills);
}

public void Event_WaveFailed(Event event, const char[] name, bool dontBroadcast)
{
    if (g_State == Probe_Running)
    {
        g_State = Probe_Failed;
        strcopy(g_FailureReason, sizeof(g_FailureReason), "wave_failed");
        LogError("WAVEPROBE game reported wave failed");
    }
}

public void Event_PlayerDeath(Event event, const char[] name, bool dontBroadcast)
{
    int bot = GetClientOfUserId(event.GetInt("userid"));
    if (bot > 0 && IsClientInGame(bot) && GetEntProp(bot, Prop_Send, "m_bIsMiniBoss") != 0)
    {
        char weapon[64];
        event.GetString("weapon", weapon, sizeof(weapon));
        float origin[3];
        GetClientAbsOrigin(bot, origin);
        LogMessage("WAVEPROBE death %N step %d attacker %d weapon %s custom %d bits %d at %.0f %.0f %.0f", bot, g_BotHits[bot],
            GetClientOfUserId(event.GetInt("attacker")), weapon, event.GetInt("customkill"), event.GetInt("damagebits"),
            origin[0], origin[1], origin[2]);
    }
    if (bot > 0 && g_BotKillPending[bot])
    {
        g_BotKills++;
        g_BotKillPending[bot] = false;
    }
    if (bot > 0)
    {
        g_BotUserId[bot] = 0;
        g_BotDeadline[bot] = 0.0;
    }
}

// tf_bot_flag_kill_on_touch makes a robot that picks up the bomb commit
// suicide, which keeps an unguarded hatch from ending a population test. A
// boss with phases dies that way at full health, past every IfHealthBelow
// its popfile names: accursed's Chief took the bomb at its spawn and never
// killed the entity wave 6 waits on. So a robot with a boss health bar drops
// the bomb as it takes it, and lives until the wear-down reaches it.
public void Event_FlagEvent(Event event, const char[] name, bool dontBroadcast)
{
    if (g_State != Probe_Running || event.GetInt("eventtype") != 1) return;
    int carrier = event.GetInt("player");
    if (carrier < 1 || carrier > MaxClients || !IsClientInGame(carrier) || GetClientTeam(carrier) != g_EnemyTeam
        || !HasEntProp(carrier, Prop_Send, "m_bUseBossHealthBar") || GetEntProp(carrier, Prop_Send, "m_bUseBossHealthBar") == 0)
    {
        return;
    }
    int flag = -1;
    while ((flag = FindEntityByClassname(flag, "item_teamflag")) != -1)
    {
        if (GetEntPropEnt(flag, Prop_Send, "moveparent") == carrier || GetEntPropEnt(flag, Prop_Send, "m_hOwnerEntity") == carrier)
        {
            AcceptEntityInput(flag, "ForceDrop");
        }
    }
}

public void Event_PlayerSpawn(Event event, const char[] name, bool dontBroadcast)
{
    int bot = GetClientOfUserId(event.GetInt("userid"));
    if (bot > 0) RegisterEnemyBot(bot);
}

static void RegisterEnemyBot(int bot)
{
    if (g_State != Probe_Running || !IsWaveRobot(bot))
    {
        return;
    }
    int userid = GetClientUserId(bot);
    if (g_BotUserId[bot] == userid) return;
    g_BotUserId[bot] = userid;
    g_BotDeadline[bot] = GetGameTime() + KillDelay();
    g_BotHits[bot] = 0;
    g_BotSpawns++;
    g_BotKillPending[bot] = false;
    CreateTimer(0.1, Timer_MarkScriptedBot, userid, TIMER_FLAG_NO_MAPCHANGE);
}

// SigMod's TimerBot is a wave clock, not an enemy the player is meant to
// defeat. Its "timer" tag is visible to VScript, so mark it through the same
// RunScriptCode input the authored missions use and leave it alive while the
// real room groups are cleared. The game retires it at wave completion.
// MvM gives a bot client the next robot when one dies, with the same userid,
// and the targetname stays on the client: every spawn decides the mark again,
// or a Giant Pyro spawned into a timer's old slot was left alive for good.
public Action Timer_MarkScriptedBot(Handle timer, any userid)
{
    int bot = GetClientOfUserId(userid);
    if (bot > 0 && IsClientInGame(bot) && IsWaveRobot(bot))
    {
        SetVariantString("if (self.HasBotTag(\"timer\")) self.KeyValueFromString(\"targetname\", \"waveprobe_timer\"); else if (self.GetName() == \"waveprobe_timer\") self.KeyValueFromString(\"targetname\", \"\")");
        AcceptEntityInput(bot, "RunScriptCode");
    }
    return Plugin_Stop;
}

// The wave's icons in the objective resource: support (1 << 1) and limited
// support (1 << 5) are the robots a wave does not wait on.
#define PROBE_ICON_SUPPORT ((1 << 1) | (1 << 5))

static bool IsSupportIcon(const char[] icon)
{
    int resource = FindEntityByClassname(-1, "tf_objective_resource");
    if (resource == -1) return false;
    static const char names[][] = { "m_iszMannVsMachineWaveClassNames", "m_iszMannVsMachineWaveClassNames2" };
    static const char flags[][] = { "m_nMannVsMachineWaveClassFlags", "m_nMannVsMachineWaveClassFlags2" };
    char listed[64];
    for (int t = 0; t < sizeof(names); t++)
    {
        if (!HasEntProp(resource, Prop_Send, names[t])) continue;
        int count = GetEntPropArraySize(resource, Prop_Send, names[t]);
        for (int i = 0; i < count; i++)
        {
            GetEntPropString(resource, Prop_Send, names[t], listed, sizeof(listed), i);
            if (StrEqual(listed, icon, false)
                && (GetEntProp(resource, Prop_Send, flags[t], _, i) & PROBE_ICON_SUPPORT) != 0)
            {
                return true;
            }
        }
    }
    return false;
}

// A robot a wave spawned, on the enemy's team, the gray one or the players'.
// The populator gives every robot it spawns a class icon; the defender bots
// and the probe's own player have none. A robot on the players' team is an
// ally: trespasser lost its wave when the probe killed its survivors, which
// are support, and its finale waits on the military it sends, which are not.
// So an ally whose icon the wave lists as support is left alone.
static bool IsWaveRobot(int bot)
{
    if (bot == g_Defender || !IsClientInGame(bot)) return false;
    int team = GetClientTeam(bot);
    if (team == g_EnemyTeam) return true;
    if (team < 1 || !IsFakeClient(bot) || !HasEntProp(bot, Prop_Send, "m_iszClassIcon")) return false;
    char icon[64];
    GetEntPropString(bot, Prop_Send, "m_iszClassIcon", icon, sizeof(icon));
    if (icon[0] == '\0') return false;
    return team != g_PlayerTeam || !IsSupportIcon(icon);
}

// Players wear a robot down, and a scripted boss changes phase at the health
// thresholds its popfile names (IfHealthBelow): a suicide went past all of
// them, and a wave waiting on the next phase never ended. So the probe sets
// the robot's health an eighth lower each half second, which no resistance
// or buddha mode turns aside, and then makes it commit suicide.
#define PROBE_WEAR_STEPS 8

static void WearDown(int bot, float now)
{
    int health = GetClientHealth(bot);
    if (g_BotHits[bot] >= PROBE_WEAR_STEPS || health <= 1)
    {
        ForcePlayerSuicide(bot);
        g_BotDeadline[bot] = now + 5.0;
        return;
    }
    int maxHealth = GetEntProp(bot, Prop_Data, "m_iMaxHealth");
    if (maxHealth < health) maxHealth = health;
    int step = maxHealth / PROBE_WEAR_STEPS;
    if (step < 1) step = 1;
    int next = health - step;
    SetEntityHealth(bot, next < 1 ? 1 : next);
    if (maxHealth >= 1000)
    {
        LogMessage("WAVEPROBE wear %N step %d read %d set %d now %d", bot, g_BotHits[bot], health, next < 1 ? 1 : next, GetClientHealth(bot));
    }
    g_BotHits[bot]++;
    g_BotDeadline[bot] = now + 0.5;
}

static bool IsScriptedTimerBot(int bot)
{
    char name[64];
    GetEntPropString(bot, Prop_Data, "m_iName", name, sizeof(name));
    return StrEqual(name, "waveprobe_timer");
}

public void OnEntityCreated(int entity, const char[] classname)
{
    if (g_State == Probe_Running && StrEqual(classname, "tank_boss"))
    {
        RegisterTank(entity);
    }
}

static int RegisterTank(int tank)
{
    // A tank with no edict (SigMod makes some entities server-side only)
    // comes back from FindEntityByClassname as a reference already, and
    // EntIndexToEntRef throws on it, which ended the probe's tick before it
    // could kill the tank.
    int ref = tank < 0 ? tank : EntIndexToEntRef(tank);
    int slot = -1;
    for (int i = 0; i < PROBE_MAX_TANKS; i++)
    {
        if (g_TankRef[i] == ref) return i;
        if (slot < 0 && (g_TankRef[i] == INVALID_ENT_REFERENCE
            || EntRefToEntIndex(g_TankRef[i]) == INVALID_ENT_REFERENCE))
        {
            slot = i;
        }
    }
    if (slot < 0)
    {
        g_State = Probe_Failed;
        strcopy(g_FailureReason, sizeof(g_FailureReason), "tank_overflow");
        LogError("WAVEPROBE more than %d active tanks", PROBE_MAX_TANKS);
        return -1;
    }
    g_TankRef[slot] = ref;
    g_TankIndex[slot] = tank;
    g_TankDeadline[slot] = GetGameTime() + KillDelay();
    g_TankSpawns++;
    g_TankKillPending[slot] = false;
    return slot;
}

public void OnEntityDestroyed(int entity)
{
    if (entity > 0 && entity <= 2048) g_NpcDeadline[entity] = 0.0;
    for (int i = 0; i < PROBE_MAX_TANKS; i++)
    {
        if (g_TankKillPending[i] && g_TankIndex[i] == entity)
        {
            g_TankKills++;
            g_TankKillPending[i] = false;
            g_TankRef[i] = INVALID_ENT_REFERENCE;
            g_TankIndex[i] = -1;
            return;
        }
    }
}

// The spawn ordinal and seed determine a stable 15-25 second delay. Random
// runtime state would make a failed mission harder to reproduce.
static float KillDelay()
{
    g_SpawnSerial++;
    int value = (g_SpawnSerial * 1103515245 + g_Seed * 12345) & 0x7fffffff;
    return 15.0 + float(value % 1001) / 100.0;
}

// Game seconds the wave's robot count may stand still before the probe does
// what only a player can: stand on a capture area the players' team takes.
#define PROBE_CAPTURE_STALL 300.0

// Trespasser's finale ends when RED captures its landing zone, a capture area
// the players reach; no robot the probe kills moves the wave past it. When the
// count has stood still that long, every enabled capture area fires the output
// a capture by the players' team fires, and the probe waits as long again.
static void CaptureWhenStalled(float now)
{
    int resource = FindEntityByClassname(-1, "tf_objective_resource");
    int remaining = resource == -1 ? -1 : GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveEnemyCount");
    if (remaining != g_StallRemaining)
    {
        g_StallRemaining = remaining;
        g_StallSince = now;
        return;
    }
    if (now - g_StallSince < PROBE_CAPTURE_STALL) return;
    g_StallSince = now;
    char output[16];
    strcopy(output, sizeof(output), g_PlayerTeam == 2 ? "OnCapTeam1" : "OnCapTeam2");
    int area = -1;
    while ((area = FindEntityByClassname(area, "trigger_capture_area")) != -1)
    {
        if (HasEntProp(area, Prop_Data, "m_bDisabled") && GetEntProp(area, Prop_Data, "m_bDisabled") != 0) continue;
        LogMessage("WAVEPROBE wave stood still at %d robots: %s on capture area %d", remaining, output, area);
        FireEntityOutput(area, output, g_Defender > 0 ? g_Defender : -1);
    }
}

public Action Timer_Probe(Handle timer)
{
    if (g_Defender > 0) EnsureDefender();
    if (g_State != Probe_Armed && g_State != Probe_Running)
    {
        return Plugin_Continue;
    }
    EnsureDefender();
    if (g_Defender > 0 && IsClientInGame(g_Defender)
        && !IsPlayerAlive(g_Defender))
    {
        TF2_RespawnPlayer(g_Defender);
    }
    SampleDefenders();
    if (g_State != Probe_Running)
    {
        return Plugin_Continue;
    }

    float now = GetGameTime();
    CaptureWhenStalled(now);
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (!IsClientInGame(bot) || !IsWaveRobot(bot) || !IsPlayerAlive(bot))
        {
            g_BotUserId[bot] = 0;
            g_BotKillPending[bot] = false;
            continue;
        }
        RegisterEnemyBot(bot);
        if (!IsScriptedTimerBot(bot) && now >= g_BotDeadline[bot])
        {
            if (!g_BotKillPending[bot])
            {
                g_BotKillPending[bot] = true;
                g_KillAttempts++;
            }
            WearDown(bot, now);
        }
    }

    // A wave spawn's skeletons and Halloween bosses are NPCs, neither players
    // nor tanks, and the wave spawn waits for them to die as for any robot:
    // dreadwood's wave 3 waited on thirty skeletons until the time limit, and
    // trespasser's first wave on a Horseless Headless Horsemann.
    static const char npcClasses[][] = { "tf_zombie", "headless_hatman", "eyeball_boss", "merasmus" };
    for (int c = 0; c < sizeof(npcClasses); c++)
    {
        int npc = -1;
        while ((npc = FindEntityByClassname(npc, npcClasses[c])) != -1)
        {
            if (npc <= MaxClients || npc > 2048) continue;
            if (GetEntProp(npc, Prop_Send, "m_iTeamNum") == g_PlayerTeam) continue;
            if (g_NpcDeadline[npc] == 0.0)
            {
                g_NpcDeadline[npc] = now + 15.0;
            }
            else if (now >= g_NpcDeadline[npc])
            {
                g_KillAttempts++;
                SDKHooks_TakeDamage(npc, g_Defender, g_Defender, 1000000.0);
                g_NpcDeadline[npc] = now + 5.0;
            }
        }
    }

    int tank = -1;
    while ((tank = FindEntityByClassname(tank, "tank_boss")) != -1)
    {
        int slot = RegisterTank(tank);
        if (slot < 0) return Plugin_Continue;
        if (now >= g_TankDeadline[slot])
        {
            g_TankKillPending[slot] = true;
            g_KillAttempts++;
            SDKHooks_TakeDamage(tank, g_Defender, g_Defender, 1000000.0);
            g_TankDeadline[slot] = now + 5.0;
        }
    }
    return Plugin_Continue;
}

static bool IsDefenderBot(int client)
{
    return client != g_Defender && IsClientInGame(client) && IsFakeClient(client)
        && !IsClientSourceTV(client) && GetClientTeam(client) == g_PlayerTeam;
}

static bool EntityBounds(int entity, float mins[3], float maxs[3])
{
    if (!HasEntProp(entity, Prop_Data, "m_vecMins") || !HasEntProp(entity, Prop_Data, "m_vecMaxs"))
    {
        return false;
    }
    float origin[3];
    GetEntPropVector(entity, Prop_Data, "m_vecAbsOrigin", origin);
    GetEntPropVector(entity, Prop_Data, "m_vecMins", mins);
    GetEntPropVector(entity, Prop_Data, "m_vecMaxs", maxs);
    AddVectors(mins, origin, mins);
    AddVectors(maxs, origin, maxs);
    return true;
}

// The same rooms the defender mod's spawn-exit watch reads: enabled
// func_respawnroom brushes owned by the bots' team or by nobody.
static bool InDefenderSpawn(const float point[3])
{
    int room = -1;
    while ((room = FindEntityByClassname(room, "func_respawnroom")) != -1)
    {
        int team = GetEntProp(room, Prop_Send, "m_iTeamNum");
        if (team != 0 && team != g_PlayerTeam) continue;
        if (HasEntProp(room, Prop_Data, "m_bDisabled") && GetEntProp(room, Prop_Data, "m_bDisabled") != 0) continue;
        float mins[3], maxs[3];
        if (!EntityBounds(room, mins, maxs)) continue;
        if (point[0] >= mins[0] && point[0] <= maxs[0] && point[1] >= mins[1] && point[1] <= maxs[1]
            && point[2] >= mins[2] && point[2] <= maxs[2])
        {
            return true;
        }
    }
    return false;
}

// The hatch: where the invaders deliver the bomb.
static void FindHatch()
{
    int zone = -1;
    while ((zone = FindEntityByClassname(zone, "func_capturezone")) != -1)
    {
        int team = GetEntProp(zone, Prop_Send, "m_iTeamNum");
        if (team != 0 && team != g_EnemyTeam) continue;
        float mins[3], maxs[3];
        if (!EntityBounds(zone, mins, maxs)) continue;
        for (int axis = 0; axis < 3; axis++)
        {
            g_HatchCenter[axis] = (mins[axis] + maxs[axis]) * 0.5;
        }
        g_HasHatch = true;
        return;
    }
}

static float FlatDistance(const float a[3], const float b[3])
{
    float dx = a[0] - b[0];
    float dy = a[1] - b[1];
    return SquareRoot(dx * dx + dy * dy);
}

static void SampleDefenders()
{
    if (!g_HasHatch) FindHatch();
    float now = GetGameTime();
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (!IsDefenderBot(bot)) continue;
        if (!IsPlayerAlive(bot))
        {
            g_DefAlive[bot] = false;
            continue;
        }
        float here[3];
        GetClientAbsOrigin(bot, here);
        float center[3];
        center = here;
        center[2] += 40.0;
        bool inSpawn = InDefenderSpawn(center);
        int userid = GetClientUserId(bot);
        if (g_DefUserId[bot] != userid)
        {
            g_DefUserId[bot] = userid;
            g_DefFirstAt[bot] = now;
            g_DefLeftAt[bot] = -1.0;
            g_DefAnchor[bot] = here;
            g_DefAnchorAt[bot] = now;
            g_DefStillMax[bot] = 0.0;
            g_DefStillAt[bot] = here;
            g_DefStillInSpawn[bot] = inSpawn;
            g_DefIdleFrom[bot] = now;
            g_DefIdleMax[bot] = 0.0;
            g_DefIdleAt[bot] = here;
            g_DefLast[bot] = here;
            g_DefTeleports[bot] = 0;
            g_DefHatchMin[bot] = -1.0;
            g_DefAlive[bot] = false;
            g_DefLives[bot] = 0;
            g_DefLeftMax[bot] = 0.0;
        }
        if (!g_DefAlive[bot])
        {
            g_DefAlive[bot] = true;
            g_DefLives[bot]++;
            g_DefLifeAt[bot] = now;
            g_DefLifeLeft[bot] = false;
            g_DefLast[bot] = here;
            g_DefAnchor[bot] = here;
            g_DefAnchorAt[bot] = now;
            g_DefIdleFrom[bot] = now;
        }
        if (g_DefLeftAt[bot] < 0.0 && !inSpawn)
        {
            g_DefLeftAt[bot] = now;
        }
        if (!g_DefLifeLeft[bot] && !inSpawn)
        {
            g_DefLifeLeft[bot] = true;
            g_DefLeftMax[bot] = FloatMax(g_DefLeftMax[bot], now - g_DefLifeAt[bot]);
        }
        if (GetVectorDistance(here, g_DefLast[bot]) > DEF_TELEPORT_JUMP)
        {
            g_DefTeleports[bot]++;
            g_DefAnchor[bot] = here;
            g_DefAnchorAt[bot] = now;
            g_DefIdleFrom[bot] = now;
        }
        g_DefLast[bot] = here;
        if (g_HasHatch)
        {
            float hatch = GetVectorDistance(here, g_HatchCenter);
            if (g_DefHatchMin[bot] < 0.0 || hatch < g_DefHatchMin[bot]) g_DefHatchMin[bot] = hatch;
        }
        // Standing still only counts once the wave runs: before it, waiting at
        // the front is the job.
        if (g_State != Probe_Running || FlatDistance(here, g_DefAnchor[bot]) > DEF_STILL_RADIUS)
        {
            g_DefAnchor[bot] = here;
            g_DefAnchorAt[bot] = now;
            g_DefIdleFrom[bot] = now;
            continue;
        }
        float still = now - g_DefAnchorAt[bot];
        if (still > g_DefStillMax[bot])
        {
            g_DefStillMax[bot] = still;
            g_DefStillAt[bot] = g_DefAnchor[bot];
            g_DefStillInSpawn[bot] = inSpawn;
        }
        if ((GetClientButtons(bot) & (IN_ATTACK | IN_ATTACK2)) != 0 || !EnemyInReach(here))
        {
            g_DefIdleFrom[bot] = now;
            continue;
        }
        float idle = now - g_DefIdleFrom[bot];
        if (idle > g_DefIdleMax[bot])
        {
            g_DefIdleMax[bot] = idle;
            g_DefIdleAt[bot] = here;
        }
    }
}

static bool EnemyInReach(const float here[3])
{
    float there[3];
    for (int enemy = 1; enemy <= MaxClients; enemy++)
    {
        if (!IsClientInGame(enemy) || GetClientTeam(enemy) != g_EnemyTeam || !IsPlayerAlive(enemy)) continue;
        GetClientAbsOrigin(enemy, there);
        if (GetVectorDistance(here, there) <= DEF_IDLE_REACH) return true;
    }
    for (int i = 0; i < PROBE_MAX_TANKS; i++)
    {
        int tank = EntRefToEntIndex(g_TankRef[i]);
        if (tank == INVALID_ENT_REFERENCE || tank <= 0) continue;
        GetEntPropVector(tank, Prop_Data, "m_vecAbsOrigin", there);
        if (GetVectorDistance(here, there) <= DEF_IDLE_REACH) return true;
    }
    return false;
}

// One line per RED bot, times in game seconds since the bot was first seen.
// left is -1 for a bot that never left its spawn room.
public Action Command_Defenders(int client, int argc)
{
    float now = GetGameTime();
    for (int bot = 1; bot <= MaxClients; bot++)
    {
        if (!IsDefenderBot(bot) || g_DefUserId[bot] != GetClientUserId(bot)) continue;
        float here[3];
        GetClientAbsOrigin(bot, here);
        float center[3];
        center = here;
        center[2] += 40.0;
        char name[64];
        GetClientName(bot, name, sizeof(name));
        ReplyToCommand(client,
            "WAVEPROBE_DEF client=%d class=%d alive=%d seen=%.1f left=%.1f lives=%d leftmax=%.1f spawnnow=%.1f inspawn=%d stillmax=%.1f stillspawn=%d stillnow=%.1f still=%.0f,%.0f,%.0f idlemax=%.1f idle=%.0f,%.0f,%.0f at=%.0f,%.0f,%.0f hatchmin=%.0f hatchnow=%.0f teleports=%d name=%s",
            bot, view_as<int>(TF2_GetPlayerClass(bot)), IsPlayerAlive(bot),
            now - g_DefFirstAt[bot],
            g_DefLeftAt[bot] < 0.0 ? -1.0 : g_DefLeftAt[bot] - g_DefFirstAt[bot],
            g_DefLives[bot], g_DefLeftMax[bot],
            IsPlayerAlive(bot) && !g_DefLifeLeft[bot] ? now - g_DefLifeAt[bot] : 0.0,
            InDefenderSpawn(center), g_DefStillMax[bot], g_DefStillInSpawn[bot],
            g_State == Probe_Running ? now - g_DefAnchorAt[bot] : 0.0,
            g_DefStillAt[bot][0], g_DefStillAt[bot][1], g_DefStillAt[bot][2],
            g_DefIdleMax[bot], g_DefIdleAt[bot][0], g_DefIdleAt[bot][1], g_DefIdleAt[bot][2],
            here[0], here[1], here[2],
            g_DefHatchMin[bot], g_HasHatch ? GetVectorDistance(here, g_HatchCenter) : -1.0,
            g_DefTeleports[bot], name);
    }
    ReplyToCommand(client, "WAVEPROBE_DEF_END hatch=%d", g_HasHatch);
    return Plugin_Handled;
}

static float FloatMax(float a, float b)
{
    return a > b ? a : b;
}

public Action Command_ProtectBots(int client, int argc)
{
    if (argc < 1)
    {
        ReplyToCommand(client, "WAVEPROBE protect_bots=%d", g_ProtectBots);
        return Plugin_Handled;
    }
    char arg[8];
    GetCmdArg(1, arg, sizeof(arg));
    g_ProtectBots = StringToInt(arg) != 0;
    ReplyToCommand(client, "WAVEPROBE protect_bots=%d", g_ProtectBots);
    return Plugin_Handled;
}
