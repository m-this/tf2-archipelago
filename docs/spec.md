# TF2 MvM Archipelago: design spec

Status: draft, nothing implemented. This is the consolidated reading of
[`discord-mvm-thread.md`](./discord-mvm-thread.md) plus the architecture
decisions in [`adr/`](./adr/). Where the thread and this file disagree, this
file wins and the divergence is called out.

## Scope

Mann vs Machine only. Not competitive TF2, not casual, not payload.

MvM is the one TF2 mode with a real progression shape already in it: a mission
is an ordered list of waves, waves are pass or fail, there is a shop with
persistent upgrades, and a tour is an ordered list of missions. That maps onto
Archipelago's regions and locations without inventing anything. Plain TF2 does
not, which is why the thread went straight to MvM and why we follow it.

Target deployment: one self-hosted `srcds` in a container, one Archipelago
server in a container, one bridge process. Friends connect with a stock TF2
client and install nothing.

### Non-goals

- No client-side mod. Vanilla clients must be able to join (ADR 0002).
- No support for official Valve Mann Up or Boot Camp servers. Those cannot run
  plugins, so there is nowhere for the integration to live.
- No attempt to submit this upstream to `ArchipelagoMW/Archipelago` in v1.
  Revisit once it actually generates and has been played end to end.

## Architecture

Three processes, one shared source of truth.

```
  gamedata/ (Go)  ──generates──>  apworld/tf2_mvm/data/*.json
        │                                    │
        │ compiled in                        │ read at generation time
        v                                    v
    bridge (Go)  <──websocket──>  Archipelago server (container)
        ^
        │ HTTP + JSON on 127.0.0.1
        v
  SourceMod plugin  (inside the srcds container)
```

**`gamedata/` (Go)** owns every MvM fact: maps, missions, waves per mission,
difficulty tiers, the weapon list, upgrade names, canteen types, robot
templates, and the id assigned to each. It compiles into the bridge and it
exports JSON for the apworld. One table, two consumers, no drift. ADR 0001.

**`apworld/tf2_mvm/` (Python)** is deliberately thin: read the exported JSON,
build items and locations from it, declare regions and access rules, expose
the YAML options. No MvM knowledge hardcoded in Python beyond the rules
themselves.

**`bridge/` (Go)** is the Archipelago client. It holds the websocket session,
handles reconnect and replay, deduplicates received items, persists what has
already been applied, and exposes a small local HTTP API to the plugin. ADR
0002.

**`plugin/` (SourcePawn)** is the only thing that sees the game. It detects
objectives and reports them, and it applies unlocks and traps. It has no
Archipelago knowledge: it speaks in MvM terms (`wave_cleared`,
`grant_weapon_slot`) and the bridge does the translation.

## Slot model

The thread never resolves this and it is the load-bearing question, because
the item and location tables depend on the answer.

**Decision: one slot for the whole server.** Progression is collective. Every
player on RED shares the same unlocked weapons, upgrades and missions.

Rationale: MvM is cooperative and balanced around a coordinated team of six.
A per-player slot means one friend has no primary weapon while another has no
melee, on a mode where wave 6 of an Advanced mission does not care about your
randomizer. It also multiplies the check count by the player count and makes
the seed unplayable if someone leaves halfway.

Consequence: the AP slot belongs to the server, not to a Steam account. The
bridge holds one connection. Whoever is on the server is playing that slot.

Per-player slots stay possible later as an option, but not in v1, and the
gamedata id scheme must not assume a single slot forever.

## Locations (checks)

Every check must be something the SourceMod plugin can observe with certainty
and only once. The MvM achievements are deliberately excluded: they are per
Steam account, so a veteran who already owns them will never fire
`achievement_earned` and the seed would be unwinnable for them and winnable
for a new account. That is a silent failure, which is worse than a missing
feature.

| Location group | Trigger | Count | Option |
| --- | --- | --- | --- |
| Wave clear | Wave N of mission M completed | waves per mission, summed | `wave_checks` |
| Mission clear | Final wave of mission M completed | one per mission in the pool | `mission_checks` |
| Tour clear | Every mission in a tour completed | one per tour | `tour_checks`, Campaign only |
| Money bonus | Wave completed with an A+ credit rating | one per wave | `money_checks` |
| Shop purchase | A check bought at the upgrade station for $100 to $400, then turned in | configurable | `shop_checks` |
| Tank / boss kill | Tank destroyed, Giant or boss robot killed | per mission, capped | `boss_checks` |

Wave clear is the backbone and the only group that is on by default. Everything
else is opt-in, because a run's length has to be tunable and the wave count
alone already gives roughly 6 to 8 checks per mission.

Shop checks are the interesting one and the riskiest. Roseburst's two variants:

- **Wave turn-in**: buy the check in the shop, clear the wave, receive it.
- **Mission turn-in**: buy it, clear the whole mission, receive it.

Both require the plugin to inject a purchasable entry into the upgrade station
UI, which is the single largest unknown in this spec. Flagged in Open
questions.

Damonj17's "upgrades count as checks like shops in other games" idea, where
opening the station dumps a wall of hints, is fun and belongs in an option, not
in the default.

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

The starting state has to be playable: adeleine64DS's "you could also require
that you have that class/weapon before you can buy the upgrade" is the right
instinct, but the generator must guarantee at least one usable class with at
least one usable weapon in sphere 0, or wave 1 is unwinnable and the seed is
dead.

## Allied bots

MvM waves are tuned for six players. A solo run without help is not a
randomizer, it is a wall.

Roseburst's `Allied Mercs` option (Off / Fill 6 / Fill 10 / Scavenge) is the
answer, and it is better than Damonj17's damage multiplier: a multiplier
compensates for missing damage but not for a missing Medic, a missing Engineer
or a missing Scout collecting money. Bots at least occupy the roles.

Open problem the thread identified and did not solve: RED bots do not buy
upgrades. adeleine64DS's suggestion, that bots simply share the player's
unlocked upgrades, is the cheapest fix and the one to try first.

Bot loadouts are themselves items under `Robot Templates` or `Single
Templates`, which turns "who is on my team" into part of the progression. Keep
that. It is the most original idea in the thread.

## Goals

- **Final Boss**: one mission from the hardest available tier is flagged, and
  clearing it wins.
- **Missionsanity**: clear X% of the missions in the pool.
- **Australium Hunt**: N junk Australium items are shuffled into the multiworld
  and you need a percentage of them. This is the only goal that makes the run
  depend on other players' worlds, which is the point of a multiworld, so it
  should not be treated as an afterthought.

## Traps and DeathLink

DeathLink in MvM needs a definition the thread does not give. A death in MvM is
cheap: you respawn at the next wave or after a short timer, and only a full
team wipe fails the wave. So DeathLink on individual death is noise. Options,
to be decided: fire on wave failure instead, or fire on individual death but
only outside the respawn grace window.

Traps from the thread, all plugin-side:

- Forced bad canteen or upgrade (Return to Spawn, Heavy Rage)
- Spawned Sentry Buster, Engineer, Sniper or Spy
- Map event triggers (Rottenburg's barrier, Mannhattan's capture points)
- Jarate on the whole team
- Stunned allied bots
- An extra Giant or boss

Traps that can make a wave mathematically unloseable-to-winnable are fine.
Traps that can corrupt the run state are not: nothing may permanently remove a
received item.

## Open questions

1. **Shop check injection.** Can a SourceMod plugin add an arbitrary entry to
   the MvM upgrade station UI, or only intercept purchases of existing ones? If
   only the latter, `shop_checks` has to be redesigned around hijacking an
   existing upgrade slot. This blocks the whole `shop_checks` group.
2. **Wave and mission identity.** Community missions from Potato.tf and
   Moonlight.tf are `.pop` files with no stable global id. `gamedata/` needs a
   naming scheme that survives a mission being renamed or a pack being
   updated, or old seeds break. Probably `map_name/pop_file_basename`.
3. **Allied bot upgrade sharing.** Does the RED bot upgrade path actually exist
   in current TF2, or was it removed? Needs testing on a live server before
   `Allied Mercs` can be specified properly.
4. **Wave count per mission.** Not derivable without parsing the `.pop` files.
   Either we parse them in `gamedata/` at build time, or we hardcode counts for
   the Valve missions and refuse community missions in v1.
5. **DeathLink semantics.** See above.
6. **Reconnection during a wave.** If the bridge loses the AP server mid-wave,
   the plugin keeps producing checks. The bridge must queue them durably and
   replay on reconnect. What happens if the bridge itself dies mid-wave is a
   separate problem and needs the queue on disk, not in memory.
