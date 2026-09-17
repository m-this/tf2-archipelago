#!/bin/sh
# Build CBaseNPC and Actions from source, for Linux, into $BOTS_OUT.
#
# The Docker image takes this path so it carries the patches in
# deploy/patches/cbasenpc. `make bots-from-source` takes it on a laptop, which
# is how a TF2 update that moves a signature CBaseNPC detours gets a fix of ours
# before upstream ships one.
#
# Every source is fetched by the commit versions.env names, never by a branch,
# so one commit of this repository builds one binary and a cache keyed on these
# inputs cannot hold a stale one. Submodules follow from their parent's commit.
#
# Linux only: the Windows .dll needs MSVC. Needs git, python3, clang and a
# 32-bit toolchain (gcc-multilib g++-multilib). TF2's dedicated server is
# 32-bit, and CBaseNPC has no 64-bit build at all.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$root/deploy/env/versions.env"

work=${BOTS_WORK:-$root/deploy/bots/build}
out=${BOTS_OUT:-$work/package}
patches="$root/deploy/patches"
src="$work/src"

# $1 repository, $2 commit, $3 directory under $src, then the directories a
# sparse checkout keeps, if any.
#
# A checkout already at the commit is reused and reset, so the patches always
# apply to upstream's tree and never to the previous run's.
checkout() {
	repo=$1 commit=$2 dir="$src/$3"
	shift 3
	if [ "$(git -C "$dir" rev-parse HEAD 2>/dev/null)" != "$commit" ]; then
		echo "fetching $repo@$commit"
		rm -rf "$dir"
		git init --quiet "$dir"
		git -C "$dir" remote add origin "https://github.com/$repo"
		filter=
		if [ $# -gt 0 ]; then
			git -C "$dir" sparse-checkout set --cone "$@"
			filter=--filter=blob:none
		fi
		git -C "$dir" fetch --quiet --depth 1 $filter origin "$commit"
		git -C "$dir" checkout --quiet --detach FETCH_HEAD
	fi
	git -C "$dir" reset --quiet --hard
	git -C "$dir" clean --quiet -fdx

	head=$(git -C "$dir" rev-parse HEAD)
	[ "$head" = "$commit" ] || {
		echo "$repo: checked out $head, versions.env pins $commit" >&2
		exit 1
	}
}

# $1 repository, $2 tag, $3 commit. The release download and the include
# checkout in build.sh go by the tag, this build by the commit, so the two have
# to name the same source.
tag_is() {
	ref=$(git ls-remote --exit-code "https://github.com/$1" "refs/tags/$2")
	tagged=${ref%%"	"*}
	[ "$tagged" = "$3" ] || {
		echo "$1: tag $2 is $tagged, versions.env pins $3" >&2
		exit 1
	}
}

# CBaseNPC registers safetyhook over SSH, which no build host has a key for.
submodules() {
	dir="$src/$1"
	git -C "$dir" -c url."https://github.com/".insteadOf=git@github.com: \
		submodule update --quiet --init --recursive --depth 1
	status=$(git -C "$dir" submodule status --recursive)
	unpinned=$(printf '%s\n' "$status" | grep -v '^ ' || true)
	[ -z "$unpinned" ] || {
		echo "$1: submodules not at the commits it records:" >&2
		echo "$unpinned" >&2
		exit 1
	}
}

apply_patches() {
	for patch in "$patches/$1"/*.patch; do
		[ -e "$patch" ] || return 0
		echo "applying $1/$(basename "$patch")"
		git -C "$src/$1" apply --whitespace=nowarn "$patch"
	done
}

tag_is TF2-DMB/CBaseNPC "$CBASENPC_VERSION" "$CBASENPC_COMMIT"
tag_is Vinillia/actions.ext "$ACTIONS_VERSION" "$ACTIONS_COMMIT"

# The SDK is 633 MB checked out whole and the build reads four directories of
# it. game/server is one of them: enginecallback.h lives there.
checkout alliedmodders/hl2sdk "$HL2SDK_COMMIT" hl2sdk-tf2 public game common lib/public/linux
checkout alliedmodders/sourcemod "$SOURCEMOD_SRC_COMMIT" sourcemod
checkout alliedmodders/metamod-source "$METAMOD_SRC_COMMIT" metamod
checkout alliedmodders/ambuild "$AMBUILD_COMMIT" ambuild
checkout TF2-DMB/CBaseNPC "$CBASENPC_COMMIT" cbasenpc
checkout Vinillia/actions.ext "$ACTIONS_COMMIT" actions

submodules sourcemod
submodules cbasenpc
submodules actions

apply_patches cbasenpc
apply_patches actions

# The tf2 manifest points the linker at lib/linux; the 32-bit libraries it
# actually links against live in lib/public/linux. CBaseNPC resolves the real
# path itself, Actions does not.
ln -sfn public/linux "$src/hl2sdk-tf2/lib/linux"

# AMBuild runs from its checkout rather than a pip install: nothing touches
# the host's Python and nothing unpinned comes from PyPI.
export PYTHONPATH="$src/ambuild" PATH="$src/ambuild/scripts:$PATH"

# clang, not gcc: Actions declares a __cdecl function-pointer alias in a
# template that gcc rejects outright.
export CC=clang CXX=clang++

# $1 directory under $src, then extra configure arguments.
build() {
	name=$1
	shift
	build="$src/$name/build"
	echo "building $name"
	mkdir "$build"
	# Both extensions print __DATE__. Dated by their commit, a rebuild of the
	# same pins on another day is the same binary.
	SOURCE_DATE_EPOCH=$(git -C "$src/$name" log -1 --format=%ct)
	export SOURCE_DATE_EPOCH
	# One && chain: set -e does not apply inside the left side of an ||.
	(cd "$build" &&
		python3 ../configure.py --hl2sdk-root "$src" --mms-path "$src/metamod" \
			--sm-path "$src/sourcemod" --sdks tf2 --targets x86 "$@" &&
		python3 -c 'from ambuild2.run import cli_run; cli_run()') >"$src/$name.log" 2>&1 || {
		cat "$src/$name.log" >&2
		exit 1
	}
}

# --extension-only: the bundled example plugin does not compile with a 1.12
# compiler, and nothing here ships it.
build cbasenpc --extension-only
build actions

mkdir -p "$out/addons/sourcemod/extensions" "$out/addons/sourcemod/gamedata"
cp "$src/cbasenpc/build/package/addons/sourcemod/extensions/cbasenpc.ext.2.tf2.so" \
	"$src/actions/build/package/addons/sourcemod/extensions/actions.ext.2.tf2.so" \
	"$out/addons/sourcemod/extensions/"
cp "$src/cbasenpc/build/package/addons/sourcemod/gamedata/cbasenpc.txt" \
	"$src/actions/build/package/addons/sourcemod/gamedata/actions.games.txt" \
	"$out/addons/sourcemod/gamedata/"

echo "built the extensions from source into $out"
