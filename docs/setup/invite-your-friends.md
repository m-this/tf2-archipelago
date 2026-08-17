# Invite your friends

## Open the port

The stack publishes `SRCDS_PORT`, `27015` by default, on UDP and on TCP.

```sh
SRCDS_PORT=27015
```

Forward that port to the machine on your router. Open it in the firewall of the
machine. UDP carries the game, so a closed UDP port means that nobody can join.

Nothing else needs to be reachable. The randomizer server and the bridge stay
inside the stack.

## The connect command

Your friends open the developer console in Team Fortress 2 and type:

```
connect your.server.address:27015
```

Replace the address with the public address of your machine. Replace the port
if you changed `SRCDS_PORT`.

The console is off by default in Team Fortress 2. Options, then Advanced, then
"Enable developer console". The key that opens it is `` ` `` on a US keyboard.

## The server password

```sh
SRCDS_PW=
```

An empty value lets anybody who knows the address join. Set it to keep the
server to the people you told:

```sh
SRCDS_PW=friends-only
```

Your friends then type this before connecting:

```
password friends-only
connect your.server.address:27015
```

`SRCDS_PW` is not `SRCDS_RCONPW`. `SRCDS_PW` lets a player in. `SRCDS_RCONPW`
lets an admin run commands. Never give out the second one.

## Hide the server from the public list

```sh
SRCDS_TOKEN=0
```

A Game Server Login Token is what puts a dedicated server in the public server
browser. The value `0` means that the server has no token. It runs, your
friends can connect to it with the address, and it does not appear in the
public list.

That is what you want for an evening with friends. Leave it at `0`.

To list the server publicly, get a token from
[steamcommunity.com/dev/managegameservers](https://steamcommunity.com/dev/managegameservers)
and put it here. Then set `SRCDS_PW` as well, or strangers will join a
randomized run in progress.

## What to tell them

Send them these three things:

1. The connect command with your address.
2. The server password, if you set one.
3. [Archipelago for MvM players](../archipelago-for-mvm-players.md), or the
   short version: the classes and the weapons start locked, clearing waves
   unlocks them, everybody shares.

They install nothing. A standard Team Fortress 2 client is the whole
requirement.

## When a player cannot connect

The server answers a query but refuses the join. Almost always this is Steam
authentication against a server that has no Steam session.

A server with no Game Server Login Token (`SRCDS_TOKEN=0`, the default) never
logs in to Steam. Its console says so:

```
Could not establish connection to Steam servers.  (Result = 8)
version : ... insecure (secure mode enabled, disconnected from Steam3)
```

With `SRCDS_LAN=1`, which is the default, that does not matter: LAN mode skips
authentication entirely. If somebody changed it, put it back:

```
rcon sv_lan 1
```

Check the rest in this order:

1. The server answers on the address you are giving people. From the host:
   `docker run --rm --network container:tf2-archipelago-srcds-1 curlimages/curl -s ifconfig.me`
   is not the answer you want. Use the LAN address of the machine, the one
   `ip -4 addr` shows on the network your friends are on.
2. Everyone is on the same network. LAN mode refuses addresses outside the
   private ranges, and a guest network on the same router is a different one.
3. The host machine is awake. It is a laptop, and it sleeps.

Going online later needs a real `SRCDS_TOKEN` from
[steamcommunity.com/dev/managegameservers](https://steamcommunity.com/dev/managegameservers)
and `SRCDS_LAN=0`. One without the other refuses every player.
