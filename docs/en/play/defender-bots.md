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

They are not human. Engineers build too close to the robots, and the robots
spot the spies late. A bot never does the clever thing your friend does. They
are good enough to make a wave winnable, which is what they are for.

## Turning them down, or off

Two settings in `.env`:

| Variable | Default | What it does |
| --- | --- | --- |
| `SRCDS_BOTS` | `1` | `0` keeps them off the field until an admin runs `sm_addbots` |
| `SRCDS_BOT_TEAM_SIZE` | `6` | How many players the server fills RED to, humans included |

Lower `SRCDS_BOT_TEAM_SIZE` for a harder run. At `4`, three friends get one
bot. Set `SRCDS_BOTS=0` when six of you play and the slots are yours.

Both take effect at the next map load. `make restart` is the certain way.

On Windows the launcher asks the same two questions. See
[Install on Windows](../setup/install-windows.md).

## Who wrote them

[OfficerSpy/TF2-MvM-Defender-TFBots][mod], GPL-3.0, plus five dependencies:
CBaseNPC, Actions, TF2Attributes, TF Econ Data and TF2Utils. The server builds
them from source, with two fixes of our own in `deploy/patches/`, whose README
says why each exists.

The bots' behaviour is the mod's. Report a bot that walks into a wall to that
repository, not to this one.

[mod]: https://github.com/OfficerSpy/TF2-MvM-Defender-TFBots

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
