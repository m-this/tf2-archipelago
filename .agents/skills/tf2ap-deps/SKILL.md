---
name: tf2ap-deps
description: Update the dependencies of tf2-archipelago and the two repositories it builds on, m-this/tf2-mvm-bots-go and m-this/sigsegv-mvm-win. Use after a TF2 update (the tf2-build-watch issue "TF2 server build N is live"), when asked to bump SourceMod, Metamod, SigMod, the bots, Archipelago or the tooling, to check what is outdated, or to look for fixes worth taking from rafradek/sigsegv-mvm upstream. Covers the order the game-keyed pins must move in, how each one is proven, and the traps met doing it.
---

# Updating tf2-archipelago's dependencies

Three repositories, one chain. A TF2 update breaks them from the bottom up, so
fix them in this order and prove each step before the next.

| Repository | What it pins | Where |
| --- | --- | --- |
| m-this/sigsegv-mvm-win | SigMod for Windows (branch `main`) and for Linux (branch `master`, rafradek's code rebuilt) | `gamedata/sigsegv/windows.txt`, `tools/winport/overrides.json` |
| m-this/tf2-mvm-bots-go | the defender bots and the SourceMod they compile with | `plugin/testbed/versions.env` |
| m-this/tf2-archipelago | everything a server runs | `deploy/env/versions.env` (the only place a pin lives; read its comments first) |

The bots are pinned by tf2-archipelago's `go.mod`, not by `versions.env`:
`go get github.com/m-this/tf2-mvm-bots-go@vX.Y.Z` moves them.

## After a TF2 update

The watcher (`.github/workflows/tf2-build-watch.yml`) opens an issue naming the
new ServerVersion. It lists the pins keyed to the game. Work through it:

1. **Windows SigMod addresses.** In sigsegv-mvm-win, the scheduled `Address
   table` workflow (`addrs.yml`) carries `overrides.json` to the new build with
   `tools/winport/carry.py`. Dispatch it on `main` with `commit=true`. Read its
   log: "carried N of M overrides" and the list of entries that did not carry.
   Each one that did not carry is resolved by hand against the new
   `server.dll` (disassembly, call graph from known callers, the Linux
   `server_srv.so` symbols as reference) and added to `overrides.json`. Never
   guess an address: a wrong one faults at runtime, a missing one only drops a
   detour.
2. **Windows bed.** Dispatch `windows.yml` on the branch with
   `archipelago_ref=main`. It builds `package-windows.zip` and plays every
   SigMod community mission over 8 shards. A crash names a module and offset;
   look up the function and check whether an address it calls is unresolved
   ("called unresolved function" in the console) before suspecting the code.
3. **Windows release.** Push a date tag `YYYYMMDD` on `main`
   (`gh api repos/m-this/sigsegv-mvm-win/git/refs -f ref=refs/tags/YYYYMMDD -f sha=...`).
   The tag run builds, publishes the release, and plays the bed. Take the
   SHA-256 from the release asset (`gh release view TAG --json assets`), not
   from memory: re-running the workflow on a tag rebuilds and republishes the
   zip, and its checksum changes.
4. **Linux SigMod.** Look at `alliedmodders/hl2sdk` branch `tf2` since the last
   build. Unchanged SDK means the Linux build is the same binary: prove it on
   the new game build with `linuxbed.yml` (`archipelago_ref=main`, `bots=true`)
   and keep the pin. Changed SDK: dispatch `main.yml` (Build and Release) on
   `master`, play its artifact with `linuxbed.yml sigmod_run=<run id>`, then
   publish `linux-YYYYMMDD`.
5. **SourceMod.** A TF2 update can move the KeyValues layout SourceMod reads
   (alliedmodders/sourcemod#2587). The symptom is silent: bots shop and buy
   nothing. Prove purchases on the new build with `deploy/run-botprobe.sh`
   (credits drop between waves, refusals stay rare) before setting
   `SOURCEMOD_TF2_SERVER_VERSION`.
6. **Bump tf2-archipelago.** `SIGSEGV_MVM_WINDOWS_VERSION` and its SHA-256,
   `SIGSEGV_MVM_*` if a Linux release was made, `SOURCEMOD_TF2_SERVER_VERSION`,
   then `TF2_SERVER_VERSION` last: moving it is what closes the watcher's issue,
   so it says the rest was checked. Update the facts in each pin's comment
   (which run checked what), add a CHANGELOG entry in player terms, and close
   the issue from the PR.

## SourceMod, Metamod and the extensions

- Latest drops: `https://sm.alliedmods.net/smdrop/<branch>/sourcemod-latest-linux`
  and `https://mms.alliedmods.net/mmsdrop/<branch>/mmsource-latest-linux`.
- Stay on the newest drop of the stable branch. SourceMod 1.13 raised the
  extension API to 9 and refuses every extension built for 1.12 ("Extension
  version is too old to load (8, max is 9)"; the 9 is the minimum it accepts,
  the message calls it the max). SigMod (both platforms), REST in Pawn, Actions
  and CBaseNPC are therefore all built against the SourceMod commit of the pin:
  SigMod by m-this/sigsegv-mvm-win (`sourcemod_ref` input of `main.yml` on
  `master` and of `windows.yml` on `main`), the other three by
  `deploy/bots/build-extensions.sh`, which `.github/workflows/extensions.yml`
  runs on Linux and Windows and the image runs in a stage of its own. Moving
  the SourceMod commit means the patches under `deploy/patches/` still apply
  and the Extensions run builds, then both beds with the new bundle.
- `SOURCEMOD_SRC_COMMIT` is the commit the drop was built from, found from the
  build number in alliedmodders/sourcemod.
- The bots compile with their own drop (`SOURCEMOD_VERSION` in
  tf2-mvm-bots-go's `plugin/testbed/versions.env`) and require a runtime of at
  least `SOURCEMOD_RUNTIME_MIN`. Bump them together when the runtime moves.
- REST in Pawn, CBaseNPC, Actions, tf2attributes, TFEconData and tf2utils are
  built from source, patched from `deploy/patches/`. The first three are
  extensions (binaries per SourceMod API), the other three are plugins (bytecode:
  a plugin compiled by an older spcomp still runs on a newer SourceMod). A bump
  means the patches still apply; read `deploy/patches/README.md`.
- On 1.13 `Address` is an int64 and a plugin compiled there passes one to a
  native by reference, and gets one back through a hidden first parameter. A
  native that reads its Address with `GetNativeCell` (tf2attributes v1.7.5,
  every CBaseNPC native) then reads a stack address. The plugins are therefore
  on their int64 lines (TFEconData x64 tags, tf2attributes#71), and the CBaseNPC
  and Actions includes declare their pointer parameters `any`, which is what
  their C++ reads. Check a new release of any of them against this before
  bumping it.

## Upstream SigMod

rafradek/sigsegv-mvm `master` is merged into sigsegv-mvm-win `master`. The value
is in its open pull requests, which upstream does not merge often. For each:

```bash
cd sigsegv-mvm-win
git fetch https://github.com/rafradek/sigsegv-mvm pull/N/head:refs/remotes/upr/N
git cherry origin/master upr/N   # + means not taken yet
git cherry origin/main upr/N
```

Take a fix into `master` (Linux) and, where the code is shared, into `main`
(Windows), then play both beds. A patch that hardcodes bytes or an absolute
address for one game build will break on the next one: mask the address
instead of copying the new bytes.

## Tooling and everything else

The rest of `versions.env` (Go tools, buf, Node, ruff, Tailscale, HonKit,
Archipelago), `go.mod`, `launcher/web` npm packages, `uv.lock`, Docker base
images and GitHub Actions versions. Check each against its latest stable
release, respect pins that move together, and let CI prove the bump.

- `proto/buf.gen.yaml` pins the connect plugin to the major `go.mod` requires.
  An unpinned remote plugin moved to connect v2 once and broke the launcher.
- A golangci-lint that cannot read the Go toolchain's export data fails the
  `go` job on every PR, main included: bump it with the toolchain.

## Traps

- **Workflow files need the `workflow` scope.** The `gh` HTTPS token lacks it,
  so a push touching `.github/workflows/` is refused. Push those over SSH, or
  run `gh auth refresh -s workflow`.
- **Test against the current waveprobe.** The beds build the probe from
  tf2-archipelago at `archipelago_ref`. A failing wave is the probe's, the
  mission's or SigMod's: compare with a run on main before calling it a
  regression, and play a flaky mission alone several times (`probe_args`)
  before trusting one result.
- **This may be a production Docker host.** Never stop, kill or remove
  containers or volumes you did not create. Start probes with unique names and
  remove only those.
- **One fact, one place.** A version lives in `versions.env` (or `go.mod` for
  the bots). Never copy it into a second file.
- Commits are Conventional Commits with a short factual body. Pull requests are
  opened, never merged: the maintainer merges.
