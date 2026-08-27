# gamedata

Go package. The single source of truth for every Mann vs Machine fact this
project knows, and for every Archipelago id. Read [ADR
0001](../docs/en/adr/0001-go-owns-the-game-data.md) before touching anything here.

## What is here

| File | Holds |
| --- | --- |
| `gamedata.go` | Game name, format version, difficulty tiers, item classifications |
| `maps.go` | The 7 maps |
| `missions.go` | The 29 Valve missions: pop file, name, map, tier, wave count |
| `classes.go` | The 9 classes |
| `slots.go` | Primary, Secondary, Melee |
| `ids.go` | The base id and every derivation from it |
| `locations.go` | The 210 checks, and the objective the plugin reports for each |
| `items.go` | The item pool template |
| `weapons.go` | Weapons available to defender bot loadouts |
| `buff_weapons_generated.go` | Append-only buff weapon catalog and legacy item IDs |
| `weapon_effects.go` | Positive weapon buff effects |
| `validate.go` | Every invariant the id scheme rests on |
| `export.go` | Writes `apworld/tf2_mvm/data/*.json` |

Not here yet, and deliberately not in v1: weapon ownership, canteens, allied
robot templates, and traps. Weapon ownership still uses progressive slots;
`WeaponBuffs` adds item-specific bonuses without unlocking a weapon.

`tools/import_tf2_weapons.py` reads Valve's installed item schema to discover
new buff targets. Its output is reviewed and appended because a game update
must never renumber existing Archipelago ids.

## When the weapon tables land

The starting point is `worlds/tf2/Items.py` in
[ALPHAMARIOX's fork](https://github.com/ArchipelagoMW/Archipelago/compare/main...ALPHAMARIOX:TF2-MvM-Archipelago:main),
556 lines of Python dicts: `credits_table`, `upgrades_table`, `class_table`,
`weapon_table` (210 lines, the big one), `weapon_slot_table`, `canteens_table`,
`robots_table`, `wave_table`, `mission_table`, `map_table` and `trap_table`.

Port them to Go. Do not vendor the Python. Once translated, this package owns
them and the Python side reads the exported JSON.

The `Group(IntFlag)` enum in the same file is a clean categorisation scheme.
Port it to a Go bitmask over the same names, so the two stay comparable.

Treat that fork as a data dump, not an implementation. It has no `World`
subclass, no regions, no rules and no client, and it does not generate.

### Two defects in it, which this package must not inherit

- **The game name disagrees with itself.** `Items.py` declares
  `"Team Fortress 2 Mann Vs. Machine"` and `Locations.py` declares
  `"Team Fortress 2"`. The game string is the multiworld's primary key for a
  slot, so it has to be one value in one place. This package picks it once and
  exports it.
- **Nothing there enforces id stability.** Never renumber an Archipelago id
  once a seed exists. The fork's two empty converter stubs were meant to solve
  that. [ADR 0001](../docs/en/adr/0001-go-owns-the-game-data.md) says what this
  project does instead, and `validate.go` enforces it.

## What does not go here

No Archipelago protocol code (that is `bridge/`). No logic rules (that is
`apworld/`). No game interaction (that is `plugin/`). This package is data and
the functions that shape it, nothing else.

## The id rules

These exist because a seed is immutable and there is no way to detect a
renumbered id at play time. A wrong id does not throw, it silently hands the
player the wrong item.

1. Every entity carries an explicit id literal in the source. No id is derived
   from slice position, map iteration order, or a hash of the name.
2. Ids are append-only. New entities take the next free id.
3. Ids are never reused. A deleted entity leaves a tombstone holding its id.
4. A test asserts uniqueness across the whole space, and asserts that no id
   present in `testdata/ids-frozen.json` has changed value.

Adding a mission is a struct literal plus a regenerate. Renaming one is fine.
Deleting one means a tombstone.

The frozen file is keyed on what an id derives from — `kind/pop file/wave`,
`class/scout/0` — and never on a display name. The names are UNVERIFIED and
expected to be corrected before the first seed is played, and a key built from
one would report that correction as nine deleted entities. `-freeze` only ever
adds keys, so it cannot retire the old ones for you.

Location and item ids are derived, not written down one by one:

```
wave clear      base + mission_id*100 + wave
mission clear   base + mission_id*100 + 99
mission ticket  base + 1_000_000 + 1_000 + mission_id
class           base + 1_000_000 + 2_000 + class_id
```

The derivation is stable because the ids it derives from are. A mission may
therefore hold at most 98 waves, which the tests check.

## Export

The exported JSON under `apworld/tf2_mvm/data/` is **committed**, so that the
apworld can be zipped and handed to someone without a Go toolchain. CI
regenerates it and fails if the committed copy differs.

```sh
go generate ./gamedata            # rewrite the export
go test ./gamedata/ -freeze       # record ids that are new, never move an old one
```
