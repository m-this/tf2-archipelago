# Mann vs Archipelago

This project turns a Team Fortress 2 Mann vs Machine server into a randomizer.
The classes, the weapon slots and the missions start locked. The team
clears waves to unlock them. Everybody on the server shares the same
unlocks.

You host it with Docker. The stack runs two containers: a Team Fortress 2
dedicated server, and a small program that connects it to the randomized
session. The session itself runs on `archipelago.gg`, and the stack hosts it
for you if you prefer. Your friends join with a standard Team Fortress 2 client
and install nothing.

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
6. [Install](setup/install.md) gets the stack running.
7. [Invite your friends](setup/invite-your-friends.md) opens the server.
8. [Testing](operate/testing.md) gives the step-by-step to confirm each
   behavior on a live server. Read it before the first session.

## The short version

```sh
cp deploy/.env.example .env   # then set SRCDS_RCONPW
make seed                     # upload the file to archipelago.gg, open a room
                              # then set AP_HOST and AP_PORT in .env
make up
make logs
```

The first start downloads about 14 GB of game files.
