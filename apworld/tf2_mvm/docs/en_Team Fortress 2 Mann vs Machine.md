# Team Fortress 2 Mann vs Machine

## What does randomization do to this game?

Mann vs Machine is Team Fortress 2's co-operative wave defence mode: six
players hold a bomb-carrying robot army off a hatch, wave after wave, buying
upgrades between waves.

A run draws a set of missions out of the 29 Valve ones. Clearing a wave is a
check, and so is clearing a mission. You do not start with the whole roster:
the mercenary classes and the loadout slots you may use are items, and so is
the ticket that lets the server load a given mission. Harder tiers ask for more
of a team before the logic considers them beatable, so an Expert mission drawn
early sits locked until the run has handed out most of the classes.

## What is the goal?

Either of two, picked in the YAML:

- **Final Boss**: the hardest mission drawn is flagged. Clear it and the run is
  over.
- **Missionsanity**: clear a share of every mission drawn, in any order.

## Which items can be in another player's world?

All of them: mission tickets, mercenary classes, the progressive weapon slot
and the cash bundles that pad the pool.

## What does another world's item look like in Mann vs Machine?

The server announces it in chat as it arrives. There is no in-game model for
someone else's item, and the plugin does not interrupt a wave to show you one.

## Do I need to install anything?

No. The randomizer lives entirely on the server: a SourceMod plugin watches the
game and a small bridge process talks to the Archipelago server. Join with a
stock, unmodified Team Fortress 2 client.

Whoever is hosting has work to do; see the setup guide.

## A note on the slot

One Archipelago slot covers the whole server rather than one per player.
Progression is collective: everyone on RED shares the same unlocked classes,
slots and missions. Mann vs Machine is balanced around a coordinated team of
six, and a per-player randomizer would leave one friend without a primary
weapon while another has no melee.
