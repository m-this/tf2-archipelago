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
 *
 * Every trap after Team Jarate was contributed by Lee Harrold.
 */

/* How a trap puts its condition on a player.
 *
 * Most are a plain TF2_AddCondition. Fire, bleeding and the slow are not: the
 * condition is only the visible half, and the damage or the speed comes from
 * the burn, bleed or stun the game records beside it, which only the matching
 * native sets up.
 */
enum TrapHow
{
    TrapHow_Condition,
    TrapHow_Ignite,
    TrapHow_Bleed,
    TrapHow_Slow,
}

enum struct TrapEffect
{
    char key[32];
    char label[24];
    char verb[16];
    TrapHow how;
    TFCond condition;
    // Long enough to lose a fight, short enough that the wave is not over.
    float duration;
}

// How much of a player's speed the slow trap takes away.
#define TrapSlowdown 0.5

/* The keys are gamedata's, checked against this file by a Go test, because
 * nothing in SourcePawn can check itself against that table. Jarate's line has
 * to keep saying what it said before the others existed.
 */
static TrapEffect g_TrapEffects[] =
{
    { "team_jarate", "Jarate", "soaked", TrapHow_Condition, TFCond_Jarated, 10.0 },
    { "team_mad_milk", "Mad Milk", "soaked", TrapHow_Condition, TFCond_Milked, 10.0 },
    { "team_on_fire", "Fire", "ignited", TrapHow_Ignite, TFCond_OnFire, 10.0 },
    { "team_slow_down", "Slow Down", "slowed", TrapHow_Slow, TFCond_Dazed, 10.0 },
    { "team_bleed", "Bleed", "bleeding", TrapHow_Bleed, TFCond_Bleeding, 10.0 },
    { "team_marked_for_death", "Marked For Death", "marked", TrapHow_Condition, TFCond_MarkedForDeath, 10.0 },
    { "team_gas", "Gas", "gassed", TrapHow_Condition, TFCond_Gas, 10.0 },
};

/* Fires the trap named by key. False for a key this plugin has never heard of. */
stock bool Traps_Apply(const char[] key)
{
    for (int i = 0; i < sizeof(g_TrapEffects); i++)
    {
        if (StrEqual(key, g_TrapEffects[i].key))
        {
            AP_Announce("Trap: %s on the whole team. %d %s.",
                g_TrapEffects[i].label, Traps_HitTeam(g_TrapEffects[i]), g_TrapEffects[i].verb);
            return true;
        }
    }
    return false;
}

/* Everyone on RED, bots included. Spare the bots and this is a trap on the
 * humans only, and on a small server the bots are most of the team.
 *
 * Counts the condition after setting it, not the calls made. The natives are
 * silent on a player the game refuses, a Pyro the fire cannot light for one,
 * and a count of calls would report a trap that hit nobody as a trap that
 * worked.
 */
static int Traps_HitTeam(TrapEffect effect)
{
    int hit = 0;
    for (int client = 1; client <= MaxClients; client++)
    {
        if (MvM_IsDefender(client) && IsPlayerAlive(client))
        {
            Traps_Hit(client, effect);
            if (TF2_IsPlayerInCondition(client, effect.condition))
            {
                hit++;
            }
        }
    }
    return hit;
}

// The player is their own attacker: the trap has no robot to blame.
static void Traps_Hit(int client, TrapEffect effect)
{
    switch (effect.how)
    {
        case TrapHow_Ignite:
        {
            TF2_IgnitePlayer(client, client, effect.duration);
        }
        case TrapHow_Bleed:
        {
            TF2_MakeBleed(client, client, effect.duration);
        }
        case TrapHow_Slow:
        {
            TF2_StunPlayer(client, effect.duration, TrapSlowdown, TF_STUNFLAG_SLOWDOWN);
        }
        default:
        {
            TF2_AddCondition(client, effect.condition, effect.duration);
        }
    }
}
