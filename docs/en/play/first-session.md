# The first session

## Before anybody joins

The plugin already writes every game event it sees and every call it makes to
the console and the SourceMod log, so the record of the evening is on disk
whatever you do. It has a louder mode that puts the same lines in the chat,
which is the fastest way to watch what the server does with a wave as it
happens:

```
rcon_password your-SRCDS_RCONPW
rcon tf2ap_debug 2
```

Type that in the Team Fortress 2 console after you connect. Keep the launcher's
log, or `make logs`, in view at the same time.

## What a player sees

Eight seconds after joining, each player gets this in the chat:

```
[AP] This server runs an Archipelago randomizer.
[AP] The run locks the classes and the weapon slots until it finds them. All players share the unlocks.
[AP] Mission: mvm_decoy. Each wave you clear is a check.
[AP] Unlocked classes: scout, medic
[AP] Unlocked slots: primary
[AP] Type !ap to speak to the multiworld. Examples: !ap hint Class: Scout and !ap missing.
```

The two unlock lines are the state of the run at that moment. A player who
joins late sees what the team has already found.

The delay keeps the welcome out of the map load, where it would scroll past
unread.

## What happens during a wave

- Bots fill the RED team to six when the wave begins, and stay filled for the
  rest of it. See [The bots on your team](defender-bots.md).
- The class menu refuses a class that the run has not unlocked, with a line in
  the chat.
- Locked weapon slots stay empty at every spawn, at the resupply locker and at
  the upgrade station.
- Entering the upgrade station opens a numbered summary of any Archipelago
  buffs on the current loadout. `sm_ap_buffs` opens it again.
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
