# Troubleshooting

Three things can be wrong:

1. The plugin does not see the game.
2. The plugin cannot reach the bridge.
3. The bridge cannot reach the room.

This page finds out which. If you ask for help, send the debug bundle first:
**Settings**, then **Debug logs**, in the launcher. It holds the launcher log,
the server console and your settings, without passwords.

Open `summary.txt` in it before anything else. It names the TF2 build the
server was running beside every version the launcher installed, and says when
two of them cannot work together. `collected.txt` beside it says which files
the bundle could not take, and why.

## Read the logs

- **Launcher:** the log at the bottom of the **Play** tab. **Filter the log**
  narrows it.
- **Docker:** `make logs` follows every service. For one service:

```sh
docker compose --project-directory . \
  --env-file deploy/env/versions.env --env-file .env \
  -f deploy/compose.yml logs -f bridge
```

Replace `bridge` with `srcds`, or with `archipelago` when the stack hosts the
session itself. `make ps` lists the containers.

## Ask the game server

Type `sm_ap_status` in the rcon box under the log, or `rcon
sm_ap_status` in the game console. The answer looks like this:

```text
[AP] version 0.1.0, mvm yes, mission mvm_decoy, wave 3 of 8
[AP] events: begin_wave yes, wave_complete yes, mission_complete yes
[AP] unlocks held at sequence 6, 0 objective(s) waiting to be sent
[AP] classes: scout, medic
[AP] slots: primary
[AP] Last bridge error: ...
```

Read it in this order:

- `mvm no`: the plugin does not think this is Mann vs Machine. Nothing is
  reported on a map that is not an MvM map.
- `events: ... no`: your game version does not send that event. Report it.
  With `wave_complete no`, the plugin watches the wave counter instead.
- `unlocks NOT FETCHED`: the plugin never got an answer from the bridge. Until
  it does, it locks nothing.
- `N objective(s) waiting to be sent`: the bridge is not answering. The plugin
  retries every five seconds.
- `Last bridge error`: the last thing that went wrong.

`sm_ap_resync` asks the bridge for the unlock set again. Try it first when the
unlocks in the chat look stale.

## Ask the bridge

The bridge serves one page with everything it knows. In the launcher, the
**Play** tab shows it. With Docker, the page is on loopback inside the
game server's network namespace:

```sh
docker run --rm --network container:tf2-archipelago-srcds-1 \
  curlimages/curl:latest -s 127.0.0.1:24680/healthz
```

| Field | What it tells you |
| --- | --- |
| `connected` | Whether the session with the room is up right now |
| `slot` | The name of your server in the session |
| `missions` | The missions the run drew |
| `seed` | The identity of the current session |
| `checks` | How many checks the run holds |
| `items` | How many items the run received |
| `acked_seq` | How far the plugin confirmed it applied |
| `goal_sent` | Whether the run is finished |
| `last_check` and `last_check_at` | The last check and when it landed |
| `wave_drift` | Missions whose wave count the game disagrees with |
| `wave_failures` | Every wave the team lost, worst first |
| `last_error` | The last failure on the room side |

`last_check` answers "did that wave count". `wave_failures` answers "which
waves stopped us", which is the question to ask before you change the team
size or the bots.

### Metrics

The bridge also serves the same numbers as Prometheus metrics on port `24681`, on
loopback by default. `BRIDGE_METRICS_BIND` in `.env` opens it to another
machine.

```sh
curl -s 127.0.0.1:24681/metrics
```

| Metric | What it tells you |
| --- | --- |
| `tf2ap_session_connected` | 1 while the session with the room is up |
| `tf2ap_session_missions` | How many missions the run drew |
| `tf2ap_run_checks_total`, `tf2ap_run_items_total` | Checks sent, items received |
| `tf2ap_run_acked_seq` | How far the plugin confirmed it applied |
| `tf2ap_run_goal_sent` | 1 once the run is finished |
| `tf2ap_run_last_check_timestamp_seconds` | When the last check landed |
| `tf2ap_mission_wave_drift` | One series per mission the game and the tables disagree about. None is the healthy case. |
| `tf2ap_wave_lost_total` | One series per wave the team lost |
| `tf2ap_run_info` | The seed and slot the numbers belong to |
| `tf2ap_game_up` | 1 when the game server answered a query on that scrape |
| `tf2ap_game_players`, `_bots`, `_players_human` | Who is on the server |
| `tf2ap_game_map` | The mission it is on |

The player counts come from a query to the game server, cached for ten
seconds. The game server refuses a source that queries it too often, so do not
lower that.

## When the room is down

Nothing is lost. The bridge writes each check to disk before it answers the
game server, and sends it upstream afterwards. The bridge reconnects on its
own. Cleared waves keep counting. Received items arrive when the room comes
back.

A bridge that never connects once is a different problem. Check the room
address. With Docker, also check `AP_TLS`: a room on `archipelago.gg` needs
`AP_TLS=true`, a room hosted in the stack needs `AP_TLS=false`.

## When the bridge is down

The plugin holds its checks in memory and retries every five seconds. The chat
says once that the bridge is unreachable.

In the launcher, the bridge runs beside the game server. **Restart** brings
both back. With Docker, the bridge restarts with the game server. The checks
are on disk, and the unlock set is rebuilt from them.

If the bridge's state file is lost, the run is not. The room holds the same
list of checks and sends it at each connection. Only the item history is lost.

## Recovering a check by hand

There is one gap: the seconds between a cleared wave and the bridge taking the
check. The plugin's queue is in memory. If the game server crashes while the
bridge is unreachable, that queue is gone.

The plugin writes every check to the SourceMod log twice: once when it queues
it, and once when the bridge has it on disk.

```text
objective wave_cleared mvm_decoy wave 3 (mission length 8) queued for the bridge
objective wave_cleared mvm_decoy wave 3 is on the bridge's disk
```

A `queued` line with no matching `on the bridge's disk` line is a check that
never landed. Replay it, on the map the check belongs to:

```text
sm_ap_report wave_cleared 3
```

## Docker: never restart the game server on its own

The bridge lives inside the game server's network namespace. `docker compose
up -d srcds` alone leaves the bridge attached to a namespace that no longer
exists. It reports healthy and reaches nothing.

Recreate the whole stack:

```sh
make up
```

## When the wave counts are wrong

Every wave count comes from the wiki, and the game is the authority. A wrong
count makes a mission clear fire one wave early, or never.

The bridge compares the length the game reports with its table and serves the
disagreements as `wave_drift`:

```json
"wave_drift": [
  {"popfile": "mvm_decoy", "tables": 8, "observed": 7}
]
```

A mission that appears there is a row to correct in `gamedata/missions.go`.
Report it. The check still counts.

## Messages in the chat

| The chat says | What it means |
| --- | --- |
| `[AP] The run did not unlock mvm_decoy. Its checks still count.` | The server runs a mission whose ticket the run has not found. A warning, not a refusal. |
| `[AP] The bridge speaks API version 2 and this plugin speaks 1.` | You updated one half and not the other. **Repair** in the launcher, or `make build` and `make up` with Docker. |
| `[AP] The bridge is unreachable.` | See [When the bridge is down](#when-the-bridge-is-down). |

## Windows: the launcher will not start

- SmartScreen or Defender blocked it. See
  [Install on Windows](../setup/install-windows.md#windows-will-warn-you).
- The install folder is full. The game server needs about 20 GB free.
- Something else uses the game port. Change **Game port** on the **Game
  server** page.

## Windows: Generate seed fails

- The launcher cannot find the Archipelago app. Set **Archipelago app** on the
  **Player options** page to the app's folder.
- The Archipelago version does not match. `tf2ap.exe -version` prints the one
  the launcher pins. Install that one.
- The pool is too small for the items. Press **Check Run Selection** on the
  **Missions** page. Add missions, or turn on victory caches. See
  [Run options](../setup/shape-of-the-run.md#more-checks).
