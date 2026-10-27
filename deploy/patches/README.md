# Patches to the defender bots and their dependencies

`deploy/bots/build.sh` applies these to the pinned upstream checkouts before it
compiles. A patch that no longer applies fails the build. That is the signal
to rebase it on the new upstream, or to drop it because upstream fixed it.

Every tree patched here is a clone of somebody else's repository. That is the
rule and not an accident: `git apply` resolves a patch's paths from the
repository root rather than from the directory it is given, so a tree with no
`.git` of its own is patched against this repository, where those paths do not
exist, and the patch is skipped without a word. A checkout brings its own
`.git`; a directory copied out of the module cache does not.

Which is why the defender mod is not here, and cannot be. It is
`m-this/tf2-mvm-bots-go`, a Go module this repository depends on, so its changes
are made there and released as a tag. `go get
github.com/m-this/tf2-mvm-bots-go@vX.Y.Z` is how this build takes a new one: the
go.mod requirement is the only pin, and there is nothing in `versions.env` to
move with it. A fix carried here instead was carried nowhere: the patch never
applied, and when a cached build tree happened to hold its output anyway, the
generated-code check failed on it.

| Patch | Why |
| --- | --- |
| `cbasenpc/0001-refresh-tools-nav-cache-on-map-start.patch` | Only for a from-source build of the extension. Rebuilds CBaseNPC's tools-side navigation cache after every map activates, so Defender Bots do not receive an empty area list on community-map changelevels. |
| `metamod/0001-sourcehook-read-clang-cl-vcall-thunks.patch` | Only for the Windows CBaseNPC built by `deploy/bots/build-cbasenpc-windows.sh`. SourceHook finds a virtual function's slot by reading the compiler's vcall thunk and knows only MSVC's; clang-cl's thunk loads the slot into a register first. Without it every `VCall<>::Init(&Class::Method)` throws and the server shuts down at map start. |
| `cbasenpc/0002-build-on-sourcemod-1.13.patch` | SourceMod 1.13 merged `IPluginContext` into `IPluginRuntime`. `FindPluginByContext` takes the context itself now, so `context->GetContext()` is gone; `GetFunctionByNameEx`, which walked `GetPublicsNum`/`GetPublicByIndex` and which nothing calls, is dropped. |
| `cbasenpc/0003-address-is-a-cell-in-this-extension.patch` | SourceMod 1.13 makes `Address` an int64, and a plugin compiled there passes one to a native by reference. CBaseNPC's natives and callbacks read and push a 32-bit cell (`PawnAddressToPtr(params[n])`, `PushCell(PtrToPawnAddress(...))`), so every `Address` its includes declare is `any`, which is what crosses; the deprecated `CAnimationOverlay` does its arithmetic in that cell. Includes only. Upstream has no 1.13 work (TF2-DMB/CBaseNPC#77, closed not planned). m-this/tf2-mvm-bots-go carries the same patch for its test-bed. |
| `cbasenpc/0004-compute-to-target-reads-no-vector.patch` | `Path.ComputeToTarget` passes its entity index to `LocalToPhysAddr` and reads a goal vector from it that nothing uses, left over from `ComputeToPos`. SourcePawn 1.12 answered with a pointer into the plugin's memory and the read was harmless. 1.13 refuses the address and leaves the pointer unset, so the read faults (`PawnVectorToVector`, `vecAddr=0x16`) as soon as a defender bot paths to a robot, and the server dies a few seconds into the wave. The dead lines go. |
| `actions/0001-do-not-treat-sdk-warnings-as-errors.patch` | Only for a from-source build. `-Werror` against the current hl2sdk fails on `#pragma warning` blocks in `vstdlib/random.h`. |
| `actions/0002-offset-a-cell-not-an-address.patch` | `BehaviorAction.Get` and `Set` cast the methodmap to `Address`, which 1.13 refuses: a methodmap is a cell and `Address` is an int64. The sum is taken in the cell and widened when it is passed. Includes only; the defender mod's test-bed carries the same patch. |
| `actions/0003-push-a-vector-as-three-cells.patch` | `PushArray` takes a cell count, and the propagation passes `sizeof(Vector)`, 12, for a `Vector` argument or return. With `SM_PARAM_COPYBACK` on `SelectTargetPoint` the plugin's 12 cells are copied back over a 3-cell `Vector`, which overwrites the lambda next to it on the stack, and the next call through it faults (`ProcessHandlerEx<Vector ...>`, `shared.h:212`). Every TFBot runs `SelectTargetPoint` each tick once the defender mod listens on it, robots included, so it crashed the server within seconds of a wave start, even at `SRCDS_BOTS=0`, where the mod stays loaded. |
| `actions/0004-keep-the-rtti-words-in-front-of-a-vtable-copy.patch` | v4 runs every handler on a per-action copy of the vtable (`ProcessorVtableManager::slot`), but copies only the function slots. `vptr[-2]` and `vptr[-1]`, offset-to-top and the type info, then read the slot's own pointers, so a `dynamic_cast` on an action inside a handler faults in `__dynamic_cast`. SigMod does one in `EngieBot_Wrangler`'s `OnCommandString` detour, which an `interrupt_action` in a popfile reaches (bigrock reverse, condemned, creepside). The patch copies the two words from the action's own vtables. |
| `tf2attributes/0001-gamedata-for-tf2-11138802.patch` | The pinned commit is FlaminSarge/tf2attributes#71 from June, before TF2 11087207 and 11138802 moved the two Windows signatures (`GetAttributeDefinitionByName`, then `GetAttributeDefinition`, whose table moved from `+0x1BC` to `+0x22C`). These are upstream's (`747a80b`, `eb4006a8`) with the PR's `windows64` lines kept. Drop it when #71 is merged into a tag that carries both. |
| `tf2utils/0001-narrow-an-offset-read-as-an-address.patch` | An offset read through the plugin's `GameConfGetAddressOffset` goes into an `any`, and 1.13 refuses the implicit int64 narrowing. The value came from `GameConfGetOffset`, an int, so `view_as<int>` loses nothing. |
| `sourcemod/0001-dhooks-set-int64-address-on-virtual-hooks.patch` | SourceMod 1.13.0-git7477 to git7481 never set `DHooksInfo::int64_address` for a virtual hook (`DHooksManager` in `vhook.cpp`; only the detour path does). A callback declared `(Address pThis, ...)` then gets either the pointer or, when the byte happens to be nonzero, the address of an int64 on the plugin's stack, so `view_as<IVision>(pThis).GetBot()` faults in CBaseNPC with a small address. Linux mostly reads zero from fresh pages; a Windows server read garbage on every map start. The extension is built from the pinned SourceMod commit and replaces the drop's `dhooks.ext`. Drop it when a drop carries the fix. |
| `sourcemod/0002-build-only-dhooks.patch` | Only for `build-extensions.sh`. The root `AMBuildScript` builds every extension, the core and the plugins; this leaves dhooks. |
| `ripext/0001-build-on-sourcemod-1.13.patch` | `sm_stringhashmap.h` is `sm_hashmap.h` in 1.13, and `StringHashMap` lives in `SourceMod`. `-Werror` is dropped, because the bundled zlib's K&R definitions are an error to clang 15 and later. The package script no longer copies `ca-bundle.crt`, which upstream's CI generates from a download; the image and the launcher keep the release's copy, and only take `rip.ext` from this build. |

Every extension is built from source, and so is DHooks: SourceMod 1.13 raised the extension API to
9 and refuses anything built for 8, which is every upstream release of CBaseNPC,
Actions and REST in Pawn. `build-extensions.sh` builds the three against the
SourceMod commit `versions.env` pins, on Linux for the Docker image (a stage of
its own) and on a Windows runner for the launcher
(`.github/workflows/extensions.yml`). The CBaseNPC patch that matters at runtime
is 0001, the map-lifecycle fix.

Two more facts the build depends on, explained in the scripts. The defender
mod and the three gamedata plugins compile with the `spcomp` the mod names in
its `plugin/testbed/versions.env`, a 1.13 drop. Actions is built from master at
`ACTIONS_COMMIT`, not from the v4.0.1 tag, which predates the API its own
release ships.
