# What the randomizer changes

Everything on this page happens on the server. The players install nothing.
For the words, see [Archipelago words, for MvM players](archipelago-for-mvm-players.md).

## The classes start locked

- Each of the nine mercenaries is an item.
- A run starts with one to four of them. The tier of the easiest mission decides
  how many. See [Starting kit by tier](#starting-kit-by-tier).
- Picking a locked class in the class menu does nothing, and the chat says why.
- A player already on a class that gets locked keeps playing until the next
  spawn. The server never forces a respawn.

## The weapon slots start locked

- There are three weapon slots: primary, secondary, melee.
- By default one item, `Progressive Weapon Slot`, opens them one at a time.
  There are three copies of it in the pool.
- A locked slot is empty. The server removes the weapon at spawn, at the
  resupply locker and at the upgrade station. You are never left holding
  nothing: the server switches you to a weapon you still have.
- Which weapon you put in an open slot is your choice. The run does not
  randomize weapons. Scattergun or Force-A-Nature, both work once primary is
  open.

The first slot that opens is the one the class needs most, so the order is
not the same for every class:

| Class | First | Second | Third |
| --- | --- | --- | --- |
| Scout, Soldier, Pyro, Demoman, Heavy, Sniper | Primary | Secondary | Melee |
| Medic | Secondary (Medigun) | Primary | Melee |
| Engineer | Melee (Wrench) | Primary | Secondary |
| Spy | Melee (Knife) | Secondary (Sapper) | Primary |

The [run option](setup/shape-of-the-run.md#weapon-slots-per-class) **Weapon
slots per class** gives each class its own slot items instead. That is
eighteen items instead of three, so it needs a longer run.

## The missions sit behind tickets

- Each mission has its own `Mission Ticket` item.
- The run starts with one mission open. Tickets open the others.
- The plugin does not refuse a map. If the server runs a mission the run has
  not unlocked, the chat says so and the waves still count.
- Logic is per mission, not per wave. A ticket puts the whole mission in logic
  at once.

The generator also asks for some classes and slots before it calls a mission
beatable. These counts are low on purpose. A hard wave is still possible.

### Starting kit by tier

| Tier of the mission | Classes | Weapon slots |
| --- | --- | --- |
| Normal | 1 | 1 |
| Intermediate | 2 | 1 |
| Advanced | 3 | 2 |
| Expert | 4 | 3 |
| Haunted | 5 | 3 |

The run starts with the kit its easiest mission needs.

## A cleared wave is a check

Each mission gives these checks:

- one per wave the team clears,
- one when the team clears the mission,
- one for the first tank the team destroys in that mission,
- one for the first giant the team kills in that mission.

Mannhattan's missions have no tank, so no tank check. A lost wave gives nothing
and costs nothing. The team replays it, as in normal MvM.

[Run options](setup/shape-of-the-run.md#more-checks) can add more checks:
victory caches, milestones, a check per giant and per tank.

## The items you receive

| Item | What it does |
| --- | --- |
| `Class: Scout` and the eight others | Opens that class for everybody |
| `Progressive Weapon Slot` | Opens the next weapon slot for everybody |
| `Mission Ticket: ...` | Puts that mission in the run |
| Weapon buffs | A permanent bonus on one weapon family: more damage, more pellets, faster reload, and so on. The upgrade station shows a list of the buffs on your loadout. |
| `Cash Bundle` | 200 credits for each player on RED, paid at the next upgrade station visit |
| `Trap: Team Jarate` | Ten seconds of Jarate for the whole team, during the next wave |
| `Trap: Team Mad Milk`, `On Fire`, `Slow Down`, `Bleed`, `Marked For Death`, `Gas` | The same, with that effect instead. Only in a run that asks for them. |
| `Grappling Hook` | Turns Mannpower's hook on for everybody for the rest of the run. Off by default. |

Buffs, cash and traps fill the checks left over after the classes, slots and
tickets. [Run options](setup/shape-of-the-run.md#rewards) decide the mix.

Classes, slots and tickets are permanent. The server applies them again after
any restart. Cash is paid once, and it belongs to the mission like any other
credits.

## DeathLink

DeathLink is off unless the seed asks for it. With it on:

- your team loses a wave, and every other DeathLink player dies;
- another DeathLink player dies, and everybody on RED dies, bots included.

The plugin only kills. The game decides whether the wave is lost, as always.
A wave lost to an arriving death is not sent back out.

## What does not change

- The upgrade station, the credits, the canteens, the wave layouts and the
  robots are stock MvM.
- Weapons themselves are not randomized. Neither are upgrades.
- The run belongs to the server. Nothing goes on anybody's Steam account.

Next: [Requirements](setup/requirements.md).
