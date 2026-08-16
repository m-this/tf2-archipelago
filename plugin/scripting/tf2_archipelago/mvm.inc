/**
 * Reading the game.
 *
 * Everything in this file is the plugin's only contact with Mann vs Machine
 * itself, and most of it is UNVERIFIED: the entity properties and the event
 * fields below are what the community documents, checked against nothing but
 * a compiler. Every one of them fails soft and says so in the log, so the
 * first live server tells us which of them are real rather than the plugin
 * silently reporting nothing.
 */

// tf_objective_resource carries the MvM run's state. One exists per map.
static int FindObjectiveResource()
{
    return FindEntityByClassname(-1, "tf_objective_resource");
}

/**
 * Whether this map is running Mann vs Machine at all. Everything the plugin
 * does hangs off this: on a payload map it must be inert.
 */
stock bool MvM_IsActive()
{
    return GameRules_GetProp("m_bPlayingMannVsMachine") != 0;
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
    if (resource != -1)
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
 * The wave being played, counting from 1.
 *
 * UNVERIFIED: m_nMannVsMachineWaveCount. The mvm_begin_wave event is the
 * primary source and this is the cross-check, which is why a disagreement is
 * worth a log line rather than a silent pick.
 */
stock int MvM_WaveFromGame()
{
    int resource = FindObjectiveResource();
    if (resource == -1)
    {
        return 0;
    }
    return GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveCount");
}

/**
 * How many waves the mission has. UNVERIFIED, same caveat.
 */
stock int MvM_MaxWavesFromGame()
{
    int resource = FindObjectiveResource();
    if (resource == -1)
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
        if (!IsClientInGame(client) || IsFakeClient(client))
        {
            continue;
        }
        int held = GetEntProp(client, Prop_Send, "m_nCurrency");
        SetEntProp(client, Prop_Send, "m_nCurrency", held + amount);
        paid++;
    }
    AP_Announce("Received %d credits for %d player(s).", amount, paid);
}
