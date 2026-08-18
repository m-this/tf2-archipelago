# tf2-archipelago: Domain Context

Glossary of the terms used across `gamedata/`, `bridge/`, `apworld/` and
`plugin/`. Two vocabularies meet in this project and they use some of the same
words for different things, so this file defines both.

Last updated: 2026-08-18.

## Archipelago vocabulary

**Multiworld**
One generated game session spanning several players and several games. The
multiworld can place an item belonging to one player in another player's
world.

**Slot**
One participant in a multiworld. A name identifies each slot, and a slot
plays exactly one game with one YAML. In this project a slot is **the TF2
server**, not a Steam account. See `docs/en/spec.md`, "Slot model".

**Seed**
The generated multiworld, immutable once generated. Every id it references
must still mean the same thing when someone plays it. That constraint
drives the id rules in ADR 0001.

**Location** (also **check**)
A place in a world that can hold an item. Checking a location tells the
server "whatever was here has been found". In this project a location is an
MvM objective: a wave cleared, a mission cleared, a tank destroyed.

**Item**
What a location contains. An item can belong to any slot in the multiworld,
so clearing a wave in TF2 can hand a sword to someone playing Zelda.
Classified as `progression`, `useful`, `filler` or `trap`.

**Progression item**
An item that an access rule can need to reach another location. Weapon
slots, mission tickets and upgrade packages are progression here. Getting
the classification wrong produces seeds that are unwinnable or trivial.

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
the seed is unplayable from the start. See the starting-state requirement in
`docs/en/spec.md`.

**DeathLink**
An opt-in convention where a death in one world kills every other DeathLink
participant. Here a death is the team losing a wave, because an individual
death in MvM is routine and only a full wipe fails a wave. Outbound, a lost
wave is a `Bounce`; inbound, a `Bounced` kills everyone on RED, bots
included, which fails the wave. The plugin does not send that loss back out.
See `docs/en/spec.md`, "Traps and DeathLink".

**Trap**
An item with a negative effect. A first-class classification in Archipelago,
not a hack.

## Mann vs Machine vocabulary

**Mission**
An ordered sequence of waves on one map, defined by a `.pop` file. Has a
difficulty tier: Normal, Intermediate, Advanced, Expert or Nightmare. "Mean
Machines" is a mission.

**Wave**
One assault within a mission. Pass or fail as a unit: a wiped team replays
the wave instead of losing the mission. It is the atomic unit of progress,
and therefore the main location group.

**Tour**
An ordered set of missions played as a campaign. Valve's tours (Operation
Two Cities and so on) stay fixed. We generate ours when a run picks the
`Campaign` mission order.

**Upgrade station**
The in-mission shop. Players spend collected credits on persistent per-weapon
upgrades between waves. Upgrades persist for the whole mission and reset when
it ends.

**Credits / money**
Cash that destroyed robots drop. A player must walk over it to collect
it, and uncollected cash is lost at wave end. Collecting all of it in a
wave earns an **A+ rating**, the trigger for the money-bonus location
group.

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
smaller than six. Their loadout comes from an unlocked robot template. They
share the player's unlocked upgrades directly, since RED bots do not buy
upgrades on their own.

**`.pop` file**
The plaintext population file defining a mission's waves and robots. The only
authority on how many waves a mission has. `gamedata/` hardcodes wave counts
for Valve's missions from the wiki instead of parsing them; v1 does not
support community missions.

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

**Launcher**
The Windows exe, `tf2ap.exe`. Installs the game server and its mods, and runs
the bridge in-process next to `srcds.exe`. Its window holds the log, the start
and stop buttons and an rcon box. The Windows equivalent of the compose stack.

**Defender bots**
OfficerSpy's MvM Defender TFBots and its five dependencies, staged by
`deploy/bots/build.sh` and patched from `deploy/patches/`. They fill the RED
team so a wave balanced for six players is winnable by fewer.

**Objective**
The plugin's vocabulary for what the bridge calls a location. The plugin
reports `wave_cleared{mission, wave}`; the bridge turns that into a location
id. Keeping the two vocabularies separate is what lets the plugin stay
Archipelago-agnostic.

**Grant**
The plugin's vocabulary for what the bridge calls a received item. The bridge
sends `grant_weapon_slot{slot}`; the plugin never sees the item id behind it.

**State grant / effect grant**
The two kinds of grant, told apart by `ItemKind.OneShot` in `gamedata`. A
state grant is a fact that stays true, such as a playable class, an open
loadout slot, or an available mission. Applying one twice equals applying
it once, so the bridge can resend it whenever the plugin asks.

An effect happens once and ends: credits get paid, or a trap fires.
Applying one twice pays money nobody earned, or fires a trap nobody
deserved. So the bridge sends each effect once, and stops once the plugin
acknowledges it. Only state grants appear in the unlock set.

**Acknowledgement**
The sequence number the plugin reports back to the bridge for what it
applied. The bridge never resends an effect at or below that number. It
lives on the bridge, on disk, because the plugin remembers nothing across
a reload. A reload is exactly when a repeat send otherwise happens.

The acknowledgement also works as the cursor the plugin resumes from.
That is why the unlock set reports it, not the length of the item list.
A cursor set above an unapplied effect loses that effect for good.

**Wave drift**
The game reports a mission length that `gamedata` disagrees with. Every
wave count in the tables comes from the wiki, and a wrong one makes a
mission clear fire early or never. The plugin sends what the game says
with each check, and the bridge serves the disagreements at `/healthz`
without ever refusing the check.

**Sequence**
The plugin's cursor over what the bridge granted, counting received items
rather than grants. The bridge skips an item it cannot read, leaving a gap,
so upgrading the bridge does not change what a sequence means. `GET
/grants?since=N` and the `seq` in an unlock set are both this number.

**Unlock set**
The full set of grants currently in effect for the slot. Authoritative on the
bridge, on disk. The plugin re-fetches it after any reload or map change rather
than trying to remember it.
