/** Seed-generated rules attached to the mission currently being played. */

#define MissionModifier_LowGravityValue 400.0
#define MissionModifier_HighGravityValue 1200.0
#define MissionModifier_FragileDamageScale 1.25
#define MissionModifier_ResistanceScale 0.50
#define MissionModifier_OverclockedSpeedScale 1.15
#define MissionModifier_MiniSpeedScale 1.25
#define MissionModifier_MiniHealthScale 0.67
#define MissionModifier_MiniModelScale 0.50
#define MissionModifier_RobotCritChance 0.33
#define MissionModifier_PlayerSpreadPenalty 2.0
#define MissionModifier_FirstShotSpreadScale 1.0
#define MissionModifier_ProjectileDeviationDegrees 5.0
#define MissionModifier_GrenadeDeviationDegrees 9.0
#define MissionModifier_FlameSpreadDegrees 8.0
#define MissionModifier_SniperDeviationDegrees 1.95
#define MissionModifier_LooseFootingFriction 0.25
#define MissionModifier_LooseFootingAcceleration 2.5
#define MissionModifier_LooseFootingForceScale 3.0
// Six RED seats and four spare client slots let defender bots turn over mid-wave
// on a 32-slot server without fighting the population manager for BLU slots.
#define MissionModifier_SurgeMaxInvaders 22
#define MissionModifier_SurgeSupportBudget 22
#define MissionModifier_SurgeStallSeconds 20.0
#define MissionModifier_SurgeRecoverySeconds 4.0
#define MissionModifier_SurgeSupportStartDelay 0.0
#define MissionModifier_SurgeSupportSpawnInterval 0.10
// A robot's SigMod templates can outlive it: Brain Taker's crumpkins wait for a
// RED player to collect them, six edicts each. At Bot Surge's refill rate they
// pile up until ED_Alloc runs out of the engine's 2048 and srcds dies, so the
// surge stands down near the limit and resumes once the map has room again.
#define MissionModifier_SurgeEdictsThrottle 1800
#define MissionModifier_SurgeEdictsResume 1500
#define MissionModifier_MaxWaves 64
#define MissionModifier_MaxEntities 2049
#define MissionModifier_TankRoofHeight 145.0
#define MissionModifier_TankSentryClearance 12.0
#define MissionModifier_BotSurgeHudInterval 0.25

enum struct MissionModifierSupportSpawner
{
    Address address;
    int wave;
    int maxActive;
    int waitBeforeBits;
    int waitBits;
    bool synthetic;
}

enum struct MissionModifierTimedSpawner
{
    Address address;
    int waitBeforeBits;
    int waitBits;
}

int g_ActiveMissionModifiers;
int g_SeedMissionModifiers;
bool g_DebugMissionModifierOverride;
char g_ModifierMission[64];
ConVar g_CvarGravity;
ConVar g_CvarFriction;
ConVar g_CvarAccelerate;
ConVar g_CvarMaxInvaders;
ConVar g_CvarSigRobotLimitOverride;
ConVar g_CvarSigSpawnStallFix;
float g_DefaultGravity = 800.0;
float g_DefaultFriction = 4.0;
float g_DefaultAccelerate = 10.0;
int g_DefaultMaxInvaders = 22;
int g_DefaultSigRobotLimitOverride;
bool g_DefaultSigSpawnStallFix;
bool g_RobotSpeedApplied[MAXPLAYERS + 1];
bool g_RobotHadSpeedAttribute[MAXPLAYERS + 1];
float g_RobotOriginalSpeed[MAXPLAYERS + 1];
bool g_RobotScaleApplied[MAXPLAYERS + 1];
float g_RobotOriginalScale[MAXPLAYERS + 1];
bool g_InaccuracyApplied[MAXPLAYERS + 1];
bool g_InaccuracyHadSpread[MAXPLAYERS + 1];
bool g_InaccuracyHadFirstShotSpread[MAXPLAYERS + 1];
bool g_InaccuracyHadProjectileSpread[MAXPLAYERS + 1];
bool g_InaccuracyHadFlameSpread[MAXPLAYERS + 1];
float g_InaccuracyOriginalSpread[MAXPLAYERS + 1];
float g_InaccuracyOriginalFirstShotSpread[MAXPLAYERS + 1];
float g_InaccuracyOriginalProjectileSpread[MAXPLAYERS + 1];
float g_InaccuracyOriginalFlameSpread[MAXPLAYERS + 1];
bool g_LooseFootingApplied[MAXPLAYERS + 1];
bool g_LooseFootingHadDamageForce[MAXPLAYERS + 1];
bool g_LooseFootingHadSelfDamageForce[MAXPLAYERS + 1];
bool g_LooseFootingHadAirblast[MAXPLAYERS + 1];
bool g_LooseFootingHadVerticalAirblast[MAXPLAYERS + 1];
float g_LooseFootingOriginalDamageForce[MAXPLAYERS + 1];
float g_LooseFootingOriginalSelfDamageForce[MAXPLAYERS + 1];
float g_LooseFootingOriginalAirblast[MAXPLAYERS + 1];
float g_LooseFootingOriginalVerticalAirblast[MAXPLAYERS + 1];
bool g_TankMiniApplied[MissionModifier_MaxEntities];
float g_TankOriginalScale[MissionModifier_MaxEntities];
float g_TankOriginalSpeed[MissionModifier_MaxEntities];
int g_TankOriginalHealth[MissionModifier_MaxEntities];
int g_TankOriginalMaxHealth[MissionModifier_MaxEntities];
int g_TankSentryRef[MissionModifier_MaxEntities];
Handle g_BotSurgeHud;
bool g_BotSurgeHudShown[MAXPLAYERS + 1];
bool g_BotCountHudEnabled[MAXPLAYERS + 1];
int g_WaveSpawnMaxActiveOffset = -1;
int g_WaveSpawnSpawnCountOffset = -1;
int g_WaveSpawnHasTFBotOffset = -1;
int g_WaveSpawnWaitBeforeOffset = -1;
int g_WaveSpawnWaitOffset = -1;
int g_WaveSpawnSupportOffset = -1;
int g_WaveSpawnLimitedSupportOffset = -1;
int g_WaveSpawnActiveBotsOffset = -1;
int g_WaveSpawnActiveBotCountOffset = -1;
float g_BotSurgeLastProgress;
float g_BotSurgeRecoveryUntil;
int g_BotSurgeMainRemaining = -1;
int g_BotSurgeWave;
bool g_BotSurgeCulled[MAXPLAYERS + 1];
ArrayList g_BotSurgeSupportSpawners;
ArrayList g_BotSurgeTimedSpawners;
bool g_BotSurgeReloadQueued;
bool g_BotSurgeMissionStarted;
bool g_BotSurgeThrottled;
Handle g_KVMakeCopy;
Handle g_KVAddSubKey;
Handle g_KVSetInt;
Handle g_KVSetFloat;
Handle g_KVSetString;
Handle g_KVFindKey;
Handle g_KVRemoveSubKey;
Handle g_KVDeleteThis;
Handle g_RecipientFilterAddRecipient;
bool g_SniperTracerApplied[MissionModifier_MaxEntities];
bool g_SniperTracerHadAttribute[MissionModifier_MaxEntities];
float g_SniperTracerOriginalAttribute[MissionModifier_MaxEntities];
Address g_BotSurgeWaveCandidate[MissionModifier_MaxWaves];
Address g_BotSurgeWaveCandidateSource[MissionModifier_MaxWaves];
int g_BotSurgeWaveCandidateMax[MissionModifier_MaxWaves];
int g_BotSurgeWaveCandidateSpawnCount[MissionModifier_MaxWaves];
bool g_BotSurgeWaveHasSupport[MissionModifier_MaxWaves];
Address g_BotSurgeSyntheticKV[MissionModifier_MaxWaves];
int g_BotSurgeCapturedWaves;
int g_BotSurgeParsingWave = -1;
bool g_BotSurgeSynthesisInstalled;

void MissionModifiers_Init()
{
    g_BotSurgeSupportSpawners = new ArrayList(sizeof(MissionModifierSupportSpawner));
    g_BotSurgeTimedSpawners = new ArrayList(sizeof(MissionModifierTimedSpawner));
    g_CvarGravity = FindConVar("sv_gravity");
    g_CvarFriction = FindConVar("sv_friction");
    g_CvarAccelerate = FindConVar("sv_accelerate");
    g_CvarMaxInvaders = FindConVar("tf_mvm_max_invaders");
    g_CvarSigRobotLimitOverride = FindConVar("sig_mvm_robot_limit_override");
    g_CvarSigSpawnStallFix = FindConVar("sig_pop_wavespawn_spawnbot_stall_fix");
    if (g_CvarGravity != null)
    {
        g_DefaultGravity = g_CvarGravity.FloatValue;
    }
    if (g_CvarFriction != null)
    {
        g_DefaultFriction = g_CvarFriction.FloatValue;
    }
    if (g_CvarAccelerate != null)
    {
        g_DefaultAccelerate = g_CvarAccelerate.FloatValue;
    }
    if (g_CvarMaxInvaders != null)
    {
        g_DefaultMaxInvaders = g_CvarMaxInvaders.IntValue;
    }
    if (g_CvarSigRobotLimitOverride != null)
    {
        g_DefaultSigRobotLimitOverride = g_CvarSigRobotLimitOverride.IntValue;
    }
    if (g_CvarSigSpawnStallFix != null)
    {
        g_DefaultSigSpawnStallFix = g_CvarSigSpawnStallFix.BoolValue;
    }
    MissionModifiers_InitBotSurgeSupportHook();
    MissionModifiers_InitSniperInaccuracyHook();
    Chamber_Init();
    Mental_Init();
    CreateTimer(0.05, Mental_Timer, _, TIMER_REPEAT);
    g_BotSurgeHud = CreateHudSynchronizer();
    CreateTimer(MissionModifier_BotSurgeHudInterval,
        Timer_MissionModifiersBotSurgeHud, _, TIMER_REPEAT);
    CreateTimer(1.0, Timer_MissionModifiersBotSurgeRecovery, _, TIMER_REPEAT);
    for (int client = 1; client <= MaxClients; client++)
    {
        if (IsClientInGame(client))
        {
            MissionModifiers_HookClient(client);
        }
    }
}

static void MissionModifiers_InitSniperInaccuracyHook()
{
    GameData config = LoadGameConfigFile("tf2_archipelago");
    DynamicDetour fireBullet = config != null
        ? DynamicDetour.FromConf(config, "FX_FireBullets") : null;
    DynamicDetour railBeam = config != null
        ? DynamicDetour.FromConf(config, "CTFPlayer::MaybeDrawRailgunBeam") : null;
    if (config != null)
    {
        StartPrepSDKCall(SDKCall_Raw);
        if (PrepSDKCall_SetFromConf(config, SDKConf_Signature,
            "CRecipientFilter::AddRecipient"))
        {
            PrepSDKCall_AddParameter(SDKType_CBasePlayer, SDKPass_Pointer);
            g_RecipientFilterAddRecipient = EndPrepSDKCall();
        }
    }
    if (fireBullet == null
        || !fireBullet.Enable(Hook_Pre, MissionModifiers_FXFireBullets)
        || !fireBullet.Enable(Hook_Post, MissionModifiers_FXFireBulletsPost))
    {
        LogError("Faulty Calibration sniper deviation disabled: FX_FireBullets could not be hooked.");
    }
    if (railBeam == null || g_RecipientFilterAddRecipient == null
        || !railBeam.Enable(Hook_Pre, MissionModifiers_RailBeam))
    {
        LogError("Faulty Calibration authoritative sniper tracer disabled.");
    }
    delete fireBullet;
    delete railBeam;
    delete config;
}

public MRESReturn MissionModifiers_FXFireBullets(DHookParam params)
{
    int weapon = params.Get(1);
    int client = params.Get(2);
    if (!(g_ActiveMissionModifiers & MissionModifier_FaultyCalibration)
        || !MvM_IsPlayer(client) || weapon <= MaxClients
        || !IsValidEntity(weapon) || params.IsNull(3) || params.IsNull(4))
    {
        return MRES_Ignored;
    }

    char classname[48];
    GetEntityClassname(weapon, classname, sizeof(classname));
    if (StrContains(classname, "tf_weapon_sniperrifle", false) != 0)
    {
        return MRES_Ignored;
    }

    float angles[3];
    params.GetVector(4, angles);
    angles[0] += GetRandomFloat(-MissionModifier_SniperDeviationDegrees,
        MissionModifier_SniperDeviationDegrees);
    angles[1] += GetRandomFloat(-MissionModifier_SniperDeviationDegrees,
        MissionModifier_SniperDeviationDegrees);
    params.SetVector(4, angles);

    Address tracer = TF2Attrib_GetByName(weapon, "sniper fires tracer");
    g_SniperTracerHadAttribute[weapon] = tracer != Address_Null;
    g_SniperTracerOriginalAttribute[weapon] = tracer != Address_Null
        ? TF2Attrib_GetValue(tracer) : 0.0;
    g_SniperTracerApplied[weapon] = TF2Attrib_SetByName(weapon,
        "sniper fires tracer", 1.0);
    TF2Attrib_ClearCache(weapon);
    return MRES_ChangedHandled;
}

public MRESReturn MissionModifiers_FXFireBulletsPost(DHookParam params)
{
    int weapon = params.Get(1);
    if (weapon > MaxClients && weapon < MissionModifier_MaxEntities
        && IsValidEntity(weapon) && g_SniperTracerApplied[weapon])
    {
        if (g_SniperTracerHadAttribute[weapon])
        {
            TF2Attrib_SetByName(weapon, "sniper fires tracer",
                g_SniperTracerOriginalAttribute[weapon]);
        }
        else
        {
            TF2Attrib_RemoveByName(weapon, "sniper fires tracer");
        }
        TF2Attrib_ClearCache(weapon);
        g_SniperTracerApplied[weapon] = false;
        g_SniperTracerHadAttribute[weapon] = false;
        g_SniperTracerOriginalAttribute[weapon] = 0.0;
    }
    return MRES_Ignored;
}

public MRESReturn MissionModifiers_RailBeam(int client, DHookParam params)
{
    if ((g_ActiveMissionModifiers & MissionModifier_FaultyCalibration)
        && MvM_IsPlayer(client) && !params.IsNull(1)
        && g_RecipientFilterAddRecipient != null)
    {
        SDKCall(g_RecipientFilterAddRecipient, params.GetAddress(1), client);
    }
    return MRES_Ignored;
}

/* Expand the mission's own endless support WaveSpawns while Bot Surge is
 * active. These remain population-manager bots: they keep the exact template,
 * spawn point and support semantics authored for this wave, and disappear when
 * the non-support enemies are done. Limited support is progression and is left
 * untouched.
 *
 * This deliberately happens after Parse. Reading or recreating popfile
 * KeyValues in SourcePawn would discard SigMod's extended syntax; the four
 * guarded offsets are the much smaller interface to the engine object.
 */
static void MissionModifiers_InitBotSurgeSupportHook()
{
    GameData config = LoadGameConfigFile("tf2_archipelago");
    if (config == null)
    {
        LogError("Bot Surge support fillers disabled: tf2_archipelago gamedata is missing.");
        return;
    }
    g_WaveSpawnMaxActiveOffset = config.GetOffset("CWaveSpawnPopulator::m_maxActive");
    g_WaveSpawnSpawnCountOffset = config.GetOffset("CWaveSpawnPopulator::m_spawnCount");
    g_WaveSpawnHasTFBotOffset = config.GetOffset("CWaveSpawnPopulator::m_bHasTFBotSpawner");
    g_WaveSpawnWaitBeforeOffset = config.GetOffset("CWaveSpawnPopulator::m_waitBeforeStarting");
    g_WaveSpawnWaitOffset = config.GetOffset("CWaveSpawnPopulator::m_waitBetweenSpawns");
    g_WaveSpawnSupportOffset = config.GetOffset("CWaveSpawnPopulator::m_bSupportWave");
    g_WaveSpawnLimitedSupportOffset = config.GetOffset("CWaveSpawnPopulator::m_bLimitedSupport");
    g_WaveSpawnActiveBotsOffset = config.GetOffset("CWaveSpawnPopulator::m_activeBots");
    g_WaveSpawnActiveBotCountOffset = config.GetOffset("CWaveSpawnPopulator::m_activeBotsCount");
    DynamicDetour populationHook = DynamicDetour.FromConf(config, "CPopulationManager::Parse");
    DynamicDetour waveHook = DynamicDetour.FromConf(config, "CWave::Parse");
    DynamicDetour waveSpawnHook = DynamicDetour.FromConf(config, "CWaveSpawnPopulator::Parse");

    StartPrepSDKCall(SDKCall_Raw);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "KeyValues::MakeCopy"))
    {
        PrepSDKCall_SetReturnInfo(SDKType_PlainOldData, SDKPass_Plain);
        g_KVMakeCopy = EndPrepSDKCall();
    }
    StartPrepSDKCall(SDKCall_Raw);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "KeyValues::AddSubKey"))
    {
        PrepSDKCall_AddParameter(SDKType_PlainOldData, SDKPass_Plain);
        g_KVAddSubKey = EndPrepSDKCall();
    }
    StartPrepSDKCall(SDKCall_Raw);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "KeyValues::SetInt"))
    {
        PrepSDKCall_AddParameter(SDKType_String, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_PlainOldData, SDKPass_Plain);
        g_KVSetInt = EndPrepSDKCall();
    }
    StartPrepSDKCall(SDKCall_Raw);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "KeyValues::SetFloat"))
    {
        PrepSDKCall_AddParameter(SDKType_String, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_Float, SDKPass_Plain);
        g_KVSetFloat = EndPrepSDKCall();
    }
    StartPrepSDKCall(SDKCall_Raw);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "KeyValues::SetString"))
    {
        PrepSDKCall_AddParameter(SDKType_String, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_String, SDKPass_Pointer);
        g_KVSetString = EndPrepSDKCall();
    }
    StartPrepSDKCall(SDKCall_Raw);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "KeyValues::FindKey"))
    {
        PrepSDKCall_SetReturnInfo(SDKType_PlainOldData, SDKPass_Plain);
        PrepSDKCall_AddParameter(SDKType_String, SDKPass_Pointer);
        PrepSDKCall_AddParameter(SDKType_Bool, SDKPass_Plain);
        g_KVFindKey = EndPrepSDKCall();
    }
    StartPrepSDKCall(SDKCall_Raw);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "KeyValues::RemoveSubKey"))
    {
        PrepSDKCall_AddParameter(SDKType_PlainOldData, SDKPass_Plain);
        g_KVRemoveSubKey = EndPrepSDKCall();
    }
    StartPrepSDKCall(SDKCall_Raw);
    if (PrepSDKCall_SetFromConf(config, SDKConf_Signature, "KeyValues::deleteThis"))
    {
        g_KVDeleteThis = EndPrepSDKCall();
    }
    delete config;
    if (g_WaveSpawnMaxActiveOffset < 0 || g_WaveSpawnSpawnCountOffset < 0
        || g_WaveSpawnHasTFBotOffset < 0 || g_WaveSpawnWaitBeforeOffset < 0
        || g_WaveSpawnWaitOffset < 0
        || g_WaveSpawnSupportOffset < 0 || g_WaveSpawnLimitedSupportOffset < 0
        || populationHook == null
        || !populationHook.Enable(Hook_Pre, MissionModifiers_PopulationParsing)
        || !populationHook.Enable(Hook_Post, MissionModifiers_PopulationParsed)
        || waveHook == null
        || !waveHook.Enable(Hook_Pre, MissionModifiers_WaveParsing)
        || !waveHook.Enable(Hook_Post, MissionModifiers_WaveParsed)
        || waveSpawnHook == null
        || !waveSpawnHook.Enable(Hook_Post, MissionModifiers_WaveSpawnParsed))
    {
        LogError("Bot Surge support fillers disabled: CWaveSpawnPopulator layout could not be hooked.");
    }
    if (g_WaveSpawnActiveBotsOffset < 0 || g_WaveSpawnActiveBotCountOffset < 0)
    {
        LogError("Bot Surge stall recovery disabled: support bot handles are unavailable on this server.");
    }
    if (g_KVMakeCopy == null || g_KVAddSubKey == null || g_KVSetInt == null
        || g_KVSetFloat == null || g_KVSetString == null || g_KVFindKey == null
        || g_KVRemoveSubKey == null || g_KVDeleteThis == null)
    {
        LogError("Bot Surge synthetic support disabled: KeyValues calls could not be prepared.");
    }
    delete populationHook;
    delete waveHook;
    delete waveSpawnHook;
}

static bool MissionModifiers_CanSynthesizeSupport()
{
    return g_KVMakeCopy != null && g_KVAddSubKey != null && g_KVSetInt != null
        && g_KVSetFloat != null && g_KVSetString != null && g_KVFindKey != null
        && g_KVRemoveSubKey != null && g_KVDeleteThis != null;
}

static void MissionModifiers_DeleteWaveCandidates()
{
    for (int wave = 0; wave < MissionModifier_MaxWaves; wave++)
    {
        if (g_KVDeleteThis != null
            && g_BotSurgeWaveCandidate[wave] != Address_Null)
        {
            SDKCall(g_KVDeleteThis, g_BotSurgeWaveCandidate[wave]);
        }
        g_BotSurgeWaveCandidate[wave] = Address_Null;
        g_BotSurgeWaveCandidateSource[wave] = Address_Null;
        g_BotSurgeWaveCandidateMax[wave] = 0;
        g_BotSurgeWaveCandidateSpawnCount[wave] = 0;
        g_BotSurgeWaveHasSupport[wave] = false;
        g_BotSurgeSyntheticKV[wave] = Address_Null;
    }
    g_BotSurgeCapturedWaves = 0;
    g_BotSurgeParsingWave = -1;
    g_BotSurgeSynthesisInstalled = false;
}

static void MissionModifiers_RemoveKV(Address parent, const char[] key)
{
    Address child = view_as<Address>(SDKCall(g_KVFindKey, parent, key, false));
    if (child == Address_Null)
    {
        return;
    }
    SDKCall(g_KVRemoveSubKey, parent, child);
    SDKCall(g_KVDeleteThis, child);
}

static Address MissionModifiers_MakeSyntheticSupport(int wave)
{
    Address filler = view_as<Address>(SDKCall(g_KVMakeCopy,
        g_BotSurgeWaveCandidate[wave]));
    if (filler == Address_Null)
    {
        return Address_Null;
    }

    char name[48];
    Format(name, sizeof(name), "__tf2ap_bot_surge_%d", wave + 1);
    int spawnCount = g_BotSurgeWaveCandidateSpawnCount[wave];
    int totalCount = (999999 / spawnCount) * spawnCount;
    SDKCall(g_KVSetString, filler, "Name", name);
    SDKCall(g_KVSetString, filler, "StartWaveWarningSound", "");
    SDKCall(g_KVSetString, filler, "FirstSpawnWarningSound", "");
    SDKCall(g_KVSetString, filler, "LastSpawnWarningSound", "");
    SDKCall(g_KVSetString, filler, "DoneWarningSound", "");
    SDKCall(g_KVSetInt, filler, "TotalCurrency", 0);
    SDKCall(g_KVSetInt, filler, "TotalCount", totalCount);
    SDKCall(g_KVSetInt, filler, "MaxActive", MissionModifier_SurgeMaxInvaders);
    SDKCall(g_KVSetInt, filler, "Support", 1);
    SDKCall(g_KVSetFloat, filler, "WaitBeforeStarting",
        MissionModifier_SurgeSupportStartDelay);
    SDKCall(g_KVSetFloat, filler, "WaitBetweenSpawns",
        MissionModifier_SurgeSupportSpawnInterval);
    MissionModifiers_RemoveKV(filler, "WaitForAllSpawned");
    MissionModifiers_RemoveKV(filler, "WaitForAllDead");
    MissionModifiers_RemoveKV(filler, "WaitBetweenSpawnsAfterDeath");
    MissionModifiers_RemoveKV(filler, "StartWaveOutput");
    MissionModifiers_RemoveKV(filler, "FirstSpawnOutput");
    MissionModifiers_RemoveKV(filler, "LastSpawnOutput");
    MissionModifiers_RemoveKV(filler, "DoneOutput");
    return filler;
}

public MRESReturn MissionModifiers_PopulationParsing(Address populationManager,
    DHookReturn result)
{
    // The old objects are about to be destroyed. Forget their addresses before
    // the new mission's WaveSpawns are parsed and recorded below.
    g_BotSurgeSupportSpawners.Clear();
    g_BotSurgeTimedSpawners.Clear();
    g_BotSurgeReloadQueued = false;
    g_BotSurgeMissionStarted = false;
    g_BotSurgeRecoveryUntil = 0.0;
    g_BotSurgeThrottled = false;
    g_BotSurgeMainRemaining = -1;
    g_BotSurgeWave = 0;
    g_BotSurgeParsingWave = -1;
    for (int wave = 0; wave < MissionModifier_MaxWaves; wave++)
    {
        g_BotSurgeSyntheticKV[wave] = Address_Null;
        g_BotSurgeWaveCandidateSource[wave] = Address_Null;
    }
    if (!(g_ActiveMissionModifiers & MissionModifier_BotSurge))
    {
        MissionModifiers_DeleteWaveCandidates();
    }
    return MRES_Ignored;
}

public MRESReturn MissionModifiers_PopulationParsed(Address populationManager,
    DHookReturn result)
{
    if (g_ActiveMissionModifiers & MissionModifier_BotSurge)
    {
        AP_Debug("Bot Surge removed spawn delays from %d wave spawners.",
            g_BotSurgeTimedSpawners.Length);
    }
    if (g_CvarMaxInvaders == null)
    {
        return MRES_Ignored;
    }
    // Every robot and defender is a client. Keep the population file from
    // consuming RED's six seats on an ordinary 32-slot Source server. Some
    // SigMod missions ask for limits such as 45, which only work with its much
    // more invasive extra-slot mod.
    int defenderSeats = Bots_DefenderTeamSize();
    int limit = MaxClients - defenderSeats;
    if ((g_ActiveMissionModifiers & MissionModifier_BotSurge)
        && limit > MissionModifier_SurgeMaxInvaders)
    {
        limit = MissionModifier_SurgeMaxInvaders;
    }
    if (limit > 0 && g_CvarMaxInvaders.IntValue > limit)
    {
        AP_Debug("The mission requested %d simultaneous invaders; capped at %d to reserve RED's %d seats.",
            g_CvarMaxInvaders.IntValue, limit, defenderSeats);
        g_CvarMaxInvaders.IntValue = limit;
    }
    return MRES_Ignored;
}

public MRESReturn MissionModifiers_WaveParsing(Address waveObject,
    DHookReturn result, DHookParam params)
{
    g_BotSurgeParsingWave++;
    int wave = g_BotSurgeParsingWave;
    if (!(g_ActiveMissionModifiers & MissionModifier_BotSurge)
        || !MissionModifiers_CanSynthesizeSupport()
        || wave < 0 || wave >= g_BotSurgeCapturedWaves
        || g_BotSurgeWaveHasSupport[wave]
        || g_BotSurgeWaveCandidate[wave] == Address_Null)
    {
        return MRES_Ignored;
    }

    Address waveKV = params.GetAddress(1);
    Address filler = MissionModifiers_MakeSyntheticSupport(wave);
    if (waveKV != Address_Null && filler != Address_Null)
    {
        SDKCall(g_KVAddSubKey, waveKV, filler);
        g_BotSurgeSyntheticKV[wave] = filler;
        g_BotSurgeSynthesisInstalled = true;
        AP_Debug("Bot Surge synthesized population-managed support for wave %d.", wave + 1);
    }
    else if (filler != Address_Null)
    {
        SDKCall(g_KVDeleteThis, filler);
    }
    return MRES_Ignored;
}

public MRESReturn MissionModifiers_WaveParsed(Address waveObject,
    DHookReturn result, DHookParam params)
{
    if (g_ActiveMissionModifiers & MissionModifier_BotSurge)
    {
        MissionModifiers_ExpandWaveSupportSpawners(g_BotSurgeParsingWave);
    }
    if (!(g_ActiveMissionModifiers & MissionModifier_BotSurge)
        && g_BotSurgeParsingWave + 1 > g_BotSurgeCapturedWaves)
    {
        int wave = g_BotSurgeParsingWave;
        if (wave >= 0 && wave < MissionModifier_MaxWaves
            && g_BotSurgeWaveCandidateSource[wave] != Address_Null)
        {
            g_BotSurgeWaveCandidate[wave] = view_as<Address>(SDKCall(
                g_KVMakeCopy, g_BotSurgeWaveCandidateSource[wave]));
            g_BotSurgeWaveCandidateSource[wave] = Address_Null;
        }
        g_BotSurgeCapturedWaves = g_BotSurgeParsingWave + 1;
        AP_Debug("Bot Surge indexed wave %d: endless support %s, filler candidate max %d.",
            g_BotSurgeCapturedWaves,
            g_BotSurgeWaveHasSupport[g_BotSurgeParsingWave] ? "yes" : "no",
            g_BotSurgeWaveCandidateMax[g_BotSurgeParsingWave]);
    }
    return MRES_Ignored;
}

public MRESReturn MissionModifiers_WaveSpawnParsed(Address waveSpawn,
    DHookReturn result, DHookParam params)
{
    if (!result.Value || waveSpawn == Address_Null)
    {
        return MRES_Ignored;
    }
    bool support = LoadFromAddress(waveSpawn
        + view_as<Address>(g_WaveSpawnSupportOffset), NumberType_Int8) != 0;
    bool limited = LoadFromAddress(waveSpawn
        + view_as<Address>(g_WaveSpawnLimitedSupportOffset), NumberType_Int8) != 0;
    Address maxActiveAddress = waveSpawn + view_as<Address>(g_WaveSpawnMaxActiveOffset);
    int maxActive = LoadFromAddress(maxActiveAddress, NumberType_Int32);
    int spawnCount = LoadFromAddress(waveSpawn
        + view_as<Address>(g_WaveSpawnSpawnCountOffset), NumberType_Int32);
    bool hasTFBot = LoadFromAddress(waveSpawn
        + view_as<Address>(g_WaveSpawnHasTFBotOffset), NumberType_Int8) != 0;
    Address waitBeforeAddress = waveSpawn + view_as<Address>(g_WaveSpawnWaitBeforeOffset);
    Address waitAddress = waveSpawn + view_as<Address>(g_WaveSpawnWaitOffset);
    int waitBeforeBits = LoadFromAddress(waitBeforeAddress, NumberType_Int32);
    int waitBits = LoadFromAddress(waitAddress, NumberType_Int32);
    if (hasTFBot && spawnCount > 0)
    {
        MissionModifierTimedSpawner timedSpawner;
        timedSpawner.address = waveSpawn;
        timedSpawner.waitBeforeBits = waitBeforeBits;
        timedSpawner.waitBits = waitBits;
        g_BotSurgeTimedSpawners.PushArray(timedSpawner);
        if (g_ActiveMissionModifiers & MissionModifier_BotSurge)
        {
            MissionModifiers_AccelerateSpawner(timedSpawner);
        }
    }

    // Only a real TFBot spawn with a positive count has its refill interval
    // accelerated. This also covers endless spawns whose MaxActive is 999.
    // The capacity bound below applies only to filler-candidate selection;
    // zero-count scripted WaveSpawns must keep their authored timers.
    if (maxActive < 1 || maxActive > 256)
    {
        return MRES_Ignored;
    }

    int wave = g_BotSurgeParsingWave;
    Address waveSpawnKV = params.GetAddress(1);
    bool synthetic = wave >= 0 && wave < MissionModifier_MaxWaves
        && waveSpawnKV == g_BotSurgeSyntheticKV[wave];
    if (wave >= 0 && wave < MissionModifier_MaxWaves && support && !limited
        && !(g_ActiveMissionModifiers & MissionModifier_BotSurge))
    {
        g_BotSurgeWaveHasSupport[wave] = true;
    }
    if (!support && !(g_ActiveMissionModifiers & MissionModifier_BotSurge)
        && MissionModifiers_CanSynthesizeSupport()
        && wave >= 0 && wave < MissionModifier_MaxWaves
        && waveSpawnKV != Address_Null
        && hasTFBot
        && maxActive > g_BotSurgeWaveCandidateMax[wave])
    {
        if (spawnCount >= 1 && spawnCount <= MissionModifier_SurgeMaxInvaders)
        {
            // This callback is nested inside CWave::Parse. Calling MakeCopy or
            // deleteThis here can invalidate parser-owned KeyValues state.
            // Remember the best child and make one detached copy in the wave's
            // post hook, after every WaveSpawn has finished parsing.
            g_BotSurgeWaveCandidateSource[wave] = waveSpawnKV;
            g_BotSurgeWaveCandidateMax[wave] = maxActive;
            g_BotSurgeWaveCandidateSpawnCount[wave] = spawnCount;
        }
    }
    if (!support || limited)
    {
        return MRES_Ignored;
    }

    MissionModifierSupportSpawner spawner;
    spawner.address = waveSpawn;
    spawner.wave = wave;
    spawner.maxActive = maxActive;
    spawner.waitBeforeBits = waitBeforeBits;
    spawner.waitBits = waitBits;
    spawner.synthetic = synthetic;
    g_BotSurgeSupportSpawners.PushArray(spawner);
    return MRES_Ignored;
}

static void MissionModifiers_AccelerateSpawner(MissionModifierTimedSpawner spawner)
{
    float wait = view_as<float>(spawner.waitBits);
    if (wait > MissionModifier_SurgeSupportSpawnInterval && wait < 3600.0)
    {
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnWaitOffset),
            view_as<int>(MissionModifier_SurgeSupportSpawnInterval), NumberType_Int32);
    }
}

static void MissionModifiers_AccelerateRecordedSpawners()
{
    MissionModifierTimedSpawner spawner;
    for (int index = 0; index < g_BotSurgeTimedSpawners.Length; index++)
    {
        g_BotSurgeTimedSpawners.GetArray(index, spawner);
        MissionModifiers_AccelerateSpawner(spawner);
    }
    AP_Debug("Bot Surge shortens refill intervals for %d recorded robot spawners.",
        g_BotSurgeTimedSpawners.Length);
}

static void MissionModifiers_RestoreSpawnerTimings()
{
    MissionModifierTimedSpawner spawner;
    for (int index = 0; index < g_BotSurgeTimedSpawners.Length; index++)
    {
        g_BotSurgeTimedSpawners.GetArray(index, spawner);
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnWaitBeforeOffset),
            spawner.waitBeforeBits, NumberType_Int32);
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnWaitOffset),
            spawner.waitBits, NumberType_Int32);
    }
}

static void MissionModifiers_ExpandSupportSpawner(MissionModifierSupportSpawner spawner,
    int boostedMax)
{
    StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnMaxActiveOffset),
        boostedMax, NumberType_Int32);

    Address waitAddress = spawner.address + view_as<Address>(g_WaveSpawnWaitOffset);
    float wait = view_as<float>(spawner.waitBits);
    if (wait > MissionModifier_SurgeSupportSpawnInterval && wait < 3600.0)
    {
        StoreToAddress(waitAddress,
            view_as<int>(MissionModifier_SurgeSupportSpawnInterval), NumberType_Int32);
    }
    AP_Debug("Bot Surge expanded an endless support spawner from %d to %d active "
        ... "(start %.1fs, refill %.1fs -> %.1fs).",
        spawner.maxActive, boostedMax, view_as<float>(spawner.waitBeforeBits), wait,
        MissionModifier_SurgeSupportSpawnInterval);
}

static void MissionModifiers_ExpandRecordedSupportSpawners()
{
    for (int wave = 0; wave <= g_BotSurgeParsingWave; wave++)
    {
        MissionModifiers_ExpandWaveSupportSpawners(wave);
    }
}

static void MissionModifiers_ExpandWaveSupportSpawners(int wave)
{
    if (wave < 0)
    {
        return;
    }
    int supportGroups;
    MissionModifierSupportSpawner spawner;
    for (int index = 0; index < g_BotSurgeSupportSpawners.Length; index++)
    {
        g_BotSurgeSupportSpawners.GetArray(index, spawner);
        if (spawner.wave == wave)
        {
            supportGroups++;
        }
    }
    if (supportGroups == 0)
    {
        return;
    }
    // Keep the fast, crowded support behavior. The stall monitor below gives
    // finite WaveSpawns a short opening if support occupies every BLU slot.
    int boostedMax = MissionModifier_SurgeSupportBudget / supportGroups;
    if (boostedMax < 1)
    {
        boostedMax = 1;
    }
    for (int index = 0; index < g_BotSurgeSupportSpawners.Length; index++)
    {
        g_BotSurgeSupportSpawners.GetArray(index, spawner);
        if (spawner.wave == wave)
        {
            MissionModifiers_ExpandSupportSpawner(spawner, boostedMax);
        }
    }
}

static void MissionModifiers_PauseSupportSpawners(bool paused)
{
    MissionModifierSupportSpawner spawner;
    for (int index = 0; index < g_BotSurgeSupportSpawners.Length; index++)
    {
        g_BotSurgeSupportSpawners.GetArray(index, spawner);
        if (spawner.wave != g_BotSurgeWave - 1)
        {
            continue;
        }
        // Restore the fast population limit after the finite groups get a turn.
        int maxActive = MissionModifier_SurgeSupportBudget;
        int groups = 0;
        for (int other = 0; other < g_BotSurgeSupportSpawners.Length; other++)
        {
            MissionModifierSupportSpawner candidate;
            g_BotSurgeSupportSpawners.GetArray(other, candidate);
            if (candidate.wave == spawner.wave)
            {
                groups++;
            }
        }
        if (groups > 0)
        {
            maxActive /= groups;
        }
        if (maxActive < 1)
        {
            maxActive = 1;
        }
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnMaxActiveOffset),
            paused ? 0 : maxActive, NumberType_Int32);
    }
}

static int MissionModifiers_CountEdicts()
{
    int count;
    int limit = GetMaxEntities();
    for (int entity = 1; entity < limit; entity++)
    {
        if (IsValidEdict(entity))
        {
            count++;
        }
    }
    return count;
}

// Back to the mission's own pace: authored refill intervals and support limits,
// and no synthesized support at all.
static void MissionModifiers_ThrottleBotSurge()
{
    MissionModifiers_RestoreSpawnerTimings();
    MissionModifierSupportSpawner spawner;
    for (int index = 0; index < g_BotSurgeSupportSpawners.Length; index++)
    {
        g_BotSurgeSupportSpawners.GetArray(index, spawner);
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnMaxActiveOffset),
            spawner.synthetic ? 0 : spawner.maxActive, NumberType_Int32);
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnWaitOffset),
            spawner.waitBits, NumberType_Int32);
    }
    g_BotSurgeThrottled = true;
    g_BotSurgeRecoveryUntil = 0.0;
}

static void MissionModifiers_ResumeBotSurge()
{
    g_BotSurgeThrottled = false;
    MissionModifiers_AccelerateRecordedSpawners();
    MissionModifiers_ExpandRecordedSupportSpawners();
}

void MissionModifiers_OnWaveBegin(int wave)
{
    g_BotSurgeWave = wave;
    g_BotSurgeLastProgress = GetGameTime();
    g_BotSurgeRecoveryUntil = 0.0;
    g_BotSurgeMainRemaining = -1;
}

void MissionModifiers_OnWaveEnd()
{
    if (g_BotSurgeThrottled)
    {
        MissionModifiers_ResumeBotSurge();
    }
    else if (g_BotSurgeRecoveryUntil > 0.0)
    {
        MissionModifiers_PauseSupportSpawners(false);
    }
    g_BotSurgeRecoveryUntil = 0.0;
    g_BotSurgeWave = 0;
}

bool MissionModifiers_IsCulledSupport(int client)
{
    return client > 0 && client <= MaxClients && g_BotSurgeCulled[client];
}

static void MissionModifiers_MarkSupportBots(bool supportBots[MAXPLAYERS + 1])
{
    if (g_WaveSpawnActiveBotsOffset < 0 || g_WaveSpawnActiveBotCountOffset < 0)
    {
        return;
    }
    MissionModifierSupportSpawner spawner;
    for (int index = 0; index < g_BotSurgeSupportSpawners.Length; index++)
    {
        g_BotSurgeSupportSpawners.GetArray(index, spawner);
        if (spawner.wave != g_BotSurgeWave - 1)
        {
            continue;
        }
        int count = LoadFromAddress(spawner.address
            + view_as<Address>(g_WaveSpawnActiveBotCountOffset), NumberType_Int32);
        // One spawner with a count this cannot be says nothing about the rest.
        if (count < 0 || count > MaxClients)
        {
            continue;
        }
        Address handles = view_as<Address>(LoadFromAddress(spawner.address
            + view_as<Address>(g_WaveSpawnActiveBotsOffset), NumberType_Int32));
        if (handles == Address_Null)
        {
            continue;
        }
        for (int bot = 0; bot < count; bot++)
        {
            int client = LoadEntityFromHandleAddress(handles + view_as<Address>(bot * 4));
            if (client > 0 && client <= MaxClients && MissionModifiers_IsRobot(client))
            {
                supportBots[client] = true;
            }
        }
    }
}

static int MissionModifiers_MainEnemiesRemaining(int resource)
{
    // The wave enemy count is a fixed total. Main-wave icon counts fall when
    // robots die, while endless support deaths do not change this sum.
    static const char counts[][] = {
        "m_nMannVsMachineWaveClassCounts", "m_nMannVsMachineWaveClassCounts2"
    };
    static const char flags[][] = {
        "m_nMannVsMachineWaveClassFlags", "m_nMannVsMachineWaveClassFlags2"
    };
    int remaining;
    for (int part = 0; part < 2; part++)
    {
        if (!HasEntProp(resource, Prop_Send, counts[part])
            || !HasEntProp(resource, Prop_Send, flags[part]))
        {
            return -1;
        }
        int length = GetEntPropArraySize(resource, Prop_Send, counts[part]);
        if (length <= 0 || length != GetEntPropArraySize(resource, Prop_Send, flags[part]))
        {
            return -1;
        }
        for (int icon = 0; icon < length; icon++)
        {
            int iconFlags = GetEntProp(resource, Prop_Send, flags[part], _, icon);
            if ((iconFlags & 1) && !(iconFlags & 2))
            {
                remaining += GetEntProp(resource, Prop_Send, counts[part], _, icon);
            }
        }
    }
    return remaining;
}

public Action Timer_MissionModifiersBotSurgeRecovery(Handle timer)
{
    if (!(g_ActiveMissionModifiers & MissionModifier_BotSurge) || g_BotSurgeWave < 1)
    {
        return Plugin_Continue;
    }
    float now = GetGameTime();
    int edicts = MissionModifiers_CountEdicts();
    if (!g_BotSurgeThrottled && edicts >= MissionModifier_SurgeEdictsThrottle)
    {
        MissionModifiers_ThrottleBotSurge();
        AP_Debug("Bot Surge paused in wave %d: %d of %d edicts in use.",
            g_BotSurgeWave, edicts, GetMaxEntities());
    }
    else if (g_BotSurgeThrottled && edicts <= MissionModifier_SurgeEdictsResume)
    {
        MissionModifiers_ResumeBotSurge();
        g_BotSurgeLastProgress = now;
        AP_Debug("Bot Surge resumed in wave %d: %d edicts in use.", g_BotSurgeWave, edicts);
    }
    if (g_BotSurgeThrottled)
    {
        return Plugin_Continue;
    }
    if (g_BotSurgeRecoveryUntil > 0.0)
    {
        if (now >= g_BotSurgeRecoveryUntil)
        {
            MissionModifiers_PauseSupportSpawners(false);
            g_BotSurgeRecoveryUntil = 0.0;
            g_BotSurgeLastProgress = now;
        }
        return Plugin_Continue;
    }
    int resource = FindEntityByClassname(-1, "tf_objective_resource");
    if (resource == -1)
    {
        return Plugin_Continue;
    }
    int enemies = MissionModifiers_MainEnemiesRemaining(resource);
    if (enemies <= 0)
    {
        return Plugin_Continue;
    }
    if (g_BotSurgeMainRemaining < 0 || enemies < g_BotSurgeMainRemaining)
    {
        g_BotSurgeLastProgress = now;
    }
    g_BotSurgeMainRemaining = enemies;
    if (now - g_BotSurgeLastProgress < MissionModifier_SurgeStallSeconds)
    {
        return Plugin_Continue;
    }
    // Tanks and finite robots still fighting are progress, not a support stall.
    if (FindEntityByClassname(-1, "tank_boss") != -1)
    {
        return Plugin_Continue;
    }
    bool supportBots[MAXPLAYERS + 1];
    for (int client = 1; client <= MaxClients; client++)
    {
        supportBots[client] = false;
    }
    MissionModifiers_MarkSupportBots(supportBots);
    int alive;
    int support;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (MissionModifiers_IsRobot(client) && IsPlayerAlive(client))
        {
            alive++;
            if (supportBots[client])
            {
                support++;
            }
        }
    }
    if (alive == 0 || support != alive)
    {
        return Plugin_Continue;
    }
    MissionModifiers_PauseSupportSpawners(true);
    g_BotSurgeRecoveryUntil = now + MissionModifier_SurgeRecoverySeconds;
    g_BotSurgeLastProgress = now;
    AP_Debug("Bot Surge recovery: wave %d held at %d main enemies with only %d endless support robots for 20 seconds; clearing BLU support for 4 seconds.",
        g_BotSurgeWave, enemies, support);
    for (int client = 1; client <= MaxClients; client++)
    {
        if (supportBots[client] && IsPlayerAlive(client))
        {
            g_BotSurgeCulled[client] = true;
            ForcePlayerSuicide(client);
        }
    }
    return Plugin_Continue;
}

static void MissionModifiers_RestoreSupportSpawners()
{
    MissionModifierSupportSpawner spawner;
    for (int index = 0; index < g_BotSurgeSupportSpawners.Length; index++)
    {
        g_BotSurgeSupportSpawners.GetArray(index, spawner);
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnMaxActiveOffset),
            spawner.maxActive, NumberType_Int32);
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnWaitBeforeOffset),
            spawner.waitBeforeBits, NumberType_Int32);
        StoreToAddress(spawner.address + view_as<Address>(g_WaveSpawnWaitOffset),
            spawner.waitBits, NumberType_Int32);
    }
}

bool MissionModifiers_ShouldReloadBotSurgeMission()
{
    // The objective resource already calls the initial upgrade period wave
    // one. mvm_begin_wave is the authoritative boundary: before it, reparsing
    // the mission cannot interrupt combat.
    if (g_BotSurgeReloadQueued || g_BotSurgeMissionStarted)
    {
        return false;
    }
    if (!(g_ActiveMissionModifiers & MissionModifier_BotSurge))
    {
        if (!g_BotSurgeSynthesisInstalled)
        {
            return false;
        }
        g_BotSurgeReloadQueued = true;
        return true;
    }
    if (g_BotSurgeSynthesisInstalled)
    {
        return false;
    }
    bool needsSynthesis;
    for (int wave = 0; wave < g_BotSurgeCapturedWaves; wave++)
    {
        if (!g_BotSurgeWaveHasSupport[wave]
            && g_BotSurgeWaveCandidate[wave] != Address_Null)
        {
            needsSynthesis = true;
            break;
        }
    }
    if (!needsSynthesis)
    {
        return false;
    }
    g_BotSurgeReloadQueued = true;
    return true;
}

void MissionModifiers_Shutdown()
{
    g_SeedMissionModifiers = 0;
    MissionModifiers_Reset();
    Chamber_Shutdown();
    MissionModifiers_DeleteWaveCandidates();
    delete g_BotSurgeSupportSpawners;
    g_BotSurgeSupportSpawners = null;
    delete g_BotSurgeTimedSpawners;
    g_BotSurgeTimedSpawners = null;
    delete g_BotSurgeHud;
    g_BotSurgeHud = null;
    delete g_KVMakeCopy;
    delete g_KVAddSubKey;
    delete g_KVSetInt;
    delete g_KVSetFloat;
    delete g_KVSetString;
    delete g_KVFindKey;
    delete g_KVRemoveSubKey;
    delete g_KVDeleteThis;
}

void MissionModifiers_OnMapStart()
{
    g_SeedMissionModifiers = 0;
    g_DebugMissionModifierOverride = false;
    g_BotSurgeSupportSpawners.Clear();
    g_BotSurgeTimedSpawners.Clear();
    g_BotSurgeReloadQueued = false;
    g_BotSurgeMissionStarted = false;
    g_BotSurgeRecoveryUntil = 0.0;
    g_BotSurgeThrottled = false;
    g_BotSurgeMainRemaining = -1;
    g_BotSurgeWave = 0;
    MissionModifiers_Reset();
    MissionModifiers_DeleteWaveCandidates();
}

void MissionModifiers_HookClient(int client)
{
    Mental_ReleaseClient(client);
    SDKUnhook(client, SDKHook_OnTakeDamagePost, Mental_OnTakeDamagePost);
    SDKHook(client, SDKHook_OnTakeDamagePost, Mental_OnTakeDamagePost);
    Chamber_HookClient(client);
    g_BotSurgeHudShown[client] = false;
    g_BotCountHudEnabled[client] = false;
    g_BotSurgeCulled[client] = false;
    SDKUnhook(client, SDKHook_OnTakeDamage, MissionModifiers_OnTakeDamage);
    SDKHook(client, SDKHook_OnTakeDamage, MissionModifiers_OnTakeDamage);
}

public Action Timer_MissionModifiersBotSurgeHud(Handle timer)
{
    if (g_BotSurgeHud == null)
    {
        return Plugin_Continue;
    }
    int aliveRobots;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (MissionModifiers_IsRobot(client) && IsPlayerAlive(client))
        {
            aliveRobots++;
        }
    }
    int limit = g_CvarMaxInvaders != null
        ? g_CvarMaxInvaders.IntValue : MissionModifier_SurgeMaxInvaders;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!IsClientInGame(client) || IsFakeClient(client) || IsClientSourceTV(client))
        {
            continue;
        }
        if (!g_BotCountHudEnabled[client])
        {
            if (g_BotSurgeHudShown[client])
            {
                ClearSyncHud(client, g_BotSurgeHud);
                g_BotSurgeHudShown[client] = false;
            }
            continue;
        }
        SetHudTextParams(0.80, 0.08, MissionModifier_BotSurgeHudInterval + 0.10,
            242, 164, 55, 235, 0, 0.0, 0.0, 0.0);
        ShowSyncHudText(client, g_BotSurgeHud,
            "BOT COUNT\nAlive BLU robots: %d / %d", aliveRobots, limit);
        g_BotSurgeHudShown[client] = true;
    }
    return Plugin_Continue;
}

public Action Command_BotCountHud(int client, int args)
{
    if (client < 1 || !IsClientInGame(client))
    {
        ReplyToCommand(client, "[AP] sm_ap_botcount must be run by an in-game player.");
        return Plugin_Handled;
    }

    bool enabled = !g_BotCountHudEnabled[client];
    if (args >= 1)
    {
        char action[8];
        GetCmdArg(1, action, sizeof(action));
        if (StrEqual(action, "on", false))
        {
            enabled = true;
        }
        else if (StrEqual(action, "off", false))
        {
            enabled = false;
        }
        else
        {
            ReplyToCommand(client, "[AP] Usage: sm_ap_botcount [on|off]");
            return Plugin_Handled;
        }
    }
    g_BotCountHudEnabled[client] = enabled;
    if (!enabled && g_BotSurgeHudShown[client])
    {
        ClearSyncHud(client, g_BotSurgeHud);
        g_BotSurgeHudShown[client] = false;
    }
    ReplyToCommand(client, "[AP] Live enemy robot counter %s.",
        enabled ? "enabled" : "disabled");
    return Plugin_Handled;
}

void MissionModifiers_OnPlayerSpawn(int client)
{
    if (client > 0)
    {
        g_BotSurgeCulled[client] = false;
        RequestFrame(MissionModifiers_ApplyClientFrame, GetClientUserId(client));
    }
}

void MissionModifiers_OnPlayerDeath(int client)
{
    Mental_ReleaseClient(client);
    // A tf_bot client slot is reused for different wave members. Restore the
    // outgoing robot while its native template scale is still known, so the
    // next occupant (especially a giant) supplies its own scale as the base.
    MissionModifiers_RestoreRobot(client);
    MissionModifiers_RestorePlayerInaccuracy(client);
    MissionModifiers_RestoreLooseFooting(client);
}

public void MissionModifiers_ApplyClientFrame(any userid)
{
    int client = GetClientOfUserId(userid);
    if (client > 0)
    {
        MissionModifiers_ApplyClient(client);
    }
}

void MissionModifiers_OnInventoryApplied(int client)
{
    if (client > 0)
    {
        RequestFrame(MissionModifiers_ApplyClientFrame, GetClientUserId(client));
    }
}

int MissionModifiers_FlagForKey(const char[] key)
{
    int flag;
    if (MissionModifiers_TryFlagForKey(key, flag)) return flag;
    AP_Error("The seed named an unknown mission modifier: %s", key);
    return 0;
}

static bool MissionModifiers_TryFlagForKey(const char[] key, int &flag)
{
    if (StrEqual(key, "low_gravity", false)) flag = MissionModifier_LowGravity;
    else if (StrEqual(key, "high_gravity", false)) flag = MissionModifier_HighGravity;
    else if (StrEqual(key, "blast_plating", false)) flag = MissionModifier_BlastPlating;
    else if (StrEqual(key, "thermal_shielding", false)) flag = MissionModifier_ThermalShielding;
    else if (StrEqual(key, "ballistic_plating", false)) flag = MissionModifier_BallisticPlating;
    else if (StrEqual(key, "fragile_mercenaries", false)) flag = MissionModifier_FragileMercenaries;
    else if (StrEqual(key, "overclocked_servos", false)) flag = MissionModifier_OverclockedServos;
    else if (StrEqual(key, "loaded_dice", false)) flag = MissionModifier_LoadedDice;
    else if (StrEqual(key, "weaponized_tanks", false)) flag = MissionModifier_WeaponizedTanks;
    else if (StrEqual(key, "miniature_menace", false)) flag = MissionModifier_MiniatureMenace;
    else if (StrEqual(key, "bot_surge", false)) flag = MissionModifier_BotSurge;
    else if (StrEqual(key, "faulty_calibration", false)) flag = MissionModifier_FaultyCalibration;
    else if (StrEqual(key, "loose_footing", false)) flag = MissionModifier_LooseFooting;
    else if (StrEqual(key, "mental", false)) flag = MissionModifier_Mental;
    else if (StrEqual(key, "make_it_count", false)) flag = MissionModifier_OneInTheChamber;
    else return false;
    return true;
}

static int MissionModifiers_AllFlags()
{
    // "all" still means one compatible gravity rule, never both.
    return ((1 << 15) - 1) & ~MissionModifier_HighGravity;
}

static void MissionModifiers_Reset()
{
    Mental_Reset();
    Chamber_Reset();
    MissionModifiers_RestoreSupportSpawners();
    MissionModifiers_RestoreSpawnerTimings();
    g_BotSurgeRecoveryUntil = 0.0;
    g_BotSurgeThrottled = false;
    g_BotSurgeLastProgress = GetGameTime();
    for (int client = 1; client <= MaxClients; client++)
    {
        MissionModifiers_RestorePlayerInaccuracy(client);
        MissionModifiers_RestoreLooseFooting(client);
        MissionModifiers_RestoreRobot(client);
    }
    for (int entity = MaxClients + 1; entity < MissionModifier_MaxEntities; entity++)
    {
        MissionModifiers_RestoreTank(entity);
        MissionModifiers_RemoveTankSentry(entity);
    }
    g_ActiveMissionModifiers = 0;
    g_ModifierMission[0] = '\0';
    if (g_CvarGravity != null)
    {
        g_CvarGravity.FloatValue = g_DefaultGravity;
    }
    if (g_CvarFriction != null)
    {
        g_CvarFriction.FloatValue = g_DefaultFriction;
    }
    if (g_CvarAccelerate != null)
    {
        g_CvarAccelerate.FloatValue = g_DefaultAccelerate;
    }
    if (g_CvarMaxInvaders != null)
    {
        g_CvarMaxInvaders.IntValue = g_DefaultMaxInvaders;
    }
    if (g_CvarSigRobotLimitOverride != null)
    {
        g_CvarSigRobotLimitOverride.IntValue = g_DefaultSigRobotLimitOverride;
    }
    if (g_CvarSigSpawnStallFix != null)
    {
        g_CvarSigSpawnStallFix.BoolValue = g_DefaultSigSpawnStallFix;
    }
    Bots_SetMissionRobotHealthScale(1.0);
}

void MissionModifiers_Apply(const char[] popFile, int flags)
{
    g_SeedMissionModifiers = flags;
    if (g_DebugMissionModifierOverride)
    {
        return;
    }
    MissionModifiers_Activate(popFile, flags);
}

static void MissionModifiers_Activate(const char[] popFile, int flags)
{
    if (StrEqual(g_ModifierMission, popFile, false) && g_ActiveMissionModifiers == flags)
    {
        return;
    }
    MissionModifiers_Reset();
    strcopy(g_ModifierMission, sizeof(g_ModifierMission), popFile);
    g_ActiveMissionModifiers = flags;

    if (g_CvarGravity != null)
    {
        if (flags & MissionModifier_LowGravity)
        {
            g_CvarGravity.FloatValue = MissionModifier_LowGravityValue;
        }
        else if (flags & MissionModifier_HighGravity)
        {
            g_CvarGravity.FloatValue = MissionModifier_HighGravityValue;
        }
    }
    if (g_CvarFriction != null && (flags & MissionModifier_LooseFooting))
    {
        g_CvarFriction.FloatValue = MissionModifier_LooseFootingFriction;
    }
    if (g_CvarAccelerate != null && (flags & MissionModifier_LooseFooting))
    {
        g_CvarAccelerate.FloatValue = MissionModifier_LooseFootingAcceleration;
    }
    Bots_SetMissionRobotHealthScale(
        (flags & MissionModifier_MiniatureMenace) ? MissionModifier_MiniHealthScale : 1.0);
    if (g_CvarMaxInvaders != null && (flags & MissionModifier_BotSurge))
    {
        g_CvarMaxInvaders.IntValue = MissionModifier_SurgeMaxInvaders;
    }
    if (g_CvarSigRobotLimitOverride != null && (flags & MissionModifier_BotSurge))
    {
        g_CvarSigRobotLimitOverride.IntValue = MissionModifier_SurgeMaxInvaders;
    }
    if (g_CvarSigSpawnStallFix != null && (flags & MissionModifier_BotSurge))
    {
        g_CvarSigSpawnStallFix.BoolValue = true;
    }
    if (flags & MissionModifier_BotSurge)
    {
        MissionModifiers_AccelerateRecordedSpawners();
        MissionModifiers_ExpandRecordedSupportSpawners();
    }
    for (int client = 1; client <= MaxClients; client++)
    {
        MissionModifiers_ApplyClient(client);
    }
    int tank = -1;
    while ((tank = FindEntityByClassname(tank, "tank_boss")) != -1)
    {
        MissionModifiers_ApplyTank(tank);
    }
    if (flags != 0)
    {
        MissionModifiers_Announce(0);
    }
}

bool MissionModifiers_IsRobot(int client)
{
    return client >= 1 && client <= MaxClients && IsClientInGame(client)
        && IsFakeClient(client) && GetClientTeam(client) == TeamBlue;
}

static bool MissionModifiers_IsCombatant(int client)
{
    return MvM_IsDefender(client) || MissionModifiers_IsRobot(client);
}

static void MissionModifiers_ApplyClient(int client)
{
    MissionModifiers_RestorePlayerInaccuracy(client);
    MissionModifiers_RestoreLooseFooting(client);
    if ((g_ActiveMissionModifiers & MissionModifier_FaultyCalibration)
        && MvM_IsPlayer(client))
    {
        MissionModifiers_SaveClientAttribute(client, "spread penalty",
            MissionModifier_PlayerSpreadPenalty,
            g_InaccuracyHadSpread[client], g_InaccuracyOriginalSpread[client]);
        MissionModifiers_SaveClientAttribute(client, "mult_spread_scale_first_shot",
            MissionModifier_FirstShotSpreadScale,
            g_InaccuracyHadFirstShotSpread[client],
            g_InaccuracyOriginalFirstShotSpread[client]);
        MissionModifiers_SaveClientAttribute(client, "projectile spread angle penalty",
            MissionModifier_ProjectileDeviationDegrees,
            g_InaccuracyHadProjectileSpread[client],
            g_InaccuracyOriginalProjectileSpread[client]);
        MissionModifiers_SaveClientAttribute(client, "flame_spread_degree",
            MissionModifier_FlameSpreadDegrees,
            g_InaccuracyHadFlameSpread[client], g_InaccuracyOriginalFlameSpread[client]);
        g_InaccuracyApplied[client] = true;
        TF2Attrib_ClearCache(client);
    }
    if ((g_ActiveMissionModifiers & MissionModifier_LooseFooting)
        && MissionModifiers_IsCombatant(client))
    {
        MissionModifiers_SaveClientAttribute(client, "damage force increase hidden",
            MissionModifier_LooseFootingForceScale,
            g_LooseFootingHadDamageForce[client],
            g_LooseFootingOriginalDamageForce[client]);
        MissionModifiers_SaveClientAttribute(client, "self dmg push force increased",
            MissionModifier_LooseFootingForceScale,
            g_LooseFootingHadSelfDamageForce[client],
            g_LooseFootingOriginalSelfDamageForce[client]);
        MissionModifiers_SaveClientAttribute(client, "airblast vulnerability multiplier hidden",
            MissionModifier_LooseFootingForceScale,
            g_LooseFootingHadAirblast[client],
            g_LooseFootingOriginalAirblast[client]);
        MissionModifiers_SaveClientAttribute(client, "airblast vertical vulnerability multiplier",
            MissionModifier_LooseFootingForceScale,
            g_LooseFootingHadVerticalAirblast[client],
            g_LooseFootingOriginalVerticalAirblast[client]);
        g_LooseFootingApplied[client] = true;
        TF2Attrib_ClearCache(client);
    }
    MissionModifiers_ApplyRobot(client);
    Chamber_ClampClient(client);
}

static void MissionModifiers_SaveClientAttribute(int client, const char[] name,
    float value, bool &hadAttribute, float &original)
{
    Address attribute = TF2Attrib_GetByName(client, name);
    hadAttribute = attribute != Address_Null;
    original = hadAttribute ? TF2Attrib_GetValue(attribute) : 0.0;
    if (!TF2Attrib_SetByName(client, name, value))
    {
        LogError("A mission modifier could not apply %s to %N.", name, client);
    }
}

static void MissionModifiers_RestoreClientAttribute(int client, const char[] name,
    bool hadAttribute, float original)
{
    if (hadAttribute)
    {
        TF2Attrib_SetByName(client, name, original);
    }
    else
    {
        TF2Attrib_RemoveByName(client, name);
    }
}

static void MissionModifiers_RestorePlayerInaccuracy(int client)
{
    if (client < 1 || client > MaxClients)
    {
        return;
    }
    if (g_InaccuracyApplied[client] && IsClientInGame(client))
    {
        MissionModifiers_RestoreClientAttribute(client, "spread penalty",
            g_InaccuracyHadSpread[client], g_InaccuracyOriginalSpread[client]);
        MissionModifiers_RestoreClientAttribute(client, "mult_spread_scale_first_shot",
            g_InaccuracyHadFirstShotSpread[client],
            g_InaccuracyOriginalFirstShotSpread[client]);
        MissionModifiers_RestoreClientAttribute(client, "projectile spread angle penalty",
            g_InaccuracyHadProjectileSpread[client],
            g_InaccuracyOriginalProjectileSpread[client]);
        MissionModifiers_RestoreClientAttribute(client, "flame_spread_degree",
            g_InaccuracyHadFlameSpread[client], g_InaccuracyOriginalFlameSpread[client]);
        TF2Attrib_ClearCache(client);
    }
    g_InaccuracyApplied[client] = false;
    g_InaccuracyHadSpread[client] = false;
    g_InaccuracyHadFirstShotSpread[client] = false;
    g_InaccuracyHadProjectileSpread[client] = false;
    g_InaccuracyHadFlameSpread[client] = false;
    g_InaccuracyOriginalSpread[client] = 0.0;
    g_InaccuracyOriginalFirstShotSpread[client] = 0.0;
    g_InaccuracyOriginalProjectileSpread[client] = 0.0;
    g_InaccuracyOriginalFlameSpread[client] = 0.0;
}

static void MissionModifiers_RestoreLooseFooting(int client)
{
    if (client < 1 || client > MaxClients)
    {
        return;
    }
    if (g_LooseFootingApplied[client] && IsClientInGame(client))
    {
        MissionModifiers_RestoreClientAttribute(client, "damage force increase hidden",
            g_LooseFootingHadDamageForce[client],
            g_LooseFootingOriginalDamageForce[client]);
        MissionModifiers_RestoreClientAttribute(client, "self dmg push force increased",
            g_LooseFootingHadSelfDamageForce[client],
            g_LooseFootingOriginalSelfDamageForce[client]);
        MissionModifiers_RestoreClientAttribute(client,
            "airblast vulnerability multiplier hidden",
            g_LooseFootingHadAirblast[client],
            g_LooseFootingOriginalAirblast[client]);
        MissionModifiers_RestoreClientAttribute(client,
            "airblast vertical vulnerability multiplier",
            g_LooseFootingHadVerticalAirblast[client],
            g_LooseFootingOriginalVerticalAirblast[client]);
        TF2Attrib_ClearCache(client);
    }
    g_LooseFootingApplied[client] = false;
    g_LooseFootingHadDamageForce[client] = false;
    g_LooseFootingHadSelfDamageForce[client] = false;
    g_LooseFootingHadAirblast[client] = false;
    g_LooseFootingHadVerticalAirblast[client] = false;
    g_LooseFootingOriginalDamageForce[client] = 0.0;
    g_LooseFootingOriginalSelfDamageForce[client] = 0.0;
    g_LooseFootingOriginalAirblast[client] = 0.0;
    g_LooseFootingOriginalVerticalAirblast[client] = 0.0;
}

static void MissionModifiers_ApplyRobot(int client)
{
    MissionModifiers_RestoreRobot(client);
    if (!MissionModifiers_IsRobot(client))
    {
        return;
    }

    float speedScale = 1.0;
    if (g_ActiveMissionModifiers & MissionModifier_OverclockedServos)
    {
        speedScale *= MissionModifier_OverclockedSpeedScale;
    }
    if (g_ActiveMissionModifiers & MissionModifier_MiniatureMenace)
    {
        speedScale *= MissionModifier_MiniSpeedScale;
        if (HasEntProp(client, Prop_Send, "m_flModelScale"))
        {
            g_RobotOriginalScale[client] = GetEntPropFloat(client, Prop_Send, "m_flModelScale");
            SetEntPropFloat(client, Prop_Send, "m_flModelScale",
                g_RobotOriginalScale[client] * MissionModifier_MiniModelScale);
            g_RobotScaleApplied[client] = true;
        }
    }
    if (speedScale == 1.0)
    {
        return;
    }

    Address attribute = TF2Attrib_GetByName(client, "major move speed bonus");
    g_RobotHadSpeedAttribute[client] = attribute != Address_Null;
    g_RobotOriginalSpeed[client] = g_RobotHadSpeedAttribute[client]
        ? TF2Attrib_GetValue(attribute) : 1.0;
    TF2Attrib_SetByName(client, "major move speed bonus",
        g_RobotOriginalSpeed[client] * speedScale);
    g_RobotSpeedApplied[client] = true;
}

static void MissionModifiers_RestoreRobot(int client)
{
    if (client < 1 || client > MaxClients)
    {
        return;
    }
    if (IsClientInGame(client))
    {
        if (g_RobotSpeedApplied[client])
        {
            if (g_RobotHadSpeedAttribute[client])
            {
                TF2Attrib_SetByName(client, "major move speed bonus", g_RobotOriginalSpeed[client]);
            }
            else
            {
                TF2Attrib_RemoveByName(client, "major move speed bonus");
            }
        }
        if (g_RobotScaleApplied[client] && HasEntProp(client, Prop_Send, "m_flModelScale"))
        {
            SetEntPropFloat(client, Prop_Send, "m_flModelScale", g_RobotOriginalScale[client]);
        }
    }
    g_RobotSpeedApplied[client] = false;
    g_RobotHadSpeedAttribute[client] = false;
    g_RobotOriginalSpeed[client] = 1.0;
    g_RobotScaleApplied[client] = false;
    g_RobotOriginalScale[client] = 1.0;
}

void MissionModifiers_OnEntityCreated(int entity, const char[] classname)
{
    Chamber_EntityCreated(entity, classname);
    if (entity > MaxClients && entity < MissionModifier_MaxEntities
        && StrEqual(classname, "tank_boss"))
    {
        SDKHook(entity, SDKHook_SpawnPost, MissionModifiers_TankSpawned);
    }
    if (entity > MaxClients && entity < MissionModifier_MaxEntities
        && MissionModifiers_IsProjectile(classname))
    {
        SDKHook(entity, SDKHook_SpawnPost, MissionModifiers_ProjectileSpawned);
    }
}

public void MissionModifiers_ProjectileSpawned(int projectile)
{
    SDKUnhook(projectile, SDKHook_SpawnPost, MissionModifiers_ProjectileSpawned);
    RequestFrame(MissionModifiers_DeviateProjectile, EntIndexToEntRef(projectile));
}

public void MissionModifiers_DeviateProjectile(any reference)
{
    if (!(g_ActiveMissionModifiers & MissionModifier_FaultyCalibration))
    {
        return;
    }
    int projectile = EntRefToEntIndex(reference);
    if (projectile <= MaxClients || !IsValidEntity(projectile)
        || !HasEntProp(projectile, Prop_Data, "m_vecAbsVelocity"))
    {
        return;
    }
    int owner = MissionModifiers_ProjectileOwner(projectile);
    if (!MvM_IsPlayer(owner))
    {
        return;
    }

    float velocity[3];
    GetEntPropVector(projectile, Prop_Data, "m_vecAbsVelocity", velocity);
    float speed = GetVectorLength(velocity);
    if (speed < 1.0)
    {
        return;
    }
    float angles[3], direction[3];
    float deviation = MissionModifier_ProjectileDeviationDegrees;
    char classname[48];
    if (GetEntityClassname(projectile, classname, sizeof(classname))
        && StrEqual(classname, "tf_projectile_pipe", false))
    {
        deviation = MissionModifier_GrenadeDeviationDegrees;
    }
    GetVectorAngles(velocity, angles);
    angles[0] += GetRandomFloat(-deviation, deviation);
    angles[1] += GetRandomFloat(-deviation, deviation);
    GetAngleVectors(angles, direction, NULL_VECTOR, NULL_VECTOR);
    ScaleVector(direction, speed);
    TeleportEntity(projectile, NULL_VECTOR, NULL_VECTOR, direction);
}

static bool MissionModifiers_IsProjectile(const char[] classname)
{
    return StrContains(classname, "tf_projectile_", false) == 0
        && !StrEqual(classname, "tf_projectile_sentryrocket", false);
}

static int MissionModifiers_OwnerThroughEntity(int entity)
{
    if (entity > 0 && entity <= MaxClients)
    {
        return entity;
    }
    if (entity > MaxClients && IsValidEntity(entity)
        && HasEntProp(entity, Prop_Send, "m_hOwnerEntity"))
    {
        return GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity");
    }
    return -1;
}

static int MissionModifiers_ProjectileOwner(int projectile)
{
    if (HasEntProp(projectile, Prop_Send, "m_hThrower"))
    {
        int owner = MissionModifiers_OwnerThroughEntity(
            GetEntPropEnt(projectile, Prop_Send, "m_hThrower"));
        if (owner > 0 && owner <= MaxClients)
        {
            return owner;
        }
    }
    int owner = MissionModifiers_OwnerThroughEntity(projectile);
    if (owner > MaxClients)
    {
        owner = MissionModifiers_OwnerThroughEntity(owner);
    }
    if (owner > 0 && owner <= MaxClients)
    {
        return owner;
    }
    int launcher = HasEntProp(projectile, Prop_Send, "m_hLauncher")
        ? GetEntPropEnt(projectile, Prop_Send, "m_hLauncher") : -1;
    owner = MissionModifiers_OwnerThroughEntity(launcher);
    if (owner > 0 && owner <= MaxClients)
    {
        return owner;
    }
    launcher = HasEntProp(projectile, Prop_Send, "m_hOriginalLauncher")
        ? GetEntPropEnt(projectile, Prop_Send, "m_hOriginalLauncher") : -1;
    owner = MissionModifiers_OwnerThroughEntity(launcher);
    if (owner > 0 && owner <= MaxClients)
    {
        return owner;
    }
    return -1;
}

public void MissionModifiers_TankSpawned(int tank)
{
    SDKUnhook(tank, SDKHook_SpawnPost, MissionModifiers_TankSpawned);
    MissionModifiers_ApplyTank(tank);
}

void MissionModifiers_OnEntityDestroyed(int entity)
{
    if (entity <= MaxClients || entity >= MissionModifier_MaxEntities)
    {
        return;
    }
    Mental_ForgetEntity(entity);
    MissionModifiers_RemoveTankSentry(entity);
    g_TankMiniApplied[entity] = false;
    g_TankOriginalScale[entity] = 1.0;
    g_TankOriginalSpeed[entity] = 0.0;
    g_TankOriginalHealth[entity] = 0;
    g_TankOriginalMaxHealth[entity] = 0;
}

static void MissionModifiers_ApplyTank(int tank)
{
    if (tank <= MaxClients || tank >= MissionModifier_MaxEntities || !IsValidEntity(tank))
    {
        return;
    }
    MissionModifiers_RestoreTank(tank);
    MissionModifiers_RemoveTankSentry(tank);

    if (g_ActiveMissionModifiers & MissionModifier_MiniatureMenace)
    {
        if (HasEntProp(tank, Prop_Send, "m_flModelScale"))
        {
            g_TankOriginalScale[tank] = GetEntPropFloat(tank, Prop_Send, "m_flModelScale");
            SetEntPropFloat(tank, Prop_Send, "m_flModelScale",
                g_TankOriginalScale[tank] * MissionModifier_MiniModelScale);
        }
        if (HasEntProp(tank, Prop_Data, "m_speed"))
        {
            g_TankOriginalSpeed[tank] = GetEntPropFloat(tank, Prop_Data, "m_speed");
            SetEntPropFloat(tank, Prop_Data, "m_speed",
                g_TankOriginalSpeed[tank] * MissionModifier_MiniSpeedScale);
        }
        if (HasEntProp(tank, Prop_Data, "m_iHealth"))
        {
            g_TankOriginalHealth[tank] = GetEntProp(tank, Prop_Data, "m_iHealth");
            SetEntProp(tank, Prop_Data, "m_iHealth",
                RoundToFloor(float(g_TankOriginalHealth[tank]) * MissionModifier_MiniHealthScale));
        }
        if (HasEntProp(tank, Prop_Data, "m_iMaxHealth"))
        {
            g_TankOriginalMaxHealth[tank] = GetEntProp(tank, Prop_Data, "m_iMaxHealth");
            SetEntProp(tank, Prop_Data, "m_iMaxHealth",
                RoundToFloor(float(g_TankOriginalMaxHealth[tank]) * MissionModifier_MiniHealthScale));
        }
        g_TankMiniApplied[tank] = true;
    }
    if (g_ActiveMissionModifiers & MissionModifier_WeaponizedTanks)
    {
        MissionModifiers_AddTankSentry(tank);
    }
}

static void MissionModifiers_RestoreTank(int tank)
{
    if (tank <= MaxClients || tank >= MissionModifier_MaxEntities
        || !g_TankMiniApplied[tank])
    {
        return;
    }
    if (IsValidEntity(tank))
    {
        if (HasEntProp(tank, Prop_Send, "m_flModelScale"))
        {
            SetEntPropFloat(tank, Prop_Send, "m_flModelScale", g_TankOriginalScale[tank]);
        }
        if (g_TankOriginalSpeed[tank] > 0.0 && HasEntProp(tank, Prop_Data, "m_speed"))
        {
            SetEntPropFloat(tank, Prop_Data, "m_speed", g_TankOriginalSpeed[tank]);
        }
        if (g_TankOriginalHealth[tank] > 0 && HasEntProp(tank, Prop_Data, "m_iHealth"))
        {
            SetEntProp(tank, Prop_Data, "m_iHealth", g_TankOriginalHealth[tank]);
        }
        if (g_TankOriginalMaxHealth[tank] > 0 && HasEntProp(tank, Prop_Data, "m_iMaxHealth"))
        {
            SetEntProp(tank, Prop_Data, "m_iMaxHealth", g_TankOriginalMaxHealth[tank]);
        }
    }
    g_TankMiniApplied[tank] = false;
}

static void MissionModifiers_AddTankSentry(int tank)
{
    int sentry = CreateEntityByName("obj_sentrygun");
    if (sentry <= MaxClients || sentry >= MissionModifier_MaxEntities)
    {
        AP_Error("Weaponized Tanks could not create a sentry for tank %d.", tank);
        return;
    }
    DispatchKeyValue(sentry, "teamnum", "3");
    DispatchKeyValue(sentry, "defaultupgrade", "1");
    DispatchSpawn(sentry);
    ActivateEntity(sentry);
    SetEntProp(sentry, Prop_Send, "m_iTeamNum", TeamBlue);
    if (HasEntProp(sentry, Prop_Data, "m_takedamage"))
    {
        SetEntProp(sentry, Prop_Data, "m_takedamage", 0);
    }
    // A building keeps its normal collision after SetParent. When its hull is
    // embedded in the tank, Source repeatedly resolves the parent against the
    // world and can push the tank out of sight. The mount is invulnerable and
    // dies with its tank, so it does not need a solid hull of its own.
    if (HasEntProp(sentry, Prop_Send, "m_CollisionGroup"))
    {
        SetEntProp(sentry, Prop_Send, "m_CollisionGroup", 2);
    }
    if (HasEntProp(sentry, Prop_Data, "m_usSolidFlags"))
    {
        int solidFlags = GetEntProp(sentry, Prop_Data, "m_usSolidFlags");
        SetEntProp(sentry, Prop_Data, "m_usSolidFlags", solidFlags | 4);
    }

    g_TankSentryRef[tank] = EntIndexToEntRef(sentry);
    MissionModifiers_PositionTankSentry(tank, sentry);
    SDKUnhook(tank, SDKHook_ThinkPost, MissionModifiers_TankThinkPost);
    SDKHook(tank, SDKHook_ThinkPost, MissionModifiers_TankThinkPost);
}

public void MissionModifiers_TankThinkPost(int tank)
{
    int sentry = EntRefToEntIndex(g_TankSentryRef[tank]);
    if (sentry > MaxClients && IsValidEntity(sentry))
    {
        MissionModifiers_PositionTankSentry(tank, sentry);
    }
}

static void MissionModifiers_PositionTankSentry(int tank, int sentry)
{
    float origin[3];
    float angles[3];
    GetEntPropVector(tank, Prop_Send, "m_vecOrigin", origin);
    if (HasEntProp(tank, Prop_Data, "m_angRotation"))
    {
        GetEntPropVector(tank, Prop_Data, "m_angRotation", angles);
    }
    float tankScale = HasEntProp(tank, Prop_Send, "m_flModelScale")
        ? GetEntPropFloat(tank, Prop_Send, "m_flModelScale") : 1.0;
    if (tankScale <= 0.0)
    {
        tankScale = 1.0;
    }
    // Building initialization and upgrade-model changes can reset this after
    // DispatchSpawn. Keep the mounted sentry locked to its tank's live scale,
    // just as the follow hook keeps their positions together.
    if (HasEntProp(sentry, Prop_Send, "m_flModelScale"))
    {
        SetEntPropFloat(sentry, Prop_Send, "m_flModelScale", tankScale);
    }
    float tankMaxs[3];
    float unscaledTankTop = MissionModifier_TankRoofHeight;
    if (HasEntProp(tank, Prop_Send, "m_vecMaxs"))
    {
        GetEntPropVector(tank, Prop_Send, "m_vecMaxs", tankMaxs);
        // A tank's collision hull can be shorter than its rendered body. It
        // is useful for taller custom tanks, but must not pull the sentry down
        // into a normal tank where its shots immediately hit the roof.
        if (tankMaxs[2] > unscaledTankTop)
        {
            unscaledTankTop = tankMaxs[2];
        }
    }
    // Model scale does not change the transmitted collision bounds. Scale the
    // roof and clearance together so a miniature sentry stays seated just
    // above its miniature tank instead of floating at the full-size height.
    float tankTop = (unscaledTankTop + MissionModifier_TankSentryClearance) * tankScale;
    origin[2] += tankTop;
    TeleportEntity(sentry, origin, angles, NULL_VECTOR);
}

static void MissionModifiers_RemoveTankSentry(int tank)
{
    if (tank <= MaxClients || tank >= MissionModifier_MaxEntities)
    {
        return;
    }
    if (IsValidEntity(tank))
    {
        SDKUnhook(tank, SDKHook_ThinkPost, MissionModifiers_TankThinkPost);
    }
    int sentry = EntRefToEntIndex(g_TankSentryRef[tank]);
    g_TankSentryRef[tank] = 0;
    if (sentry > MaxClients && IsValidEntity(sentry))
    {
        RemoveEntity(sentry);
    }
}

public Action MissionModifiers_OnTakeDamage(int victim, int &attacker, int &inflictor,
    float &damage, int &damagetype, int &weapon, float damageForce[3],
    float damagePosition[3], int damagecustom)
{
    bool changed;
    if (MissionModifiers_IsRobot(victim))
    {
        if ((g_ActiveMissionModifiers & MissionModifier_BlastPlating) && (damagetype & DMG_BLAST))
        {
            damage *= MissionModifier_ResistanceScale;
            changed = true;
        }
        if ((g_ActiveMissionModifiers & MissionModifier_ThermalShielding)
            && (damagetype & (DMG_BURN | DMG_IGNITE)))
        {
            damage *= MissionModifier_ResistanceScale;
            changed = true;
        }
        if ((g_ActiveMissionModifiers & MissionModifier_BallisticPlating)
            && (damagetype & DMG_BULLET))
        {
            damage *= MissionModifier_ResistanceScale;
            changed = true;
        }
    }
    if ((g_ActiveMissionModifiers & MissionModifier_FragileMercenaries)
        && MvM_IsPlayer(victim))
    {
        damage *= MissionModifier_FragileDamageScale;
        changed = true;
    }
    return changed ? Plugin_Changed : Plugin_Continue;
}

public Action TF2_CalcIsAttackCritical(int client, int weapon, char[] weaponname, bool &result)
{
    if (!(g_ActiveMissionModifiers & MissionModifier_LoadedDice)
        || !MissionModifiers_IsRobot(client))
    {
        return Plugin_Continue;
    }
    result = GetURandomFloat() < MissionModifier_RobotCritChance;
    return Plugin_Changed;
}

public Action Command_MissionModifiers(int client, int argc)
{
    MissionModifiers_Announce(client);
    return Plugin_Handled;
}

public Action Command_MissionModifierDebug(int client, int argc)
{
    if (argc == 0)
    {
        ReplyToCommand(client,
            "[AP] Usage: sm_ap_modifier <on|off|toggle> <key|all> | clear | seed");
        ReplyToCommand(client,
            "[AP] Keys: low_gravity, high_gravity, blast_plating, thermal_shielding, ballistic_plating, fragile_mercenaries, overclocked_servos, loaded_dice, weaponized_tanks, miniature_menace, bot_surge, faulty_calibration, loose_footing, mental, make_it_count");
        MissionModifiers_Announce(client);
        return Plugin_Handled;
    }

    char action[16];
    GetCmdArg(1, action, sizeof(action));
    int flags = g_ActiveMissionModifiers;
    if (StrEqual(action, "clear", false))
    {
        g_DebugMissionModifierOverride = true;
        flags = 0;
    }
    else if (StrEqual(action, "seed", false))
    {
        g_DebugMissionModifierOverride = false;
        flags = g_SeedMissionModifiers;
    }
    else
    {
        if (argc < 2)
        {
            ReplyToCommand(client, "[AP] %s needs a modifier key or all.", action);
            return Plugin_Handled;
        }
        char key[32];
        GetCmdArg(2, key, sizeof(key));
        int flag;
        if (StrEqual(key, "all", false))
        {
            flag = MissionModifiers_AllFlags();
        }
        else if (!MissionModifiers_TryFlagForKey(key, flag))
        {
            ReplyToCommand(client, "[AP] Unknown mission modifier: %s", key);
            return Plugin_Handled;
        }

        if (StrEqual(action, "on", false))
        {
            if (flag == MissionModifier_LowGravity) flags &= ~MissionModifier_HighGravity;
            if (flag == MissionModifier_HighGravity) flags &= ~MissionModifier_LowGravity;
            flags |= flag;
        }
        else if (StrEqual(action, "off", false))
        {
            flags &= ~flag;
        }
        else if (StrEqual(action, "toggle", false))
        {
            flags ^= flag;
            if (flag == MissionModifier_LowGravity && (flags & flag))
                flags &= ~MissionModifier_HighGravity;
            if (flag == MissionModifier_HighGravity && (flags & flag))
                flags &= ~MissionModifier_LowGravity;
        }
        else
        {
            ReplyToCommand(client, "[AP] Unknown action: %s (use on, off, toggle, clear, or seed)", action);
            return Plugin_Handled;
        }
        g_DebugMissionModifierOverride = true;
    }

    char popFile[64];
    MvM_PopFile(popFile, sizeof(popFile));
    MissionModifiers_Activate(popFile, flags);
    if (MissionModifiers_ShouldReloadBotSurgeMission())
    {
        ReplyToCommand(client,
            "[AP] Reloading this mission to update its Bot Surge support waves.");
        Missions_ReloadCurrentForModifiers();
    }
    if (flags == 0)
    {
        MissionModifiers_Announce(client);
    }
    return Plugin_Handled;
}

void MissionModifiers_AnnounceWave()
{
    g_BotSurgeMissionStarted = true;
    if (g_ActiveMissionModifiers != 0)
    {
        MissionModifiers_Announce(0);
    }
}

void MissionModifiers_Announce(int client)
{
    if (g_ActiveMissionModifiers == 0)
    {
        AP_PrintToClient(client, "No mission modifiers are active.");
        return;
    }
    char names[192];
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_LowGravity, "Low Gravity");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_HighGravity, "High Gravity");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_BlastPlating, "Blast Plating");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_ThermalShielding, "Thermal Shielding");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_BallisticPlating, "Ballistic Plating");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_FragileMercenaries, "Fragile Mercenaries");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_OverclockedServos, "Overclocked Servos");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_LoadedDice, "Loaded Dice");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_WeaponizedTanks, "Weaponized Tanks");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_MiniatureMenace, "Miniature Menace");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_BotSurge, "Bot Surge");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_FaultyCalibration, "Faulty Calibration");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_LooseFooting, "Loose Footing");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_Mental, "Mental");
    MissionModifiers_AddName(names, sizeof(names), MissionModifier_OneInTheChamber, "Make it Count");
    if (client == 0)
    {
        AP_Announce("Mission modifiers: %s", names);
    }
    else
    {
        AP_PrintToClient(client, "Mission modifiers: %s", names);
    }
}

static void MissionModifiers_AddName(char[] names, int maxlength, int flag, const char[] name)
{
    if (!(g_ActiveMissionModifiers & flag))
    {
        return;
    }
    if (names[0] != '\0')
    {
        StrCat(names, maxlength, ", ");
    }
    StrCat(names, maxlength, name);
}

#include "mission_mental.inc"
#include "mission_chamber.inc"
