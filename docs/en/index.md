# Mann vs Archipelago

This project turns a Team Fortress 2 Mann vs Machine server into a randomizer.
The classes, the weapon slots and the missions start locked. The team unlocks
them by clearing waves. Everybody on the server shares the same unlocks.

You host it with Docker. The stack runs three containers: a Team Fortress 2
dedicated server, a randomizer server, and a small program that connects the
two. Your friends join with a standard Team Fortress 2 client and install
nothing.

This book is for the host. It assumes that you know Mann vs Machine and that
you never used a randomizer of this kind. It defines every word before it uses
it.

## Read in this order

1. [Archipelago for MvM players](archipelago-for-mvm-players.md) gives you the
   vocabulary. Read it first.
2. [What the randomizer changes](what-the-randomizer-changes.md) says what is
   different from a normal MvM server.
3. [Requirements](setup/requirements.md) and [Install](setup/install.md) get
   the stack running.
4. [The shape of the run](setup/shape-of-the-run.md) sets the length and the
   difficulty of an evening.
5. [Invite your friends](setup/invite-your-friends.md) opens the server.
6. [Testing](operate/testing.md) lists what still needs a real test, and
   what each test needs. Read it before the first session.

## The short version

```sh
cp deploy/.env.example .env   # then set SRCDS_RCONPW
make up
make logs
```

The first start downloads about 14 GB of game files.
