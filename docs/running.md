# Host a server

This page is for the host. A player needs
[archipelago-101.md](archipelago-101.md) and the address of the server.

## What you need

- Docker with the compose plugin.
- About 20 GB of disk space. The game server downloads about 14 GB at the first
  start.
- Two cores and 4 GB of memory for six players.
- One UDP port that your friends can reach. The game uses it.

## Start the stack

1. Copy the example environment file: `cp deploy/.env.example .env`.
2. Set `SRCDS_RCONPW`. It has no default and the stack refuses to start without
   it.
3. Start the stack: `make up`.
4. Read the output: `make logs`.

`make up` builds three images and starts them:

| Service | What it does | Ports |
| --- | --- | --- |
| `archipelago` | Hosts the multiworld on the seed that it generates at the first start | none, internal |
| `srcds` | Runs the Team Fortress 2 dedicated server | `27015/udp` and `/tcp`, the only public ports |
| `bridge` | Holds the Archipelago session and answers the plugin | none, loopback inside the network namespace of the game server |

The first start is slow: the game files arrive before the server does. Each
later start takes seconds.

## The shape of the run

Set these in `.env` before the first start. The stack generates the seed once
and then keeps it.

| Variable | Default | What it decides |
| --- | --- | --- |
| `MVM_MISSION_COUNT` | `8` | How many missions the run uses. Eight take about one evening. |
| `MVM_DIFFICULTY` | `intermediate` | The easiest tier that the run can draw. It also draws every tier above it. |
| `MVM_GOAL` | `final_boss` | `final_boss` or `missionsanity` |
| `MVM_MISSIONSANITY_PERCENTAGE` | `80` | How much of the run Missionsanity asks for |

A change to these values does nothing on its own. To start a new run:

```sh
docker compose -f deploy/compose.yml down
docker volume rm tf2-archipelago_apoutput
make up
```

The bridge reads the new seed, drops the checks of the previous run and starts
again. No other event makes it drop them.

## Connect

A player opens the console and connects:

```
connect your.server.address:27015
password <SRCDS_PW>      // only if you set one
```

All the players join RED. The host selects the mission: `SRCDS_STARTMAP` sets
it. Change it with rcon between missions, or edit `.env` and restart the stack.

## What to expect

The plugin writes a welcome in the chat eight seconds after a player spawns. The
welcome names the server, the mission, the unlocks and the `!ap` command.

The plugin also writes each wave that the team clears and each item that the run
receives. It writes the failures the same way. The chat says when the bridge
cannot reach the multiworld, so that nobody decides that the randomizer is
broken.

## When something looks wrong

Read the output of the three services:

```sh
make logs
docker compose -f deploy/compose.yml logs bridge
```

An admin has four commands in the game:

```
sm_ap_status          // mission, wave, events, unlocks, queue depth, last error
sm_ap_resync          // ask the bridge for the unlock set again
sm_ap_report wave_cleared 3    // report a check by hand
tf2ap_debug 1         // write every bridge call and game event to the chat
```

Run `sm_ap_status` first. Nobody checked the plugin against a live server, and
the line about which events exist answers most questions.

**The bridge does not lose a check.** It writes the check to disk before it
answers the game server, and it sends the check upstream after that. An
Archipelago server that is down for an hour costs nothing: the checks arrive
when it comes back. A restart of any of the three services costs nothing either.

## What CI tests, and what nobody tested

CI runs the part below the plugin against a real Archipelago server on a real
seed. Start the same test with `make integration`.

Nobody ran the plugin. This machine had no Team Fortress 2 server. The game
events that the plugin hooks and the entity properties that it reads are
guesses with fallbacks. Set `tf2ap_debug 1` for the first session.
