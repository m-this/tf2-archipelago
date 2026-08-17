# Install

Run everything from the root of the repository. [Without the
repository](#without-the-repository), at the end of this page, does the same
with two downloaded files.

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

That password unlocks the remote console of the game server. You need it for
the admin commands. Nobody else needs it.

## 3. Give the stack a session to play

The stack plays a session that already exists. [Create the
session](create-the-session.md) makes one, hosts it on `archipelago.gg`, and
gives you the address of a room. Write that address into `.env`:

```sh
AP_HOST=archipelago.gg
AP_PORT=12345
AP_TLS=true
```

`SRCDS_RCONPW`, `AP_HOST` and `AP_PORT` are the three values with no default.
The stack refuses to start without one of them, and it prints which one.

## 4. Start the stack

```sh
make up
make logs
```

`make up` builds two images and starts two containers. `make logs` follows the
output of both. Stop following with Ctrl-C. That does not stop the stack.

## What the first start looks like

The build compiles the game plugin and the bridge. It takes a few minutes.

Then, in this order:

1. The game server starts and downloads about 14 GB of game files. This is the
   long part. How long depends on your connection.
2. The stack installs the plugin into the game files once they arrive. The log says
   `[AP] installed the plugin and ripext into /home/steam/tf-dedicated/tf`.
3. The bridge joins the room. Its log says `connected to archipelago slot=tf2
   missions=8`, and the room page says
   `tf2 (Team #1) playing Team Fortress 2 Mann vs Machine has joined`.

Every later start takes seconds. The stack downloads nothing again.

## The services

| Service | What it does | Ports |
| --- | --- | --- |
| `srcds` | Runs the Team Fortress 2 dedicated server and the plugin | `27015/udp` and `27015/tcp`, the only public ports |
| `bridge` | Holds the session with the room and answers the plugin | none, loopback inside the network namespace of the game server |

A third service, `archipelago`, hosts the session on this machine instead of on
`archipelago.gg`. It starts only with `COMPOSE_PROFILES=selfhost` in `.env`. See
[Create the session](create-the-session.md).

The bridge shares the network namespace of the game server. That is how the
plugin reaches it at `127.0.0.1` and nothing outside the machine can.
Restarting the game server restarts the bridge with it. That costs seconds, not
progress: the bridge writes every check to disk.

## The commands

| Command | What it does |
| --- | --- |
| `make seed` | Make a session in `seed/`, to upload to `archipelago.gg` |
| `make up` | Start the stack |
| `make logs` | Follow the output of the services |
| `make ps` | List the containers and their state |
| `make down` | Stop the stack. Keeps the game files and the run. |
| `make restart` | `make down` then `make up` |
| `make build` | Rebuild the images |
| `make clean` | Stop the stack and delete every volume, including the 14 GB of game files |
| `make check` | Run everything that the continuous integration runs |
| `make integration` | Start a real randomizer server and a real bridge, and drive them the way the plugin does |
| `make dist` | Build everything a release attaches into `dist/`: the `.apworld`, the plugin, the exported data, and the compose file below |

`make clean` deletes the game files. Use `make down` unless you mean it.

## Where the stack keeps things

| Volume | Holds | Delete it to |
| --- | --- | --- |
| `tf2-archipelago_tf2game` | The 14 GB of game files, SourceMod and the plugin | Download everything again |
| `tf2-archipelago_bridgestate` | The checks and the unlocks of the current run | Nothing useful. The bridge rebuilds the checks from the room. |
| `tf2-archipelago_apoutput` | The session, with `COMPOSE_PROFILES=selfhost` only | [Start a new run](../operate/start-a-new-run.md) |

The sessions themselves are files in `seed/`, in the repository. Git ignores
that directory, and nothing deletes it for you.

## Without the repository

Every release attaches a `compose.yaml` that names published images instead of
building them, and the `.env.example` that goes with it. A machine with Docker
needs nothing else: no clone, no Go, and no compiler.

```sh
mkdir mann-vs-archipelago && cd mann-vs-archipelago
base=https://github.com/m-this/tf2-archipelago/releases/latest/download
curl -fsSLO "$base/compose.yaml"
curl -fsSL -o .env "$base/.env.example"
```

Set `SRCDS_RCONPW` in `.env`, then make a session and start:

```sh
docker compose --profile seed run --rm seed   # writes ./seed
docker compose up -d
docker compose logs -f
```

Steps 3 and 4 of [Create the session](create-the-session.md) apply as they
stand. Upload the file from `seed/`, create a room, and write the port of the
room into `AP_PORT`.

The images come from `ghcr.io/m-this/tf2-archipelago`. The `compose.yaml` that
you download pins them to the release it came from, so the stack keeps the
version you installed. `TF2AP_VERSION` in `.env` picks another one:

```sh
TF2AP_VERSION=v1.0.0
```

```sh
docker compose pull && docker compose up -d
```

The commands in the table above are `make` targets, and they need the
repository. `docker compose` does each of them on its own: `up -d`, `logs -f`,
`ps`, `down`, and `down -v`.

The [releases page](https://github.com/m-this/tf2-archipelago/releases) also
attaches `tf2_mvm.apworld` and `tf2_archipelago.smx`, for an Archipelago install
or a game server that this compose file does not run.
