# bridge

Go. The Archipelago client. The only component that knows the AP protocol, and
via `gamedata/` the only component that knows the id mapping. Read [ADR
0002](../docs/en/adr/0002-server-side-plugin-with-a-go-bridge.md) first.

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
- **State can be replayed, effects cannot.** A class or a slot granted twice is
  the same as granted once, so it is resent whenever it is asked for. Credits
  are not, and neither are the traps that come later: an effect is sent once and
  is held back for good once the plugin acknowledges it. `ItemKind.OneShot` in
  `gamedata` is what tells the two apart, so a kind added there needs no second
  list here.
- **The state file is not the only copy.** Archipelago holds the same check list
  for the slot and sends it in `Connected`, and the bridge adopts whatever it is
  missing. Losing the file costs a run's item history, not its checks.
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
| `POST` | `/objective` | `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":3,"waves_total":6}` | `204` once the check is on disk |
| `POST` | `/objective` | `{"kind":"mission_cleared","popfile":"mvm_coaltown"}` | `204` |
| `GET` | `/unlocks` | | `{"resume_from":6,"unlocks":{"class":[…],"weapon_slot":[…],"mission_ticket":[…]}}` |
| `GET` | `/missions` | | the run's missions in the order the seed drew them, each with the map its popfile runs on and whether its ticket is held |
| `GET` | `/grants?since=6` | | `{"seq":8,"grants":[…]}`, held open until there is something past that sequence |
| `POST` | `/grants/ack` | `{"seq":8}` | `204`. Everything through that sequence is applied, so no effect below it is sent again |
| `GET` | `/messages?since=-1` | | the multiworld's chat, long-polled. A negative sequence means "start from now" |
| `POST` | `/say` | `{"text":"!hint Class: Scout"}` | `204`, `403` for a command that cannot be undone, `429` for a flood, `413` for a line too long, `503` when there is no multiworld to say it to |
| `GET` | `/healthz` | | the API version, the session, the run, and any mission the game says is a different length than the tables do |

An objective naming a mission or a wave that does not exist is a `400`, not a
silent drop: it means the plugin and the tables disagree.

`waves_total` is optional and never changes the answer. The wave counts in
`gamedata` come from the wiki and none has been checked against a running
server; a wrong one on the goal mission makes a seed unwinnable and nothing at
play time would notice. So the plugin sends what the game says, the bridge logs
the disagreement once per mission and serves it in `/healthz` as `wave_drift`.
The check itself still counts: the wave was cleared either way.

`/unlocks` is keyed by grant kind rather than by a field per kind. A kind added
to `gamedata` shows up with no change on either side of the wire, and a plugin
too old to know it skips it instead of failing to parse.

`/missions` exists because a popfile name does not carry the map it runs on:
`mvm_ghost_town_666` runs on `mvm_ghost_town`, so trimming the name is a guess
that fails. `gamedata` already holds the pairing, and the plugin needs it to
change mission. The unlock set answers "may we play this"; this answers "what is
there and how is it loaded".

`resume_from` is the acknowledged sequence, not how far the item list reaches,
and the difference is a lost effect. The unlock set carries state only, so a
cash bundle that landed while no plugin was listening is not in it. A cursor set
to the length of the item list would sit above that bundle and nothing would
ever hand it over. Resuming from the acknowledged sequence re-sends some state
the plugin already holds, which costs nothing by definition, and every effect it
never got.

`/grants` always answers with the sequence the bridge is at, even when it has
nothing new. That is how the plugin learns it is *ahead*: the only way to be
ahead is that the run restarted underneath it, and then it has to refetch the
unlock set rather than wait for grants that will never arrive.

**A sequence counts received items, not grants.** An item id this binary cannot
read is skipped, and skipping it leaves a gap rather than moving what follows.
Counting grants instead would renumber every later one the day a larger
`gamedata` makes that id readable, and the plugin would reapply grants it
already has and miss ones it does not — with nothing to notice it.

Chat is the one thing here that is not durable. A check is a fact about a run
and must survive anything; a line someone typed while the game server was
restarting is gone, the way it would be in any other chat. `/say` refuses
rather than queues for the same reason: a message that lands ten minutes late
is worse than one that was refused, and the player is standing right there to
be told.

Credits appear in `/grants` and never in `/unlocks`, because the unlock set is
state and credits are an effect. Re-applying a cash bundle on every map change
would print money.

That leaves the case where the plugin's cursor moves backwards, which happens on
a reload, a resync, and a seed change. State is simply resent. An effect is held
back once `/grants/ack` has carried it, and the acknowledgement is on disk, so a
plugin that reloads mid-run does not collect the whole run's cash a second time.
The plugin does not have to tell the two apart: it acknowledges what it applied,
and the bridge decides what that means.

The acknowledgement only moves forward, and never past the items that exist. A
resend shorter than what was held drags it back down with the list, because an
acknowledgement left past the end would suppress the fresh effects landing in
those slots, which is the one thing it must never do.

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
