/**
 * Traps.
 *
 * A trap is an ordinary Archipelago item with a negative effect: somebody in
 * another world opens a chest and this team pays for it. The seed decides how
 * many are in the pool; a seed with none never sends one.
 *
 * docs/en/spec.md sets the rule this file rests on. A trap can cost the team a
 * wave. A trap never touches what the run already holds: nothing here removes
 * an unlock, spends credits or writes persistent state.
 */

// Long enough to lose a fight, short enough that the wave is not over. Jarate
// is +35% damage taken and no crits for the whole team at once.
#define TrapJarateDuration 10.0

/* Fires the trap named by key. False for a key this plugin has never heard of.
 *
 * The keys are gamedata's, checked against this file by a Go test, because
 * nothing in SourcePawn can check itself against that table.
 */
stock bool Traps_Apply(const char[] key)
{
    if (StrEqual(key, "team_jarate"))
    {
        AP_Announce("Trap: Jarate on the whole team. %d soaked.", Traps_JarateTeam());
        return true;
    }
    return false;
}

/* Everyone on RED, bots included. Spare the bots and this is a trap on the
 * humans only, and on a small server the bots are most of the team.
 *
 * Counts the condition after setting it, not the calls made. TF2_AddCondition
 * is silent on a player the game refuses, and a count of calls would report a
 * trap that soaked nobody as a trap that worked.
 */
static int Traps_JarateTeam()
{
    int soaked = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (MvM_IsDefender(client) && IsPlayerAlive(client))
        {
            TF2_AddCondition(client, TFCond_Jarated, TrapJarateDuration);
            if (TF2_IsPlayerInCondition(client, TFCond_Jarated))
            {
                soaked++;
            }
        }
    }
    return soaked;
}
