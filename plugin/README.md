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
| Debug | Console and the log at `tf2ap_debug 1`, chat as well at `2` | Every bridge call, every queued objective, every wave event |

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
| `sm_ap_status` | generic | The whole picture: mission, wave, which events exist, the unlock set, the missions, the chat relay, the queue depth, the last bridge error |
| `sm_ap_resync` | generic | Fetch the unlock set again |
| `sm_ap_mission [number\|popfile]` | changemap | List the run's missions, or switch to one the run has unlocked |
| `sm_ap_report <kind> [wave]` | root | Report an objective by hand: `wave_cleared`, `mission_cleared` or `death` |

In the chat, `!ap status` prints the run for the player who asked, and asks
the bridge whether it is connected. `!mission` lists the missions.
`!mission <n>` switches, for an admin. The plugin refuses a mission the run
has not unlocked.

The plugin also decides which mission plays. It starts on
`tf2ap_start_mission`. It moves off any mission the run does not hold or has
not unlocked. After a mission clear, it loads the next unlocked mission that
is not cleared, `tf2ap_next_mission_delay` seconds later. If the game changes
the level first, the next map start loads the planned mission.

`sm_ap_report` tests the wiring without playing a wave. It also sends a
check by hand when the game fails to fire the expected event.

## ConVars

| ConVar | Default | What it does |
| --- | --- | --- |
| `tf2ap_bridge_url` | `http://127.0.0.1:24680` | Base URL of the bridge. Loopback. |
| `tf2ap_announce` | `1` | Announce grants and cleared waves in chat |
| `tf2ap_chat` | `1` | Show what the rest of the multiworld says |
| `tf2ap_debug` | `1` | Every bridge call and game event: `0` none, `1` console and log, `2` chat as well |
| `tf2ap_start_mission` | empty | The popfile the server starts on |
| `tf2ap_next_mission_delay` | `30` | Seconds from a mission clear to the next mission. `0` leaves it to the game |
| `tf2ap_bot_upgrades_chat` | `0` | Say what the defender bots buy at the upgrade station |

The plugin reports a lost wave only while a wave it saw start is running.
The game fires `mvm_wave_failed` while a mission loads. A live server sent a
Death Link to the whole multiworld fourteen seconds after a map change, with
nobody playing.

The plugin does two things for the defender bots. When a human arrives and
RED is full of bots, it kicks one bot so the human has a seat. With
`tf2ap_bot_upgrades_chat 1`, it names every `MVM_Upgrade` a bot sends in the
chat. It reads the upgrade names out of `scripts/items/mvm_upgrades.txt`.

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
the difference. Run the first live session with `tf2ap_debug 2`, which puts the
same lines in the chat where somebody is watching.

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
- **`gamedata/` owns wave counts**, rather than parsing mutable `.pop` files at
  runtime. Valve counts are hardcoded and community counts come from the
  versioned `community.json` manifest.
