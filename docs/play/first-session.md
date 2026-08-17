# The first session

## Before anybody joins

Read [What nobody tested](../operate/what-nobody-tested.md). The half of this
project that runs inside the game has never run on a live Team Fortress 2
server. Your first session is partly a diagnostic session. Plan for that and it
is fine. Ignore it and you will not know whether a missing check is a bug or
the game.

Turn on the loud mode before the first wave:

```
rcon_password your-SRCDS_RCONPW
rcon tf2ap_debug 1
```

Type that in the Team Fortress 2 console after you connect. With it on, the
server writes every game event it sees and every call it makes into the chat.
It is noisy. It is also the only way to find out which events this game version
actually sends.

Keep `make logs` open in a terminal on the host at the same time.

## What a player sees

Eight seconds after joining, each player gets this in the chat:

```
[AP] This server runs an Archipelago randomizer.
[AP] The run locks the classes and the weapon slots until it finds them. All players share the unlocks.
[AP] Mission: mvm_decoy. Each wave you clear is a check.
[AP] Unlocked classes: scout, medic
[AP] Unlocked slots: primary
[AP] Type !ap to speak to the multiworld. Examples: !ap hint Scout and !ap missing.
```

The two unlock lines are the state of the run at that moment. A player who
joins late sees what the team has already found.

The delay keeps the welcome out of the map load, where it would scroll past
unread.

## What happens during a wave

- The class menu refuses a class that the run has not unlocked, with a line in
  the chat.
- Locked weapon slots stay empty at every spawn, at the resupply locker and at
  the upgrade station.
- Each wave that the team clears writes `[AP] Wave 3 cleared.` in the chat.
- Each item that the run receives writes `[AP] Unlocked: Class: Pyro` or
  `[AP] The run received 200 credits for 4 player(s).`
- The lines of the other players in the randomized session arrive in the same
  chat.
- Anything that goes wrong is written in red, whatever the other settings are.
  A player will see it before you do.

## Who does what

**The host** connects like everybody else, and additionally holds the console
password. The host runs the admin commands, chooses the map between missions,
and reads the logs on the machine.

**The players** clear waves. There is nothing for them to configure. Their only
commands are `!ap` and `!apchat`, in [Chat commands](chat-commands.md).

## The first check

The moment that proves the whole chain is the first cleared wave. Watch for
three things, in this order:

1. `[AP] Wave 1 cleared.` in the game chat. The plugin saw the wave.
2. `check recorded` in the bridge log, on the host. The check is on disk.
3. `tf2 sent <item> to <somebody>` in the randomizer log. The multiworld has
   it.

If step 1 does not happen, the plugin did not see the wave. Run `rcon
sm_ap_status` and read the `events:` line. See
[Troubleshooting](../operate/troubleshooting.md).

If step 1 happens and step 2 does not, the plugin cannot reach the bridge. The
chat says so in red.

## Ending the evening

`make down` stops the stack and keeps everything. The next `make up` continues
the same run with the same session, the same checks and the same unlocks.

Nothing expires. A run can sit for a week.
