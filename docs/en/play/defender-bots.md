# The bots on your team

Team Fortress 2 balances every Mann vs Machine wave for six players on RED.
With two, the robots get through. Wave 1 of an Advanced mission is not
winnable by a pair, so a run of two people would stall on its first evening.

The server fills the empty slots for you. Nothing to install, nothing to type.

## What they do

RED is filled to six when a wave begins, and stays filled for the rest of it:
a bot that dies is replaced within a second. They pick their classes, fight,
spend their own credits at the upgrade station between waves, and ready
themselves, so a wave starts when *you* press F4.

They are not human. Engineers build too close to the robots, spies get spotted
late, and a bot will never do the clever thing your friend does. They are good
enough to make a wave winnable, which is what they are for.

## Turning them down, or off

Two settings in `.env`:

| Variable | Default | What it does |
| --- | --- | --- |
| `SRCDS_BOTS` | `1` | `0` keeps them off the field until an admin runs `sm_addbots` |
| `SRCDS_BOT_TEAM_SIZE` | `6` | How many players RED is filled to, humans included |

Lower `SRCDS_BOT_TEAM_SIZE` for a harder run: at `4`, three friends get one
bot. Set `SRCDS_BOTS=0` when six of you are playing and the slots are yours.

Both take effect at the next map load. `make restart` is the certain way.

## Who wrote them

[OfficerSpy/TF2-MvM-Defender-TFBots][mod], GPL-3.0, plus five dependencies:
CBaseNPC, Actions, TF2Attributes, TF Econ Data and TF2Utils. The server builds
them from source at image build time, with two fixes of our own carried in
`deploy/patches/`. [MvM Defender TFBots and the build-from-source
question](../mvm-defender-bots.md) is the long version.

They are a work in progress upstream, and so is their behaviour here. A bot
that walks into a wall is a bug worth reporting to that repository, not to this
one.

[mod]: https://github.com/OfficerSpy/TF2-MvM-Defender-TFBots

## On a server that is not this image

Every release attaches `tf2-defender-bots.zip`. It holds the whole stack —
plugins, extensions for Linux and Windows, gamedata and the per-map navigation
hints — rooted at `addons/`, so installing it is one unzip into the game
directory (`tf/`) next to SourceMod.

Then set the three convars in `server.cfg`, which is what this image's
entrypoint does for you:

```
sm_redbots_manager_mode 2
sm_redbots_manager_defender_team_size 6
sm_redbots_manager_min_players -1
```

`mode 2` spawns the bots when a wave begins. `min_players -1` matters more than
it looks: the mod's own ready-up gate defaults to 3 and counts RED *before* the
wave, where a solo player has no bots yet, so leaving it on blocks the F4 that
would have spawned them.
