# bridge

Go. The Archipelago client. The only component that knows the AP protocol, and
via `gamedata/` the only component that knows the id mapping. Read [ADR
0002](../docs/adr/0002-server-side-plugin-with-a-go-bridge.md) first.

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

## Layout

| Package | Holds |
| --- | --- |
| `cmd/bridge` | Entrypoint: config, signals, the two goroutines |
| `internal/config` | The environment, read once at startup |
| `internal/state` | The check list, the item list, and everything derived from them |
| `internal/apclient` | The Archipelago session and the goal condition |
| `internal/chat` | The multiworld's conversation, bounded and not durable |
| `internal/httpapi` | The routes the plugin calls |

There is no separate queue package. The durable queue *is* the check list in
`state`: written before the plugin is answered, and resent in full on every
reconnect. Archipelago ignores repeats, and 210 ids is not a set worth tracking
acknowledgements for.

Nothing is shared with `plugin/` except the wire format, which is documented in
the HTTP API and nowhere else.

## The API

| Method | Path | Body | Returns |
| --- | --- | --- | --- |
| `POST` | `/objective` | `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":3}` | `204` once the check is on disk |
| `POST` | `/objective` | `{"kind":"mission_cleared","popfile":"mvm_coaltown"}` | `204` |
| `GET` | `/unlocks` | | `{"seq":6,"classes":[…],"slots":[…],"missions":[…]}` |
| `GET` | `/grants?since=6` | | `{"seq":8,"grants":[…]}`, held open until there is something past that sequence |
| `GET` | `/messages?since=-1` | | the multiworld's chat, long-polled. A negative sequence means "start from now" |
| `POST` | `/say` | `{"text":"!hint Scout"}` | `204`, or `503` when there is no multiworld to say it to |
| `GET` | `/healthz` | | the session state and the missions in the seed |

An objective naming a mission or a wave that does not exist is a `400`, not a
silent drop: it means the plugin and the tables disagree.

`/grants` always answers with the sequence the bridge is at, even when it has
nothing new. That is how the plugin learns it is *ahead*: the only way to be
ahead is that the run restarted underneath it, and then it has to refetch the
unlock set rather than wait for grants that will never arrive.

Chat is the one thing here that is not durable. A check is a fact about a run
and must survive anything; a line someone typed while the game server was
restarting is gone, the way it would be in any other chat. `/say` refuses
rather than queues for the same reason: a message that lands ten minutes late
is worse than one that was refused, and the player is standing right there to
be told.

Credits appear in `/grants` and never in `/unlocks`. A cash bundle is applied
once when it arrives; re-applying it on every map change would print money.

## Configuration

| Variable | Default | What it does |
| --- | --- | --- |
| `AP_HOST` | `archipelago` | Archipelago server host |
| `AP_PORT` | `38281` | Archipelago server port |
| `AP_TLS` | `false` | `wss://` instead of `ws://` |
| `AP_SLOT_NAME` | `tf2` | The slot to claim. One slot for the whole game server. |
| `AP_PASSWORD` | empty | Multiworld password |
| `BRIDGE_LISTEN` | `127.0.0.1:24680` | Plugin-facing address. Loopback. |
| `BRIDGE_STATE` | `/data/bridge.json` | The state file |
| `BRIDGE_POLL_TIMEOUT` | `25s` | How long `/grants` is held open |

## Running it against a real server

```sh
go build ./bridge/cmd/bridge
AP_HOST=127.0.0.1 AP_SLOT_NAME=TF2 BRIDGE_STATE=./state/bridge.json ./bridge
curl -X POST localhost:24680/objective \
  -d '{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":1}'
```

The check shows up in the Archipelago server log as
`TF2 sent <item> to TF2 (Crash Course Wave 1)`.
