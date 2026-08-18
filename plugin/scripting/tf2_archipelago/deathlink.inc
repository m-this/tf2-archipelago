/**
 * Death Link.
 *
 * A death in another world takes the team down, and the team losing a wave
 * is what counts as a death here. An individual death in Mann vs Machine is
 * routine, so it is not one; only a wave failure is a loss the whole team
 * shares, which is what a Death Link is on the other side too.
 *
 * Whether the seed wants it is the bridge's knowledge. This side kills when
 * told to and reports every lost wave; the bridge drops the report when the
 * seed did not ask.
 */

// A received death kills the team, and a dead team loses the wave. That loss
// must not go back out as a death of its own: long enough for the game to
// notice the wipe, short enough not to swallow the next real loss.
#define DeathLinkEchoWindow 20.0

// What the bridge last said. For chat only: the report goes either way.
bool g_DeathLinkOn;

float g_DeathLinkAppliedAt;

void DeathLink_Received(const char[] source, const char[] cause)
{
    if (!MvM_IsActive())
    {
        AP_Debug("A death arrived from %s, but this map is not Mann vs Machine.", source);
        return;
    }
    int killed = MvM_KillTeam();
    g_DeathLinkAppliedAt = GetEngineTime();
    if (cause[0] != '\0')
    {
        AP_Announce("Death Link: %s", cause);
    }
    else
    {
        AP_Announce("Death Link: %s died.", source);
    }
    AP_Debug("The death from %s took %d player(s) down.", source, killed);
}

// The team lost a wave. Reported unless the team was killed by a Death Link a
// moment ago, in which case the loss is that death, not a new one.
void DeathLink_WaveFailed(const char[] popFile, int wave)
{
    if (g_DeathLinkAppliedAt > 0.0 && GetEngineTime() - g_DeathLinkAppliedAt < DeathLinkEchoWindow)
    {
        AP_Debug("The wave was lost to a Death Link. The plugin does not send it back.");
        return;
    }
    Bridge_ReportDeath(popFile, wave);
}
