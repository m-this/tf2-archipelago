# Install on Linux

One file. No Docker, no clone, no compiler. This is the same program as the
Windows launcher. It draws the same interface in the terminal rather than in a
window. You get the log, the run's missions, the Bot Switcher, the rcon line
and the same eight tabs of settings. Nothing to install for it, and it works over SSH.

Download `tf2ap-linux-amd64` from the
[latest release](https://github.com/m-this/tf2-archipelago/releases/latest),
make it executable, and run it.

```sh
curl -fsSLO https://github.com/m-this/tf2-archipelago/releases/latest/download/tf2ap-linux-amd64
chmod +x tf2ap-linux-amd64
./tf2ap-linux-amd64
```

## What happens

It asks for your Archipelago room address. Then it installs everything:
SteamCMD, the TF2 dedicated server, Metamod:Source, SourceMod, the plugin, and
the bots that fill your team. The game server is about 14 GB, and the first
start takes a while because of it. Every later start takes seconds.

Then the interface opens on it.

| Key | What it does |
| --- | --- |
| `s` | Start the server, or stop it |
| `r` | Restart it |
| `j` | Start Team Fortress 2 and join |
| `c` | Copy the join line, for sending to a friend |
| `,` | The settings, in the eight tabs the window uses |
| `tab` | Between the run, the Bot Switcher and the log |
| `i` | The rcon line. `esc` gives the keys back |
| `p` | On the run's tab, load the mission the cursor is on |
| `a` | On the Bot Switcher, hand the bot team to the running server |
| `q` | Quit, which stops the server |

`tf2ap-linux-amd64 -console` prints the log and nothing else, which is what a
service or a `screen` session wants: an interface that draws over the whole
screen writes nothing useful into a file. Stop that one with Ctrl+C.

## What you need

| Thing | What you need |
| --- | --- |
| Linux | 64-bit, glibc |
| Disk | About 20 GB free |
| Memory | 4 GB for six players |
| Processor | Two cores |
| Network | Nothing, for friends on the same network. One forwarded port only if you pick that route. |

The TF2 dedicated server is a 32-bit program. On a 64-bit distribution it
needs the 32-bit C, C++ and curl libraries. On Debian and Ubuntu:

```sh
sudo dpkg --add-architecture i386
sudo apt update
sudo apt install lib32gcc-s1 lib32stdc++6 libcurl3t64-gnutls:i386
```

Fedora calls the C library `glibc.i686`, Arch calls it `lib32-glibc`. SteamCMD
and the server name a library plainly when they cannot find it.

No Docker, no Steam client, no Steam account for the server itself.

## The Archipelago session

The launcher runs the TF2 server. The multiworld session is separate. Mann vs
Machine is not one of the games that ship with Archipelago, so the seed
generator stays with the official app.

1. Install the official
   [Archipelago](https://github.com/ArchipelagoMW/Archipelago/releases) app.
   The launcher looks for it in `~/Archipelago`, `/opt/Archipelago` and `/ap`.
2. Run `./tf2ap-linux-amd64 -yaml tf2.yaml` to write the player file, and drop
   it into the app's `Players/` folder.
3. Generate there, then upload the archive at
   [archipelago.gg/uploads](https://archipelago.gg/uploads) to open a room.
4. Give the launcher the room address.

See [Create the session](create-the-session.md) for the full detail.

## Settings

`./tf2ap-linux-amd64 -configure` walks every setting in the terminal.
Everything also reads an environment variable, under the name
`deploy/.env.example` uses:

```sh
AP_ROOM=archipelago.gg:12345 SRCDS_BOT_TEAM_SIZE=4 ./tf2ap-linux-amd64
```

`./tf2ap-linux-amd64 -env` prints every name it reads and marks the ones your
environment already sets.

## Commands

| Command | What it does |
| --- | --- |
| `tf2ap-linux-amd64` | Install whatever the server needs, then run it |
| `tf2ap-linux-amd64 -room <host:port>` | Set the room address, then run |
| `tf2ap-linux-amd64 -configure` | Edit every setting, then exit |
| `tf2ap-linux-amd64 -install` | Install or repair the server, then exit |
| `tf2ap-linux-amd64 -status` | Show the settings and the install state |
| `tf2ap-linux-amd64 -yaml <path>` | Write the Archipelago player file, then exit |
| `tf2ap-linux-amd64 -env` | List the environment variables, then exit |
| `tf2ap-linux-amd64 -version` | Print the version and the pinned tool versions |

## Where it keeps things

| Path | Holds |
| --- | --- |
| `~/tf2-archipelago/` | The game files, SourceMod and SteamCMD |
| `~/tf2-archipelago/tf2.yaml` | The player file |
| `~/tf2-archipelago/bridge-state/` | The checks and unlocks of the run |
| `~/.config/tf2ap/config.json` | Your saved settings |

`TF2AP_INSTALL_ROOT` moves the first three, for a second disk.

## The other way to run it

[Install with Docker](install.md) runs the same software as two containers.
Take that one to keep the server off your own account, or to run it on a
machine that already has a Docker stack. This page is the shorter road.

See [Invite your friends](invite-your-friends.md) to open the server up, and
[Troubleshooting](../operate/troubleshooting.md) when something looks wrong.
