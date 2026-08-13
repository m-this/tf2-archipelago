# ADR 0001 — Go owns the MvM game data; the apworld is a thin Python reader

- **Status**: Accepted
- **Date**: 2026-08-13
- **Deciders**: project owner
- **Related**: `docs/spec.md`, `docs/prior-art.md`, ADR 0002

## Context

An Archipelago game integration needs two things that both depend on the same
facts about the game:

1. An **apworld**, which runs inside the Archipelago generator process and
   must be Python. It declares the item pool, the location list, the regions,
   the access rules and the YAML options.
2. A **runtime client**, which holds the multiworld session and translates
   between game events and Archipelago messages.

Both need the same tables: every MvM map, every mission and its wave count and
difficulty tier, every weapon and which class and slot it belongs to, every
upgrade line, every canteen, every allied robot template, and a stable numeric
id for each of those.

The project convention is that everything custom is written in Go. The apworld
being Python is not negotiable: the Archipelago generator imports it and calls
into it, so it runs in that process or not at all.

That leaves the question of where the tables live. `ALPHAMARIOX`'s fork puts
them in Python (`worlds/tf2/Items.py`, 556 lines of dicts) and leaves the
id-assignment functions as empty stubs, which is precisely the part that is
hard.

## Decision

**`gamedata/` is a Go package and it is the only place MvM facts are written
down.** It compiles into the bridge, and it has an exporter that writes JSON
into `apworld/tf2_mvm/data/`. The Python apworld reads that JSON at import
time and builds its item and location tables from it.

Specifically:

- Every table from `ALPHAMARIOX`'s `Items.py` is ported to Go structs, not
  vendored as Python. The `Group` bitmask is ported as a Go bitmask with the
  same member names so the two remain comparable.
- **Ids are assigned in Go and are append-only.** Each entity carries an
  explicit id literal in the source. Ids are never renumbered, never reused
  after deletion, and a removed entity keeps its id reserved with a tombstone.
  A test asserts that no two entities share an id and that no id present in
  the committed export has changed. Renumbering an id silently invalidates
  every seed ever generated, and there is no way to detect that at play time,
  so it is guarded at commit time instead.
- The exported JSON is **committed**, not gitignored, and the exporter runs in
  CI to verify the committed copy matches the Go source. Committing it means
  the apworld is a standalone artifact: someone can zip `apworld/tf2_mvm/` and
  hand it to a friend without a Go toolchain.
- The game name string, the AP base id offset, and the data format version all
  live in `gamedata/` and are exported alongside the tables. The apworld
  refuses to load a data file whose format version it does not know.
- Python holds the logic that cannot be data: region graph construction,
  access rules, fill hooks, and the `Options` classes. `Options.py` is adapted
  from the fork rather than generated, because option classes are code with
  docstrings that the AP website renders, not a table.

## Consequences

**Positive**

- One place to edit. Adding a community mission is a Go struct literal and a
  regenerate, not two edits that can disagree.
- The bridge and the generator cannot disagree about what "wave 3 of Mean
  Machines" means, because they are reading the same numbers from the same
  origin.
- Id stability becomes a testable property in a language with a test runner we
  already run in CI, rather than a convention nobody checks.
- Porting the fork's tables to Go forces a read of all 556 lines, which is how
  the game-name inconsistency in the fork was found in the first place.

**Negative**

- A generation step exists, and a stale export is a real failure mode. CI
  catching it is the mitigation, not a fix.
- Contributors from the Archipelago community will expect a normal apworld and
  will find a JSON blob and a Go package. The apworld directory needs a README
  that says so in the first paragraph.
- Two languages for one conceptual layer. Accepted: the alternative is Python
  everywhere, which loses the bridge, or Go everywhere, which is impossible.
- If this is ever submitted upstream, the JSON-reading apworld may not be
  accepted as-is. That is a v2 problem and `spec.md` already declares upstream
  submission out of scope for v1.

## Alternatives considered

- **A normal hand-written Python apworld** (finish `ALPHAMARIOX`'s files).
  Rejected: the bridge then needs its own copy of the same tables in Go, and
  the two drift the first time anyone adds a mission. It is also the option
  that most directly contradicts the "custom code is Go" convention.
- **A Manual apworld** (JSON plus hooks, fully generated from Go). Rejected:
  Manual cannot express real accessibility rules, so a seed could place the
  final mission's ticket behind that same mission. The whole point of the
  Mission Order and Goal options in `spec.md` is a logic graph, and Manual has
  no graph.
- **Generate the Python source from Go** rather than JSON read by Python.
  Rejected: generated Python is unreadable in review, unbreakpointable in a
  debugger, and the diff on every regeneration is noise. JSON is data and
  reads like data.
- **Keep the tables in Python and expose them to Go over a socket.** Rejected
  out of hand: a network dependency between the generator and the runtime for
  something that is a static table.
