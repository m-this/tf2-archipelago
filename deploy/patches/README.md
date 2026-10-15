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
| `tf2attributes/0001-drop-pragma-unused-before-declaration.patch` | `#pragma unused` sits above the function it names, and spcomp 1.12 resolves it before the declaration exists. Nothing compiles without this. |
| `tf2attributes/0002-gamedata-for-tf2-11087207.patch` | TF2 11087207 rewrote `CEconItemSchema::GetAttributeDefinitionByName` as a lookup in a new name map, so the Windows signature matches nothing. The new one wildcards the map's offset. Drop it once upstream ships a gamedata for this build. |
| `actions/0001-drop-libudis86-and-asm-sources.patch` | Only for a from-source build of the extension. SourceMod removed `public/libudis86` and `public/asm` in commit `e07c120c`; the AMBuilder still lists them. |
| `actions/0002-do-not-treat-sdk-warnings-as-errors.patch` | Only for a from-source build. `-Werror` against the current hl2sdk fails on `#pragma warning` blocks in `vstdlib/random.h`. |
| `actions/0003-terminate-handle-instead-of-assigning-index.patch` | Only for a from-source build. `CBaseHandle::operator=` takes an `IHandleEntity*`, not an index. |
| `cbasenpc/0002-build-on-sourcemod-1.13.patch` | SourceMod 1.13 merged `IPluginContext` into `IPluginRuntime`. `FindPluginByContext` takes the context itself now, so `context->GetContext()` is gone; `GetFunctionByNameEx`, which walked `GetPublicsNum`/`GetPublicByIndex` and which nothing calls, is dropped. |
| `actions/0004-build-on-sourcemod-1.13.patch` | Two headers forward-declare `SourcePawn::IPluginContext` as a class. It is a typedef of `IPluginRuntime` in 1.13, so they include `sp_vm_api.h` instead. |
| `sourcemod/0001-dhooks-set-int64-address-on-virtual-hooks.patch` | SourceMod 1.13.0-git7477 never sets `DHooksInfo::int64_address` for a virtual hook (`DHooksManager` in `vhook.cpp`; only the detour path does). A callback declared `(Address pThis, ...)` then gets either the pointer or, when the byte happens to be nonzero, the address of an int64 on the plugin's stack, so `view_as<IVision>(pThis).GetBot()` faults in CBaseNPC with a small address. Linux mostly reads zero from fresh pages; a Windows server read garbage on every map start. The extension is built from the pinned SourceMod commit and replaces the drop's `dhooks.ext`. Drop it when a drop carries the fix. |
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
mod compiles with SourceMod 1.12.0-git7164's `spcomp`, and git7246's segfaults
on it. Actions stays at v3.9.2: v4.0.0 has no TF2 build, and its TF2 source
does not compile.
