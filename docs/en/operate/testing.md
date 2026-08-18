# Testing

`make integration` covers seed generation, the Archipelago server and the
bridge, on every run. The plugin ran on a real server on 2026-08-17, over
rcon, with no game client. The three MvM events exist, the mission and
its length read correctly, and a check reached the multiworld.

That run left four things unconfirmed: a wave clearing, credits arriving,
a weapon slot enforced, and a class moved. This page is the step-by-step
to confirm each one, and what to watch while you test.

## 1. Connect to your server

Start the stack, then join with a stock TF2 client, like a player.
See [Invite your friends](../setup/invite-your-friends.md) for the connect
command and the join password.

```sh
make up
```

```
connect your.server.address:27015
```

## 2. Get admin

Two separate things count as admin here, and you likely want both.

**Rcon**, for server commands (`sv_cheats`, `tf_bot_kill`, and the checks
below):

```sh
# in .env
SRCDS_RCONPW=your-password
```

```
rcon_password your-SRCDS_RCONPW
rcon sm_ap_status
```

If `sm_ap_status` answers, rcon works.

The same commands run from the host, with no game client. Run every `rcon`
line on this page either way:

```sh
make rcon CMD="sm_ap_status"
```

The server reads `SRCDS_RCONPW` once, at boot. An edit to `.env` after that
changes nothing until `make restart`. Until then the file and the server
hold two different passwords. `make rcon` reports the refusal instead of an
empty answer.

**Chat commands** (`!mission`, `sm_ap_mission` in chat), for the
`SRCDS_ADMIN_STEAMIDS` allowlist:

```sh
# in .env
SRCDS_ADMIN_STEAMIDS=STEAM_0:1:XXXXXXX
```

Comma-separate more than one value. Change it, then restart the stack
(`make down && make up`). Confirm the change from the game chat:

```
!ap
```

A non-admin gets nothing back from `!ap`; an admin does.

## 3. Turn on the loud mode

Do this before every test below. It puts every bridge call and game event
in the chat. Watch the chat during each test.

```
rcon tf2ap_debug 1
rcon sm_ap_status
```

A healthy `sm_ap_status` answer looks like this:

```
[AP] version 0.1.0, mvm yes, mission mvm_decoy_intermediate, wave 0 of 7
[AP] events: begin_wave yes, wave_complete yes, mission_complete yes
[AP] unlocks held at sequence 5, 0 objective(s) waiting to be sent
[AP] classes: soldier, heavy
[AP] slots: primary
```

`mission` has to be the mission, not the map. `wave 0 of 7` has to match
the mission's real length. `unlocks held` has to say held, not NOT
FETCHED.

## 4. Load a mission the run actually holds

```
rcon sm_ap_mission <popfile>
```

Pick a popfile from `/missions` (or `sm_ap_status`'s `mission` field) that
is in this run's pool. A popfile outside the pool still loads: the plugin
says so, and counts the checks anyway. That makes it a confusing thing to
test against by accident.

Do this before every test below, even on a server that just started.
`SRCDS_STARTMAP` names a map, not a mission. A map loads that map's own
default mission, and the run does not hold it.

`mvm_decoy` starts Doe's Drill, eight waves. The run's Decoy mission is
`mvm_decoy_intermediate`, Doe's Doom, seven waves. `gamedata` holds both,
so nothing fails loudly. The waves you clear count as Doe's Drill's
checks, the run never asked for them, and the run does not move.

`sm_ap_status` tells the two apart. Its `mission` field has to read
`mvm_decoy_intermediate`, not `mvm_decoy`. The plugin also writes this to
the error log at every map load:

```
The run did not unlock mvm_decoy. Its checks still count.
```

That line means the server is on the wrong mission. Do not ignore it.

## 5. Clear a wave alone

MvM will not start a wave until a human readies up. Ready up, then run
this command block. It ends the wave without a real playthrough:

```
rcon sv_cheats 1
rcon mp_disable_respawn_times 1
god
rcon tf_bot_kill all
rcon tf_mvm_tank_kill
```

`god` runs in your own console, not through rcon: rcon runs as the
server, and the server is not a player. `tf_bot_kill all` kills what is
alive right now, but the mission keeps sending the rest of the wave. Run
it again every few seconds until the wave ends. This clears the wave the
way the game does, so `mvm_wave_complete` fires from a genuine clear, not
a shortcut.

`tf_mvm_force_victory` and `tf_mvm_jump_to_wave` also exist. Do not use
them here: they skip the event under test.

### Play a wave rather than end it

To play a wave alone, weaken the robots. `tf_mvm_skill` is a cheat cvar,
so turn `sv_cheats` on first:

```sh
make rcon CMD="sv_cheats 1"
make rcon CMD="tf_mvm_skill 1"
```

`tf_mvm_skill` runs 1 to 5 and defaults to 3. At 1, one player can win a
wave. A map reload drops it, so set it again after `sm_ap_mission`.

Credits are the other half, because an MvM defender is its upgrades. Run
this in the game client's own console, not through rcon: it acts on a
player, and rcon is the server.

```
currency_give 30000
```

The credits spend at the upgrade station like any other. The command also
writes `m_nCurrency`, which test 7 below reads, so a run that used it
cannot confirm a Cash Bundle payout.

`god`, also in the client's console, is the shortest way to reach a late
wave with the events intact.

### Bots on the RED team

The server ships them. RED is filled to six when a wave begins, and the
bots pick classes, fight, buy their own upgrades at the station and ready
themselves. Nothing has to be typed: `SRCDS_BOTS=1` is the default, and
`SRCDS_BOT_TEAM_SIZE` sets the number. `SRCDS_BOTS=0` leaves them to an
admin's `!addbots`.

So one tester alone is a full team. Press F4 and the wave starts.

Count them with the bridge's metrics, not with `status`. `status` lists
every fake client, robots included, so it reads about 25 for a team of
five mid-wave. A2S counts only the defenders:

```sh
curl -s 127.0.0.1:24681/metrics | grep tf2ap_game_
```

`tf2ap_game_bots 5` with `tf2ap_game_players_human 1` is a full RED team.

Leave `tf_bot_difficulty` alone. It is one setting for every bot on the
server, robots included. Raise it for the team and the robots get the same
raise. Robots already alive keep the value they spawned under.

To take the bots out mid-session, without restarting:

```sh
make rcon CMD="sm_redbots_manager_mode 0"
make rcon CMD="sm_purgebots"
```

The convar first: `sm_purgebots` on its own removes them and the next
wave brings them straight back. `sm_addbots` puts them back by hand, and
the next map load returns to whatever `SRCDS_BOTS` says, because
server.cfg sets the convar again. `sm_addbots` and `sm_purgebots` are
admin commands, so the Steam id has to be in `SRCDS_ADMIN_STEAMIDS` or
the call has to come over rcon.

**Do not use `tf_bot_add` for this.** It raises `tf_bot_quota`, and in MvM
a quota above 0 spawns bots and never stops: the count climbs on its own
and waves never end. The mod adds with `noquota` and does not have that
problem. If you have used `tf_bot_add` by hand, `tf_bot_quota 0` before
`tf_bot_kick all`, or the server replaces every bot you kill, for good.

**Confirms:** `mvm_wave_complete` fires and the plugin reports the check.
`mvm_begin_wave` gave the plugin the right wave number beforehand.
Compare `sm_ap_status`'s `wave N of M` before and after. Watch for:

```
[AP] Wave 1 cleared.
```

in chat.

## 6. Clear a whole mission

Repeat step 5 for every wave of the loaded mission.

**Confirms:** `mvm_mission_complete` fires on the last wave, not one wave
early or late. It also confirms the last-wave fallback stays unused.
Check `wave_drift` on the bridge's health page. See
[Troubleshooting](troubleshooting.md) to confirm the game's wave count
agrees with `gamedata`.

## 7. Confirm credits pay out

Needs a Cash Bundle item received mid-wave. Any item works. If the queue
is empty, check `sm_ap_status`'s "objective(s) waiting" line, or send one
from another connected game.

**Confirms:** `m_nCurrency` updates for the player on the server. With two
or more players connected, confirm the payout loop pays the whole team,
not just one player.

## 8. Confirm a locked weapon slot stays empty

Needs a locked slot. Visit the resupply locker, or the upgrade station.

**Confirms:** the plugin removes the weapon, rather than warning and
leaving it in place. If a player holds the removed weapon, they must end
up with something else in hand. Watch for that.

## 9. Confirm a locked class moves the player

Join a locked class, then respawn.

**Confirms:** the move happens at the next spawn, not mid-wave. Forcing a
respawn mid-wave costs the player the rest of the wave for free.

## 10. Confirm mission name pairing

No Archipelago server is necessary. Vanilla MvM is enough. Load each
ambiguous popfile by hand, and read the in-game mission name off
`sm_ap_status`:

```
rcon sm_ap_mission <popfile>
rcon sm_ap_status
```

**Confirms:** `gamedata/missions.go`'s display name matches the file.
Check the two Decoy intermediates, the two Coal Town intermediates, the
two Mannworks intermediates, and the three advanced groups.

## 11. Confirm a second game in the same multiworld

Needs an Archipelago server seeded with TF2 and at least one other game,
both connected.

**Confirms:** an item from the other game reaches TF2, and a TF2 check
reaches the other game.

## 12. Confirm a restart mid-mission

With a wave in progress:

```sh
make down && make up
```

**Confirms:** the bridge's queue and the plugin's unlock set survive the
restart without losing or repeating a check.
