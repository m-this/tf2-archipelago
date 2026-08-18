# MvM Defender TFBots and the build-from-source question

Research note, not a decision. Checked 2026-08-18 against the GitHub API, the
release artifacts, and the mod's source. The decision (bundle the mod, and
download prebuilt versus build from source) lands in a later ADR.

## Why this matters

Valve tunes MvM waves for six defenders. A solo or small-team Archipelago run
needs the empty RED slots filled with bots that actually play, or wave 1 is
unwinnable and the seed is dead. `docs/en/spec.md` already names this as the
reason `Allied Mercs` exists, and `docs/en/operate/testing.md` currently tells
a tester that making bots fight "needs a third-party plugin, which is a
personal choice and not part of this repository." This note is the research for
changing that.

## The mod

`OfficerSpy/TF2-MvM-Defender-TFBots` — "TFBots that can play Mann vs Machine."
GPL-3.0, SourcePawn, 21 stars, last pushed 2026-07-10. The README calls it a
"constant work-in-progress" and warns of reported incompatibility with external
mods such as `sigsegv-mvm`. The initial AI is a port of
`Pelipoika/TF2_Idlebot` ("MvM AFK Bot"); the internal PathFollower and the tank
weapon score system are adapted from `sigsegv-mvm`'s C++.

### Verified facts

Checked 2026-08-18 against the GitHub API and the release artifacts.

| Thing | Value | How it was checked |
| --- | --- | --- |
| Latest release | `1.5.5`, published 2026-03-17 | GitHub releases API |
| Release asset for SourceMod 1.12 | `TF2-MvM-Defender-TFBots-sm1.12.7164.zip`, 161 KB | `unzip -l` on the asset |
| Also ships | a `sm1.13.7166` build of the same tag | same |
| Zip layout | roots at `configs/`, `gamedata/`, `plugins/` | `unzip -l` |
| Zip contents | `plugins/tf2_defenderbots.smx`, `gamedata/tf2.defenderbots.txt`, `configs/defenderbots/bot_names.txt`, `configs/defenderbots/map/*.cfg` (28 maps), `configs/defenderbots/mission/*.txt` | `unzip -l` |
| Compile-time dep | `OfficerSpy/SM_Stock_OfficerSpy` (sp.toml) | `sp.toml` in the repo root |
| Runtime deps | TF2Attributes, TF Econ Data, TF2Utils, CBaseNPC, Actions, ripext | README "Requirements" |
| Commits on `main` since 1.5.5 | 3 (spylurk.sp + two README updates) | `compare/1.5.5...main` |

The zip unpacks straight into a SourceMod tree (`addons/sourcemod/{plugins,
gamedata,configs}`), so installing it is one `unzip` and one `cp`. The configs
ship per-map navigation hints for 28 maps, including the 7 Valve MvM maps this
project cares about (`mvm_decoy`, `mvm_coaltown`, `mvm_mannworks`,
`mvm_bigrock`, `mvm_mannhattan`, `mvm_rottenburg`, `mvm_ghost_town`).

## How the mod fills RED, and what it needs to do it by default

Read from `source/tf2_defenderbots.sp` and `source/redbots3/events.sp` on
2026-08-18. The mod has three manager modes, picked by the
`sm_redbots_manager_mode` convar:

| Mode | convar value | When bots spawn |
| --- | --- | --- |
| `MANAGER_MODE_MANUAL_BOTS` | `0` (**the default**) | An admin adds them by hand (`sm_addbots`, `!votebots`). |
| `MANAGER_MODE_READY_BOTS` | `1` | A human readies up (F4). The readystate listener calls `ManageDefenderBots(true)`, which spawns bots and starts the imbalance monitor. |
| `MANAGER_MODE_AUTO_BOTS` | `2` | `mvm_begin_wave` fires and the handler calls `ManageDefenderBots(true)`. Bots spawn when a wave begins. |

In every mode, once `g_bBotsEnabled` is true, `Timer_CheckBotImbalance` runs
every second and tops RED up to `sm_redbots_manager_defender_team_size`
(**default 6**) for the rest of the round. So the fill target is 6 by default;
what changes between modes is the trigger.

Related convar defaults, read from the `CreateConVar` calls:

| Convar | Default | Meaning |
| --- | --- | --- |
| `sm_redbots_manager_mode` | `0` | MANUAL_BOTS. The mod does nothing automatic out of the box. |
| `sm_redbots_manager_defender_team_size` | `6` | The fill target. |
| `sm_redbots_manager_min_players` | `3` | Minimum humans+bots before ready-up is allowed. Normal uses this; Intermediate adds 1, Advanced 2, Expert 3, Nightmare 4. `-1` disables. |
| `sm_redbots_manager_bot_lineup_mode` | `0` | RANDOM class picks for the bots. |
| `sm_redbots_manager_kick_bots` | `1` | Kick bots on wave failure/completion. |

**Implication for this project.** The mod does not fill RED by default. To get
the behaviour the spec wants (solo/2/3 fills to 6), the image has to set
`sm_redbots_manager_mode 2` (AUTO_BOTS) in a SourceMod config, the same way
`deploy/srcds/tf2_archipelago.cfg` sets our own convars. With AUTO_BOTS, the
solo path is: the human readies up (allowed by `tf_mvm_min_players_to_start 1`
already in our `server.cfg`), `mvm_begin_wave` fires, the mod spawns bots to
fill RED to 6, and the imbalance timer keeps it there for the wave.

### Bots ready themselves

`docs/en/operate/testing.md` currently says "MvM waits for every RED player,
and a bot never readies itself, so F4 does nothing while one is on the team,"
and tells a tester to run `bot_command all tournament_player_readystate 1` as a
workaround. The defender mod removes that workaround's reason to exist.

`HandleTeamPlayerCountChanged` (in `tf2_defenderbots.sp`, called on
`player_team` for RED changes) runs the ready loop: when every RED member is
ready, it unreadies one (preferring a human, else a bot) so the wave does not
start and another bot can slot in; if it unreadied a bot, it re-readies that
bot 0.2 s later via `Timer_ReadyPlayer` → `SetPlayerReady(bot, true)`. So the
bots are readied by the plugin, not by the player. The testing doc's manual
ready-up step becomes redundant once the mod is installed.

## Coexistence with this project's plugin

Read from our `plugin/scripting/tf2_archipelago/mvm.inc` and the defender mod's
`events.sp` on 2026-08-18. The two plugins operate on disjoint sets of clients,
and the one place they touch the same game state is settled.

**Enforcement does not reach bots.** Our `MvM_IsPlayer` (mvm.inc:56) filters
`!IsFakeClient(client)`. Weapon slot enforcement, class enforcement, the credit
payout in `MvM_GrantCredits`, and the unlock-set application all go through it,
so defender bots (which are `tf_bot` fake clients) are excluded. The defender
mod's `Timer_PlayerSpawn` / `Event_PlayerSpawn` operates on fake clients only.
No overlap.

**Bot spawning does not touch the quota.** The mod spawns with
`tf_bot_add <n> red <class> expert noquota` (tf2_defenderbots.sp:1936), so it
does not raise `tf_bot_quota`. The "zero the quota after every add" warning in
testing.md is about manual `tf_bot_add`, not the mod.

**Both hook `mvm_begin_wave`.** Our plugin records the wave number from it; the
defender mod uses it as the AUTO_BOTS trigger. SourceMod allows multiple hooks
on one event, so both fire. No collision.

**One behaviour change for the operator.** The mod is built with
`CHANGETEAM_RESTRICTIONS` defined, which bans a non-admin from switching BLU →
RED for 30 s (extending by 10 s per repeat). Our plugin never forces team
changes, so this does not break us, but it is a rule the operator inherits.

**One resolved interaction worth recording.** CBaseNPC issue #59 (closed
2025-02-06) reported that an older CBaseNPC replaced the gamerules proxy and
made `GameRules_GetRoundState` throw `Property "m_iRoundState" not found on the
gamerules proxy` for every other plugin. Our `MvM_IsActive` reads
`m_bPlayingMannVsMachine` off the same proxy, and the defender mod calls
`GameRules_GetRoundState` throughout. Fixed in CBaseNPC 1.12.0.114 and later
(we would pin 1.15.4.126), but it is the clearest evidence that CBaseNPC
patches game-rules internals other plugins depend on. See the CBaseNPC section
below.

## The five runtime dependencies

Three are gamedata-backed SourcePawn plugins: a `.smx` plus a signatures
`.txt`, no compiled extension, architecture-independent. Two are compiled C++
extensions: a `.so` linked against the `tf2` HL2SDK branch, ABI-coupled to the
SourceMod and Metamod in the base image.

### Verified facts

Checked 2026-08-18 against each repo's latest release assets and, for the two
extensions, the ELF class of the shipped `.so`.

| Dependency | Repo | Pinned version | Type | Runtime files |
| --- | --- | --- | --- | --- |
| TF2Attributes | `FlaminSarge/tf2attributes` | `v1.7.5` | gamedata plugin | `tf2attributes.smx`, `gamedata/tf2.attributes.txt` |
| TF Econ Data | `nosoop/SM-TFEconData` | `0.19.1.5` | gamedata plugin | `tf_econ_data.smx`, `gamedata/tf2.econ_data.txt` |
| TF2Utils | `nosoop/SM-TFUtils` | `1.4.0.2` | gamedata plugin | `tf2utils.smx`, `gamedata/tf2.utils.nosoop.txt` |
| CBaseNPC | `TF2-DMB/CBaseNPC` | `1.15.4.126` | compiled extension | `extensions/cbasenpc.ext.2.tf2.so`, `gamedata/cbasenpc.txt` |
| Actions | `Vinillia/actions.ext` | `v3.9.2` | compiled extension | `extensions/actions.ext.2.tf2.so`, `gamedata/actions.games.txt` |

Notes:

- TF2Attributes, TF Econ Data and TF2Utils ship only the `.smx`, the gamedata
  `.txt` and the `.inc` from their releases. No compiled `.so`. The `.smx` is
  SourcePawn bytecode, so it loads on any SourceMod 1.12 install regardless of
  arch.
- CBaseNPC's Linux tarball (`cbasenpc1.15.4.126_linux.tar.gz`, 3.5 MB) roots at
  `addons/sourcemod/` and carries `extensions/cbasenpc.ext.2.tf2.so`,
  `gamedata/cbasenpc.txt`, and the `scripting/include/` tree (build-time only
  at runtime). The `.so` is a **32-bit ELF**. `AMBuildScript` enables only the
  `tf2` SDK (every other SDK is commented out) and `platformSpec` for tf2 is
  `WinLinuxMac` with `x86` only, so no 64-bit build is produced.
- Actions v3.9.2's zip roots at `actions/` and carries `actions.ext.2.tf2.so`
  in **both** `extensions/` (32-bit, 9.6 MB) and `extensions/x64/` (64-bit,
  10.1 MB), plus L4D and L4D2 builds. See the next section for why the pin is
  v3.9.2 and not the latest.

### UNVERIFIED

- Whether the two compiled `.so`s actually load against the SourceMod in
  `cm2network/tf2:sourcemod` (SourceMod 1.12.0-git7246). They are built against
  the alliedmodders HL2SDK `tf2` branch and a SourceMod source snapshot, and
  the image's SourceMod is from the same 1.12 line, but a vtable or gamedata
  mismatch surfaces only at `sm exts list` after `make up`. This is the same
  "unverified until a human is on the server" class as the rest of milestone 4
  in `TODO.md`.

## 32-bit and 64-bit

Checked 2026-08-18 against the image metadata and the shipped binaries.

- The `cm2network/tf2:sourcemod` image is `amd64`, but TF2's dedicated server
  (Steam app `232330`) is a 32-bit binary. The image ships `steamcmd/linux32`
  and SourceMod 1.12 ships 32-bit extensions. So a TF2 srcds loads 32-bit
  SourceMod extensions, not 64-bit.
- CBaseNPC ships a 32-bit `cbasenpc.ext.2.tf2.so` only. `AMBuildScript`
  declares x86-only for tf2. There is no 64-bit CBaseNPC build to download, and
  issue #49 ("Prepare extension for x64 build") is open. A 64-bit build is not
  available and not needed for TF2.
- Actions v3.9.2 ships both a 32-bit and a 64-bit `actions.ext.2.tf2.so`. The
  64-bit copy is unused for TF2. Actions v4.0.0 ships no TF2 build at all (see
  below), so for v4.x the 32-bit TF2 `.so` is only reachable by building from
  source.

So the prebuilt path covers 32-bit for both extensions at the pinned versions.
The "build it ourselves" question only arises for a newer Actions than v3.9.2,
where the maintainer has not published a TF2 binary at all.

## Actions v4.0.0 and the missing TF2 binary

The Actions extension is the one dependency whose latest release does not
carry a TF2 build. Checked 2026-08-18.

| Release | TF2 `.so` in the zip? | L4D / L4D2 |
| --- | --- | --- |
| `v3.9.2` (2025-06-10) | yes, 32-bit and 64-bit | yes |
| `v3.9.3` | no release asset (the tag exists, the zip 404s) | n/a |
| `v4.0.0` (2026-03-26) | **no** | L4D2 only |

Two findings make this a packaging gap on the maintainer's side, not a
deliberate drop:

1. The v4.0.0 source still builds TF2. `AMBuilder` keeps `tf2` in the
   `gameFiles` map (`source/sdk/tf2/actions_processor.cpp`,
   `source/sdk/tf2/actions_tools_tf2.cpp`), and `PackageScript` copies whatever
   `Extension.extensions` produced. No commit between v3.9.2 and v4.0.0 removes
   TF2 from the build matrix; the six commits are "remove std::any", "per call
   swap", "defer vtables", "add new file to the builder", "fix clang build",
   "update gamedata". The TF2 build is simply not in the artifact the
   maintainer uploaded.
2. There is no CI workflow file in the repo (no `.github/workflows/`), so
   releases are built and packaged by hand. No issue or discussion mentions TF2
   being dropped. The repo has 4 issues total (#28, #29 open; #24, #27 closed),
   all L4D-specific.

So a v4.x TF2 `.so` is only available by building from source. A v3.9.2 TF2
`.so` is available prebuilt and is the last one shipped. The four issues are
all L4D2 crash reports; none affect TF2.

## The TF2 SDK open-sourcing, and whether it helps here

`prior-art.md` already notes in one line that "Valve published the TF2 client
and server source in the February 2025 SDK drop." This section is the longer
read the user asked for.

The key point: the February 2025 Valve release is **not on the build path** for
these extensions. SourceMod extensions build against `alliedmodders/hl2sdk`
(the `tf2` branch of that mirror has existed for years), Metamod:Source source,
and SourceMod source, all wired together by AMBuild. That mirror is a cleaned,
buildable snapshot of the SDK headers Valve shipped to modders long before
2025; it is what `AMBuildScript` in both CBaseNPC and Actions resolves via
`hl2sdk-manifests` and `findSdkPath`. Valve's 2025 drop is a separate, larger
artifact that these projects do not consume.

### alliedmodders/hl2sdk maintenance

Checked 2026-08-18 against the issue tracker (the core API for commit dates
was rate-limited; recency is read off the most recently updated issues, which
is a reliable proxy for maintenance).

| Signal | Value |
| --- | --- |
| Most recently updated issue | 2026-08-01 (#78, BMS `EmitGameSound` crash, closed) |
| Open CS2 support tracker | #132, opened 2026-07-28 |
| Clang 19 compile fix | #363, closed 2026-05-23 |
| Clang 22 / vscript fix | #398, closed |
| C++20 / msvc tier1 fix | #366, open |
| `x86_64` compile fixes | #337, #301, both closed |

The mirror is actively maintained through 2026. The tf2 branch is kept
buildable against modern compilers (clang 19/22, msvc C++20). No open issue
reports the 32-bit tf2 build being broken; the x86_64 path has had fixes
(#337, #301) but 32-bit is the well-trodden path for TF2 and is what we need.

### What building from source actually requires

Read from `configure.py`, `AMBuildScript`, `hl2sdk-manifests/manifests/tf2.json`,
and a shallow checkout of each source on 2026-08-18. The Makefile in the
actions.ext repo root is a leftover SourceMod sample scaffold (it references
`sample`, `hl2sdk-ob`, `mmsource-1.9`); the real build is AMBuild 2.2, the
same as CBaseNPC.

```
git clone --depth 1 --branch tf2  --single-branch https://github.com/alliedmodders/hl2sdk       hl2sdk-tf2
git clone --depth 1 --branch 1.12-dev --single-branch https://github.com/alliedmodders/sourcemod  sm-src
git clone --depth 1 --branch master --single-branch https://github.com/alliedmodders/metamod-source mms-src
cd <cbasenpc or actions source>
python configure.py --hl2sdk-root <parent-of-hl2sdk-tf2> --mms-path <mms-src> --sm-path <sm-src> \
    --sdks tf2 --targets x86
ambuild
```

`SdkHelpers.ambuild` does not auto-checkout the SDK. It expects `hl2sdk-tf2`
present at `--hl2sdk-root` or via the `HL2SDKTF2` env var. The SourceMod branch
is `1.12-dev` (not `1.12` — that branch does not exist). Metamod:Source builds
off `master`.

**Path mismatch in the tf2 manifest.** The manifest's `postlink_libs` for
linux x86 references `lib/linux/mathlib_i486.a` and `lib/linux/tier1_i486.a`,
but the actual 32-bit Linux prebuilt libs live at `lib/public/linux/` in the
hl2sdk tf2 branch:

```
lib/public/linux/mathlib_i486.a     1.1 MB
lib/public/linux/tier1_i486.a       3.1 MB
lib/public/linux/libtier0_srv.so    239 KB
lib/public/linux/libvstdlib_srv.so  413 KB
```

`lib/linux/` does not exist. AMBuild resolves relative to `sdk['path']`, so
this is either a manifest bug that the projects paper over with a symlink, or
AMBuild's `getLists` handles the path differently than the manifest declares.
Either way, a from-source build that fails to link against `mathlib_i486.a`
should look here first. The 64-bit libs are at `lib/public/linux64/` and are
not needed for TF2.

**Measured checkout footprints** (shallow clone, sparse checkout of only the
build paths, no `.git`):

| Source | Branch / tag | Full shallow | Sparse (no `.git`) |
| --- | --- | --- | --- |
| `alliedmodders/hl2sdk` | `tf2` | 633 MB (517 MB tree + 117 MB `.git`) | **24 MB** |
| `alliedmodders/sourcemod` | `1.12-dev` | 61 MB | **3.3 MB** |
| `alliedmodders/metamod-source` | `master` | 7.1 MB | **5.2 MB** |
| `TF2-DMB/CBaseNPC` | `1.15.4.126` | 2.2 MB | 1.8 MB |
| `Vinillia/actions.ext` | `v4.0.0` | 14 MB | 14 MB |
| `OfficerSpy/SM_Stock_OfficerSpy` | `main` | 416 KB | 416 KB |

The hl2sdk sparse checkout uses `--cone` mode with
`public public/engine public/mathlib public/vstdlib public/tier0 public/tier1
public/toolframework public/game/server game/shared common lib/public/linux` —
everything the tf2 manifest's `include_paths` and `postlink_libs` reference,
nothing else. That drops 189 MB of Windows `.lib` files (`lib/public/x86/`,
`lib/public/x64/`) and the 117 MB `.git` directory. Total sparse footprint
across all six sources: **~50 MB**.

Concretely, building both compiled extensions from source needs:

- `alliedmodders/hl2sdk` on the `tf2` branch, sparse-checked-out to the 24 MB
  of headers and 32-bit Linux prebuilt libs the build links against.
- `alliedmodders/sourcemod` on `1.12-dev`, sparse-checked-out to `public/` and
  `core/` (where `smsdk_ext.cpp` lives at `public/smsdk_ext.cpp`).
- `alliedmodders/metamod-source` on `master`, for the `core/` and
  `core/sourcehook/` includes.
- The CBaseNPC and actions.ext source trees themselves (pinned tags).
- AMBuild 2.2 (`pip install ambuild`) and a 32-bit C++ toolchain
  (`gcc-multilib` / `-m32` on a 64-bit host), because TF2 srcds is 32-bit and
  CBaseNPC produces no 64-bit build.
- For the defender mod itself: `spcomp` (already fetched by
  `plugin/build.sh`) plus the `.inc` include files from all six deps plus
  `SM_Stock_OfficerSpy`, and the mod's own `source/` tree.

None of that is the February 2025 Valve drop. The Valve release would matter if
we were writing our own extension from scratch against fresh headers, or if the
alliedmodders mirror fell out of maintenance. Neither is the case today.

### Would a pinned v4.x Actions build need patches?

Yes, and more than patches. The commit-log reading below was wrong; the build
proves it. See "The from-source build, done" for what actually happens.

The six commits between v3.9.2 and v4.0.0 are "remove std::any", "per call
swap", "defer vtables", "add new file to the builder", "fix clang build",
"update gamedata". None removes TF2 from `gameFiles`, which is why the TF2
build looked like a packaging slip. It is not: the TF2 source was never
recompiled after the processor API changed, and it no longer builds.

## Licenses

Checked 2026-08-18 against each repo's `LICENSE` file and, where there is none,
the source header. A release image that ships the compiled artifacts
redistributes them, so this matters.

| Dependency | License | Where declared |
| --- | --- | --- |
| MvM Defender TFBots | GPL-3.0 | `LICENSE` file |
| TF Econ Data | GPL-3.0 | `LICENSE` file |
| TF2Utils | GPL-3.0 | `LICENSE` file |
| CBaseNPC | GPL-3.0 | `extension/smsdk_config.h` header (no top-level `LICENSE` file) |
| Actions | GPL-3.0 | `LICENSE` file |
| TF2Attributes | **none found** | no `LICENSE` file, no source header, empty `README.md` |

**TF2Attributes is the one blocker.** It has no license file, no license
header in `scripting/tf2attributes.sp` or the `.inc`, and an empty README.
Under default copyright that is "all rights reserved" — no redistribution
right. It has been distributed on the AlliedModders forums
(`forums.alliedmods.net/showthread.php?t=210221`) since 2012 with implicit
permission to use, and it is the most widely depended-on TF2 SourceMod plugin,
but forum distribution is not a license grant for bundling the compiled `.smx`
into a release image. The safe options are: get explicit confirmation from
FlaminSarge, or have the image download the `.smx` at runtime (the entrypoint
already installs into a volume on first start, so this is a small change to
`srcds-entrypoint.sh` rather than a Dockerfile layer). The runtime-download
path also sidesteps GPL redistribution questions for every GPL dep, though
GPL-3.0 permits conveyance of the compiled form alongside the offer of source,
so the GPL ones are fine to bundle if we point at the upstream source.

## CBaseNPC is the fragile dependency

Read from the CBaseNPC issue tracker on 2026-08-18. CBaseNPC is the dependency
most likely to break on a TF2 update, and the one whose breakage takes the
defender mod down with it. The defender mod's gamedata
(`gamedata/tf2.defenderbots.txt`) uses Linux mangled symbols
(`@_ZN9CTFPlayer24PostInventoryApplicationEv`, `@g_MannVsMachineUpgrades`,
etc.) which are stable across TF2 builds, but CBaseNPC itself uses direct
signatures and vtable detours that move when Valve ships a srcds update.

| Issue | State | What it tells us |
| --- | --- | --- |
| #69 "cbasenpc.ext.2.tf2.so causing crash loops" | closed 2025-11-17 | A TF2 update made the `.so` crash-loop; the reporter runs the defender mod. Fixed in a later release. |
| #71 "Crashing with latest release" | closed 2025-05-03 | Same class: a release stopped working on current TF2. |
| #62 "Update Windows gamedata for 2/18/25 TF2 update" | closed 2025-02-20 | `CBaseEntity::CBaseEntity`, `CBaseAnimating::ResetSequence`, `CTFGameRules::ApplyOnDamageModifyRules` signatures changed in a TF2 update. |
| #51 "x32 Windows Gamedata Update for 4/18/2024 TF2 Update" | closed | Same pattern, earlier update. |
| #59 "Extension causes the SDK Tool's native GameRules_GetRoundState to throw exceptions" | closed 2025-02-06 | An older CBaseNPC replaced the gamerules proxy and broke `m_iRoundState` for every other plugin. Our `MvM_IsActive` reads `m_bPlayingMannVsMachine` off the same proxy; the defender mod calls `GameRules_GetRoundState` throughout. Fixed in 1.12.0.114+. |
| #49 "Prepare extension for x64 build" | open | 64-bit is not ready. We need 32-bit, so this does not block us. |

The pattern: every TF2 srcds update risks moving a signature CBaseNPC detours,
and the extension crashes or fails to load until a new release ships with
updated gamedata. Pinning 1.15.4.126 (current) is safe today, but it is the one
dependency an operator may have to bump mid-run. The integration test cannot
catch this; only a live server after a TF2 update does.

## Open issues in the defender mod that affect us

Read from the defender mod's issue tracker on 2026-08-18 (11 issues total).

| Issue | State | Effect on us |
| --- | --- | --- |
| #13 "Bots can't go to the upgrade station" | **open** (2026-07-07) | `CBaseEntity.WorldSpaceCenter` throws in `redbots3/behavior/gotoupgrade.sp::CTFBotGotoUpgrade_Update`. Bot upgrade-buying is currently broken on the latest release and on `main` (the 3 commits since 1.5.5 do not touch it). See below. |
| #5 "Crashes newer compilers" | open | A build issue for the mod itself, not for us: we ship the prebuilt `.smx`. |
| #4 "[MvM] RED bots need balancing on Spy detection" | open | Balance, not correctness. |
| #7 "[MvM] Engineer bots build very close to robots" | open | Balance. |
| #1 "[Linux] Bots Cannot Buy Upgrades Properly" | closed 2024-05-26 | An earlier Linux upgrade bug, resolved. |
| #11 "Crashing server on startup unknown reason" | closed 2026-02-16 | Resolved; the report runs srcds 32-bit + SM 1.12 build 7164, our exact stack. |

**#13 reopens the spec's "share player upgrades" question.** The defender mod
is built with `METHOD_MVM_UPGRADES` defined: bots buy their own upgrades at the
station via `VS_GrantOrRemoveAllUpgrades`, and on wave failure the mod can
refund them (`redbots_manager_keep_bot_upgrades`). That design supersedes the
spec's "bots share the player's unlocked upgrades directly" line and resolves
the `UNVERIFIED` item in `TODO.md` ("Allied bot upgrade sharing"). But with #13
open, the buy-upgrades path crashes on `WorldSpaceCenter`, so bots cannot reach
the station on the current release. Until #13 is fixed, bots are under-upgraded
relative to a human team, and the spec's "share player upgrades" fallback
becomes relevant again. Either we wait for #13, we ship the mod and accept
under-upgraded bots, or we implement the share-upgrades path in our plugin on
top of the mod.

## What we build, and what we download

Two constraints decided this, and neither was visible when the section above
was written: CI minutes are scarce, and the server has to run on Windows too.

Compiling the plugins is what buys us anything — the patches are in them, and
`.smx` is bytecode, so one build serves Linux and Windows. Compiling the two
extensions buys nothing on the normal path: our patches to them are build
fixes, not behaviour, the binary would be Linux-only (the `.dll` needs MSVC),
and a from-scratch C++ build is several CPU-minutes per run. Both pinned
versions ship a 32-bit TF2 `.so` **and** `.dll` upstream, which is exactly
what a Windows server needs and what we could never cross-compile.

So: **the four plugins are compiled from patched source, the two extensions
are downloaded from the pinned releases.** `deploy/bots/build.sh` does both and
stages one tree that serves either platform. The from-source path for the
extensions stays, in `deploy/bots/build-extensions.sh` behind
`BOTS_BUILD_EXTENSIONS=1`, because the day a TF2 update moves a CBaseNPC
signature is the day we need to patch and rebuild without waiting for a tag.

Cold, the whole thing takes about 30 seconds and no C++ toolchain.

### How patching works

Patches live in `deploy/patches/<dep>/` as a series of `.patch` files applied
at build time to the pinned source checkout. The build flow for each dep:

1. Shallow-clone the pinned tag from GitHub (or pull from cache).
2. Apply `deploy/patches/<dep>/*.patch` in alphabetical order, `git apply
   --check` first, fail the build on a rejected hunk.
3. Build with `spcomp`; `ambuild` too when the extensions are being built here
   rather than downloaded.
4. Copy the build outputs (`.so`, `.smx`, gamedata `.txt`, configs) into the
   staged SourceMod tree.

A patch that no longer applies is a CI failure, not a silent skip. That is the
signal to rebase the patch against the new upstream, or to drop it if upstream
fixed the issue. The pin in `versions.env` moves deliberately; the patches
follow.

### What gets built, and what does not

| Dep | Built from source? | Why |
| --- | --- | --- |
| CBaseNPC | no, downloaded | The release ships a 32-bit `.so` and `.dll`, which is both platforms. Buildable here with `BOTS_BUILD_EXTENSIONS=1` when a TF2 update breaks it. |
| Actions | no, downloaded | Same, at v3.9.2. v4.0.0 has no TF2 binary and no TF2 source that compiles, so v4.x is not a destination. |
| MvM Defender TFBots | yes, `spcomp` | SourcePawn plugin. We patch `#13` (upgrade-station crash) and any other open bug we hit, then recompile. |
| TF2Attributes | yes, `spcomp` | SourcePawn plugin. No license on the prebuilt `.smx`; compiling from source under GPL-3.0 (the `.inc` and `.sp` are on GitHub) gives us a binary we can redistribute. |
| TF Econ Data | yes, `spcomp` | SourcePawn plugin. Trivial to compile; keeps the build uniform. |
| TF2Utils | yes, `spcomp` | Same. |

The four plugins are compiled, the two extensions are downloaded. The compiler
is not the one `plugin/build.sh` uses: our own plugin builds with
`SOURCEMOD_VERSION`, and the defender mod needs
`DEFENDERBOTS_SOURCEMOD_VERSION` because the newer spcomp segfaults on it.

### The TF2Attributes license question under this approach

Building from source changes the TF2Attributes situation. The `.sp` source is
on GitHub under `FlaminSarge/tf2attributes` with no license file, but the
project has been publicly distributed and modified by the SourceMod community
since 2012. Compiling it ourselves produces a binary whose provenance is the
public source, not a redistributed upload. This is cleaner than bundling a
prebuilt `.smx` with no license, though it does not fully resolve the
"all rights reserved" default: the source itself has no license grant. The
practical answer is that every TF2 SourceMod project depends on this plugin
and FlaminSarge has never objected to redistribution, but the formal answer
still needs either a license file upstream or explicit confirmation. Until
then, the image compiles it at build time rather than shipping the upstream
`.smx`. The release artifact does carry our build of it, which is the
conveyance the "all rights reserved" default leaves unresolved; the GPL-3.0
deps around it are fine, since we point at their sources.

## CI cost

Checked 2026-08-18 against the GitHub billing docs and the repo's own metadata.

The repository is public, so Actions minutes are unlimited on standard runners
and GHCR is free. The cache allowance is 10 GB per repository, against ~25 MB
of downloads and staged output here. Cost is not the constraint; wall-clock on
a shared runner is, which is the reason the shipped path does not compile C++
at all.

One cache entry covers it, in the `plugin` job of `ci.yml`:

```
key: bots-${{ hashFiles('deploy/env/versions.env', 'deploy/patches/**', 'deploy/bots/*.sh') }}
path: deploy/bots/build
```

Those three inputs are the only things that change what the script produces:
the pins, the patches, and the script itself. On a hit the step is four spcomp
runs. On a miss it is about 30 seconds of downloads on top. The staged tree,
the upstream checkouts, the spcomp drop and the release zips all live under
that one path, so there is nothing to key separately.

The from-source extension build is not in CI. It is a local escape hatch for a
broken TF2 update, and putting a multi-minute C++ build on every push to buy
nothing is the trade this section exists to refuse.

## The from-source build, done

Ran on 2026-08-18 in a `debian:13` container with `gcc-multilib g++-multilib
clang` and AMBuild from git (`pip install ambuild` finds nothing on PyPI; the
package name resolves only from `alliedmodders/ambuild`). Every one of the six
deps now produces an artifact. Every problem below was found by running the
build, not by reading the source.

| Artifact | Built | Size |
| --- | --- | --- |
| `cbasenpc.ext.2.tf2.so` | 1.15.4.126, clang, 32-bit ELF | 8.8 MB |
| `actions.ext.2.tf2.so` | v3.9.2 + 3 patches, clang, 32-bit ELF | 8.1 MB |
| `tf2_defenderbots.smx` | 1.5.5 + our #13 patch, spcomp 1.12.0.7164 | 146 KB |
| `tf2attributes.smx` | v1.7.5 + 1-line patch | 17 KB |
| `tf_econ_data.smx` | 0.19.1.5 | 19 KB |
| `tf2utils.smx` | master | 19 KB |

### What the build needs that the earlier reading missed

- **The sparse checkout list was too narrow.** `game/server` is on the include
  path (`enginecallback.h` comes from there) but was not in the cone list. The
  working set is `public game common lib/public/linux`, which is 86 MB, not
  24 MB. Still worth doing: the full tree is 633 MB.
- **`lib/linux` really is missing, and it really does break the build.**
  Actions fails at link time on `lib/linux/libvstdlib_srv.so`. A
  `ln -s public/linux lib/linux` in the SDK checkout fixes it. CBaseNPC
  resolves `lib/public/linux` itself and does not need the symlink.
- **Submodules are not optional, and CBaseNPC's use an SSH URL.**
  `third_party/safetyhook` is registered as `git@github.com:...`, so a CI
  checkout without an SSH key fails on `submodule update`. The build needs
  `git config --global url."https://github.com/".insteadOf git@github.com:`.
- **CBaseNPC must be configured `--extension-only`.** Its example plugin
  (`scripting/cbasenpc_example.sp`) does not compile with a 1.12 spcomp:
  `CBaseNPC npc = CBaseNPC();` is now `error 170: creating new object
  'CBaseNPC' requires using 'new'`. We do not ship that plugin. Configure
  still needs `--spcomp-path` and `--sm-api-path` even with `--extension-only`.

### Actions v4.0.0 does not build for TF2

Not a packaging gap. `source/sdk/tf2/actions_processor.cpp` still calls the
old API:

    constexpr HashValue hash = compile::hash("&ActionProcessor::OnSight");
    return ProcessHandler(hash, this, &ActionProcessor::OnSight, me, subject);

`ProcessHandler` now takes `const HashFunction&`, and `HashFunction` carries a
vtable `offset` alongside the hash. The L4D2 side was ported — it passes
`gProcessorFunctions->OnSight` — and the TF2 side was not. That is 48 callsites
in one file, each needing an entry in a TF2 `ProcessorFunctions` table with the
right vtable offset. Porting it means deriving those offsets against the TF2
`Action` vtable, which is exactly the work the maintainer skipped. **Pin
v3.9.2.**

### Three patches to build Actions v3.9.2

1. `AMBuilder` lists `public/libudis86/*.c` and `public/asm/asm.c` from the
   SourceMod tree. Both directories were deleted in SourceMod commit `e07c120c`
   ("CDetour safetyhook", 2024-05-21), so they are gone from every 1.12 build
   we would compile against. Drop the two `project.sources +=` lines; CDetour
   is safetyhook-based now and needs neither.
2. `AMBuildScript` passes `-Werror` with `-Wall`. Against the current hl2sdk
   that is fatal on both compilers: clang trips on the MSVC `#pragma warning`
   blocks in `public/vstdlib/random.h` (`-Wunknown-pragmas`), gcc trips on
   `-Wdangling-else` in `public/tier0/platform.h` and on `[[nodiscard]]` from
   safetyhook inside SourceMod's own `CDetour/detours.cpp`. Change it to
   `-Wno-error`.
3. `source/actions_natives.h:233` does `handle = INVALID_EHANDLE_INDEX` on a
   `CBaseHandle&`. The SDK's `operator=` takes only `const IHandleEntity*`.
   Use `handle.Term()`.

With those three, v3.9.2 builds clean under clang. Under gcc it does not:
`source/actions_tools.h` uses `K(__cdecl*)(T...)` in a template alias that gcc
rejects. **Build the extensions with clang.**

### spcomp 1.12.0.7246 segfaults on the defender mod

Our `plugin/build.sh` pins SourceMod 1.12.0-git7246. Both `spcomp` and
`spcomp64` from that drop die with SIGSEGV on `tf2_defenderbots.sp` — no output,
no diagnostic, exit 139. This is upstream issue #5 ("Crashes newer compilers")
seen from our side. The mod's own CI uses 1.12.7164, and that drop's `spcomp64`
compiles it fine: 57 warnings, no errors, 146 KB `.smx`.

So the build needs two SourcePawn compilers: 7246 for our plugin, 7164 for the
defender mod. The alternative is bisecting the compiler crash between 7164 and
7246 and reporting it, which is a real contribution but not on the path to a
working image.

### Two more SourcePawn compile fixes

- **TF2Attributes does not compile with any 1.12 spcomp**, at master or at
  v1.7.5: `tf2attributes.sp:1139` is `#pragma unused UnloadAttributeValue`
  placed *above* the function it names, and 1.12 resolves the pragma before the
  declaration exists. Deleting that one line builds it. This matters for the
  license question — the fix is ours, so what we ship is our build of public
  source, and there is still no license grant on that source.
- **TF Econ Data must be the `0.19.1.5` tag, not master.** Master's
  `tf_econ_data/rarity_definition.sp` calls `LoadAddressFromAddress`, which
  exists in no released stocksoup. The tag builds. Both it and TF2Utils need
  `nosoop/stocksoup` on the include path, which is a seventh build-time
  dependency the earlier list did not name.

## Defender mod #13: root cause and patch

`FindClosestUpgradeStation` (`redbots3/behavior/gotoupgrade.sp`) ends with

    return stations[GetRandomInt(0, stationcount - 1)];

When no station is reachable — `IsPathToVectorPossible` fails for every
`func_upgradestation`, which is what happens to a bot that spawned outside the
spawn room, or on a map whose stations sit in a nav-disconnected area —
`stationcount` is 0, `GetRandomInt(0, -1)` is called out of range, and the
function returns an unset slot of an uninitialised array. That value lands in
`m_iStation[actor]`.

`CTFBotGotoUpgrade_OnStart` half-handles it: `<= MaxClients` catches 0 and
pretends the bot is in the upgrade zone, but then falls through to
`WorldSpaceCenter(m_iStation[actor])` on that same invalid entity. `_Update`
has the same shape with its `IsValidEntity` guard commented out. Either call is
the `CBaseEntity.WorldSpaceCenter` exception in issue #13.

The patch is in `deploy/patches/defenderbots/0001-guard-missing-upgrade-station.patch`:
return `-1` on an empty station list, make the round-state branch in `OnStart`
an `else` so it cannot run on an invalid station, and restore the validity
check in `_Update` as an `action.Done`. The mod compiles with it applied, same
warning count as without. A bot with no reachable station now buys as if it
were at a station (the mod's existing fallback) instead of throwing.

Not yet confirmed on a live server: whether that is the whole of #13, or only
the crash the reporter hit. Only a real MvM round answers that.

## What is settled and what is still open

Settled by this research:

- The mod fills RED to 6 in every mode, but only AUTO_BOTS (`sm_redbots_manager_mode
  2`) does it without a human trigger. The image must set that convar.
- The mod readies its own bots, so the testing doc's `bot_command all
  tournament_player_readystate 1` workaround becomes redundant.
- The mod and our plugin coexist: our enforcement excludes fake clients, the
  mod operates on fake clients, and both hooking `mvm_begin_wave` is fine.
- TF2Attributes is the only dep without a license. The other five are GPL-3.0.
  Building from source sidesteps the redistribution question for the binary,
  but the source itself still has no license grant.
- TF2 srcds is 32-bit; CBaseNPC ships 32-bit only; Actions v3.9.2 ships both,
  v4.x ships no TF2 build. Building from source is the only path to Actions
  v4.x for TF2.
- `alliedmodders/hl2sdk` tf2 branch is actively maintained through 2026. The
  32-bit Linux prebuilt libs exist at `lib/public/linux/` (not `lib/linux/`
  as the tf2 manifest references).
- All six deps build from source, verified end to end on 2026-08-18. That path
  is kept for the day a TF2 update breaks CBaseNPC, behind
  `BOTS_BUILD_EXTENSIONS=1`. It needs clang, two spcomp versions, a
  `lib/linux` symlink, an SSH-to-HTTPS submodule rewrite and the patches above.
- The shipped path downloads the two extensions instead. Both pins carry a
  32-bit TF2 `.so` and `.dll`, so one staged tree serves a Linux and a Windows
  server, and the whole stage takes ~30 seconds with no C++ toolchain. The four
  plugins are still compiled here: that is where our patches live, and `.smx`
  is bytecode that runs on either platform.
- Actions is pinned at v3.9.2. v4.0.0's TF2 source does not compile; porting it
  is 48 callsites plus a vtable-offset table.
- The repo is public, so GH Actions minutes are free and the 10 GB cache
  allowance is more than enough for ~70 MB of source + build outputs.
- The direction is: compile the plugins from patched source, download the two
  extensions from their pinned releases, cache aggressively.
- Defender mod #13 (bots can't buy upgrades) we patch ourselves, we do not
  wait for upstream.

Still open:

- Whether the pinned `.so`s load against `cm2network/tf2:sourcemod`'s SourceMod
  1.12.0-git7246. Only `sm exts list` after `make up` answers this.
- Whether our #13 patch is the whole fix. The root cause is understood and the
  patch is in-tree, but only a live MvM round shows whether bots reach the
  station afterwards.
- Whether to bisect the spcomp crash between 1.12.0-git7164 and git7246 and
  report it upstream, or just carry two compilers.
- What a Windows server needs beyond the zip. The `.dll`s are in it, but no
  Windows srcds has run this stack yet.
- TF2Attributes licensing: get a license file upstream or explicit confirmation
  from FlaminSarge, even under the build-from-source approach.
- The ADR that records this: why the plugins are compiled and the extensions
  are not, and what moves that line back.
