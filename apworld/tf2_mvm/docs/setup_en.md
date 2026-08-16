# Setup Guide for Team Fortress 2 Mann vs Machine

## Required software

Players need nothing beyond Team Fortress 2. The randomizer is server-side.

The host needs:

- Docker with the compose plugin.
- Roughly 20 GB of disk. The dedicated server downloads about 14 GB of game
  files on first start.
- A machine that can hold a Team Fortress 2 dedicated server and an
  Archipelago server at once. Two cores and 4 GB of RAM is enough for six
  players.

Everything else — the Archipelago server, the dedicated server, the SourceMod
plugin and the bridge that connects them — ships in the compose file for this
project.

## Configuring your YAML

Options live under `Team Fortress 2 Mann vs Machine`:

- `mission_count`: how many missions the run spans. Eight is about an evening.
- `difficulty_pool`: the easiest tier that may appear. Every harder tier comes
  with it, so `normal` means anything and `expert` means Expert and Haunted
  only.
- `goal`: `final_boss` or `missionsanity`.
- `missionsanity_percentage`: how much of the run Missionsanity wants.
- `death_link`: off by default. A death in Mann vs Machine is cheap, so this is
  noisier here than in most games.

## Joining a game

1. The host copies `.env.example` to `.env`, sets `SRCDS_RCONPW` and
   `SRCDS_HOSTNAME`, and runs `docker compose up`.
2. On first start the stack generates a seed from the YAML and hosts it. On
   later starts it reuses the seed already in `output/`.
3. Players open the Team Fortress 2 console and connect to the server's
   address, with `password <SRCDS_PW>` first if the host set one.
4. The server picks the missions. There is no map vote and no mission browser:
   which mission is loaded is part of the run.

## Where the state lives

The bridge holds the Archipelago session and the unlock set, on disk, in the
compose volume. It survives a server restart, a map change and a crash
mid-wave: checks are queued before they are acknowledged and replayed on
reconnect. If the Archipelago server is unreachable, the game keeps playing and
the checks land once it comes back.
