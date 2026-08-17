# What nobody tested

The plugin ran on a live Team Fortress 2 server for the first time on
2026-08-17. It answered. Most of what this page used to warn about is now
settled, and what is left is smaller and more precise.

Your first session with players is still partly a diagnostic session, because
one thing cannot be tested without a person at a game client: a wave.

## What the first live run settled

A real server, a real seed, the plugin driven over rcon. Every line below was a
guess before that session:

| Thing | Result |
| --- | --- |
| The plugin loads and reaches the bridge | Yes, over the loopback it shares with the game server |
| `mvm_begin_wave` exists | Yes |
| `mvm_wave_complete` exists | Yes |
| `mvm_mission_complete` exists | Yes |
| The plugin knows it is in MvM | Yes |
| It reads the mission from `m_iszMvMPopfileName` | Yes. It reported `mvm_decoy_intermediate` while the map was `mvm_decoy`, which the fallback could not have produced |
| It reads the mission length from `m_nMannVsMachineMaxWaveCount` | Yes. It reported 6 waves for Cave-in |
| The wave counts in the tables are right | Cave-in is 6 in the game and 6 in the tables |
| A reported check reaches the randomizer server | Yes: `tf2 sent Cash Bundle to tf2 (Cave-in Wave 1)` |
| An item comes back and is applied once | Yes, and the acknowledgement moved with it |
| The mission switcher changes mission and map | Yes, on the same map and across maps |

The three events existing is the important one. The whole design rests on them.

## What is still not proven

A wave. Mann vs Machine does not start one until a human player readies up, and
a bot cannot ready up. So the events exist but have never been seen to fire,
and everything downstream of a wave is still untested:

| What the plugin does | Why it might not work |
| --- | --- |
| Reports a cleared wave when `mvm_wave_complete` fires | The event exists. Nobody has seen it fire |
| Reads the wave number out of `mvm_begin_wave` | The field name may be wrong |
| Reports a cleared mission | Same, and the last-wave fallback has never run |
| Reads the wave counter from `m_nMannVsMachineWaveCount` | Only the maximum has been read so far |
| Pays credits by writing `m_nCurrency` | There was nobody on the server to pay |
| Removes a weapon from a locked slot | Never seen on a real player |
| Moves a player off a locked class at the next spawn | Never seen on a real player |

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
- Every objective is written to the SourceMod log as it is queued, so a check
  the plugin loses can be replayed by hand.

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

A healthy answer looks like this:

```
[AP] version 0.1.0, mvm yes, mission mvm_decoy_intermediate, wave 0 of 7
[AP] events: begin_wave yes, wave_complete yes, mission_complete yes
[AP] unlocks held at sequence 5, 0 objective(s) waiting to be sent
[AP] classes: soldier, heavy
[AP] slots: primary
```

`mission` has to be the mission, not the map. `wave 0 of 7` has to match the
mission's real length. `unlocks held` has to say held, not NOT FETCHED.

## Testing a wave on your own

You do not need anybody else, and you do not need to run or join anything: the
randomizer server is already in the stack and the bridge is its only client.

`sv_visiblemaxplayers` says six, but a wave starts as soon as the players who
are here have readied up, and the stack sets `tf_mvm_min_players_to_start 1`.
So connect, press the ready key, and the wave begins.

Clearing it alone is the part that needs help. From the console:

```
rcon sv_cheats 1
rcon mp_disable_respawn_times 1
god
rcon tf_bot_kill all
rcon tf_mvm_tank_kill
```

`god` is typed in your own console rather than through rcon, because rcon runs
as the server and the server is not a player. `tf_bot_kill all` kills what is
alive right now and the mission keeps sending the rest of the wave, so run it
again every few seconds until the wave ends. It ends the wave the way the game
does, which is the point: `mvm_wave_complete` fires for real.

`tf_mvm_force_victory` and `tf_mvm_jump_to_wave` also exist. Neither is worth
using here. They skip the event this is trying to observe.

Load a mission the run actually holds first, with `rcon sm_ap_mission`. On a
mission outside the run the plugin says so and counts the checks anyway, which
is correct behaviour and a confusing thing to debug against.

Then, with players on the server, watch in this order:

1. Does `[AP] Wave 1 cleared.` appear when the team clears wave 1.
2. Does a locked weapon slot stay empty after a visit to the resupply locker.
3. Does a locked class move the player at the next spawn, and not mid-wave.
4. Do credits from a Cash Bundle actually arrive.
5. Does the mission clear fire on the last wave and not one wave early. Check
   `wave_drift` in the bridge health page. See
   [Troubleshooting](troubleshooting.md).

## What testing it found

Running it turned up four faults that no amount of reading had:

- The server refused to host at all, because Mann vs Machine needs 32 slots and
  the example configuration asked for 6.
- The game's own `server.cfg` replaced the operator's rcon password with a
  published default, on a port open to the network.
- Every plugin command run from rcon aborted halfway, because the console is not
  a player and the plugin printed to it as though it were.
- The mission switcher set the mission before changing the map, and the map
  change undid it.

## The other unverified thing

Inside a difficulty tier that holds more than one mission, the pairing of
display name to mission file is a guess. The names and the files are both
certain. Which name goes with which file is not, for the two Decoy
intermediates, the two Coal Town intermediates, the two Mannworks intermediates
and the three advanced groups.

A wrong pairing gives the wrong mission name in the chat and in the hints.
Nothing worse. The mission itself, its waves and its checks are correct.
