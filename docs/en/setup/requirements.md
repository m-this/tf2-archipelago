# Requirements

## The machine

| Thing | What you need |
| --- | --- |
| Disk | About 20 GB free. The game server downloads about 14 GB at the first start. |
| Memory | 4 GB for six players. |
| Processor | Two cores. |
| Docker | Docker with the compose plugin. |
| Network | One port that your friends can reach, UDP and TCP. |

The game files stay in a Docker volume named `tf2-archipelago_tf2game`. Keep
that volume. Deleting it downloads 14 GB again.

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
