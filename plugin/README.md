# plugin

SourcePawn. Runs inside the `srcds` container. The only component that sees the
game. Read [ADR
0002](../docs/adr/0002-server-side-plugin-with-a-go-bridge.md) first.

## Building

```sh
./build.sh          # fetches the pinned toolchain into build/, compiles
```

It produces `build/tf2_archipelago.smx` and treats warnings as errors. The
compiler and the ripext includes are pinned in the script; nothing else is
needed to build.

## What you will see in game

There is no client mod, so chat is the whole diagnostic surface. Three levels:

| Level | Goes to | Contains |
| --- | --- | --- |
| Announcements | Chat, unless `tf2ap_announce 0` | Waves cleared, items received, classes and slots unlocked |
| Errors | Chat and the SourceMod log, always | Anything that went wrong, including the bridge being unreachable |
| Debug | Chat, console and the log, when `tf2ap_debug 1` | Every bridge call, every queued objective, every wave event |

Errors reach chat even with announcements muted. If the bridge is down, the
players are the ones who will notice their checks are not landing, and they
should be told why rather than deciding the randomizer is broken.

Repeated bridge failures are said once. The retry loop would otherwise fill
chat with the same line every five seconds.

## Commands

| Command | Flag | What it does |
| --- | --- | --- |
| `sm_ap` | generic | The whole picture: mission, wave, which events exist, the unlock set, the queue depth, the last bridge error |
| `sm_ap_resync` | generic | Fetch the unlock set again |
| `sm_ap_report <kind> [wave]` | root | Report an objective by hand |

`sm_ap_report` is how the wiring gets tested without playing a wave, and how a
check gets sent when the game did not fire the event it should have.

## ConVars

| ConVar | Default | What it does |
| --- | --- | --- |
| `tf2ap_bridge_url` | `http://127.0.0.1:24680` | Base URL of the bridge. Loopback. |
| `tf2ap_announce` | `1` | Announce grants and cleared waves in chat |
| `tf2ap_debug` | `0` | Echo every bridge call and game event |

## What is UNVERIFIED

Everything this plugin reads out of the game. It has been compiled, never run:
no Team Fortress 2 server was available while it was written.

- `mvm_begin_wave`, `mvm_wave_complete`, `mvm_mission_complete` are hooked with
  `HookEventEx`, so a missing one is a log line and a degraded mode rather than
  a failed load. `sm_ap` prints which of the three exist.
- If `mvm_wave_complete` turns out not to exist, a one-second timer watches
  `m_nMannVsMachineWaveCount` instead and reports a wave when the counter goes
  up.
- The mission's identity comes from `m_iszMvMPopfileName`, falling back to the
  map name, which is the right pop file for every map's default mission.
- `m_nCurrency` is how cash bundles are paid out.

None of the fallbacks change the wire contract, so the bridge cannot tell the
difference. The first live session should be run with `tf2ap_debug 1`.

## What it does

Two directions, and nothing else.

**Observe.** Detect MvM objectives and report them to the bridge:
`wave_cleared`, `mission_cleared`, `tank_destroyed`, `giant_killed`,
`money_bonus`. Game events and `SDKHooks` do the work.

**Apply.** Receive grants from the bridge and enforce them: lock and unlock
weapon slots, restrict classes, gate upgrades at the station, hand out
canteens, spawn allied bots with an unlocked template, fire traps.

## What it must not do

- **Know anything about Archipelago.** No item ids, no location ids, no slot,
  no seed. It speaks MvM vocabulary; the bridge translates. This is what lets
  the plugin be reloaded, rewritten or debugged without touching the
  multiworld.
- **Hold authoritative state.** After a reload, a map change or an `srcds`
  crash, ask the bridge for the full unlock set and apply it. Never try to
  remember what was already granted.
- **Block a game frame.** Every bridge call is asynchronous. A blocking HTTP
  call stalls the server for everyone on it.

## Dependencies

- SourceMod, current stable.
- [`ripext`](https://github.com/ErikMinekus/sm-ripext) (REST in Pawn) for
  non-blocking HTTP and JSON.

Both go into the `srcds` image (`deploy/`), pinned.

## The player-facing surface

Chat, HUD text and annotations. That is the whole list, because there is no
client mod (ADR 0002). Anything the player needs to know about a received item
or a fired trap has to fit through one of those three.

## Open questions that land here

From [`../docs/spec.md`](../docs/spec.md), the ones this component has to
answer before its part of the spec can be finished:

1. **Shop check injection.** Can a plugin add an arbitrary purchasable entry to
   the MvM upgrade station UI, or only intercept purchases of existing ones?
   The entire `shop_checks` location group depends on the answer.
2. **Allied bot upgrade sharing.** Does the RED bot upgrade path still exist in
   current TF2? Needs testing on a live server.
3. **Wave counts.** Either parse the `.pop` files or hardcode the Valve
   missions. Parsing probably belongs in `gamedata/` at build time rather than
   here at runtime, but the plugin is where the ground truth can be checked.
