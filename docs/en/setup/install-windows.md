# Install on Windows

This is the easiest way to run a Mann vs Archipelago server. One file, no
Docker, no clone, no compiler.

Download `tf2ap.exe` from the
[latest release](https://github.com/m-this/tf2-archipelago/releases/latest) and
run it. It is a single file that does everything.

## What tf2ap.exe does

1. It installs SteamCMD, the TF2 dedicated server, SourceMod and the plugin the
   first time you run it. The game server is about 14 GB; this is the long
   part, and it only happens once.
2. It asks for the configuration an evening needs: the address of your
   Archipelago room, an RCON password, and the shape of the run.
3. It starts the game server and the bridge in one process. Press Ctrl-C to
   stop both.

Every later run takes seconds. The 14 GB of game files stay on your disk, and
the configuration is saved in `%APPDATA%\tf2ap\config.json`.

## What you need

| Thing | What you need |
| --- | --- |
| Windows | 10 or 11, 64-bit. |
| Disk | About 20 GB free. The game server is about 14 GB. |
| Memory | 4 GB for six players. |
| Processor | Two cores. |
| Network | One port that your friends can reach, UDP and TCP. The default is 27015. |

You do not need Docker. You do not need the Steam client. You do not need a
Steam account for the server.

## The Archipelago session

The launcher runs the TF2 server. The multiworld session is separate, because
Mann vs Machine is not one of the games that ship with Archipelago and the
generator is Python.

1. Install the official
   [Archipelago](https://github.com/ArchipelagoMW/Archipelago/releases) app on
   Windows. It bundles its own Python.
2. Download `tf2_mvm.apworld` from the same release as `tf2ap.exe`.
3. Drop `tf2_mvm.apworld` into the `custom_worlds/` folder of your Archipelago
   install.
4. Open the Archipelago app, generate a seed, and upload it at
   [archipelago.gg/uploads](https://archipelago.gg/uploads) to create a room.

The room page gives an address like `archipelago.gg:12345`. Write the two
halves into the launcher when it asks.

See [Create the session](create-the-session.md) for the detail on each step,
including how to host the session on your own machine instead.

## The first start

Double-click `tf2ap.exe`, or run it from a terminal:

```
tf2ap.exe
```

A terminal window opens. The first start goes like this:

1. The launcher installs SteamCMD. Seconds.
2. The launcher installs the TF2 dedicated server. This is the 14 GB download.
   How long depends on your connection.
3. The launcher installs SourceMod and the plugin. Seconds.
4. The launcher asks for the configuration. Answer the prompts. Defaults from
   your last run are in brackets; press Enter to keep one.
5. The launcher starts the server. The log says
   `connected to archipelago slot=tf2`, and the room page says
   `tf2 (Team #1) playing Team Fortress 2 Mann vs Machine has joined`.

Your friends connect with the developer console:

```
connect your.server.address:27015
```

See [Invite your friends](invite-your-friends.md) for the rest.

## The commands

| Command | What it does |
| --- | --- |
| `tf2ap.exe` | Install if needed, ask for missing values, start the server |
| `tf2ap.exe -configure` | Edit every setting, then exit |
| `tf2ap.exe -install` | Install or repair the server, then exit |
| `tf2ap.exe -status` | Show the configuration and install state |
| `tf2ap.exe -version` | Print the version and the pinned tool versions |

## Where it keeps things

| Path | Holds |
| --- | --- |
| `%USERPROFILE%\tf2-archipelago\` | The game files, SourceMod and SteamCMD. Keep this. |
| `%APPDATA%\tf2ap\config.json` | Your saved configuration. |
| `%USERPROFILE%\tf2-archipelago\bridge-state\` | The checks and unlocks of the current run. |

## When something goes wrong

The launcher logs to the terminal. Scroll up to read what the game server and
the bridge said.

The most common issue is the port. Forward `27015` (or the port you set) to
your machine on your router, and open it in the Windows firewall. UDP carries
the game; a closed UDP port means nobody can join.

See [Troubleshooting](../operate/troubleshooting.md) for the rest.

## The other way to run it

If you already have a Linux machine with Docker, or you prefer the container
stack, see [Install](install.md). The two run the same software: the launcher
is the same bridge and the same plugin, just packaged for Windows without
Docker.
