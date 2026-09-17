#!/usr/bin/env bash
# Build CBaseNPC for Windows, with this repository's patches, from Linux.
#
# build-extensions.sh builds the Linux extensions from source so the in-tree
# patches under deploy/patches/cbasenpc reach Docker. The Windows launcher has
# only ever taken CBaseNPC's released binary, which does not carry them, and the
# one that matters is 0001: without it the tools-side navigation cache keeps the
# previous map's areas after a changelevel. On Windows that is not a quiet bug.
# The defender bots walk those areas at the next wave start and the server never
# returns from the walk: measured under Wine, the main thread looping between
# CBaseNPC and server.dll in a nav connection walk, with nothing logged since the
# last bot finished shopping. tf2_archipelago changes mission by changelevel, so
# every mission switch on Windows meets it.
#
# Toolchain, the same one the SigMod port uses: clang-cl and lld-link targeting
# i686-pc-windows-msvc, Microsoft's SDK and CRT from xwin, and hl2sdk-tf2's own
# x86 import libraries.
#
#     deploy/bots/build-cbasenpc-windows.sh
#
# Writes deploy/bots/build/windows/cbasenpc.ext.2.tf2.dll.
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
work="${BOTS_WORK:-$root/deploy/bots/build}"
src="$work/src/cbasenpc"
patches="$root/deploy/patches"
out="$work/windows"
obj="$out/obj"

XWIN="${XWIN:-$HOME/xwin-sdk}"
AM="${ALLIEDMODDERS:-$HOME/projects/alliedmodders}"
SDK="$AM/hl2sdk-tf2"
SM="$work/src/sourcemod"
MMS="$work/src/metamod"
JOBS="${JOBS:-$(nproc)}"

. "$root/deploy/env/versions.env"

# SourceMod and Metamod are the checkouts build-extensions.sh builds Linux
# against, fetched the same way when it has not run. The SourceMod version
# matters: CBaseNPC's AddCDetour expects the CDetour built on SafetyHook, and a
# SourceMod older than that one references asm.c's copy_bytes instead. The SDK
# comes from a full checkout, since that build's sparse one leaves out lib/public/x86.
if [ ! -d "$SM" ]; then
	git clone --quiet --depth 1 --branch "$SOURCEMOD_SRC_BRANCH" \
		https://github.com/alliedmodders/sourcemod "$SM"
	git -C "$SM" submodule update --init --recursive --depth 1 --quiet
fi
if [ ! -d "$MMS" ]; then
	git clone --quiet --depth 1 --branch "$METAMOD_SRC_BRANCH" \
		https://github.com/alliedmodders/metamod-source "$MMS"
fi

for path in "$src/extension" "$src/third_party/safetyhook/src" "$SDK/lib/public/x86" \
	"$SM/public/CDetour" "$MMS/core" "$XWIN/crt/lib/x86"; do
	[ -e "$path" ] || { echo "missing: $path" >&2; exit 1; }
done
command -v clang-cl >/dev/null || { echo "clang-cl not on PATH" >&2; exit 1; }
command -v lld-link >/dev/null || { echo "lld-link not on PATH" >&2; exit 1; }

# The same already-applied-is-not-a-failure rule build-extensions.sh uses: the
# checkout survives between runs and a second run finds its own work.
#
# metamod/0001 is Windows-only by its nature. SourceHook finds a virtual
# function's slot by reading the compiler's vcall thunk, and knew only MSVC's.
# Built with clang-cl, every VCall<>::Init(&Class::Method) in CBaseNPC came back
# "not virtual" and threw from SDK_OnLoad, which ends the server at map start.
apply_patches() {
	for patch in "$patches/$1"/*.patch; do
		[ -e "$patch" ] || break
		git -C "$2" apply --reverse --check "$patch" 2>/dev/null && continue
		echo "applying $1/$(basename "$patch")"
		git -C "$2" apply --whitespace=nowarn "$patch"
	done
}
apply_patches cbasenpc "$src"
apply_patches metamod "$MMS"

rm -rf "$obj"
mkdir -p "$obj/safetyhook" "$obj/cbasenpc"

# Microsoft's headers first, then the engine's. /MT because the CRT is linked
# statically, the way CBaseNPC's own Windows release links it, so the DLL needs
# no runtime on the server. /arch:SSE2 is what MSVC assumes for x86 unasked;
# clang-cl targets a plain i686 and refuses the SDK's SSE intrinsics without it.
common=(
	--target=i686-pc-windows-msvc /nologo /MT /Ox /Oy- /W0 /arch:SSE2 -fms-compatibility
	/imsvc "$XWIN/crt/include" /imsvc "$XWIN/sdk/include/ucrt"
	/imsvc "$XWIN/sdk/include/um" /imsvc "$XWIN/sdk/include/shared"
	/D_CRT_SECURE_NO_DEPRECATE /D_CRT_SECURE_NO_WARNINGS /D_CRT_NONSTDC_NO_DEPRECATE
	/D_ITERATOR_DEBUG_LEVEL=0 /DNDEBUG /DWIN32 /D_WINDOWS
)

# SafetyHook, with its own flags: its AMBuilder resets everything before setting
# them, and Zydis is C.
sh="$src/third_party/safetyhook"
shflags=("${common[@]}" /I"$sh/include" /I"$sh/zydis")

# CBaseNPC's own SDK table numbers TF2 11, not the 12 hl2sdk-manifests uses, and
# its sources compare against its own numbering, so it is built with its own.
ext="$src/extension"
cxxflags=(
	"${common[@]}" /TP /EHsc /std:c++17
	/FI"$root/deploy/bots/msvc-sdk-prelude.h"
	/DRAD_TELEMETRY_DISABLED /DSE_TF2=11 /DSOURCE_ENGINE=11
	/DCOMPILER_MSVC /DCOMPILER_MSVC32
	/I"$ext" /I"$ext/sdk" /I"$ext/sourcesdk" /I"$ext/sourcesdk/NextBot"
	/I"$ext/sourcesdk/NextBot/Path" /I"$ext/natives" /I"$ext/shared"
	/I"$SM/public" /I"$SM/public/extensions" /I"$SM/sourcepawn/include"
	/I"$SM/public/amtl/amtl" /I"$SM/public/amtl"
	/I"$MMS/core" /I"$MMS/core/sourcehook"
	/I"$SDK/public" /I"$SDK/public/engine" /I"$SDK/public/mathlib" /I"$SDK/public/vstdlib"
	/I"$SDK/public/tier0" /I"$SDK/public/tier1" /I"$SDK/public/game/server"
	/I"$SDK/public/toolframework" /I"$SDK/game/shared" /I"$SDK/game/server" /I"$SDK/common"
	/I"$sh/include"
)

# The source list, read out of the extension's own AMBuilder so a CBaseNPC bump
# needs nothing here. Only plain entries: SourceMod's smsdk_ext.cpp is listed
# there by os.path.join and added below, as AddCDetour adds detours.cpp.
mapfile -t sources < <(python3 - "$ext/AMBuilder" <<'PY'
import re, sys
text = open(sys.argv[1]).read()
block = re.search(r"project\.sources\s*=\s*\[(.*?)\]", text, re.S).group(1)
for name in re.findall(r"^\s*'([^']+\.cpp)',?\s*$", block, re.M):
    print(name)
PY
)
sources=("${sources[@]/#/$ext/}" "$SM/public/smsdk_ext.cpp" "$SM/public/CDetour/detours.cpp")

failed=0
echo "safetyhook: $(find "$sh/src" -name '*.cpp' ! -name 'os.linux.cpp' | wc -l) sources and Zydis"
for file in "$sh"/src/*.cpp "$sh/zydis/Zydis.c"; do
	[ "$(basename "$file")" = os.linux.cpp ] && continue
	echo "$file"
done | xargs -P "$JOBS" -I{} bash -c '
	f="{}"; o="'"$obj"'/safetyhook/$(basename "$f").obj"
	case "$f" in
	*.c) clang-cl '"$(printf '%q ' "${shflags[@]}")"' /TC -c "$f" /Fo"$o" ;;
	*) clang-cl '"$(printf '%q ' "${shflags[@]}")"' /TP /EHsc /std:c++17 -c "$f" /Fo"$o" ;;
	esac' || failed=1

echo "cbasenpc: ${#sources[@]} sources"
printf '%s\n' "${sources[@]}" | xargs -P "$JOBS" -I{} bash -c '
	f="{}"
	rel="${f//\//_}"
	clang-cl '"$(printf '%q ' "${cxxflags[@]}")"' -c "$f" /Fo"'"$obj"'/cbasenpc/${rel##*_extension_}.obj" \
		> "'"$obj"'/cbasenpc/${rel##*_extension_}.log" 2>&1 \
		|| { echo "FAILED $f"; head -20 "'"$obj"'/cbasenpc/${rel##*_extension_}.log"; exit 255; }' || failed=1

[ "$failed" -eq 0 ] || { echo "compile failed" >&2; exit 1; }

echo "linking"
lld-link /nologo /DLL /MACHINE:X86 /OUT:"$out/cbasenpc.ext.2.tf2.dll" \
	/LIBPATH:"$XWIN/crt/lib/x86" /LIBPATH:"$XWIN/sdk/lib/um/x86" /LIBPATH:"$XWIN/sdk/lib/ucrt/x86" \
	"$obj"/safetyhook/*.obj "$obj"/cbasenpc/*.obj \
	"$SDK/lib/public/x86/tier0.lib" "$SDK/lib/public/x86/tier1.lib" \
	"$SDK/lib/public/x86/vstdlib.lib" "$SDK/lib/public/x86/mathlib.lib" \
	legacy_stdio_definitions.lib kernel32.lib user32.lib advapi32.lib shell32.lib ole32.lib uuid.lib

echo "built $out/cbasenpc.ext.2.tf2.dll"
