# Install

Run everything from the root of the repository.

## 1. Write the configuration file

```sh
cp deploy/.env.example .env
```

`.env` is the one file that you edit. Git ignores it.

## 2. Set the console password

Open `.env` and set `SRCDS_RCONPW` to a password of your choice:

```sh
SRCDS_RCONPW=pick-something-long
```

This is the only value with no default. The stack refuses to start without it
and prints `set SRCDS_RCONPW in .env`.

That password unlocks the remote console of the game server. You need it for
the admin commands. Nobody else needs it.

Now is also the moment to set the shape of the run. The stack generates the
randomized session at the first start and then keeps it, so a change made later
does nothing until you start a new run. See
[The shape of the run](shape-of-the-run.md).

## 3. Start the stack

```sh
make up
make logs
```

`make up` builds three images and starts three containers. `make logs` follows
the output of all three. Stop following with Ctrl-C. That does not stop the
stack.

## What the first start looks like

The build compiles the game plugin and the bridge. It takes a few minutes.

Then, in this order:

1. The randomizer server sees an empty output directory and generates the
   session. The log says `no seed in /ap/output, generating one`, then
   `hosting /ap/output/AP_<number>.zip on port 38281`. This takes under a
   minute.
2. The game server starts and downloads about 14 GB of game files. This is the
   long part. How long depends on your connection.
3. The plugin is installed into the game files once they arrive. The log says
   `[AP] installed the plugin and ripext into /home/steam/tf-dedicated/tf`.
4. The bridge connects to the randomizer server. The randomizer log says
   `tf2 (Team #1) playing Team Fortress 2 Mann vs Machine has joined`.

Every later start takes seconds. Nothing is downloaded again and no new session
is generated.

## The three services

| Service | What it does | Ports |
| --- | --- | --- |
| `archipelago` | Generates the randomized session at the first start, then hosts it | none, internal to the stack |
| `srcds` | Runs the Team Fortress 2 dedicated server and the plugin | `27015/udp` and `27015/tcp`, the only public ports |
| `bridge` | Holds the session with the randomizer server and answers the plugin | none, loopback inside the network namespace of the game server |

The bridge shares the network namespace of the game server. That is how the
plugin reaches it at `127.0.0.1` and nothing outside the machine can.
Restarting the game server restarts the bridge with it. That costs seconds, not
progress: the bridge writes every check to disk.

## The commands

| Command | What it does |
| --- | --- |
| `make up` | Start the stack |
| `make logs` | Follow the output of the three services |
| `make ps` | List the containers and their state |
| `make down` | Stop the stack. Keeps the game files, the session and the run. |
| `make restart` | `make down` then `make up` |
| `make build` | Rebuild the three images |
| `make clean` | Stop the stack and delete every volume, including the 14 GB of game files |
| `make check` | Run everything that the continuous integration runs |
| `make integration` | Start a real randomizer server and a real bridge, and drive them the way the plugin does |

`make clean` deletes the game files. Use `make down` unless you mean it.

## Where the stack keeps things

| Volume | Holds | Delete it to |
| --- | --- | --- |
| `tf2-archipelago_tf2game` | The 14 GB of game files, SourceMod and the plugin | Download everything again |
| `tf2-archipelago_apoutput` | The generated session | [Start a new run](../operate/start-a-new-run.md) |
| `tf2-archipelago_bridgestate` | The checks and the unlocks of the current run | Nothing useful. The bridge rebuilds the checks from the randomizer server. |
