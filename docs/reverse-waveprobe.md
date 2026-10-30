# Reverse MvM wave probe

The daily and local stability sweeps use `-reverse-objectives` to attempt every
playable reverse mission: 122 wave/mode cases across twelve missions, with
Bot Surge off and on. All 2,628 catalog cases count in the denominator. The
previous profile excluded 78 playable reverse cases because only three bomb/carrier
missions had been validated; that profile remains available as `-reverse-bombs`
for comparison, but is no longer the daily default. The catalog audit also
recovered Cyberia's ten reverse wave/mode cases by recognizing its embedded
navigation; no current catalog mission is excluded.

See the [full local validation report](audits/full-reverse-20261024/REPORT.md) for
measured results, retained failures, and the navigation catalog recovery.

## Completion evidence

Full coverage means every case is attempted and reported. It does not imply
that every authored objective currently works, or that a missing simulator
witness is a production defect. Missing evidence is a probe error and counts
against verified coverage. Initial failures and longer retries are preserved.
`-experimental-reverse` remains an alias for independent investigation.
A bare invocation without a reverse flag retains the standard-only control.
Use `-reverse-only` with an enabled reverse profile to inspect its plan.

The probe uses a BLU fake player and defeats RED enemies using the existing
wear-down fixture. It preserves allied BLU robots and tanks. In reverse missions
it disables the standard probe's kill-on-bomb-pickup fixture so authored allied
carriers can survive; ordinary missions keep that fixture enabled.

For missions allowing human bomb pickup and deployment, the simulator moves the
fake player to an available bomb, then to an enabled BLU capture zone. It uses
normal collision/touch handling and the mission's own pickup/capture permissions.
If the mission forbids human pickup or deployment, it leaves bomb delivery to
its authored allied carriers. It does not change those permissions, enable
capture zones, fire capture outputs, or call winning relays. The ordinary probe's
capture-output stall workaround is disabled for BLU-player tests.

A reverse bomb pass requires the native `mvm_wave_complete` event, a BLU
`teamplay_round_win`, and a native `teamplay_flag_event` capture by a BLU
participant in that wave. Computer Crash wave 6 is an authored combat finale:
it instead requires a native death of `THE CORRUPTED` and `mvm_wave_complete`.
The target-death witness is serialized as `target_kills`; killing unrelated
support bots cannot satisfy it. A native RED round win before that target
death remains a loss. For bomb waves, a native RED round win is a failure even when
SigMod suppresses `mvm_wave_failed`. A completion
without its required native objective evidence is a probe error, not accepted coverage. JSONL includes
objective visit, pickup, and capture counts; raw logs identify capture actors.
The simulator is enabled and checked before every wave because map/plugin reloads
can reset experimental state. Missing simulator support refuses the experiment.
After a completed reverse mission, the controller allows the game's natural
mission/map transition to settle before loading another mission. If it remains
on its victory map, the next fixture explicitly reloads the map. Reverse waves
use the fake player's native ready-up command rather than `mp_restartgame`,
which can interfere with mission-spawned objectives.
Map-load failures remain explicit; automatic transitions stay enabled. Reverse mission readiness also waits for SourceMod's
`OnConfigsExecuted` callback. Seeing a map name before server/plugin configs
finish is not sufficient readiness.

## Running it

Build the controller and probe plugin using the same pinned toolchain as the
native stability bed (`make stability-bed`; see [stability setup](stability.md)).
Use an isolated disposable server with both community packs, never a live player
server. The server must use 32 player slots and keep the production robot limiter
enabled. Use the same headless wake fixture as the standard wave probe.

The production Docker entrypoint periodically syncs staged addons into the game.
Bind the candidate probe under
`/opt/tf2-archipelago/addons/sourcemod/plugins/tf2_waveprobe.smx`; updating only
`tf/addons/sourcemod/plugins/tf2_waveprobe.smx` is temporary and can be overwritten.
Verify the staged and installed SHA-256 hashes before and after every trial,
and check the running status includes `configsready`. A plugin reversion
invalidates candidate validation. Use a fixed compiled file for the container's
bind mount; an atomic replacement of a host file can leave a bind mount on its
old inode.

For a single mission:

```sh
dist/linuxbed/waveprobe -rcon 127.0.0.1:27035 \
  -mission mvm_rottenburg_rev_int_breaking_walls \
  -reverse-objectives -mode both -speed 50 -timeout 180s
```

For the daily reverse-only plan (122 planned and zero unsupported cases):

```sh
dist/linuxbed/waveprobe -plan -mission all -mode both \
  -reverse-objectives -reverse-only > reverse-plan.jsonl
```

Keep the first run's JSONL and debug snapshots before retesting a specific
nonpass with the same profile and a longer wall budget. Hosted and local runners
record `reverse_objectives=1` in their config, and retries preserve that flag. The
manual `reverse-smoke` workflow scope tests Breaking Walls on all three
environments without publishing full-catalog badges. The full daily profile includes all playable reverse missions.

## What this measures

Teleporting the fake player into objective volumes exercises real pickup,
capture permissions, authored hatch locks and engine/SigMod completion logic.
It does not validate walking routes, player survival, gate reachability, or
whether a human can physically navigate to that volume. Allied carriers use
their own AI and navigation. Future missions with different payloads or
multi-stage mechanics need native witnesses of their own; clearing RED robots
alone does not establish objective completion.

The local investigation's initial kill-only baseline, candidate snapshots,
per-wave native capture evidence and reports are saved separately under
`docs/audits/reverse-probe-20261024/` in the original workspace. This source branch
does not include the raw game logs or generated binaries.

## Additional native objectives

Shogun and Robotfactory's final wave use real control-point touch/capture logic.
The fake player selects unlocked, unowned points and stands in the capture area;
it never changes ownership, unlocks a point or writes capture progress.

Robotfactory wave 1 selects Engineer, equips the native wrench through player
input, aims at the current objective dispenser, and holds primary attack.
Native repair, metal consumption, mission regeneration, shielding and authored
Lua callbacks remain active. All five native repairs, or the authored five-building completion output,
and native BLU victory plus wave completion are required. The observed
`repair_completed` flag preserves completion evidence if victory removes the
buildings before the next health sample. Wave 2 preserves the allied VIP and observes
the three authored escort stage outputs in order; it never moves the VIP or
fires those outputs. Missing buildings or stage outputs cannot pass.

Autumnull's finale requires a native death of its named `Overseer` boss, as
Computer Crash requires `THE CORRUPTED`. Hideout wave 4 observes the authored
`boss_deploy_relay` output for tank delivery, or a native flag capture when
the authored player permissions allow it. Both require native BLU victory and
wave completion. Its combat finale requires native
BLU victory, observed enemy defeats and wave completion. These exceptions are
bound to exact mission/wave keys in the independent report validator.

Cyberia's Arctic Arrangement uses native control-point objectives on waves 1–3
and 5. Wave 4 is its authored combat/tank stage: a native BLU win, wave completion,
enemy defeats and all five enemy tank deaths (four control boxes plus the main
RED tank) are required. Bomb captures cannot substitute for either profile.
The corrected profile is compared at both 20× and 50× in the local audit.

The fixtures exercise objective completion, not a complete human playthrough.
A local Linux Docker run cannot establish native Windows or Ubuntu startup;
those remain separate hosted measurements using the same case plan.
