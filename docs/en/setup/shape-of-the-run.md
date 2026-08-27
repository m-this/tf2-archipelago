# The shape of the run

These values live in `.env`. They decide how long an evening is, how hard it
is, and what ends it.

**Set them before you make the session.** `make seed` reads them once, and the
session keeps them. A change made later does nothing until you
[start a new run](../operate/start-a-new-run.md).

## The run

| Variable | Default | Range | What it decides |
| --- | --- | --- | --- |
| `MVM_MISSION_COUNT` | `8` | 1 to 29 | How many missions the run draws |
| `MVM_DIFFICULTY` | `intermediate` | `normal`, `intermediate`, `advanced`, `expert` | The easiest tier that the run can draw |
| `MVM_GOAL` | `final_boss` | `final_boss`, `missionsanity` | What ends the run |
| `MVM_MISSIONSANITY_PERCENTAGE` | `80` | 10 to 100 | How much of the run the Missionsanity goal asks for |
| `MVM_DEATH_LINK` | `false` | `true`, `false` | A lost wave kills the linked players, and their deaths wipe the team. See below. |
| `MVM_EXCLUDED_MISSIONS` | empty | popfile names, separated by commas | Missions the run never draws. See below. |
| `MVM_START_MISSION` | `random` | a popfile name | The mission the run starts on. See below. |
| `MVM_START_CLASS` | `random` | a mercenary name | The class the run starts with. See below. |

### `MVM_MISSION_COUNT`

Most Valve missions hold six or seven waves, so eight missions is about 50
waves. That is one evening for a team that knows the mode.

Asking for more missions than the pool holds gives you the whole pool. The
generator also draws more missions than you asked for when the run needs more
places to put its unlocks. It writes a line in the log when it does.

### `MVM_DIFFICULTY`

This is the easiest tier that the run can draw. The run draws that tier and
every tier above it. `intermediate` draws intermediate, advanced and expert,
which is 25 of the 29 missions. `expert` draws the three expert missions and
the one haunted mission, which is four.

The tier of the easiest mission drawn also sets what the team starts with. A
normal run starts with one class and one weapon slot. An expert run starts with
four classes and all three weapon slots, because an expert mission is not
playable with less.

The standalone launcher omits a `haunted` preset. That tier holds only
Caliginous Caper: its wave, tank, giant and completion checks provide exactly
enough locations for the four remaining class unlocks, but a one-mission run
has no mission-ticket progression and commits the whole run to 666 robots.
Hand-authored YAML may still select it.

The launcher's **Check Run Selection** action applies the same capacity rule
before it writes or generates a player file. Saving settings and starting Test
mode also refuse an empty or undersized pool. The official Archipelago
generator repeats the validation and remains the final authority.

### `MVM_GOAL`

`final_boss` marks the hardest mission that the run drew. Clearing that mission
ends the run. This gives an evening with a clear last fight.

`missionsanity` asks for a share of the missions that the run drew, in any
order. `MVM_MISSIONSANITY_PERCENTAGE` is that share, rounded up. The default of
`80` on an eight-mission run means seven missions in any order.

The `final_boss` goal ignores `MVM_MISSIONSANITY_PERCENTAGE`.

### `MVM_DEATH_LINK`

DeathLink is a convention where a death in one game kills every other player
who turned it on. In MvM a single death is normal rather than notable, so here
a death is the wave the team lost.

With `true`, a lost wave kills every linked player in the multiworld. One of
their deaths kills everybody on your team, bots included.

The plugin only kills. It does not fail the wave. Nobody holds the hatch until
the team respawns, so the wave is usually lost, but the game decides that as
always. A wave lost that way is not sent back out.

Expect a run with DeathLink on to be harder. A wave can be lost to somebody
else's mistake.

### `MVM_START_MISSION` and `MVM_START_CLASS`

`random` leaves both to the seed. The run then starts on the easiest mission it
drew, with classes drawn at random.

Name a mission and the run always draws that one and starts there. Name a
mercenary and the run always starts with it. How many classes the run starts
with is the tier of the start mission's business, and `MVM_START_CLASS` names
one of them.

`MVM_START_MISSION=mvm_coaltown_advanced MVM_START_CLASS=Engineer` starts every
run on Ctrl+Alt+Destruction with an Engineer.

The Final Boss goal is the hardest mission the run drew. If you name that
mission as the start, clearing it wins on the spot. Generation stops and says
so. Name an easier mission, or use the Missionsanity goal.

The Windows launcher has one menu for both, and the server boots on the mission
you pick rather than on `SRCDS_STARTMAP`.

### `MVM_EXCLUDED_MISSIONS`

Missions the run never draws, whatever the tier. Name them by popfile,
separated by commas. `MVM_EXCLUDED_MISSIONS=mvm_ghost_town_666` keeps out
Caliginous Caper, one wave of 666 robots that takes an hour on its own. The
Windows launcher lists the missions with a box to untick.

## The session

| Variable | Default | What it decides |
| --- | --- | --- |
| `AP_HOST` | `archipelago.gg` | Where the session runs |
| `AP_PORT` | none | The port of the room |
| `AP_TLS` | `true` | `wss://` rather than `ws://`. A room on `archipelago.gg` answers `wss://`. |
| `AP_SLOT_NAME` | `tf2` | The name of your server inside the session |
| `AP_PASSWORD` | empty | The password of the room |

`AP_HOST`, `AP_PORT` and `AP_TLS` come from the room page. See
[Create the session](create-the-session.md), which also covers the three values
that host the session on your own machine instead.

Anybody who has the address of a room reaches that room. Set a password on the
room and repeat it in `AP_PASSWORD`, unless the address stays between friends.

`AP_SLOT_NAME` is what the session calls your server in its log and in the chat
that reaches your players. Change it if you want the lines to read better.

Playing with someone in another game needs a session that holds more than one
slot. Generation reads one file of options, written by
`deploy/archipelago-entrypoint.sh`, and a second player needs a second file
beside it. The room takes that player with no port to open.

## The game server

| Variable | Default | What it decides |
| --- | --- | --- |
| `SRCDS_HOSTNAME` | `Mann vs Archipelago` | The name of the server in the browser |
| `SRCDS_ADMIN_STEAMIDS` | empty | Who may use `!mission` and the `sm_ap_` commands in the chat. Steam ids, separated by commas. Either the 17 digit form from a profile URL or SourceMod's `STEAM_0:1:...`. |
| `SRCDS_MAXPLAYERS` | `32` | Server slots. Team Fortress 2 refuses to host MvM with fewer, and caps RED at six itself. Do not lower it. |
| `SRCDS_STARTMAP` | `mvm_decoy` | The map that the server starts on |
| `SRCDS_START_MISSION` | empty | The mission the server loads once the map is up, as a popfile name |
| `TF2AP_NEXT_MISSION_DELAY` | `30` | Seconds between a mission clear and the next mission |

The run picks the mission from there. If the loaded mission is not part of the
run, or the run has not unlocked it, the server moves to the first unlocked
mission. When the team clears a mission, the server loads the next unlocked
one after `TF2AP_NEXT_MISSION_DELAY` seconds. See
[Chat commands](../play/chat-commands.md).

`SRCDS_PORT`, `SRCDS_PW`, `SRCDS_RCONPW` and `SRCDS_TOKEN` are covered in
[Invite your friends](invite-your-friends.md).

## Where these values go

`make seed` writes them into one configuration file and generates the session
from it. After that the file is history: the session is what the run follows.
