# gamedata

Go package. The single source of truth for every Mann vs Machine fact this
project knows, and for every Archipelago id. Read [ADR
0001](../docs/adr/0001-go-owns-the-game-data.md) before touching anything here.

Nothing exists yet.

## What goes here

Go structs and table literals for:

- Maps
- Missions: map, `.pop` file, difficulty tier, wave count
- Waves, derived from the mission's wave count
- Classes
- Weapons: class, slot, whether MvM allows it
- Upgrade lines, and which weapons carry each one
- Canteen types
- Allied robot templates
- Trap definitions
- The game name string, the AP base id offset, the data format version

Plus the exporter that writes `apworld/tf2_mvm/data/*.json`, and the tests that
guard id stability.

The starting point for these tables is `worlds/tf2/Items.py` in ALPHAMARIOX's
fork, 556 lines of Python dicts. Port them to Go, do not vendor the Python.
See [`../docs/prior-art.md`](../docs/prior-art.md) for what is in there and
what is broken in it.

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
   present in the committed export has changed value.

Adding a mission is a struct literal plus a regenerate. Renaming one is fine.
Deleting one means a tombstone.

## Export

The exported JSON under `apworld/tf2_mvm/data/` is **committed**, so that the
apworld can be zipped and handed to someone without a Go toolchain. CI
regenerates it and fails if the committed copy differs.
