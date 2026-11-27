---
name: update-sigmod
description: Port SigMod (m-this/sigsegv-mvm-win) to a new TF2 build on Windows and Linux, and take fixes from rafradek/sigsegv-mvm and potato-tf upstream. Use when a TF2 update breaks SigMod (a crash at load or during a wave, "BInitAttributes failed", attribute names not found, mods in FAILED state, a server that hangs at wave start), or when asked to update, port or rebuild SigMod. tf2ap-deps covers the pins and releases that come after.
---

# Porting SigMod to a new TF2 build

SigMod hooks the game by address on Windows and by symbol on Linux, so a TF2
update breaks the two in different ways. Port both on one branch of
m-this/sigsegv-mvm-win (`mfe/fix/tf2-<build>`), prove each on its bed, and
only then move to the releases and pins in tf2ap-deps.

Get both builds of both binaries first: `server.dll` and `server_srv.so`, the
one SigMod was made for and the new one (steamcmd `download_depot` with the
manifests from `tools/winport/table.json`). Every step below compares them.

## Windows: the address table

1. Dispatch `Address table` (`addrs.yml`). It carries `overrides.json` to the
   new build with `carry.py` and commits `winport-auto-<build>`.
2. **The carry drops every entry whose function body changed, and its report
   does not list them all.** On 11267535 it named 67 and dropped 263. Diff the
   entry names of `gamedata/sigsegv/windows.txt` between `main` and the carry
   commit. That diff is the work list.
3. A dropped entry fails silently. A missing thunk returns false with no
   error: `CEconItemSchema::BInitAttributes` did, and all 389 SigMod attribute
   names went missing. A missing detour only turns its mod off.
4. Find each dropped function with `tools/winport/fuzzy.py OLD.dll NEW.dll
   RVA...`. Accept a pick only after checking it by its callers, its vtable
   slot and the strings it reads. Restore it with `build "<new build>"` in the
   format of its neighbours.
5. Check every `vtidx` of a class whose base gained a virtual. On 11267535,
   `CGameRules` got one and every `CTFGameRules` slot after it moved by one.
6. Look for two names on one address. `CBaseObject::StartUpgrading` sat on
   `StartPlacement` for two builds: its detour declared no argument for a
   function that pops four bytes, and each building placed shifted the stack.
   `detourargs.py` checks parameter counts, not this.
7. `tools/winport/detourargs.py` must report 0 mismatches, and `checktable.py`
   must pass.

## Linux: a new compiler

Valve rebuilt `server_srv.so` with GCC 14 for 11267535. A compiler change breaks
more than an address change does:

- **Clones.** GCC splits functions into `.part.N`, `.isra.N` and
  `.constprop.N` clones that take arguments in registers (`regparm(3)`). Some
  clones disappear and some new ones appear. For a function whose clone is
  gone, detour the whole function with a plain `DETOUR_DECL_MEMBER` or
  `DETOUR_DECL_STATIC`. For a clone that remains, keep `__gcc_regcall`, and
  read its prologue to see which of eax, edx and ecx it reads before writing
  (`objdump -d --start-address`). The gamedata convention is that of
  `windows.txt`: keep the `"X [clone]"` key, `sym` names the whole function for
  x86, and `sym64` keeps the clone regex for x64. Each regex must match exactly
  one symbol: check every `sym` and `sym regex` against `nm` of the new binary.
- **Return types.** A function can gain a return value in the new build.
  `CUpgrades::PlayerPurchasingUpgrade` became `bool` on Linux, as it already was
  on Windows: its callers test `%al` after the call. A detour still declared
  `void` hands them garbage. Read the call sites, not the old declaration.
- **Byte patches.** PIC code reads globals through the GOT and moves stack
  slots, so a patch whose bytes no longer match fails `Check` and its mod shows
  FAILED in `sig_list_mods`. Rewrite the buffer for the new code, and make sure
  it is unique in its window. Touch only the `_LINUX` branch: Windows has its
  own bytes. Mask addresses instead of copying them, or the next build breaks
  the patch again.
- **Extractors** (`CProp_Extract`) read offsets out of instructions. Find the
  site again in the new function.
- **Wrapper checks.** Since Linux now calls the whole function, a guard that
  only the Windows wrapper had (null player, empty nav mesh, ignored entity)
  must apply on both platforms.

## Proving it

- **Linux:** run `main.yml` on the branch, then `linuxbed.yml` with
  `sigmod_run=<run id>` and `bots=true`. The bed installs the latest successful
  Extensions run of `archipelago_ref`, so check that it carries the CBaseNPC and
  TF2Utils the new build needs. In `sig_list_mods`, no mod may be FAILED.
- **Windows:** run `windows.yml` on the branch. It plays every community
  mission over 8 shards. It runs without defender bots by default, and the Linux
  bed runs with them: before calling a mission a Windows regression, play it
  with `bots=true` on both. To replay one mission several times, list it in
  `tools/winport/bed/focus.txt` on a throwaway branch. On 11267535, trespasser
  lost two out of three without bots whatever the build.
- **A hang** leaves no core. `linuxbed.yml` takes the stacks of every thread
  once rcon stops answering. Read them before guessing.
- In both consoles, check that `BInitAttributes` passes, that the attribute
  count is right, that bots spawn and that waves are played.

## Taking from upstream

rafradek/sigsegv-mvm and potato-tf/sigsegv-mvm often fix the same update
first. Read their commits and open pull requests. potato-tf #30 was the
starting point for 11267535, and its `void PlayerPurchasingUpgrade` was wrong.
Before taking any change:

- **It must build and behave on Windows.** Upstream builds for Linux only. A
  change to a detour signature, a `#if defined _LINUX` block, a gamedata entry or
  an extractor needs its Windows counterpart: the address in `windows.txt`, the
  `_WINDOWS` branch, or `IExtractStub`. Run `detourargs.py` and both beds.
- Prove each claim against the new binary, as you would your own code.
- Keep the upstream author on commits taken as they are.

## Traps

- **Line endings.** Many files in sigsegv-mvm-win are CRLF. A script that opens
  them in text mode writes LF and the diff becomes the whole file. Read and
  write in binary, or with `newline=""`.
- **`detourargs.py` and `PLATFORM_64BITS`.** It reads the x86 Windows build, so
  a 64-bit-only declaration must count as not compiled. If a mismatch makes no
  sense, check that condition first.
- **One branch, two platforms.** Windows and Linux fixes land on the same
  branch. Pull with `--rebase` before every push, and do not let two people
  edit `windows.txt` at once.
