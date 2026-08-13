# bridge

Go. The Archipelago client. The only component that knows the AP protocol, and
via `gamedata/` the only component that knows the id mapping. Read [ADR
0002](../docs/adr/0002-server-side-plugin-with-a-go-bridge.md) first.

Nothing exists yet.

## What it does

Northbound, to the Archipelago server, over websocket:

- Holds one session for the server's slot. `Connect`, `Connected`,
  `LocationChecks`, `ReceivedItems`, `StatusUpdate`, `Bounced`, `Say`.
- Handles `ws://` and `wss://`. Reconnects on its own with backoff, and
  replays anything queued while it was down.
- Deduplicates received items. Archipelago replays the full item list on
  reconnect, so "have I already applied this" is the bridge's problem, not the
  plugin's.

Southbound, to the SourceMod plugin, over HTTP and JSON on `127.0.0.1`:

- Accepts objective reports (`wave_cleared`, `mission_cleared`,
  `tank_destroyed`, …) and translates them to location ids.
- Serves the current unlock set, so the plugin can resync after a reload or a
  map change without holding state.
- Long-poll for new grants: the plugin asks "anything since sequence N" and the
  request is held open. Nothing pushes into `srcds`.

## Invariants

These are the ones that matter. Everything else is detail.

- **A check that reaches the bridge is never lost.** Write to the durable queue
  first, return 200 second, send upstream third. Losing a wave clear that cost
  ten minutes of play is not an acceptable failure mode, and the AP server
  being down for an hour must not cost anything.
- **Checks are idempotent.** A check is identified by its location id. The
  plugin retries on timeout and cannot know whether the first attempt landed,
  so reporting the same wave clear twice has to be a no-op.
- **The bridge is the sole authority on the unlock set.** Not the plugin, not
  `srcds` memory. On-disk state survives a bridge restart.
- **Loopback only.** The bridge never binds a public interface.

## Layout, when it exists

Standard Go: `cmd/bridge/` for the entrypoint, `internal/` for the rest. Likely
packages: `apclient` (websocket session), `queue` (durable check queue),
`httpapi` (the plugin-facing API), `state` (unlock set persistence).

Nothing shared with `plugin/` except the wire format, which is documented in
the HTTP API and nowhere else.
