# TF2 MvM Archipelago: design spec

This is the consolidated reading of
[`discord-mvm-thread.md`](./discord-mvm-thread.md) plus the architecture
decisions in [`adr/`](./adr/). Where the thread and this file disagree, this
file wins, and the text calls out the divergence. See
[Testing](operate/testing.md) for what a human still has to confirm on a
live server.

## Scope

Mann vs Machine only. Not competitive TF2, not casual, not payload.

MvM is the one TF2 mode with a real progression shape already in it. Each
mission holds an ordered list of waves, and each wave is pass or fail. A
shop sells persistent upgrades, and each tour holds an ordered list of
missions.

That maps onto Archipelago's regions and locations without inventing
anything. Plain TF2 does not, which is why the thread went straight to
MvM, and why we follow it.

Target deployment: one self-hosted `srcds` in a container, one Archipelago
server in a container, one bridge process. Friends connect with a stock TF2
client and install nothing.

### Non-goals

- No client-side mod. Vanilla clients must be able to join (ADR 0002).
- No support for official Valve Mann Up or Boot Camp servers. Those cannot run
  plugins, so there is nowhere for the integration to live.
- No upstream submission to `ArchipelagoMW/Archipelago` in v1. Revisit this
  after it generates seeds and someone plays a seed end to end.
- No community missions (Potato.tf, Moonlight.tf) in v1. `.pop` files carry no
  stable global id and cannot be read off the host without a VPK extractor.
  `gamedata/` hardcodes the 29 Valve missions instead, with wave counts taken
  from the wiki.

## Architecture

Three processes, one shared source of truth.

```
  gamedata/ (Go)  ──generates──>  apworld/tf2_mvm/data/*.json
        │                                    │
        │ compiled in                        │ read at generation time
        v                                    v
    bridge (Go)  <──websocket──>  Archipelago server (archipelago.gg)
        ^
        │ HTTP + JSON on 127.0.0.1
        v
  SourceMod plugin  (inside the srcds container)
```

**`gamedata/` (Go)** owns every MvM fact: maps, missions, waves per mission,
difficulty tiers, the weapon list, upgrade names, canteen types, and robot
templates. It also owns the id assigned to each one. It compiles into the
bridge, and it exports JSON for the apworld. One table, two consumers, no
drift. ADR 0001.

**`apworld/tf2_mvm/` (Python)** is deliberately thin: read the exported JSON,
build items and locations from it, declare regions and access rules, expose
the YAML options. No MvM knowledge hardcoded in Python beyond the rules
themselves.

**`bridge/` (Go)** is the Archipelago client. It holds the websocket
session, handles reconnect and replay, and deduplicates received items. It
also records what it has already applied, and exposes a small local HTTP
API to the plugin. ADR 0002.

**`plugin/` (SourcePawn)** is the only thing that sees the game. It detects
objectives and reports them, and it applies unlocks and traps. It has no
Archipelago knowledge: it speaks in MvM terms (`wave_cleared`,
`grant_weapon_slot`) and the bridge does the translation.

## Slot model

This is the load-bearing question, because the item and location tables
depend on the answer.

**Decision: one slot for the whole server.** Progression is collective. Every
player on RED shares the same unlocked weapons, upgrades and missions.

Rationale: MvM is cooperative and balanced around a coordinated team of
six, and wave 6 of an Advanced mission does not care about your
randomizer. A per-player slot leaves one friend with no primary weapon
while another has no melee. It also multiplies the check count by the
player count, and makes the seed unplayable if someone leaves halfway.

Consequence: the AP slot belongs to the server, not to a Steam account. The
bridge holds one connection. Whoever is on the server plays that slot.

Per-player slots stay possible later as an option, but not in v1, and the
gamedata id scheme must not assume a single slot forever.

## Locations (checks)

Every check must be something the SourceMod plugin can observe with
certainty and only once. The design deliberately excludes the MvM
achievements, because Steam ties them to one account: a veteran who
already owns them will never fire `achievement_earned`. For that veteran
the seed stays unwinnable, while a new account can still win it. That
failure stays silent, which is worse than a missing feature.

| Location group | Trigger | Count | Option |
| --- | --- | --- | --- |
| Wave clear | Wave N of mission M completed | waves per mission, summed | `wave_checks` |
| Mission clear | Final wave of mission M completed | one per mission in the pool | `mission_checks` |
| Tour clear | Every mission in a tour completed | one per tour | `tour_checks`, Campaign only |
| Money bonus | Wave completed with an A+ credit rating | one per wave | `money_checks` |
| Shop purchase | A check bought at the upgrade station for $100 to $400, then turned in | configurable | `shop_checks` |
| Tank / boss kill | Tank destroyed, Giant or boss robot killed | per mission, capped | `boss_checks` |

Wave clear is the core location group, and the only one on by default.
Everything else is opt-in, because a run's length has to be tunable. The
wave count alone already gives roughly 6 to 8 checks per mission.

Shop checks are the most novel group, and the least certain to work.
Roseburst's two variants:

- **Wave turn-in**: buy the check in the shop, clear the wave, receive it.
- **Mission turn-in**: buy it, clear the whole mission, receive it.

Both need the plugin to inject a purchasable entry into the upgrade station
UI. `shop_checks` ships off by default and stays off until a live server
confirms it.

Damonj17's "upgrades count as checks like shops in other games" idea, where
opening the station shows a long list of hints, is fun. It belongs in an
option, not in the default.

## Items

| Item group | Effect | Classification |
| --- | --- | --- |
| Weapon slot | Unlocks Primary, Secondary or Melee | progression |
| Weapon | Unlocks one specific weapon for one class | progression |
| Upgrade package | Unlocks one upgrade line across every weapon that has it | progression |
| Class | Unlocks a mercenary class | progression |
| Canteen | Unlocks one power-up canteen type | useful |
| Mission ticket | Unlocks one mission or one tour | progression |
| Robot template | Unlocks one allied bot loadout | useful |
| Credits | A cash bundle at the next wave start | filler |
| Trap | See below | trap |

Two mutually exclusive ways to gate upgrades, from Roseburst:

- **Weapon Unlocks**: one check per weapon, and unlocking a weapon gives every
  upgrade on it.
- **Upgrade Packages**: one check per upgrade line, shared across weapons.
  Getting "Primary Damage" gives damage on the Minigun, the Sydney Sleeper and
  the Flamethrower alike.

Packages produce a smaller and more interesting item pool. Weapon Unlocks
produce a much bigger one. Both ship, `upgrades_shuffle` picks.

The starting state has to be playable. adeleine64DS's suggestion from the
thread, "you could also require that you have that class/weapon before you
can buy the upgrade", is the right instinct. The generator must guarantee
at least one usable class with at least one usable weapon in sphere 0.
Otherwise wave 1 is unwinnable, and the seed is dead.

## Allied bots

Valve tunes waves for six players, so a solo run with no help is not a
randomizer: it is a fight nobody can win alone.

Roseburst's `Allied Mercs` option (Off / Fill 6 / Fill 10 / Scavenge) is
the answer. It beats Damonj17's damage multiplier. A multiplier compensates
for missing damage, but not for a missing Medic, a missing Engineer, or a
missing Scout collecting money. Bots at least occupy the roles.

RED bots do not buy upgrades on their own. The design: bots share the
player's unlocked upgrades directly, adeleine64DS's suggestion from the
thread and the cheapest fix available.

Bot loadouts are themselves items under `Robot Templates` or `Single
Templates`, which turns "who is on my team" into part of the progression. Keep
that. It is the most original idea in the thread.

## Goals

- **Final Boss**: the design flags one mission from the hardest available
  tier, and clearing it wins.
- **Missionsanity**: clear X% of the missions in the pool.
- **Australium Hunt**: the generator shuffles N junk Australium items into
  the multiworld, and you need a percentage of them. This is the only goal that
  ties the run to other players' worlds. That is what a multiworld is for,
  so the design must treat this goal as central, not minor.

## Traps and DeathLink

A death in MvM is cheap: you respawn at the next wave or after a short
timer, and only a full team wipe fails it. So DeathLink on individual
death is only noise, and this project does not support it. A session that
asks for it gets one warning in the bridge log, and nothing else happens.

Traps from the thread, all plugin-side:

- Forced bad canteen or upgrade (Return to Spawn, Heavy Rage)
- Spawned Sentry Buster, Engineer, Sniper or Spy
- Map event triggers (Rottenburg's barrier, Mannhattan's capture points)
- Jarate on the whole team
- Stunned allied bots
- An extra Giant or boss

Traps that can make a wave mathematically unloseable-to-winnable are fine.
Traps that can corrupt the run state are not: nothing permanently removes
a received item.
