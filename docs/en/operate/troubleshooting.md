# Troubleshooting

Three things can be wrong: the game server does not see the game, the plugin
cannot reach the bridge, or the bridge cannot reach the randomizer server. This
page finds out which.

## Read the logs

```sh
make logs
```

That follows every service the stack runs. For one service, use the full
compose command.
The stack needs two environment files, so the short form does not work:

```sh
docker compose --project-directory . \
  --env-file deploy/env/versions.env --env-file .env \
  -f deploy/compose.yml logs -f bridge
```

Replace `bridge` with `srcds`, or with `archipelago` when the stack hosts the
session itself.

```sh
make ps
```

That lists the containers. The bridge reports `healthy` when its own interface
answers.

## Ask the game server

```
rcon_password your-SRCDS_RCONPW
rcon sm_ap_status
```

The answer is five lines:

```
[AP] version 0.1.0, mvm yes, mission mvm_decoy, wave 3 of 8
[AP] events: begin_wave yes, wave_complete yes, mission_complete yes
[AP] unlocks held at sequence 6, 0 objective(s) waiting to be sent
[AP] classes: scout, medic
[AP] slots: primary
[AP] Last bridge error: ...
```

Read them in this order:

- **`mvm no`** means that the plugin does not think this is Mann vs Machine.
  Nothing is reported on a map that is not an MvM map.
- **`events: ... no`** is the important one. It names which of the three Mann vs
  Machine game events this server actually sends. A `no` here means your game
  version does not send that event; report it. `wave_complete no` makes the
  plugin watch the wave counter instead.
- **`unlocks NOT FETCHED`** means that the plugin has never had an answer from
  the bridge. Until it does, it enforces nothing: a server where nobody can hold
  a weapon is worse than a wave played with too much kit.
- **`N objective(s) waiting to be sent`** counts the checks that the plugin
  holds. Anything above zero means that the bridge is not answering. It retries
  every five seconds.
- **`Last bridge error`** is the last thing that went wrong, in the plugin's own
  words.

`rcon sm_ap_resync` asks the bridge for the unlock set again. It is the first
thing to try when the unlocks in the chat look stale.

## Ask the bridge

The bridge serves one page with everything it knows. It lives on the loopback
interface inside the game server's network namespace, so the request has to be
made from there:

```sh
docker run --rm --network container:tf2-archipelago-srcds-1 \
  curlimages/curl:latest -s 127.0.0.1:24680/healthz
```

The container name comes from `make ps`.

The answer holds:

| Field | What it tells you |
| --- | --- |
| `api_version` | The wire version. The plugin says in the chat when it disagrees. |
| `connected` | Whether the session with the randomizer server is up right now |
| `slot` | The name of your server in the session |
| `missions` | The missions that the run drew |
| `death_link` | Whether the seed asked for Death Link |
| `seed` | The identity of the current session |
| `checks` | How many checks the run holds |
| `items` | How many items the run has received |
| `acked_seq` | How far the plugin has confirmed it applied |
| `goal_sent` | Whether the run has been declared finished |
| `last_check` and `last_check_at` | The last check and when it landed |
| `wave_drift` | Missions whose length the game disagrees with |
| `last_error` | The last failure on the randomizer side |

`last_check` answers "did that wave count".

## Watch the run over time

The same numbers are served as Prometheus metrics, on their own port, so a
dashboard can plot them instead of a person re-running the command above. That
port **is** published on the host — `BRIDGE_METRICS_BIND` decides who can reach
it, loopback by default:

```sh
curl -s 127.0.0.1:24681/metrics
```

| Metric | What it tells you |
| --- | --- |
| `tf2ap_session_connected` | 1 while the session with the randomizer server is up |
| `tf2ap_session_missions` | How many missions the run drew |
| `tf2ap_run_checks_total` / `tf2ap_run_items_total` | Checks sent, items received |
| `tf2ap_run_acked_seq` | How far the plugin confirmed it applied. Stuck behind the item count means the game server is not applying grants |
| `tf2ap_run_goal_sent` | 1 once the run is finished |
| `tf2ap_run_last_check_timestamp_seconds` | When the last check landed. Absent until one does |
| `tf2ap_mission_wave_drift` | One series per mission the game and the tables disagree about, valued at the difference. No series is the healthy case |
| `tf2ap_run_info` | The seed and slot the numbers belong to |
| `tf2ap_game_up` | 1 when the game server answered an A2S query on that scrape |
| `tf2ap_game_players` / `_bots` / `_players_human` | Who is on the server. MvM counts its robot waves as bots, so the people playing are players minus bots |
| `tf2ap_game_players_max` | What the server advertises — six, the RED slots. Not the 32 it must be started with to host MvM at all |
| `tf2ap_game_map` | The mission it is on, as a label |

The player counts come from an A2S query the bridge sends the game server, the
same thing a server browser asks. A server that does not answer reports
`tf2ap_game_up 0` and **no** counts, so a restarting srcds reads as missing
rather than as an empty server.

Two catches, both measured on a running server rather than guessed:

- srcds binds `0.0.0.0:27015` and answers a query sent to any of its interface
  addresses, but **drops** one sent to `127.0.0.1`. Address it by name
  (`srcds:27015`), even from inside its own network namespace — that is what
  `BRIDGE_GAME_QUERY` defaults to.
- srcds stops answering a source that queries it more than a few times a second
  (`sv_max_queries_sec`, over a 30 second window), and keeps refusing until that
  window drains. So the answer is cached for ten seconds: scrape `/metrics` in a
  loop and you still get one query every ten seconds. Without that, curling this
  endpoint a few times in a row makes the dashboard say the server is down.

`tf2ap_mission_wave_drift` is the one worth an alert: a wrong wave count on the
goal mission is what makes a seed unwinnable, and it cannot be repaired mid-run.

## When the randomizer server is down

Nothing is lost. The bridge writes each check to disk before it answers the
game server, and sends it upstream afterwards. A randomizer server that is down
for an hour costs nothing: the checks arrive when it comes back. The bridge
reconnects on its own, waiting longer between attempts up to thirty seconds.

Received items stop arriving while it is down. Cleared waves keep counting.

A bridge that never connects once is a different problem. Check `AP_HOST`,
`AP_PORT` and `AP_TLS`. A room on `archipelago.gg` answers `wss://` and needs
`AP_TLS=true`; a session inside the stack answers `ws://` and needs
`AP_TLS=false`. The wrong one fails every attempt, and the bridge logs the
failure each time.

## When the bridge is down

The plugin holds its checks in memory and retries every five seconds. The chat
says that the bridge is unreachable, once, so that nobody decides that the
randomizer is broken.

The bridge shares the network namespace of the game server, so restarting the
game server restarts the bridge too. It comes back on its own within seconds.
The checks are on disk and the unlock set is rebuilt from them.

If the state file of the bridge is lost, the run is not lost either. The
randomizer server holds the same list of checks and sends it at each
connection. The bridge adopts whatever it is missing. Losing the file costs the
item history, not the checks.

## Recovering a check by hand

There is one gap in all of that: the seconds between a cleared wave and the
bridge taking the check. The plugin's queue is in memory and holds at most 64
objectives. If the game server crashes while the bridge is unreachable,
whatever is in that queue is gone.

The plugin writes every objective to the SourceMod log twice: once when it
queues it, and once when the bridge has it on disk.

```sh
docker compose --project-directory . \
  --env-file deploy/env/versions.env --env-file .env \
  -f deploy/compose.yml exec srcds \
  bash -c 'grep objective /home/steam/tf-dedicated/tf/addons/sourcemod/logs/L*.log'
```

```
objective wave_cleared mvm_decoy wave 3 (mission length 8) queued for the bridge
objective wave_cleared mvm_decoy wave 3 is on the bridge's disk
```

A `queued` line with no matching `on the bridge's disk` line is a check that
never landed. Replay it:

```
rcon sm_ap_report wave_cleared 3
```

Run it on the map that the check belongs to. The plugin sends the mission that
the game is on, so replaying a Decoy check while the server runs Coal Town
records the wrong place.

## Never restart the game server on its own

The bridge lives inside the game server's network namespace, which is what puts
its API on a loopback nothing else can reach. The cost is that the game server
owns that namespace.

So `docker compose up -d srcds` on its own leaves the bridge attached to a
namespace that no longer exists. It keeps running, it still reports itself
healthy, and it can reach nothing: the plugin gets a refused connection and the
randomizer server sees the slot disconnect. `docker restart` does not fix it and
fails with `joining network namespace: No such container`.

Recreate it:

```sh
make up            # recreates the whole stack, which is always safe
```

Or, if only the bridge needs it:

```sh
docker compose --project-directory . \
  --env-file deploy/env/versions.env --env-file .env \
  -f deploy/compose.yml up -d --force-recreate bridge
```

`make up` and the Ansible role both recreate the whole project, so this only
happens when a single service is restarted by hand.

## When the wave counts are wrong

Every wave count in this project comes from the wiki, and the game is the
authority. A wrong count makes a mission clear fire one wave early, or never.

The plugin sends the mission length that the game reports with each check. The
bridge compares it with its own table and serves the disagreements as
`wave_drift`:

```json
"wave_drift": [
  {"popfile": "mvm_decoy", "tables": 8, "observed": 7}
]
```

An empty `wave_drift` after a full mission means that the table is right for
that mission. A mission that appears there is a row to correct in
`gamedata/missions.go`. The check still counts: the wave was cleared either
way.

## When a mission is not part of the run

```
[AP] The run did not unlock mvm_decoy. Its checks still count.
```

The server is running a mission whose ticket the run has not found. This is a
warning, not a refusal. The map rotation belongs to you.

## When the plugin and the bridge disagree

```
[AP] The bridge speaks API version 2 and this plugin speaks 1. Update the one that is behind.
```

One half of the stack was rebuilt and the other was not. Run `make build` and
`make up`.
