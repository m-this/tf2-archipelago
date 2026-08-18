# Mann vs Archipelago

This project turns a Team Fortress 2 Mann vs Machine server into a randomizer.
The classes, the weapon slots and the missions start locked. The team
clears waves to unlock them. Everybody on the server shares the same
unlocks.

You host it two ways. On Windows, one exe installs and runs the whole thing.
Anywhere with Docker, a two-container stack does the same: a Team Fortress 2
dedicated server, and a small program that connects it to the session. The
session itself runs on `archipelago.gg`, and the stack hosts it for you if you
prefer. Your friends join with a standard Team Fortress 2 client
and install nothing.

The server also fills the RED team with bots that play, so two people can win a
run that Valve balanced for six.

This book is for the host. It assumes that you know Mann vs Machine and that
you never used a randomizer of this kind. It defines every word before it uses
it.

## Read in this order

1. [Archipelago for MvM players](archipelago-for-mvm-players.md) gives you the
   vocabulary. Read it first.
2. [What the randomizer changes](what-the-randomizer-changes.md) says what is
   different from a normal MvM server.
3. [Requirements](setup/requirements.md) says what the machine needs.
4. [The shape of the run](setup/shape-of-the-run.md) sets the length and the
   difficulty of an evening.
5. [Create the session](setup/create-the-session.md) makes the run and puts it
   on `archipelago.gg`.
6. [Install on Windows](setup/install-windows.md) or
   [Install with Docker](setup/install.md) gets the server running.
7. [Invite your friends](setup/invite-your-friends.md) opens the server.
   [The bots on your team](play/defender-bots.md) says who fills the empty
   slots.

## The short version

On Windows, download `tf2ap.exe` from the
[latest release](https://github.com/m-this/tf2-archipelago/releases/latest) and
run it. It asks for the room address and installs the rest.

With Docker:

```sh
cp deploy/.env.example .env   # then set SRCDS_RCONPW
make seed                     # upload the file to archipelago.gg, open a room
                              # then set AP_HOST and AP_PORT in .env
make up
make logs
```

The first start downloads about 14 GB of game files.
