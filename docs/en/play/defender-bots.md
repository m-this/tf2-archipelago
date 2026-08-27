# The bots on your team

Team Fortress 2 balances every Mann vs Machine wave for six players on RED.
With two, the robots get through. Wave 1 of an Advanced mission is not
winnable by a pair, so a run of two people stalls on its first evening.

The server fills the empty slots for you. Nothing to install, nothing to type.

## What they do

The server fills RED to six when a wave begins, and keeps it full for the rest
of the wave. A bot that dies comes back within a second. The bots pick their
classes, fight, and spend their own credits at the upgrade station between
waves. They also ready themselves, so a wave starts when *you* press F4.

They are not human. The robots spot the spies late, and a bot never does the
clever thing your friend does. They are good enough to make a wave winnable,
which is what they are for.

A bot steps aside when a friend joins. RED holds six, so when it is full of
bots and a player arrives, one bot leaves and the player takes the seat. The
mod fills the team back up when the next wave begins.

The bots carry the game's own bot names, the ones a Valve server uses.

## Turning them down, or off

The settings in `.env`:

| Variable | Default | What it does |
| --- | --- | --- |
| `SRCDS_BOTS` | `1` | `0` keeps them off the field until an admin runs `sm_addbots` |
| `SRCDS_BOT_TEAM_SIZE` | `6` | How many players the server fills RED to, humans included |
| `SRCDS_BOT_CLASS_BLACKLIST` | empty | Classes the bots never play, separated by commas: `sniper,spy` |
| `SRCDS_BOT_TEAM_COMP` | empty | The classes the bots fill RED with, in order. See below. |
| `TF2AP_BOT_UPGRADES_CHAT` | `0` | `1` writes what the bots buy at the upgrade station in the chat |
| `SRCDS_BOT_HATS` | `1` | A random hat on every bot |
| `SRCDS_BOT_HAT_EFFECTS` | `0` | `1` puts a random unusual effect on that hat |

Lower `SRCDS_BOT_TEAM_SIZE` for a harder run. At `4`, three friends get one
bot. Set `SRCDS_BOTS=0` when six of you play and the slots are yours.

Change one of these against a record, not a memory. Valve tunes every wave for
six defenders, and the bots exist so fewer can win. Nobody measured how well
they do it. `wave_failures` in `/healthz` names every wave an evening
lost, worst first, and `tf2ap_wave_lost_total` plots the same thing. Play a
mission, read which wave stopped you, then change a number. See
[Troubleshooting](../operate/troubleshooting.md).

Bots are poor snipers and spies. `SRCDS_BOT_CLASS_BLACKLIST=sniper,spy` keeps
them on the classes they play well. The class names are the mod's:
`scout`, `soldier`, `pyro`, `demoman`, `heavyweapons`, `engineer`, `medic`,
`sniper`, `spy`.

A blacklist forbids classes. It does not say what the team is. A draw from the
rest gave a play-test three Spies and two Scouts on an Advanced mission.
Another team had no Engineer and lost wave 1 of Quarry twice.

`SRCDS_BOT_TEAM_COMP=engineer,medic,heavyweapons,soldier,demoman` names the
team instead. The order is the order the seats fill, so put the classes you
cannot do without first. Humans take seats before the bots do, and the last
entries are rarely reached. The names are the mod's, the same as the blacklist.

A team named here beats the blacklist. A list shorter than the empty seats
leaves the rest to the mod.

## Building a loadout

The presets cover the common builds. To make your own, open the **Loadouts**
page of the settings, pick a class, pick a weapon per slot, type a name and
press Save.

The loadout then appears at the bottom of every weapon menu for that class, on
the Team page and the Classes page. A loadout belongs to one class, because a
Medic cannot hold a Gunslinger.

Remove one and any seat still naming it plays stock. The team is not lost.

## Bending a mission for a short team

Valve tunes every wave for six defenders. The **Balancing** page of the settings
holds the two ways to help a team that is short of them.

Weapon buffs make the team stronger. They are on by default.

The three robot scales make the robots weaker: damage, health and speed. Each is
what the robots keep with one player on RED, and it rises back to 100 at six. A
full team therefore always plays the mission as Valve wrote it. All three start
at 100, which changes nothing.

Only robot damage has a measurement behind it. At 70 it does nothing. At 50 it
starts to bend a mission: one clear in eight attempts at a wave the unchanged
build lost every one of twenty four times. Health and speed carry no
measurement.

Robot speed does more than lower the difficulty. A slower robot lengthens the
wave and leaves more money on the field. It changes the pace of a mission as
well.

## Changing the team mid-mission

A lineup chosen for wave 1 is the wrong lineup for wave 5. Until now the only
fix was to restart the mission.

The launcher has a **Bot Switcher** tab, between the mission list and the log.
It shows what each seat plays and what it carries. To change the team, open the
Bots page of the settings, set the seats, then press Apply.

The mod replaces only the seats whose class changed, and leaves the others
alone. The wave continues, and the bots keep the money they earned.

In the game, `!ap bots` opens the same team as a menu. Pick a seat, then pick a
class. It needs the same admin right as the mission switch.

A change made during a wave takes effect at the next break. A bot removed
during a wave drops its buildings, and its replacement starts again from spawn
while the robots carry the bomb forward.

The game no longer lets a player inspect a teammate's upgrades. With
`TF2AP_BOT_UPGRADES_CHAT=1` the chat says what each bot buys, one line per
purchase. It is off by default because a bot buys a lot.

The last two are looks and nothing else. A bot draws a hat its class can wear
once, and wears that one for the rest of the mission, so the hat is how you
tell one Heavy from another. It draws again only if it changes class. The
effects are off by default because six unusual particles are on screen for the
whole wave. Neither touches how a bot plays, what it buys or what it carries.

War paints were here and are gone: they painted the weapon entities the upgrade
station replaces, and the server died the moment two engineers finished
shopping.

All of them take effect at the next map load. `make restart` is the certain
way.

On Windows the launcher has a **Bots** tab for the same settings. Six menus,
one per seat, name the team in order, and a loadout preset per class says what
a bot of that class spawns with. Stock weapons are the default. The three ticks
at the foot of that tab are the looks. See [Install on Windows](../setup/install-windows.md).

A bot holds its distance by what it carries, not by its class. A Brass Beast
closes in, because it cannot reposition once it is spun up; a Tomislav holds a
lane. A shotgun walks in rather than firing from minigun range.

A bot also pulls a weapon that still has ammo, rather than walking at a robot
holding an empty one. That is what a Heavy did when its minigun ran dry.

An Engineer nests near the hatch rather than in front of the robots' spawn
door. A sentry at the door meets a whole wave with no team around it yet, and
what the team gets for it is an Engineer rebuilding for the rest of the wave.
`sm_redbots_manager_engineer_nest_depth` says how far up the bomb path it may
build, as a fraction of the path: `0.4` by default, `1.0` the old spawn door.
A fraction, because the path is a few thousand units on Decoy and several
times that on Rottenburg.

At the upgrade station a bot buys damage first, and buys it for the weapon in
its hands. It used to buy at random, which is how a Heavy ended up with jump
height and a stock minigun. A few weapons decide for themselves: a Kritzkrieg
buys uber rate, a Rescue Ranger buys metal. An Engineer buys the sentry,
because that is where its damage is, and a Medic buys healing rather than a
syringe gun. Resistances come last: a bot respawns every wave.

## Who wrote them

[OfficerSpy/TF2-MvM-Defender-TFBots][mod], GPL-3.0, plus five dependencies:
CBaseNPC, Actions, TF2Attributes, TF Econ Data and TF2Utils. The server builds
them from source. TF2Attributes takes one fix of ours from `deploy/patches/`,
whose README says why.

The mod itself comes from our fork, [m-this/tf2-mvm-bots][fork]. Its `main`
branch is an upstream tag plus our changes, and `DEFENDERBOTS_VERSION` names a
tag of it.

The bots' behaviour is the mod's. Report a bot that walks into a wall to
OfficerSpy's repository, not to this one. The class blacklist and the
server-wide loadout file are ours, on the fork.

[mod]: https://github.com/OfficerSpy/TF2-MvM-Defender-TFBots
[fork]: https://github.com/m-this/tf2-mvm-bots

## On a server that is not this image

Every release attaches `tf2-defender-bots.zip`. It holds the whole stack:
plugins, extensions for Linux and Windows, gamedata and the per-map navigation
hints. The zip roots at `addons/`, so one unzip into the game directory (`tf/`)
installs it.

Then set the three convars in `server.cfg`. The image's entrypoint and the
Windows launcher both do this for you:

```
sm_redbots_manager_mode 2
sm_redbots_manager_defender_team_size 6
sm_redbots_manager_min_players -1
```

`mode 2` spawns the bots when a wave begins. `min_players -1` matters more than
it looks. The mod's own ready-up gate defaults to 3 and counts RED *before* the
wave, where a solo player has no bots yet. Leave it on, and it blocks the F4
that spawns them.
