# tf2-archipelago — Domain Context

Glossary of the terms used across `gamedata/`, `bridge/`, `apworld/` and
`plugin/`. Two vocabularies meet in this project and they use some of the same
words for different things, so both are pinned down here.

Last updated: 2026-08-13.

## Archipelago vocabulary

**Multiworld**
One generated game session spanning several players and several games. Items
belonging to one player can be placed in another player's world.

**Slot**
One participant in a multiworld, identified by name. A slot plays exactly one
game with one YAML. In this project a slot is **the TF2 server**, not a Steam
account. See `docs/spec.md`, "Slot model".

**Seed**
The generated multiworld. Immutable once generated. Every id referenced by a
seed must still mean the same thing when the seed is played, which is the
constraint that drives the id rules in ADR 0001.

**Location** (also **check**)
A place in a world where an item can be placed. Checking a location tells the
server "whatever was here has been found". In this project a location is an MvM
objective: a wave cleared, a mission cleared, a tank destroyed.

**Item**
What a location contains. May belong to any slot in the multiworld, so clearing
a wave in TF2 may hand a sword to someone playing Zelda. Classified as
`progression`, `useful`, `filler` or `trap`.

**Progression item**
An item the generator's logic can require in order to reach another location.
Weapon slots, mission tickets and upgrade packages are progression here.
Getting the classification wrong produces seeds that are unwinnable or trivial.

**apworld**
The Python package that teaches the Archipelago generator about one game: its
items, locations, regions, rules and YAML options. Ships as a zipped
`.apworld` file. Ours is `apworld/tf2_mvm/`.

**Region / access rule**
The graph the generator reasons over. A region holds locations; an access rule
is the condition for reaching one. "Wave 4 of mission M is reachable once you
hold M's ticket and a primary weapon slot" is an access rule.

**Sphere 0**
Everything reachable with no items at all. If sphere 0 contains no location,
the seed is dead on arrival. See the starting-state requirement in
`docs/spec.md`.

**DeathLink**
An opt-in convention where a death in one world kills every other DeathLink
participant. Needs a game-specific definition of "death"; ours is unresolved
(`docs/spec.md`, open question 5).

**Trap**
An item with a negative effect. A first-class classification in Archipelago,
not a hack.

## Mann vs Machine vocabulary

**Mission**
An ordered sequence of waves on one map, defined by a `.pop` file. Has a
difficulty tier: Normal, Intermediate, Advanced, Expert or Nightmare. "Mean
Machines" is a mission.

**Wave**
One assault within a mission. Pass or fail as a unit: a wiped team replays the
wave, it does not lose the mission. The atomic unit of progress, and therefore
the backbone location group.

**Tour**
An ordered set of missions played as a campaign. Valve's tours (Operation
Two Cities and so on) are fixed; ours are generated when the `Campaign`
mission order is selected.

**Upgrade station**
The in-mission shop. Players spend collected credits on persistent per-weapon
upgrades between waves. Upgrades persist for the whole mission and reset when
it ends.

**Credits / money**
Cash dropped by destroyed robots, collected by walking over it. Uncollected
cash is lost at wave end. Collecting all of it in a wave earns an **A+ rating**,
which is the trigger for the money-bonus location group.

**Canteen**
The Power Up Canteen, a consumable charged at the upgrade station: Übercharge,
Critical Hits, Ammo Refill, Building Upgrade, Recall (Return to Spawn). Recall
and its like are what makes a "forced bad canteen" trap possible.

**Giant / boss / tank**
Oversized robot variants, named boss robots, and the tank that slowly hauls a
bomb to the hatch. All three are discrete, observable kill events, which is why
they work as locations.

**Sentry Buster**
A suicide robot that spawns to destroy an Engineer's sentry. Listed as a trap
in the thread.

**Robot template**
A loadout definition for a robot: class, weapon, attributes. Used by the game
for enemy robots, and by us for **allied bots**, where a template becomes an
unlockable item.

**Allied merc / RED bot**
A `tf_bot` on the player's team, spawned by the plugin to fill out a team
smaller than six. Their loadout comes from an unlocked robot template. Whether
they can use the player's purchased upgrades is unresolved (`docs/spec.md`,
open question 3).

**`.pop` file**
The plaintext population file defining a mission's waves and robots. The only
authority on how many waves a mission has, which is why `gamedata/` may have
to parse them (`docs/spec.md`, open question 4).

## Project vocabulary

**gamedata**
The Go package that is the single source of truth for every MvM fact and every
id. Compiles into the bridge, exports JSON for the apworld. ADR 0001.

**Export**
The committed JSON under `apworld/tf2_mvm/data/`, produced by `gamedata/` and
verified in CI. Committed rather than generated at build time so the apworld
is a standalone artifact.

**Bridge**
The long-lived Go process holding the Archipelago websocket session and
exposing a loopback HTTP API to the plugin. The only component that knows both
the AP protocol and the id mapping. ADR 0002.

**Plugin**
The SourcePawn plugin inside the `srcds` container. Sees the game, knows
nothing about Archipelago, holds no authoritative state.

**Objective**
The plugin's vocabulary for what the bridge calls a location. The plugin
reports `wave_cleared{mission, wave}`; the bridge turns that into a location
id. Keeping the two vocabularies separate is what lets the plugin stay
Archipelago-agnostic.

**Grant**
The plugin's vocabulary for what the bridge calls a received item. The bridge
sends `grant_weapon_slot{slot}`; the plugin does not know an item id was
involved.

**Unlock set**
The full set of grants currently in effect for the slot. Authoritative on the
bridge, on disk. The plugin re-fetches it after any reload or map change rather
than trying to remember it.
