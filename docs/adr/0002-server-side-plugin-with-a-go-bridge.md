# ADR 0002 — Server-side SourceMod plugin talking to a Go bridge, no client mod

- **Status**: Accepted
- **Date**: 2026-08-13
- **Deciders**: project owner
- **Related**: `docs/spec.md`, ADR 0001

## Context

Something has to sit between a running TF2 server and the Archipelago
multiworld. It has to detect in-game events and turn them into
`LocationChecks`, and it has to receive `ReceivedItems` and turn them into
in-game unlocks.

Three constraints shape the answer:

1. **Friends must be able to join with a stock TF2 client.** This is a game
   people play together on a whim. Anything that requires every participant to
   install a mod will not get played.
2. **The Archipelago protocol is a websocket session with real liveness
   requirements.** The client must handle `ws://` and `wss://`, must reconnect
   after a drop, and must not lose or double-apply items across a reconnect.
3. **Only SourceMod can see the game.** Wave completion, tank deaths, the
   upgrade station, weapon restrictions: all of it is SourcePawn territory,
   inside the `srcds` process.

SourcePawn is a poor place for constraint 2. The SourceMod websocket extension
is unmaintained, `ripext` (REST in Pawn) does HTTP and JSON well but its
websocket support is Socket.IO-flavoured rather than raw, and SourcePawn has no
real concurrency model, no durable storage beyond a SQLite handle, and a
blocking call in a game frame stalls the server for everyone on it.

## Decision

Split it in two, along the line where the constraints change.

**The SourceMod plugin sees the game and nothing else.** It detects objectives
and it applies unlocks and traps. It speaks in MvM vocabulary
(`wave_cleared`, `mission_cleared`, `tank_destroyed`, `grant_weapon_slot`,
`apply_trap`) and knows nothing about Archipelago: no item ids, no location
ids, no slot, no seed. It reaches the bridge over plain HTTP and JSON via
`ripext`, non-blocking, on `127.0.0.1`.

**The Go bridge owns the Archipelago session.** It holds the websocket,
reconnects, replays, deduplicates, and persists. It is the only component that
knows the AP protocol, and via `gamedata/` (ADR 0001) it is the only component
that knows the id mapping.

**No client-side mod.** Everything the player sees is delivered through
channels a vanilla client already renders: chat, HUD text, annotations, the
existing upgrade station UI.

Supporting decisions:

- **The plugin never holds authoritative state.** After a plugin reload, a map
  change or an `srcds` crash, it asks the bridge for the full current unlock
  set and applies it. There is exactly one source of truth for "what has this
  slot received", and it is the bridge's on-disk state, not the game server's
  memory.
- **Checks are queued durably before they are acknowledged.** The plugin
  reports a check, the bridge writes it to disk, then returns 200. Only then
  does it try to send it upstream. A check that reaches the bridge is never
  lost, even if the AP server has been down for an hour. Losing a wave clear
  that took ten minutes to earn is not an acceptable failure mode.
- **Checks are idempotent by construction.** A check is identified by its
  location id, not by an event. Reporting the same wave clear twice is a
  no-op. This matters because the plugin will retry on timeout and cannot know
  whether the first attempt landed.
- **The bridge exposes long-poll, not push.** The plugin asks "anything new
  since sequence N" and the bridge holds the request open. `srcds` is not a
  server we want listening for inbound connections beyond the game port.
- **The bridge binds to loopback only.** Per the house rule on ports. The game
  port is the one exception in this stack, and it is the game port, not ours.

## Consequences

**Positive**

- Reconnect, replay, dedup, backoff and durable state are written once, in Go,
  with tests and a race detector. None of that is expressible in SourcePawn at
  a quality anyone would trust.
- The plugin stays small and stays about the game. It can be reloaded during a
  session without losing progress.
- Vanilla clients join. This is the difference between a project that gets
  played and one that gets demoed once.
- The bridge is testable without a game server: point it at an Archipelago
  server and drive its HTTP API with `curl`.

**Negative**

- Three processes to run instead of one, and a compose file that has to keep
  them in the right order.
- A local HTTP hop on every game event. Irrelevant at MvM's event rate, which
  is a handful of events per wave, but it does mean the plugin has to handle
  the bridge being down, which is one more code path.
- `ripext` becomes a hard dependency of the plugin, so it is one more thing to
  install into the `srcds` container and one more thing that can break on a
  SourceMod update.
- Anything the player must see has to fit through chat, HUD text or
  annotations. No custom UI, ever, unless the no-client-mod decision is
  revisited.

## Alternatives considered

- **Websocket straight from SourcePawn.** Rejected: the extension is
  unmaintained, and even if it worked, the reconnect and durability
  requirements land in a language with no good way to meet them.
- **A client-side mod, like most apworlds have.** Rejected on constraint 1.
  This also fails for a second reason: MvM state is server-authoritative, so a
  client mod would have to infer wave completion from what it can see, which
  is strictly worse information than the server already has.
- **No plugin at all, play on official Boot Camp or Mann Up servers**
  (Roseburst raised this as possibly "more practical"). Rejected: Valve
  servers run no plugins, so there is no way to gate weapons or upgrades, and
  the entire item side of the design disappears. What is left is an
  achievement tracker.
- **Put the AP client logic in a SourceMod extension** (C++, loaded into
  `srcds`). Rejected: it gets the concurrency but inherits the blast radius. A
  crash in the AP client would take the game server down with it, and a
  segfault mid-wave is a worse outcome than a dropped websocket.
- **Have the bridge read the `srcds` console log** instead of running a plugin.
  Rejected for the item direction: reading the log gives events but there is no
  way to write back, so nothing can be unlocked.
