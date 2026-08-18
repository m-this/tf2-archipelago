#!/bin/sh
# Build CBaseNPC and Actions from source, for Linux.
#
# Not on the normal path: build.sh downloads the pinned releases instead,
# because nothing we change lives in these two and a C++ build costs several
# CPU-minutes. Run this (BOTS_BUILD_EXTENSIONS=1 ./deploy/bots/build.sh) when a
# TF2 update moves a signature CBaseNPC detours and the fix has to be ours
# before upstream ships it. Then put the fix in deploy/patches/ and it applies
# here.
#
# Linux only. The Windows .dll needs MSVC, so on Windows the pinned upstream
# release is the only option.
#
# Needs git, curl, python3-pip, clang and a 32-bit toolchain (gcc-multilib
# g++-multilib). TF2's dedicated server is 32-bit, and CBaseNPC has no 64-bit
# build at all.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$root/deploy/env/versions.env"

work=${BOTS_WORK:-$root/deploy/bots/build}
out=${BOTS_OUT:-$work/package}
patches="$root/deploy/patches"
src="$work/src"

# CBaseNPC registers its safetyhook submodule over SSH, which no build host has
# a key for.
git config --global url."https://github.com/".insteadOf "git@github.com:"
git config --global advice.detachedHead false

submodules() {
	git -C "$src/$1" submodule update --init --recursive --depth 1 --quiet
}

# The SDK is 633 MB checked out whole and the build reads four directories of
# it. game/server is one of them: enginecallback.h lives there.
if [ ! -d "$src/hl2sdk-tf2" ]; then
	echo "fetching alliedmodders/hl2sdk@$HL2SDK_BRANCH"
	git clone --quiet --depth 1 --branch "$HL2SDK_BRANCH" --filter=blob:none --sparse \
		https://github.com/alliedmodders/hl2sdk "$src/hl2sdk-tf2"
	git -C "$src/hl2sdk-tf2" sparse-checkout set --cone public game common lib/public/linux
fi

if [ ! -d "$src/sourcemod" ]; then
	echo "fetching alliedmodders/sourcemod@$SOURCEMOD_SRC_BRANCH"
	git clone --quiet --depth 1 --branch "$SOURCEMOD_SRC_BRANCH" \
		https://github.com/alliedmodders/sourcemod "$src/sourcemod"
	submodules sourcemod
fi

if [ ! -d "$src/metamod" ]; then
	echo "fetching alliedmodders/metamod-source@$METAMOD_SRC_BRANCH"
	git clone --quiet --depth 1 --branch "$METAMOD_SRC_BRANCH" \
		https://github.com/alliedmodders/metamod-source "$src/metamod"
fi

# build.sh already cloned both of these for their includes; only the submodules
# and the patches are missing.
submodules cbasenpc
submodules actions
# Already applied is not a failure: the checkout survives between runs, so a
# second run finds its own work. See apply_patches in build.sh.
for patch in "$patches/actions"/*.patch; do
	[ -e "$patch" ] || break
	git -C "$src/actions" apply --reverse --check "$patch" 2>/dev/null && continue
	echo "applying $(basename "$patch")"
	git -C "$src/actions" apply --whitespace=nowarn "$patch"
done

# The tf2 manifest points the linker at lib/linux; the 32-bit libraries it
# actually links against live in lib/public/linux. CBaseNPC resolves the real
# path itself, Actions does not.
ln -sfn public/linux "$src/hl2sdk-tf2/lib/linux"

if [ ! -d "$work/ambuild" ]; then
	# AMBuild is not on PyPI under this name; the package comes from the repo.
	git clone --quiet --depth 1 https://github.com/alliedmodders/ambuild "$work/ambuild"
	pip install --quiet --break-system-packages "$work/ambuild"
fi

# clang, not gcc: Actions declares a __cdecl function-pointer alias in a
# template that gcc rejects outright.
export CC=clang CXX=clang++

# $1 directory, $2 extra configure arguments
build() {
	echo "building $1"
	rm -rf "$src/$1/build"
	mkdir -p "$src/$1/build"
	# shellcheck disable=SC2086 # $2 is a deliberate argument list.
	(cd "$src/$1/build" && python3 ../configure.py \
		--hl2sdk-root "$src" \
		--mms-path "$src/metamod" \
		--sm-path "$src/sourcemod" \
		--sdks tf2 --targets x86 $2 >/dev/null)
	(cd "$src/$1/build" && ambuild >/dev/null)
}

# --extension-only because the bundled example plugin does not compile with a
# 1.12 compiler and nothing here ships it. Configure wants a spcomp regardless.
sm="$work/spcomp/addons/sourcemod/scripting"
build cbasenpc "--extension-only --spcomp-path $sm/spcomp64 --sm-api-path $sm/include"
build actions ""

cp "$src/cbasenpc/build/package/addons/sourcemod/extensions/cbasenpc.ext.2.tf2.so" \
	"$src/actions/build/package/addons/sourcemod/extensions/actions.ext.2.tf2.so" \
	"$out/addons/sourcemod/extensions/"
cp "$src/cbasenpc/build/package/addons/sourcemod/gamedata/cbasenpc.txt" \
	"$src/actions/build/package/addons/sourcemod/gamedata/actions.games.txt" \
	"$out/addons/sourcemod/gamedata/"

echo "built the extensions from source into $out"
