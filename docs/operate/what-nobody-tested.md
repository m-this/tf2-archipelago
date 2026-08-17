# What nobody tested

**The SourceMod plugin has never run on a live Team Fortress 2 server.**

No Team Fortress 2 server was available on the machine that this was built on.
The plugin compiles with warnings treated as errors, and that is the whole of
what is known about it. Every game event it hooks and every entity property it
reads is a guess taken from community documentation and checked against nothing
but a compiler.

Your first session is partly a diagnostic session.

## What is proven

`make integration` starts a real randomizer server on a freshly generated
session and a real bridge, and drives the bridge exactly the way the plugin
drives it. It runs on every change. It verifies:

- The session generates, for several combinations of options.
- The randomizer server hosts it and the bridge connects to it.
- The starting classes, weapon slots and mission ticket arrive.
- A reported check reaches the randomizer server, and a repeated report counts
  once.
- An objective that does not exist is refused.
- The goal registers when the run is finished.
- Chat crosses in both directions.
- The state survives a restart.

So everything from the bridge upwards works. The seed, the item logic, the
checks, the durability and the session are not in question.

## What is not proven

Everything between the game and the bridge. Specifically:

| What the plugin does | Why it might not work |
| --- | --- |
| Detects a cleared wave with `mvm_wave_complete` | The event may not exist or may not fire |
| Reads the wave number from `mvm_begin_wave` | Same |
| Detects a cleared mission with `mvm_mission_complete` | Same |
| Reads the current mission from `m_iszMvMPopfileName` | The property name may be wrong |
| Reads the wave counter from `m_nMannVsMachineWaveCount` | Same |
| Reads the mission length from `m_nMannVsMachineMaxWaveCount` | Same |
| Pays credits by writing `m_nCurrency` | Same |
| Removes a weapon from a locked slot | Never seen on a real player |
| Moves a player off a locked class at the next spawn | Never seen on a real player |

None of this has been observed once.

## What was done about it

The plugin is built to survive being wrong.

- Every event hook is optional. A missing event degrades the plugin instead of
  failing to load it.
- Every property is tested before it is read. A missing property gives one
  warning and a fallback, not an error on every frame.
- A missing `mvm_wave_complete` falls back to a one-second timer watching the
  wave counter. A rising counter means a wave was beaten.
- A missing mission name falls back to the map name.
- Both mission-clear detectors fire on purpose. The bridge counts a check once,
  so between the two of them one should work.
- Nothing is enforced until the bridge has answered once. A server where nobody
  can hold a weapon because the bridge hiccuped is worse than a wave played with
  too much kit.
- Every failure is written to the chat in red. There is no client mod, so the
  chat is the whole diagnostic surface.

## How to find out

Turn on the loud mode for the first session:

```
rcon_password your-SRCDS_RCONPW
rcon tf2ap_debug 1
```

Then, on the first MvM map, before the first wave:

```
rcon sm_ap_status
```

The `events:` line is the answer:

```
[AP] events: begin_wave yes, wave_complete yes, mission_complete no
```

That line says which of the three events this game version actually sends. It
is the single most useful piece of information that a first session can
produce. Whatever it says, it is worth recording.

Also worth watching, in this order:

1. Does `[AP] Wave 1 cleared.` appear when the team clears wave 1.
2. Does the mission name in the welcome message match the mission you started,
   or is it the map name.
3. Does a locked weapon slot stay empty after a visit to the resupply locker.
4. Does the mission clear fire on the last wave and not one wave early. Check
   `wave_drift` in the bridge health page. See
   [Troubleshooting](troubleshooting.md).

## The other unverified thing

Inside a difficulty tier that holds more than one mission, the pairing of
display name to mission file is a guess. The names and the files are both
certain. Which name goes with which file is not, for the two Decoy
intermediates, the two Coal Town intermediates, the two Mannworks intermediates
and the three advanced groups.

A wrong pairing gives the wrong mission name in the chat and in the hints.
Nothing worse. The mission itself, its waves and its checks are correct.
