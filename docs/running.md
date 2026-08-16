# Running it

For the person hosting. Players need [archipelago-101.md](archipelago-101.md)
and the server address, nothing else.

## What you need

- Docker with the compose plugin.
- About 20 GB of disk. The game server downloads roughly 14 GB on first start.
- Two cores and 4 GB of RAM is enough for six players.
- A UDP port reachable by your friends. One, and it is the game's.

## Starting it

```sh
cp deploy/.env.example .env
$EDITOR .env          # SRCDS_RCONPW has no default and the stack refuses to start without it
make up
make logs
```

`make up` builds three images and starts them:

| Service | What it is | Ports |
| --- | --- | --- |
| `archipelago` | The multiworld server, on a seed it generates on first start | none, internal |
| `srcds` | The Team Fortress 2 dedicated server | `27015/udp` and `/tcp`, the only public ones |
| `bridge` | Holds the Archipelago session, serves the plugin | none, loopback inside the game server's network namespace |

First start is slow: the game files download before the server comes up. Later
starts take seconds.

## The run's shape

Set in `.env` before the first start, because the seed is generated once and
kept:

| Variable | Default | What it decides |
| --- | --- | --- |
| `MVM_MISSION_COUNT` | `8` | How many missions the run spans. Eight is about an evening. |
| `MVM_DIFFICULTY` | `intermediate` | The easiest tier that may appear; every harder one comes with it |
| `MVM_GOAL` | `final_boss` | `final_boss` or `missionsanity` |
| `MVM_MISSIONSANITY_PERCENTAGE` | `80` | How much of the run missionsanity wants |

Changing these later does nothing on its own. To roll a new run:

```sh
docker compose -f deploy/compose.yml down
docker volume rm tf2-archipelago_apoutput
make up
```

The bridge notices the new seed, drops the previous run's checks, and starts
over. It does not drop them for any lesser reason.

## Joining

Players open the console and connect:

```
connect your.server.address:27015
password <SRCDS_PW>      # only if you set one
```

Everyone joins RED. Which mission is loaded is the server's business:
`SRCDS_STARTMAP` decides it, and it can be changed between missions with rcon
or by editing `.env` and restarting.

## What to expect

Eight seconds after a player spawns in, the plugin tells them in chat what the
server is, which mission is loaded, what is unlocked, and how to use `!ap`.

Cleared waves are announced. Received items are announced. So are failures: if
the bridge cannot reach the multiworld, chat says so rather than leaving people
to conclude the randomizer is broken.

## When something looks wrong

```sh
make logs                                    # all three services
docker compose -f deploy/compose.yml logs bridge
```

In game, as an admin:

```
sm_ap_status          # mission, wave, which events exist, unlocks, queue depth, last error
sm_ap_resync          # ask the bridge for the unlock set again
sm_ap_report wave_cleared 3    # report a check by hand
tf2ap_debug 1         # echo every bridge call and game event into chat
```

`sm_ap_status` is the first thing to run. The plugin's reading of Mann vs
Machine has never been checked against a live server, and the line it prints
about which events exist is the answer to most of what could be wrong.

**Checks are not lost when things break.** The bridge writes a check to disk
before it tells the game server it has it, and sends it upstream afterwards. An
Archipelago server that has been down for an hour costs nothing; the checks
land when it comes back. The same is true across a restart of any of the three
services.

## What has and has not been tested

Everything up to the plugin's HTTP calls runs in CI, against a real Archipelago
server on a real generated seed: `make integration`.

The plugin itself has been compiled and never run. No Team Fortress 2 server
was available while it was written, so the game events it hooks and the entity
properties it reads are informed guesses with fallbacks. The first session
should be run with `tf2ap_debug 1`.
