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
