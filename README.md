# tf2-archipelago

An [Archipelago](https://archipelago.gg) integration for a self-hosted Team
Fortress 2 server, in Mann vs Machine mode.

**Status: four components run. One wave remains to clear.** Seed generation,
the Archipelago server, and the bridge pass an end-to-end check on every
`make integration` run. The plugin ran on a real server on 2026-08-17. The
three MvM events exist, the plugin reads the mission and its length
correctly, and a check reaches the multiworld.

One part still needs a player at a TF2 client. MvM does not start a wave
until a human player is ready. A bot cannot do that.

## Start

```sh
cp deploy/.env.example .env   # SRCDS_RCONPW has no default
make up
make logs
```

The first start downloads about 14 GB of game files. [`docs/`](./docs/) is
a full book for the host. [`docs/en/setup/install.md`](./docs/en/setup/install.md)
gives the detail. Send
[`docs/en/archipelago-for-mvm-players.md`](./docs/en/archipelago-for-mvm-players.md)
to a player new to a multiworld.

```sh
make check        # everything CI runs
make integration  # real Archipelago and bridge, driven the way the plugin drives them
```

## Why MvM

MvM is the only TF2 mode with built-in progress:

- A mission is an ordered series of waves.
- A team clears a wave, or fails it.
- A shop sells upgrades that persist.
- A round is an ordered series of missions.

This structure maps directly onto Archipelago's regions and locations.
Classic TF2 has none of it.

## How it works

Three processes. One source of truth.

```
  gamedata/ (Go)  ──generates──>  apworld/tf2_mvm/data/*.json
        │                                   │
        │ built into                        │ read at generation
        v                                   v
    bridge (Go)  <──websocket──>  Archipelago server (container)
        ^
        │ HTTP + JSON on 127.0.0.1
        v
  SourceMod plugin  (inside the srcds container)
```

The SourceMod plugin is the only part that sees the game. The Go bridge is
the only part that speaks Archipelago. `gamedata/`, in Go, is the only part
that knows what a mission or a weapon is. It exports that data as JSON for
the Python apworld.

Players connect with a stock TF2 client. They install nothing.

## Directory tree

| Directory | Language | Role |
| --- | --- | --- |
| [`gamedata/`](./gamedata/) | Go | Source of truth: maps, missions, waves, weapons, upgrades, robots, and the IDs. Exports the JSON. |
| [`bridge/`](./bridge/) | Go | Archipelago client. WebSocket, reconnection, a durable queue, and a loopback HTTP API for the plugin. |
| [`apworld/`](./apworld/) | Python | A thin apworld. It reads the exported JSON and sets the regions, the rules, and the YAML options. |
| [`plugin/`](./plugin/) | SourcePawn | Detects the objectives. Applies the unlocks and the traps. |
| [`deploy/`](./deploy/) | Compose | Archipelago server, srcds, and the bridge. |
| [`docs/`](./docs/) | Markdown | Spec, ADRs, the hosting guide, prior art, and the original Discord thread. |

## Documentation

`docs/` is a book, in English and in French. GitHub Pages publishes it at
[m-this.github.io/tf2-archipelago](https://m-this.github.io/tf2-archipelago/),
and `make docs` builds and serves the English version on `127.0.0.1:8081`.

- [`docs/en/SUMMARY.md`](./docs/en/SUMMARY.md): the book's table of
  contents. Hosting, game setup, play, and troubleshooting.
- [`docs/en/archipelago-for-mvm-players.md`](./docs/en/archipelago-for-mvm-players.md):
  for a player new to a multiworld. The vocabulary and the chat commands.
- [`docs/en/operate/testing.md`](./docs/en/operate/testing.md): what still
  needs a test, and what each test needs. Read it before the first session.
- [`docs/en/spec.md`](./docs/en/spec.md): the design. Scope, locations,
  items, goals, and open questions.
- [`docs/en/adr/`](./docs/en/adr/): the decisions, and why the alternatives
  lost.
- [`docs/en/discord-mvm-thread.md`](./docs/en/discord-mvm-thread.md): the
  original Archipelago Discord thread, copied word for word. **Damonj17**
  and **Roseburst** wrote the design.
- [`docs/en/prior-art.md`](./docs/en/prior-art.md): what exists already,
  most of all the fork
  [ALPHAMARIOX/TF2-MvM-Archipelago](https://github.com/ArchipelagoMW/Archipelago/compare/main...ALPHAMARIOX:TF2-MvM-Archipelago:main).
- [`CONTEXT.md`](./CONTEXT.md): the glossary. Archipelago's and MvM's
  vocabularies share words but not their meanings; this file fixes both.
- [`TODO.md`](./TODO.md): what blocks what.

## Credits

The design comes from the Archipelago Discord thread. **Damonj17** set the
premise and the shape of the items and the checks. **Roseburst** wrote the
entire YAML options schema. **adeleine64DS**, **Amazia**, **Snolid Ice**,
**mudkipslike**, **TheBreadstick**, **CrystalClear**, and **Pixel Silzavon**
contributed. The starting data tables come from **ALPHAMARIOX**'s fork.
