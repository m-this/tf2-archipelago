# Requirements

There are two ways to run the server. **Windows** is the easiest: one exe, no
Docker. **Docker** works on any operating system. Both run the same software.

## The machine

| Thing | What you need |
| --- | --- |
| Disk | About 20 GB free. The game server downloads about 14 GB at the first start. |
| Memory | 4 GB for six players. |
| Processor | Two cores. |
| Network | One port that your friends can reach, UDP and TCP. |

## Windows (recommended)

| Thing | What you need |
| --- | --- |
| Windows | 10 or 11, 64-bit. |
| The launcher | `tf2ap.exe` from the [latest release](https://github.com/m-this/tf2-archipelago/releases/latest). |
| Archipelago | The official [Archipelago app](https://github.com/ArchipelagoMW/Archipelago/releases), to generate the seed. |

No Docker, no clone, no compiler. See
[Install on Windows](install-windows.md).

## Docker

| Thing | What you need |
| --- | --- |
| Docker | Docker with the compose plugin. |

See [Install with Docker](install.md).

## The network

The stack publishes one port, `27015` by default, on UDP and on TCP. Set it
with `SRCDS_PORT` in `.env`.

- UDP carries the game. Without it nobody can join.
- TCP carries the remote console of the game server. You need it to run the
  admin commands in [Troubleshooting](../operate/troubleshooting.md).

The stack publishes nothing else. The bridge opens its own connection out to
the room on `archipelago.gg` and listens on loopback only. See
[Install](install.md) for the services.

The machine needs to reach `archipelago.gg` on the port of the room. A firewall
that filters outgoing traffic has to let that port through.

If the machine is behind a router, forward that port to it. If the machine has
a firewall, open that port.

## What you do not need

- No Steam account for the server. `SRCDS_TOKEN=0` runs the server without one
  and keeps it out of the public server list. See
  [Invite your friends](invite-your-friends.md).
- No Team Fortress 2 installation on the host. The container downloads its own.
- No account on `archipelago.gg`. The website hosts a session for anybody who
  uploads one. See [Create the session](create-the-session.md).

## Security note

The game server is a large C++ process that reads network traffic from anybody
who knows the address. Run it on a machine where that is acceptable. If the
same machine runs something that matters to you, decide that on purpose rather
than by default.
