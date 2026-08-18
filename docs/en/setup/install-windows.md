# Install on Windows

This is the easiest way to run a Mann vs Archipelago server. One file, no
Docker, no clone, no compiler.

Download `tf2ap.exe` from the
[latest release](https://github.com/m-this/tf2-archipelago/releases/latest) and
run it.

## What tf2ap.exe does

Double-click it and a window opens. The window holds everything an evening
needs:

- **Start**, **Stop** and **Restart**, so the server goes up and down without a
  terminal.
- A log, where the game server, the bridge and the installer write. This is
  what you read when something looks wrong.
- **Settings**, for the room address, the map, the bots and the shape of the
  run.
- An **rcon** box at the bottom. It sends a command to the server and prints
  the answer in the log. `sm_ap_status` is the one to know.

The first Start installs SteamCMD, the TF2 dedicated server, Metamod:Source,
SourceMod, the plugin and the defender bots. The game server is about 14 GB.
This is the long part, and it happens once. The log tracks it.

Closing the window stops the server. Every later run takes seconds, and the
launcher saves your answers in `%APPDATA%\tf2ap\config.json`.

## What you need

| Thing | What you need |
| --- | --- |
| Windows | 10 or 11, 64-bit. |
| Disk | About 20 GB free. The game server is about 14 GB. |
| Memory | 4 GB for six players. |
| Processor | Two cores. |
| Network | One port your friends can reach, UDP and TCP. The default is 27015. |

You do not need Docker. You do not need the Steam client. You do not need a
Steam account for the server.

## The Archipelago session

The launcher runs the TF2 server. The multiworld session is separate. Mann vs
Machine is not one of the games that ship with Archipelago, and the generator
is Python.

1. Install the official
   [Archipelago](https://github.com/ArchipelagoMW/Archipelago/releases) app. It
   carries its own Python.
2. Download `tf2_mvm.apworld` from the same release as `tf2ap.exe`.
3. Put `tf2_mvm.apworld` in the `custom_worlds/` folder of your Archipelago
   install.
4. Copy `%USERPROFILE%\tf2-archipelago\tf2.yaml` into the `Players/` folder of
   that same install. The launcher writes this file from your run settings on
   the first start. `tf2ap.exe -yaml <path>` writes it anywhere you want.
5. Open the Archipelago app and generate. Upload the archive at
   [archipelago.gg/uploads](https://archipelago.gg/uploads) to open a room.

The room page shows an address such as `archipelago.gg:12345`. Give the two
halves to the launcher when it asks.

See [Create the session](create-the-session.md) for the detail of each step.
It also shows how to host the session on your own machine.

## The first start

Double-click `tf2ap.exe`, or run it from a terminal:

```
tf2ap.exe
```

A terminal window opens. The first start goes like this:

1. The launcher asks for the room address. Paste the line from your room page,
   such as `archipelago.gg:12345`, and press Enter. This comes first, so a
   typo costs you a second rather than a download.
2. The launcher installs SteamCMD. Seconds.
3. The launcher installs the TF2 dedicated server. This is the 14 GB download.
   The time depends on your connection.
4. The launcher starts the server. The log says
   `connected to archipelago slot=tf2`, and the room page says
   `tf2 (Team #1) playing Team Fortress 2 Mann vs Machine has joined`.

Your friends connect from the developer console:

```
connect your.server.address:27015
```

See [Invite your friends](invite-your-friends.md) for the rest.

## The bots on your team

The server fills the RED team with bots that play. Team Fortress 2 balances
every wave for six defenders, so two people can win a run. The bots pick
classes, fight, buy their own upgrades and ready themselves.

`tf2ap.exe -configure` turns them off, or fills RED to fewer than six for a
harder run. [The bots on your team](../play/defender-bots.md) says what they
do and what they are bad at.

## The commands

The window covers an evening. These are for a script, or for a setting the
window does not show. Run them from a terminal: the exe attaches to it.

| Command | What it does |
| --- | --- |
| `tf2ap.exe` | Open the window |
| `tf2ap.exe -room <host:port>` | Set the room address, then open the window |
| `tf2ap.exe -console` | Run in the terminal, with no window |
| `tf2ap.exe -configure` | Edit every setting in the terminal, then exit |
| `tf2ap.exe -install` | Install or repair the server, then exit |
| `tf2ap.exe -status` | Show the settings and the install state |
| `tf2ap.exe -yaml <path>` | Write the Archipelago player file, then exit |
| `tf2ap.exe -env` | List the environment variables, then exit |
| `tf2ap.exe -version` | Print the version and the pinned tool versions |

## Settings from the environment

Every setting also reads an environment variable, under the name
`deploy/.env.example` uses. A variable wins over the saved file for that run,
and the launcher never writes it back. Start a server with no questions from a
shortcut or a `.bat` file:

```bat
set AP_ROOM=archipelago.gg:12345
set SRCDS_BOT_TEAM_SIZE=4
tf2ap.exe
```

`AP_ROOM` carries the whole address. `AP_HOST`, `AP_PORT` and `AP_TLS` set the
three parts on their own, for a `.env` file shared with the Docker stack.

`tf2ap.exe -env` prints every name it reads and marks the ones your environment
already sets.

## Where it keeps things

| Path | Holds |
| --- | --- |
| `%USERPROFILE%\tf2-archipelago\` | The game files, SourceMod and SteamCMD. Keep this. |
| `%USERPROFILE%\tf2-archipelago\tf2.yaml` | The player file for the Archipelago app. |
| `%USERPROFILE%\tf2-archipelago\bridge-state\` | The checks and unlocks of the run. |
| `%APPDATA%\tf2ap\config.json` | Your saved settings. |

`TF2AP_INSTALL_ROOT` moves the first three, for a second disk.

## When something goes wrong

The launcher writes its log to the terminal. Scroll up to read what the game
server and the bridge said.

The port is the usual cause. Forward 27015, or the port you set, to your
machine on your router, and open it in the Windows firewall. UDP carries the
game, so a closed UDP port stops everybody from joining.

See [Troubleshooting](../operate/troubleshooting.md) for the rest.

## The other way to run it

If you have a Linux machine with Docker, see [Install](install.md). The two run
the same software. The launcher holds the same bridge and the same plugin,
packaged for Windows without Docker.
