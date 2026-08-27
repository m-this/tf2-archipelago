# Chat commands

Everything here is typed in the normal Team Fortress 2 chat. There is no client
to install and no second window to keep open.

## For the players

| Type this | What the server does |
| --- | --- |
| `!ap` | Print the help |
| `!ap status` | Print the state of the run. The mission and the wave. The classes and the slots the run holds. How many missions the run unlocked and cleared. Whether the bridge is connected to the randomizer server |
| `!mission` | List the missions of the run. Mark the one that plays, the cleared ones and the locked ones |
| `!ap missing` | List the checks that nobody has found yet |
| `!ap checked` | List the checks that are already found |
| `!ap remaining` | List what is left. The randomizer server decides whether it answers before the run is over. |
| `!ap players` | List the participants in the session |
| `!ap hint Class: Scout` | Ask where an item is |
| `!ap hint_location Doe's Doom Wave 3` | Ask what a place holds |
| `!ap options` | Print the settings of the session |
| `!ap help` | Print the help of the randomizer server |
| `!ap license` | Print the licence of the randomizer server |
| `!apchat nice one` | Speak to the other players in the session |
| `!ap unlock mission` | Hand over the next mission ticket. Test mode only: a real randomizer server has never heard of it |

The game server itself answers `!ap status` and `!mission`, so they work when
the randomizer server does not answer. Every other `!ap` command goes to the
randomizer server, which prints its answer in the chat. `!apchat` sends plain
text. The lines of the other players arrive in the same chat. Team chat works
the same as all chat.

Asking where an item is costs hint points, which the session earns from checks.
Use the item names from
[Archipelago for MvM players](../archipelago-for-mvm-players.md).

The whole name, including the prefix. `!ap hint Scout` does not find
`Class: Scout`: the randomizer server matches on the whole name and refuses
anything under 75% alike, and the prefix is more than half of that name. It
answers with the name it thinks you meant, so the second try works.

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

## For the admin

An admin is a Steam id in `SRCDS_ADMIN_STEAMIDS`. Set it before the first start
and it works in the normal chat, like any other command:

| Type this | What the server does |
| --- | --- |
| `!mission 3` | Switch to the third mission in the list |
| `!mission mvm_decoy_intermediate` | Switch by mission file name |
| `!ap bots` | Open the bot team as a menu. Pick a seat, pick a class |

`!ap bots` needs the same right as the mission switch. Both decide what the
whole RED team plays for the rest of the run. A change made during a wave takes
effect at the next break, because a bot removed during a wave drops its
buildings.

A player who is not an admin is told no rather than ignored. A mission the run
has not unlocked is refused for everybody: its ticket is somewhere in the
multiworld.

Switching changes the mission, and the map with it when the two differ.

## Which mission plays

The run decides, not the map cycle:

- The server starts on `tf2ap_start_mission`. If that value is blank, it
  starts on the map's own mission.
- If the loaded mission is not part of the run, the server moves to the first
  unlocked mission that is not cleared. It does the same if the run has not
  unlocked the loaded mission.
- When the team clears a mission, the server says which mission comes next.
  It loads that mission `tf2ap_next_mission_delay` seconds later. The next
  mission is the first unlocked one that is not cleared, in the order the
  seed drew them. When the team has cleared every unlocked mission, the
  server plays one of them again until a ticket opens another.

## For the host

The same commands, and a few more, run from the remote console. Connect to the
server, open the developer console, and type:

```
rcon_password your-SRCDS_RCONPW
rcon sm_ap_status
```

| Command | What it does |
| --- | --- |
| `sm_ap_status` | Print the mission, the wave, which game events exist, the unlocks, the missions, the state of the chat relay, the queue depth, and the last error |
| `sm_ap_mission` | List the missions of the run. With an argument, switch to one |
| `sm_ap_resync` | Ask the bridge for the unlock set again |
| `sm_ap_buffs` | Open the numbered menu of buffs applying to the current loadout |

### Debug and test commands

These require root admin access. Test buffs last until the plugin, map, or run
state reloads them.

| Command | What it does |
| --- | --- |
| `sm_ap_buff_test <1-80\|effect-key\|all> [levels]` | Add effects to your active weapon. Example: `sm_ap_buff_test projectile-count 3` |
| `sm_ap_buff_give <target> <1-80\|effect-key\|all> [levels]` | Add effects to another living RED player's active weapon |
| `sm_ap_projectile_debug on` | Enable projectile diagnostics and print the active weapon state |
| `sm_ap_projectile_debug` | Print the 24 most recent projectile diagnostic lines |
| `sm_ap_projectile_debug off` | Disable projectile diagnostics |
| `sm_ap_unlock_override <on\|off>` | Temporarily allow every class and weapon slot, or restore the run's locks |
| `sm_ap_bundle [credits]` | Pay a test Cash Bundle, defaulting to 200 credits |
| `sm_ap_report wave_cleared [wave]` | Report a cleared wave by hand |
| `sm_ap_report mission_cleared` | Report a cleared mission by hand |
| `sm_ap_report death` | Report a failed wave by hand |

In chat, omit the `sm_` prefix and use `!ap_buff_test projectile-count 3`.
From the TF2 client console, send the command to the server with
`cmd sm_ap_buff_test projectile-count 3`; entering `sm_ap_buff_test` directly
asks the client for a local command and prints `Unknown command`.

The console is not a player, so it is not an admin either: rcon runs as the
server itself and reaches every command above whatever `SRCDS_ADMIN_STEAMIDS`
says.

`sm_ap_report` with no wave number uses the wave that the game is on. Reporting
the same check twice is not a problem: the bridge identifies a check by the
place it belongs to, so the second report does nothing.

## The settings

These are console variables. Set one for the session with
`rcon tf2ap_debug 2`. To keep it across restarts, edit
`cfg/sourcemod/tf2_archipelago.cfg` in the game files. The stack never
overwrites that file once it exists.

| Variable | Default | What it does |
| --- | --- | --- |
| `tf2ap_announce` | `1` | Write the cleared waves and the received items in the chat |
| `tf2ap_chat` | `1` | Write what the rest of the session says in the chat |
| `tf2ap_debug` | `1` | Every bridge call and game event. `0` writes none, `1` writes them to the console and the SourceMod log, `2` writes them to the chat as well. |
| `tf2ap_bridge_url` | `http://127.0.0.1:24680` | Where the bridge is. Loopback only. Do not change it. |
| `tf2ap_start_mission` | empty | The mission the server starts on, as a popfile name. The launcher and the image write it from their settings. |
| `tf2ap_next_mission_delay` | `30` | Seconds between a mission clear and the next mission. `0` leaves the game's own cycle to it. |
| `tf2ap_bot_upgrades_chat` | `0` | Write what the defender bots buy at the upgrade station in the chat. One line per purchase. |
| `tf2ap_bots_wait_for_players` | `1` | Hold the bots unready until every player on RED is ready. With nobody on RED they never ready, so an empty server does not play the run by itself. |
| `tf2ap_bots_backfill` | `1` | Put a bot back on RED when a player leaves between waves. During a wave the bot mod already does it. |

Errors reach the chat whatever `tf2ap_announce` is set to. A failure that
nobody can see gets blamed on the game.
