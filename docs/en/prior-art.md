# Prior art

What already exists, and what we can take from it. Checked 2026-08-13.

## ALPHAMARIOX/TF2-MvM-Archipelago

<https://github.com/ArchipelagoMW/Archipelago/compare/main...ALPHAMARIOX:TF2-MvM-Archipelago:main>

A fork of Archipelago, 3 commits ahead of `main` and 143 behind, last touched
March 2026. Three files, all under `worlds/tf2/`:

| File | Lines | State |
| --- | --- | --- |
| `Items.py` | 556 | Data tables, mostly filled. `mvm_data_to_ap_id()` and `ap_id_to_mvm_data()` are empty stubs (`""` as the body). No `item_table` assembly, no `ItemClassification` assigned. |
| `Options.py` | 214 | Complete and usable. 16 option classes plus `TeamFortress2Options(PerGameCommonOptions)` and two `OptionGroup`s. |
| `Locations.py` | 4 | An empty `TFLocation(Location)` class. Nothing else. |

There is no `__init__.py`, no `World` subclass, no regions, no rules, and no
client. It does not generate. Treat it as a data dump, not an implementation.

### What is worth taking

**The tables in `Items.py`.** This is the real value, and it is exactly the
kind of data that belongs in `gamedata/` (see ADR 0001). Fourteen dicts:

`credits_table`, `upgrades_table`, `class_table`, `weapon_table` (210 lines,
the big one), `weapon_slot_table`, `canteens_table`, `robots_table` (169
lines), `wave_table`, `mission_table`, `map_table`, `trap_table`.

The `Group(IntFlag)` enum in the same file is a clean categorisation scheme
(`base`, `credits`, `upgrades`, `tf_class`, `mvm_class`, `shop`, `mvm_map`,
`mission`, `wave`, `weapon`, `weapon_slot`, `canteen`, `robots`, `traps`).
Port it to a Go bitmask over the same names so the two stay comparable.

Port the tables to Go, do not vendor the Python. Once translated, `gamedata/`
owns them and the Python side reads the exported JSON.

**The options.** `Options.py` is close to Roseburst's outline from the
Discord thread, and we can adapt it nearly as-is. Note the naming does not
match the
thread: `ShuffleMaps` / `ShuffleMissions` / `LockClasses` / `LockWeapons` /
`LockWeaponSlots` / `ShuffleUpgrades` / `ShuffleRobots` / `ShuffleCanteens` /
`AddTraps` / `AddTrapTypes`, plus `RandomizeMissionCount` and
`LockClassesCounter` as `Range`s. Roseburst's `Mission Order`, `Goal`,
`Tour Size`, `Allied Mercs`, `Merc Loadouts`, `Giants and Bosses` and the
whole `Check Options` block have no equivalent there yet.

### Known defects to not inherit

- The game name disagrees with itself: `Items.py` declares
  `"Team Fortress 2 Mann Vs. Machine"`, `Locations.py` declares
  `"Team Fortress 2"`. The game string is the multiworld's primary key for a
  slot, so it has to be one value in one place. We pick it once, in
  `gamedata/`, and export it.
- Nothing enforces id stability. We must never renumber an Archipelago id
  once a seed exists. The fork's author meant the two stub converter
  functions to solve that problem. See ADR 0001 for how we handle it
  instead.

## Snolid Ice's MvM Manual

Referenced twice in the thread ("i made a mvm manual", "my mvm manual is
kinda rough"), but nobody posted a link. Manual apworlds are JSON-driven
and cannot express real accessibility rules, which is why we are not going
that route. Still, it is worth finding for the item and location naming,
if the author still has it.

## Archipelago itself

- Protocol: `docs/network protocol.md` in `ArchipelagoMW/Archipelago`. The
  message set we need is small: `Connect`, `Connected`, `LocationChecks`,
  `ReceivedItems`, `StatusUpdate`, `Bounced` (for DeathLink), `Say`.
- Adding a game: `docs/adding games.md` and `docs/world api.md`.
- The client must handle both `ws://` and `wss://`, and must reconnect on
  its own. That reconnect logic is the single most annoying part of a
  hand-rolled client. It is the main reason the bridge is a long-lived Go
  process, rather than something bolted into the SourceMod plugin (ADR
  0002).

## Source-side prior art

No Archipelago client exists for any Source engine game, so there is no
plugin to copy. The relevant references are the SourceMod API itself
(game events, `SDKHooks`, `TF2_` natives), plus the MvM-specific entity
and event names. The community wikis document those names, not Valve.

Valve published the TF2 client and server source in the February 2025 SDK
drop. We are not using it. A server-side-only integration keeps vanilla
clients able to join, and that matters more than anything the SDK gives
us.
