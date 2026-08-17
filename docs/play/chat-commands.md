# Chat commands

Everything here is typed in the normal Team Fortress 2 chat. There is no client
to install and no second window to keep open.

## For the players

| Type this | What the server does |
| --- | --- |
| `!ap` | Print the help |
| `!ap missing` | List the checks that nobody has found yet |
| `!ap status` | Print the state of the randomized session |
| `!ap checked` | List the checks that are already found |
| `!ap remaining` | List what is left. The randomizer server decides whether it answers before the run is over. |
| `!ap players` | List the participants in the session |
| `!ap hint Scout` | Ask where an item is |
| `!ap hint_location Doe's Doom Wave 3` | Ask what a place holds |
| `!ap options` | Print the settings of the session |
| `!ap help` | Print the help of the randomizer server |
| `!ap license` | Print the licence of the randomizer server |
| `!apchat nice one` | Speak to the other players in the session |

`!ap` sends the command to the randomizer server and prints the answer in the
chat. `!apchat` sends plain text. The lines of the other players arrive in the
same chat.

Asking where an item is costs hint points, which the session earns from checks.
Use the item names from
[Archipelago for MvM players](../archipelago-for-mvm-players.md).

## Commands that are refused

The list above is the whole list. Any other command is refused with:

```
[AP] That multiworld command cannot be sent from the game. It cannot be undone.
```

Every command in the list only reads. The ones that are missing change the run,
and nothing in a randomized session undoes them. `!release` hands the remaining
items of this server to the other participants. One line typed by one player
would end the run for everybody, and a game server has no accounts and no way
to tell who should be allowed.

So the rule is a list of what is allowed, held on the bridge. There is one list
rather than one per component.

## The other refusals

| The chat says | Why |
| --- | --- |
| `Wait a moment before speaking to the multiworld again.` | One player may speak once every three seconds |
| `Too much is going to the multiworld. Wait a moment.` | Five lines at once for the whole server, then one every three seconds |
| `That line is too long for the multiworld.` | A line is at most 300 characters |
| `The bridge has no connection to the multiworld. It refused your line.` | The randomizer server is unreachable right now |

A line is never queued. A message that lands ten minutes late is worse than one
that was refused while the player was still reading the chat.

## For the host

The admin commands run from the remote console. Connect to the server, open the
developer console, and type:

```
rcon_password your-SRCDS_RCONPW
rcon sm_ap_status
```

| Command | What it does |
| --- | --- |
| `sm_ap_status` | Print the mission, the wave, which game events exist, the unlocks, the queue depth and the last error |
| `sm_ap_resync` | Ask the bridge for the unlock set again |
| `sm_ap_report wave_cleared 3` | Report a cleared wave by hand |
| `sm_ap_report mission_cleared` | Report a cleared mission by hand |

`sm_ap_report` with no wave number uses the wave that the game is on. Reporting
the same check twice is not a problem: the bridge identifies a check by the
place it belongs to, so the second report does nothing.

## The settings

These are console variables. Set one for the session with
`rcon tf2ap_debug 1`. To keep it across restarts, edit
`cfg/sourcemod/tf2_archipelago.cfg` in the game files. The stack never
overwrites that file once it exists.

| Variable | Default | What it does |
| --- | --- | --- |
| `tf2ap_announce` | `1` | Write the cleared waves and the received items in the chat |
| `tf2ap_chat` | `1` | Write what the rest of the session says in the chat |
| `tf2ap_debug` | `0` | Write every bridge call and every game event in the chat and the console |
| `tf2ap_bridge_url` | `http://127.0.0.1:24680` | Where the bridge is. Loopback only. Do not change it. |

Errors reach the chat whatever `tf2ap_announce` is set to. A failure that
nobody can see gets blamed on the game.
