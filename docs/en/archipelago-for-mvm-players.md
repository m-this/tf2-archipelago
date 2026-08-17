# Archipelago for MvM players

Archipelago is a randomizer that spans several games at the same time. This
page defines the seven words that the rest of the book uses. Each word gets an
MvM example.

You do not need this page to play. Read it if you want to know why your Scout
has no scattergun.

## Multiworld

A multiworld is one randomized session that covers several players and several
games at once. The unlocks of each game are moved around inside the whole
session, not inside one game.

A multiworld can hold your Team Fortress 2 server, a Zelda game and a Doom
game. The shotgun that your team needs can sit behind a door in the Zelda game.
The sword that the Zelda player needs can sit behind wave 4 of Coal Town.
Nobody finishes alone.

A multiworld that holds only this Team Fortress 2 server is a normal setup. The
default `.env` produces exactly that.

## Slot

A slot is one participant in a multiworld. A slot has a name and plays one
game.

**In this project the slot is the server, not a player.** All six players on
RED share one slot and one set of unlocks. This is deliberate. Valve balances
MvM for a team of six. One slot for each player would leave one player with no
primary weapon and another player with no melee weapon, in a mode where the
team has to hold a hatch together.

The word "slot" also means a weapon slot in MvM. This book always writes
"weapon slot" for the MvM meaning.

## Seed

A seed is one generated multiworld. The generator decides once where every
unlock sits, and then nothing moves.

Your stack generates a seed at the first start and hosts it until you delete
it. The same wave holds the same unlock all evening, and all next evening. A
new seed is a new run, not a second try. See
[Start a new run](operate/start-a-new-run.md).

## Item

An item is an unlock that the multiworld hands out. In this project the items
are:

| Item | What it does |
| --- | --- |
| `Class: Scout` and the eight other classes | Makes that mercenary playable |
| `Progressive Weapon Slot` | Opens the primary, then the secondary, then the melee weapon slot |
| `Mission Ticket: Crash Course` and one per mission | Marks that mission as part of the run |
| `Cash Bundle` | Pays 200 credits to every player on the server |

An item may belong to any slot in the multiworld. Clearing a wave on your
server may hand a sword to the Zelda player. Their items arrive in your chat
when they find yours.

## Check and location

A location is a place that holds an item. A check is the act of reaching that
place and finding what it holds.

In this project a location is an MvM objective:

- Each wave that the team clears is a location. "Doe's Doom Wave 3" is one.
- Each mission that the team clears is a location. "Doe's Doom Complete" is
  one.

The 29 Valve missions hold 181 waves. With the 29 mission clears, that is 210
locations in the table. A run uses the missions that it drew, not all of them.

When your team clears wave 3 of Doe's Doom, the server checks that location.
The multiworld learns that the item at that place is found, and it sends the
item to its owner.

## Progression item

A progression item is an item that the generator's logic can require in order
to reach another location. The generator guarantees that the run is finishable
in some order using only these.

Weapon slots, class items and mission tickets are progression items here. An
advanced mission asks for three classes and two weapon slots before the
generator will put anything important behind it. A hard mission at the end of
the chain sits behind a lot of them.

Getting this classification wrong produces a run that cannot be finished, or a
run with no order in it at all. This is why the item list above is short and
fixed.

## Filler

A filler item is an item with no place in the logic. It is what pads the pool
when there are more locations than progression items.

`Cash Bundle` is the filler here. The run has 40 named items against up to 210
locations, so most waves hold either filler or an item that belongs to another
game in the multiworld.

## What this means at the keyboard

You start with a part of a loadout. One or two classes, one weapon slot, one
mission. Every wave that you clear finds something.

A class in grey, or a weapon that is not in your hands, is an item that the run
has not found yet. That is the game, not a fault.
