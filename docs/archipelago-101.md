# Archipelago, for someone who has never played one

You do not need any of this to play. Join the server, shoot robots, and the
rest happens around you. Read it if you want to know why your Scout is missing
a scattergun.

## The idea

A randomizer takes a game and shuffles where its unlocks are found. Archipelago
does that across several games at once, so the shotgun you need might be sitting
in someone else's Zelda, and the sword they need might be behind wave 4 of
Coal Town. Nobody finishes alone.

## The five words

**Multiworld** — one session spanning everybody playing. Ours can be one game
or several; the Team Fortress 2 server is one participant in it.

**Slot** — one participant. In most games a slot is a person. Here it is **the
server**: everyone on RED shares one slot, so everyone shares the same unlocks.
Mann vs Machine is balanced around a coordinated team of six, and splitting the
randomizer per player would leave one friend without a primary weapon while
another has no melee.

**Check** — a place an item can be hidden, and the act of finding it. Here a
check is a wave cleared or a mission cleared. Clearing wave 3 of Doe's Doom
tells the multiworld "whatever was hidden there has been found", and whoever it
belongs to receives it.

**Item** — what was hidden. Ours are mercenary classes, weapon slots, mission
tickets and cash. Any of them can turn up in another player's game, and their
items can turn up in ours.

**Seed** — one generated multiworld. Fixed once it is made: the same checks
hold the same items for as long as the run lasts. Deleting it and starting
again is a new run, not a retry.

## What that means at the keyboard

You start with less than a full loadout. One or two classes, one weapon slot,
one mission. Clearing waves finds items; some are yours, some are somebody
else's, and the ones that are yours arrive in chat and take effect at once.

If a class is greyed out or a weapon is missing, it has not been found yet.
That is the game, not a bug.

## Talking to the multiworld

The server does it for you, but you can ask it things in chat:

| Typed in chat | What it does |
| --- | --- |
| `!ap` | help |
| `!ap missing` | which checks are still unfound |
| `!ap status` | where the run is |
| `!ap hint Scout` | ask where an item is; costs hint points |
| `!apchat nice one` | talk to the other players in the multiworld |

What the rest of the multiworld says arrives in chat the same way.

## The goal

One of two, set when the seed is made:

- **Final Boss** — the hardest mission drawn is flagged. Clear it and the run
  is over.
- **Missionsanity** — clear a share of every mission drawn, in any order.

## What this is not

- Not a mod you install. Join with a stock client.
- Not competitive Team Fortress 2. Mann vs Machine only.
- Not saved to your Steam account. The run belongs to the server.
