# Testing

What still needs a real test, and what each one needs: an Archipelago
server, a second game in the multiworld, one player, or more than one.

- **A wave completes.** Needs: the stack up, one player, a wave in
  progress. Confirms `mvm_wave_complete` fires and the plugin reports the
  check.
- **The wave number reads correctly.** Needs: the stack up, one player, a
  wave in progress. Confirms `mvm_begin_wave` gives the plugin the right
  number.
- **A mission completes.** Needs: the stack up, one player, every wave of a
  mission cleared. Confirms `mvm_mission_complete` fires, and that the
  last-wave fallback is not needed.
- **The wave counter reads correctly.** Needs: the stack up, one player, a
  wave in progress. Confirms `m_nMannVsMachineWaveCount` matches the wave
  the game is actually on.
- **Credits pay out.** Needs: the stack up, one player, a Cash Bundle item
  received mid-wave. Confirms `m_nCurrency` updates for the player on the
  server.
- **Credits reach every player, not just one.** Needs: the stack up, two or
  more players, a Cash Bundle item received. Confirms the payout loop pays
  the whole team.
- **A locked weapon slot stays empty.** Needs: the stack up, one player, a
  visit to the resupply locker or the upgrade station with a locked slot.
  Confirms the plugin removes the weapon rather than warning and leaving it.
- **A locked class moves the player.** Needs: the stack up, one player on a
  locked class, a respawn. Confirms the move happens at the next spawn, not
  mid-wave.
- **Mission name pairing.** Needs: the stack up, no Archipelago server
  required — vanilla MvM is enough. Load each ambiguous popfile by hand
  (`sm_ap_mission <popfile>`) and read the in-game mission name off
  `sm_ap_status`. Confirms `gamedata/missions.go`'s display name matches the
  file, for the two Decoy intermediates, the two Coal Town intermediates,
  the two Mannworks intermediates, and the three advanced groups.
- **A second game in the same multiworld.** Needs: an Archipelago server
  seeded with TF2 and at least one other game, both connected. Confirms an
  item from the other game reaches TF2, and a TF2 check reaches the other
  game.
- **A restart mid-mission.** Needs: the stack up, one player, a wave in
  progress, `make down && make up` run mid-wave. Confirms the bridge's queue
  and the plugin's unlock set survive a restart without losing or repeating
  a check.

## How to trigger a wave alone

Several tests above need a cleared wave, and MvM will not start one until a
human player readies up — a bot cannot do that. One player is enough:

```
rcon sv_cheats 1
rcon mp_disable_respawn_times 1
god
rcon tf_bot_kill all
rcon tf_mvm_tank_kill
```

`god` runs in your own console, not through rcon: rcon runs as the server,
and the server is not a player. `tf_bot_kill all` kills what is alive right
now; the mission keeps sending the rest of the wave, so run it again every
few seconds until the wave ends. This clears the wave the way the game
does, which is the point: `mvm_wave_complete` fires for real.

`tf_mvm_force_victory` and `tf_mvm_jump_to_wave` also exist, and are not
worth using here — they skip the event under test.

Load a mission the run actually holds first, with `rcon sm_ap_mission`. On
a mission outside the run the plugin says so and counts the checks anyway,
which is correct behavior and a confusing thing to test against by mistake.

## What to watch, in order

1. `[AP] Wave 1 cleared.` in chat, when the team clears wave 1.
2. A locked weapon slot stays empty after a resupply locker visit.
3. A locked class moves the player at the next spawn, not mid-wave.
4. Credits from a Cash Bundle actually arrive.
5. The mission clear fires on the last wave, not one wave early. Check
   `wave_drift` on the bridge's health page. See
   [Troubleshooting](troubleshooting.md).

Turn on the loud mode first, so every bridge call and game event lands in
the chat:

```
rcon_password your-SRCDS_RCONPW
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

`mission` has to be the mission, not the map. `wave 0 of 7` has to match the
mission's real length. `unlocks held` has to say held, not NOT FETCHED.
