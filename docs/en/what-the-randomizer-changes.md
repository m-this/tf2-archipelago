# What the randomizer changes

This page assumes the vocabulary of
[Archipelago for MvM players](archipelago-for-mvm-players.md).

Everything below happens on the server. The players install nothing.

## The classes start locked

The nine mercenaries are nine separate items. A run starts with one to four of
them, depending on how hard the easiest mission of the run is.

| Tier of the starting mission | Classes at the start | Weapon slots at the start |
| --- | --- | --- |
| Normal | 1 | 1 |
| Intermediate | 2 | 1 |
| Advanced | 3 | 2 |
| Expert | 4 | 3 |

A player who picks a locked class in the class menu gets a line in the chat and
stays where they are. A player who is already on a class that the run has not
unlocked keeps playing the wave and changes class at the next spawn. The plugin
never forces a respawn: in MvM a free respawn hands back the life that dying
cost you.

## The weapon slots start locked

There are three weapon slots: primary, secondary, melee. One item,
`Progressive Weapon Slot`, opens them one at a time. There are three copies of
it in the pool.

Which slot a copy opens depends on the class you play. Six classes get the
primary slot, then the secondary, then the melee. Three do not, because their
first slot is what the class is for:

| Class | First | Second | Third |
| --- | --- | --- | --- |
| Medic | Secondary, the Medigun | Primary | Melee |
| Engineer | Melee, the Wrench and the PDAs | Primary | Secondary |
| Spy | Melee, the Knife | Secondary, the Sapper | Primary |

The count is what the run holds, not the slot. A run with one copy opens the
Medigun for a Medic and the Scattergun for a Scout, at the same moment.

A locked weapon slot is empty. The server removes the weapon each time the game
gives it to you: at spawn, at the resupply locker, and at the upgrade station.
If the weapon that goes away is the one in your hands, the server puts another
one there, so you are never left holding nothing.

Weapon ownership is not randomized. The Scattergun and the Force-A-Nature are
the same item to the slot lock: either can be equipped once Primary is open.

Individual weapons can receive useful passive buffs, however. The catalog
contains curated weapon/effect combinations, including many combinations the
stock game would never create. A buff applies to that functional weapon family
and combines with upgrades bought at the MvM station. Numeric effects add
another level each time; on/off effects such as
airborne crits, ignite, Gas Passer gasoline and Mad Milk clamp at one. Other
effects include projectile count and speed, bleed duration, afterburn damage,
extra damage, faster firing or reload, a larger clip and healing on kill.
Projectile count adds pellets to hitscan weapons and additional rockets per
trigger pull. Extra rockets ignore direct contact with their shooter while
retaining normal explosion splash damage.

The second pass adds slow on hit, 50% faster gestures and consumable use,
active health regeneration, faster deploy, meter recharge and minigun spin-up,
ammo on hit, mini-crits on kill, increased healing received, rocket-jump damage
resistance, stronger self-damage launch force, cheaper airblasts, longer Über,
revealing cloaked or disguised victims and a speed boost on hit.

Mechanic-specific effects stay with weapons that can use them: airblast with
airblast-capable flamethrowers, building and metal effects with Engineer items,
healing and Über effects with Medi Guns, and banner duration with banners.
Passive equipment and consumables do not draw combat effects. Functional
reskins share one pool, so a Pistol reward also follows the Lugermorph and
C.A.P.P.E.R., for example.

A run samples weapon/effect permutations for the checks left after its required
classes, slots and mission tickets. By default every spare check is a buff and
cash rewards are disabled. If cash is enabled, buffs occupy 75% of this spare
space and cash fills the rest. Each buff draw has a 25% chance to add another
level to a numeric permutation already in the seed; toggle effects never
repeat. The Rewards tab controls the cash split and lets tickets, classes,
slots and buffs independently be useful or required for progression. Small
runs therefore contain a subset of the catalog, and a run never creates more
items than it has locations.

Entering an upgrade station opens a numbered menu listing the buffs that apply
to the weapons in the current loadout. `sm_ap_buffs` opens the same menu on
demand.

## The missions sit behind tickets

Each of the 29 Valve missions has its own `Mission Ticket` item. The tickets
are what the generator uses to decide the order of the run.

The plugin does not refuse a map. If the server runs a mission whose ticket the
run has not found, the chat says so and the waves still count as checks. The
map rotation belongs to you, not to the randomizer.

### Logic is per mission, not per wave

A mission is one region. Every check inside it sits behind the same gate: the
waves, the tank, the giant and the clear. So a ticket puts the whole mission in
logic at once. No seed puts wave 3 in logic and leaves wave 6 out of it. The
answer to "how much of a wave am I expected to do" is the whole mission.

A ticket alone is not always enough. Each tier also asks for some classes and
weapon slots before the generator calls its missions beatable:

| Tier | Classes | Weapon slots |
| --- | --- | --- |
| Normal | 1 | 1 |
| Intermediate | 2 | 1 |
| Advanced | 3 | 2 |
| Expert | 4 | 3 |
| Haunted | 5 | 3 |

Both counts are always reachable: every seed's pool holds every class and every
weapon slot. The counts sit below what a real team wants, on purpose. Logic
decides what is possible, and a wave that is merely hard is still possible.

## A cleared wave is a check

Each wave that the team clears reports one check. Each mission that the team
clears reports one more. The first tank the team destroys in a mission reports
one more again, and so does the first giant. Both are once per mission, however
many the mission holds.

Mannhattan's three missions run on gates and hold no tank, so they have no tank
check. Every mission holds a giant, so every mission has that one. A check
nobody can reach is a run nobody can finish.

A lost wave reports nothing, and the randomizer adds no penalty. The team
replays the wave, the same as in normal MvM.

The game decides when a wave is lost, and this project does not change that
rule. The robots have to carry the bomb into the hatch. A team wipe on its own
does not lose the wave: the game respawns the team, and the wave goes on.

## DeathLink kills the team, and nothing more

DeathLink is off unless the seed asks for it. With it on, a death crosses the
multiworld both ways.

Outbound: the team loses a wave, and the bridge sends that loss as a death. The
plugin hooks the game's own `mvm_wave_failed`, so what other players receive is
what ended the wave on your screen.

Inbound: a death arrives, and the plugin kills everybody on RED, bots included.
It fires no wave-failed event. Nobody holds the hatch until the team respawns,
so the wave is usually lost, but the game decides that as always.

A wave lost to an arriving death is not sent back out. So one death cannot
travel back and forth between two linked players.

## Credits can arrive as items

`Cash Bundle` pays 200 credits. Each player on RED gets the full 200, so a
bundle that arrives with six players on the server is 1200 credits of upgrades.

A bundle pays between waves, at the upgrade station, and not before. A wave the
team loses takes it back to where the wave began, and money paid into that wave
goes back with it. Waiting for the upgrade station costs nothing, because that
is where the money is spent.

So a bundle that arrives mid-wave waits for the end of it, and one that arrives
with nobody on the server waits for somebody. The chat says how much is
waiting. Nothing is lost either way.

Credits are still the one item in this project that happens once and is over.
Classes, weapon slots and tickets are facts that stay true, so the server can
apply them again after any restart.

A bundle is paid once. The money then belongs to the mission the way any other
credits do. The end of a mission clears it, the same as in normal MvM.

## Traps arrive from other players

A trap is an item with a negative effect, and it belongs to the multiworld like
every other item. Somebody in another world opens a chest, and your team pays
for it.

`trap_percentage` decides how much of the run's spare space holds one. It is
zero by default, so a seed that did not ask gets none.

There is one trap so far. `Trap: Team Jarate` soaks everyone on RED, bots
included: ten seconds of taking 35% more damage and dealing no crits.

A trap fires during a wave. One that arrives between waves waits for the next
one, and the chat says so. Jarate on a team standing at the upgrade station is
not a trap.

A trap can cost the team a wave. No trap takes back an unlock the run has
already found.

## Everybody shares the unlocks

There is one set of unlocks for the whole server. A weapon slot that the run
finds opens for all six players at the same moment. A class that the run has
not found is locked for all six.

This also means that a player who joins in the middle of the evening gets the
current set at once. There is nothing per player to carry over.

## Nobody installs anything

Your friends connect with a standard Team Fortress 2 client. No mod, no
launcher, no account link. Everything they see arrives as chat text from the
server.

The run belongs to the server. It is not stored on anybody's Steam account.

## What does not change

The upgrade station, the credits that robots drop, the canteens, the wave
layouts and the robots themselves are stock MvM. This version randomizes the
classes, the weapon slots, the missions and the money. The individual weapons,
the upgrade lines and the canteens are not in it.
