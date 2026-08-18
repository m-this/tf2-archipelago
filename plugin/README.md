# plugin

SourcePawn. Runs inside the `srcds` container. The only component that sees the
game. Read [ADR
0002](../docs/en/adr/0002-server-side-plugin-with-a-go-bridge.md) first.

## Building

```sh
./build.sh          # fetches the pinned toolchain into build/, compiles
```

It produces `build/tf2_archipelago.smx` and treats warnings as errors. The
script pins the compiler and the ripext includes. The build needs nothing
else.

## What you will see in game

There is no client mod, so chat is the whole diagnostic surface. Three levels:

| Level | Goes to | Contains |
| --- | --- | --- |
| Announcements | Chat, unless `tf2ap_announce 0` | Waves cleared, items received, classes and slots unlocked |
| Errors | Chat and the SourceMod log, always | Anything that went wrong, including the bridge being unreachable |
| Debug | Chat, console and the log, when `tf2ap_debug 1` | Every bridge call, every queued objective, every wave event |

Errors reach chat even with announcements muted. If the bridge is down,
players notice their checks stop landing before anyone else does. The
plugin tells them why, so they do not conclude the randomizer is broken.

Repeated bridge failures get one line, not many. Otherwise the retry loop
fills chat with the same line every five seconds.

## Talking to the multiworld

Players have no Archipelago client, so the plugin is theirs:

| Typed in chat | What it does |
| --- | --- |
| `!ap` | One line of help |
| `!ap hint Class: Scout` | Runs an Archipelago server command. It adds the `!` if missing, so `!ap hint`, `!ap missing`, `!ap status` and `!ap release` all work |
| `!apchat <text>` | Says something to the other players in the multiworld, prefixed with the sender's name |

What the multiworld says comes back into chat, so a hint answered or an item
someone else found shows up in game. `tf2ap_chat 0` turns that off.

## Death Link

When the seed asks for it, the team losing a wave is the death this side
sends, and a death from anywhere else kills everyone on RED, bots included,
which loses the wave. That loss is not sent back out: for twenty seconds
after a received death, `mvm_wave_failed` is taken as its consequence. The
plugin reports every lost wave and the bridge knows whether the seed asked,
so nothing here needs configuring.

Anyone joining gets a chat welcome eight seconds after they spawn in. It
covers what the server is, which mission runs, and what they unlocked so
far, plus how to use `!ap`. Eight seconds, because a message printed
during the map load scrolls past before anyone can read it.

## Commands

| Command | Flag | What it does |
| --- | --- | --- |
| `sm_ap_status` | generic | The whole picture: mission, wave, which events exist, the unlock set, the queue depth, the last bridge error |
| `sm_ap_resync` | generic | Fetch the unlock set again |
| `sm_ap_report <kind> [wave]` | root | Report an objective by hand: `wave_cleared`, `mission_cleared` or `death` |

`sm_ap_report` tests the wiring without playing a wave. It also sends a
check by hand when the game fails to fire the expected event.

## ConVars

| ConVar | Default | What it does |
| --- | --- | --- |
| `tf2ap_bridge_url` | `http://127.0.0.1:24680` | Base URL of the bridge. Loopback. |
| `tf2ap_announce` | `1` | Announce grants and cleared waves in chat |
| `tf2ap_chat` | `1` | Show what the rest of the multiworld says |
| `tf2ap_debug` | `0` | Echo every bridge call and game event |

## What is UNVERIFIED

Everything this plugin reads out of the game. It compiles, but the author
never ran it, because no Team Fortress 2 server was available at the
time.

- The plugin hooks `mvm_begin_wave`, `mvm_wave_complete`,
  `mvm_mission_complete` and `mvm_wave_failed` with `HookEventEx`, so a
  missing one produces a log line and a degraded mode, not a failed load.
  `sm_ap_status` prints which of the four exist.
- If `mvm_wave_complete` turns out not to exist, a one-second timer watches
  `m_nMannVsMachineWaveCount` instead and reports a wave when the counter goes
  up.
- The mission's identity comes from `m_iszMvMPopfileName`, falling back to the
  map name, which is the right pop file for every map's default mission.
- `m_nCurrency` is how cash bundles are paid out.

None of the fallbacks change the wire contract, so the bridge cannot tell
the difference. Run the first live session with `tf2ap_debug 1`.

## What it does

Two directions, and nothing else.

**Observe.** Detect MvM objectives and report them to the bridge:
`wave_cleared`, `mission_cleared`, `tank_destroyed`, `giant_killed`,
`money_bonus`. Game events and `SDKHooks` do the work.

**Apply.** Receive grants from the bridge, and enforce them. Lock and
unlock weapon slots, restrict classes, and gate upgrades at the station.
Hand out canteens, spawn allied bots with an unlocked template, and fire
traps.

## What it must not do

- **Know anything about Archipelago.** No item ids, no location ids, no slot,
  no seed. It speaks MvM vocabulary; the bridge translates. This lets
  someone reload, rewrite or debug the plugin without touching the
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

## Decisions from the spec that land here

From [`../docs/en/spec.md`](../docs/en/spec.md):

- **Shop checks ship off by default.** No live-server test confirms that a
  purchasable entry can go into the MvM upgrade station UI, so
  `shop_checks` stays off until one does.
- **Allied bots share the player's unlocked upgrades directly**, since RED
  bots do not buy upgrades on their own.
- **`gamedata/` hardcodes wave counts from the wiki**, rather than parsing
  them from `.pop` files. It owns the table; v1 does not support
  community missions.
