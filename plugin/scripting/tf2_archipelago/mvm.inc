/**
 * Reading the game.
 *
 * Everything in this file is the plugin's only contact with Mann vs Machine
 * itself, and most of it is UNVERIFIED: the entity properties and the event
 * fields below are what the community documents, checked against nothing but a
 * compiler.
 *
 * So none of it is read without asking first. A property that does not exist
 * is a warning and a fallback, never a native error: SourceMod turns a bad
 * property name into a runtime error on every call, which would bury the
 * server's console in the middle of a wave.
 */

// TFTeam_Red is the only team players are on in Mann vs Machine. BLU is the
// robots, and they are bots.
#define TeamRed 2

// g_WarnedProps keeps each missing-property complaint to one line rather than
// one per frame.
StringMap g_WarnedProps;

void MvM_Init()
{
    g_WarnedProps = new StringMap();
}

/**
 * Whether a networked property exists, complaining once if it does not.
 */
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
        AP_Error("this server has no %s; the plugin is working without it", property);
    }
    return false;
}

// tf_objective_resource carries the run's state. One exists per map.
static int FindObjectiveResource()
{
    return FindEntityByClassname(-1, "tf_objective_resource");
}

/**
 * Whether this map is running Mann vs Machine at all. Everything the plugin
 * does hangs off this: on a payload map it must be inert.
 *
 * The game rules property is the real answer; the map name is the fallback,
 * and it is a good one because every Mann vs Machine map is named mvm_.
 */
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

/**
 * Whether this client is someone the plugin may touch: a human, in game, on
 * RED. The robots are bots on BLU, and stripping a robot's weapons would take
 * the wave apart.
 */
stock bool MvM_IsPlayer(int client)
{
    return client > 0 && client <= MaxClients
        && IsClientInGame(client)
        && !IsFakeClient(client)
        && GetClientTeam(client) == TeamRed;
}

/**
 * The pop file the server is running, which is the mission's identity
 * everywhere else in this project.
 *
 * UNVERIFIED: m_iszMvMPopfileName. It is documented as holding a path such as
 * scripts/population/mvm_coaltown_advanced.pop, so the directory and the
 * extension are trimmed. Falls back to the map name, which is the correct pop
 * file for every map's default mission.
 */
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

// TrimPopFileName turns scripts/population/mvm_decoy.pop into mvm_decoy.
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

/**
 * The wave being played, counting from 1, or zero when the game will not say.
 *
 * UNVERIFIED: m_nMannVsMachineWaveCount. The mvm_begin_wave event is the
 * primary source and this is the cross-check.
 */
stock int MvM_WaveFromGame()
{
    int resource = FindObjectiveResource();
    if (resource == -1 || !HasProp(resource, "m_nMannVsMachineWaveCount"))
    {
        return 0;
    }
    return GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveCount");
}

/**
 * How many waves the mission has, or zero. UNVERIFIED, same caveat.
 */
stock int MvM_MaxWavesFromGame()
{
    int resource = FindObjectiveResource();
    if (resource == -1 || !HasProp(resource, "m_nMannVsMachineMaxWaveCount"))
    {
        return 0;
    }
    return GetEntProp(resource, Prop_Send, "m_nMannVsMachineMaxWaveCount");
}

/**
 * Hand a cash bundle to everyone playing. Credits are per player in Mann vs
 * Machine, so a bundle is worth its amount to each of them rather than being
 * split; the alternative is filler that does nothing on a full team.
 *
 * UNVERIFIED: m_nCurrency on CTFPlayer.
 */
stock void MvM_GrantCredits(int amount)
{
    if (amount <= 0 || !MvM_IsActive())
    {
        return;
    }
    int paid = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (!MvM_IsPlayer(client) || !HasProp(client, "m_nCurrency"))
        {
            continue;
        }
        int held = GetEntProp(client, Prop_Send, "m_nCurrency");
        SetEntProp(client, Prop_Send, "m_nCurrency", held + amount);
        paid++;
    }
    if (paid > 0)
    {
        AP_Announce("Received %d credits for %d player(s).", amount, paid);
    }
    else
    {
        AP_Debug("received %d credits with nobody to pay", amount);
    }
}
