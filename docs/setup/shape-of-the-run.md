# The shape of the run

These values live in `.env`. They decide how long an evening is, how hard it
is, and what ends it.

**Set them before the first start.** The stack generates the session once and
then keeps it. A change made later does nothing until you
[start a new run](../operate/start-a-new-run.md).

## The run

| Variable | Default | Range | What it decides |
| --- | --- | --- | --- |
| `MVM_MISSION_COUNT` | `8` | 1 to 29 | How many missions the run draws |
| `MVM_DIFFICULTY` | `intermediate` | `normal`, `intermediate`, `advanced`, `expert` | The easiest tier that the run can draw |
| `MVM_GOAL` | `final_boss` | `final_boss`, `missionsanity` | What ends the run |
| `MVM_MISSIONSANITY_PERCENTAGE` | `80` | 10 to 100 | How much of the run the Missionsanity goal asks for |
| `MVM_DEATH_LINK` | `false` | `true`, `false` | Not implemented. See below. |

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

**Do not set `haunted`.** That tier holds one mission, Caliginous Caper, and
that mission holds one wave. Two checks is not enough room for the items that a
run needs, so generation stops with an error that names the option. The
container then restarts and prints the same error again. The tier starts
working the day Valve ships a second haunted mission.

### `MVM_GOAL`

`final_boss` marks the hardest mission that the run drew. Clearing that mission
ends the run. This gives an evening with a clear last fight.

`missionsanity` asks for a share of the missions that the run drew, in any
order. `MVM_MISSIONSANITY_PERCENTAGE` is that share, rounded up. The default of
`80` on an eight-mission run means seven missions in any order.

The `final_boss` goal ignores `MVM_MISSIONSANITY_PERCENTAGE`.

### `MVM_DEATH_LINK`

DeathLink is a convention where a death in one game kills every other player
who turned it on. It needs an agreed meaning of "death", and in MvM an
individual death is normal rather than notable.

This bridge does not do it. A session that asks for it gets one warning in the
bridge log and nothing else happens. Leave it at `false`.

## The session

| Variable | Default | What it decides |
| --- | --- | --- |
| `AP_SLOT_NAME` | `tf2` | The name of your server inside the randomized session |
| `AP_PASSWORD` | empty | The password of the session |
| `AP_PORT` | `38281` | The port of the randomizer server |

The randomizer server is not published outside the stack, so an empty
`AP_PASSWORD` is safe. Nothing on the network can reach it.

`AP_SLOT_NAME` is what the randomizer server calls your server in its own log
and in the chat that reaches your players. Change it if you want the lines to
read better.

Playing with someone in another game, on another machine, needs the randomizer
port published and a session that holds more than one participant. That is an
edit to `deploy/compose.yml`, not a value in `.env`. The file says where.

## The game server

| Variable | Default | What it decides |
| --- | --- | --- |
| `SRCDS_HOSTNAME` | `Mann vs Archipelago` | The name of the server in the browser |
| `SRCDS_ADMIN_STEAMIDS` | empty | Who may use `!mission` and the `sm_ap_` commands in the chat. Steam ids, separated by commas. Either the 17 digit form from a profile URL or SourceMod's `STEAM_0:1:...`. |
| `SRCDS_MAXPLAYERS` | `32` | Server slots. Team Fortress 2 refuses to host MvM with fewer, and caps RED at six itself. Do not lower it. |
| `SRCDS_STARTMAP` | `mvm_decoy` | The map that the server starts on |

`SRCDS_STARTMAP` takes any `mvm_` map. The run does not pick the map for you.
Change it between missions from the remote console, or edit `.env` and restart
the stack.

`SRCDS_PORT`, `SRCDS_PW`, `SRCDS_RCONPW` and `SRCDS_TOKEN` are covered in
[Invite your friends](invite-your-friends.md).

## Where these values go

The randomizer container writes them into one configuration file at the first
start and generates the session from it. After that the file is history: the
session is what the run follows.
