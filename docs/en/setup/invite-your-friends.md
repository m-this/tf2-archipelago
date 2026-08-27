# Invite your friends

## Where the players come from

A server is meant to be joined from somewhere else, so it takes connections on
its game port unless you say otherwise. Pick one of three with `SRCDS_REACH` in
`.env`:

```sh
SRCDS_REACH=port     # straight at the game port, forwarded on your router. The default.
SRCDS_REACH=steam    # over Steam's relay, with no port to open
SRCDS_REACH=lan      # this machine and the local network
```

`lan` is the whole answer for people playing in the same house, and it needs
nothing else. The other two reach the internet, and both need a login token.

A server with no token stays on the local network whatever the reach says, and
prints a line in the log saying so. It has no choice: without a token it never
gets a Steam session, and a server with no session refuses every player who
tries to join, the ones in the same house included.

> **`steam` is not finished.** No run took the relay all the way to a Team
> Fortress 2 client that joined. The launcher offers it anyway, under
> **Networking**. `port` is the one people play on.

## The login token

Both `steam` and `port` log the server in to Steam. Without a token it never
gets a Steam session, and every client that tries to join is refused with no
useful message. Get one at
[steamcommunity.com/dev/managegameservers](https://steamcommunity.com/dev/managegameservers)
for app id 440, once, and put it in Settings or in `.env`:

```sh
SRCDS_TOKEN=YOURTOKENHERE
```

The token is not a password anybody types. It identifies the server to Steam.
`SRCDS_TOKEN=0` means none, which is the right answer only for `lan`, and is
what the server has until you give it one.

## Over Steam, with no port to open

With `SRCDS_REACH=steam` the server asks Valve for an address on the Steam
Datagram Relay and prints it as soon as it has one:

```
FakeIP allocation succeeded: 169.254.13.42:20232, 20233
```

The launcher lifts that out of the log and puts it in a box above the log with
a **Copy** button. Your friends type the first address:

```
connect 169.254.13.42:20232
```

It works from anywhere. You forward nothing and you open nothing in the
firewall. The traffic goes through Valve's relays, so the people joining never
see your own address.

Two things to know. The address is a new one every time the server starts.
Send the line from this run rather than one you wrote down last week. And
nobody can add the server to their favourites, because there is no stable
address to favourite.

## Over a forwarded port

With `SRCDS_REACH=port` the players connect straight to you:

```sh
SRCDS_PORT=27015
```

Forward that port to the machine on your router, on UDP and on TCP, and open it
in the machine's firewall. UDP carries the game, so a closed UDP port means
nobody can join. Then your friends type your public address:

```
connect your.public.address:27015
```

Nothing else has to be reachable. The randomizer server and the bridge stay on
loopback.

## The developer console

The console is off by default in Team Fortress 2. Open Options, then Advanced,
then tick "Enable developer console". The key that opens it is `` ` `` on a US
keyboard.

## The server password

```sh
SRCDS_PW=
```

An empty value lets anybody who has the address join. Set it to keep the server
to the people you told:

```sh
SRCDS_PW=friends-only
```

Your friends then type this before connecting:

```
password friends-only
connect ...
```

`SRCDS_PW` is not `SRCDS_RCONPW`. `SRCDS_PW` lets a player in. `SRCDS_RCONPW`
lets an admin run commands. Never give out the second one.

## Staying off the public list

A server with `SRCDS_TOKEN=0` never logs in and never appears in the public
server browser. With a real token it can, which is what you want to avoid on a
randomized run in progress. Set `SRCDS_PW` as well, and strangers who find the
server still cannot get in.

## What to tell them

Send them these three things:

1. The connect line for whichever reach you picked. Over Steam, the one from
   this run's log.
2. The server password, if you set one.
3. [Archipelago for MvM players](../archipelago-for-mvm-players.md), or the
   short version: the classes and the weapons start locked, clearing waves
   unlocks them, everybody shares.

They install nothing. A standard Team Fortress 2 client is the whole
requirement.

## When a player cannot connect

The server answers a query but refuses the join. Almost always this is Steam
authentication against a server that has no Steam session: `SRCDS_REACH` is
`steam` or `port` and `SRCDS_TOKEN` is still `0`. The console says so:

```
Could not establish connection to Steam servers.  (Result = 8)
version : ... insecure (secure mode enabled, disconnected from Steam3)
```

Put in a real token, or go back to `SRCDS_REACH=lan`, where authentication is
skipped entirely and no token is needed.

Then check the rest in this order:

1. **On `lan`**, the player sees `LAN servers are restricted to local clients
   (class C)`. That is the server saying the address they came from is not in
   the same class C, the same first three numbers, as its own: a guest network
   on the same router is a different one, and so is a VPN, and so is the
   bridge network a server in a container sits on. Give out the address
   `ip -4 addr` shows for the network everyone is on, or leave `lan` behind:
   `port` with a token has no such rule.
2. **On `steam`**, the address is the one from this run. It changes on every
   start, and an old one goes nowhere.
3. **On `port`**, the port is forwarded on UDP as well as TCP. A TCP-only rule
   answers the query and drops the game.
4. The host machine is awake. It is a laptop, and it sleeps.
