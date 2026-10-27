#!/bin/sh
# Build CBaseNPC, Actions, REST in Pawn and a patched DHooks from source into
# $BOTS_OUT.
#
# SourceMod 1.13 raised the extension API to 9 and refuses every extension built
# for 8, so none of the three can be taken from its release any more: they are
# rebuilt against the SourceMod commit versions.env pins. A release of any of
# them is only usable again once it is built against 1.13 itself.
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
# On Linux it needs git, python3, clang and a 32-bit toolchain (gcc-multilib
# g++-multilib). On a Windows runner it needs git, Python and MSVC's x86
# environment already set up, and writes the .dll files. TF2's dedicated server
# is 32-bit, and CBaseNPC has no 64-bit build at all.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$root/deploy/env/versions.env"

case $(uname -s) in
MINGW* | MSYS* | CYGWIN*) windows=1 ;;
*) windows=0 ;;
esac
python=${PYTHON:-python3}
[ "$windows" = 1 ] && python=${PYTHON:-python}

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
	elif [ $# -gt 0 ]; then
		# A checkout left by an older version of this script may lack a
		# directory the list now names; the blobs come from the promisor.
		git -C "$dir" sparse-checkout set --cone "$@"
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
tag_is ErikMinekus/sm-ripext "$RIPEXT_VERSION" "$RIPEXT_COMMIT"
# Actions has no tag on the commit its v4.0.1 release was built from, so it goes
# by ACTIONS_COMMIT alone, here and in build.sh (versions.env says why).

# The SDK is 633 MB checked out whole and the build reads seven directories of
# it. game/server is one of them: enginecallback.h lives there. tier1 and
# mathlib, and the utils/lzma tier1 includes, are the others: hl2sdk-manifests
# builds both libraries from their AMBuilder for
# an extension built on it (Actions is one), and finds nothing to link
# when they are missing. Windows then fails to link; Linux links anyway and
# leaves ConVar_Register, g_pCVar and V_snprintf undefined, so the extension
# fails to load.
if [ "$windows" = 1 ]; then
	sdk_libs=lib/public
else
	sdk_libs=lib/public/linux
fi
checkout alliedmodders/hl2sdk "$HL2SDK_COMMIT" hl2sdk-tf2 public game common tier1 mathlib utils/lzma $sdk_libs
checkout alliedmodders/sourcemod "$SOURCEMOD_SRC_COMMIT" sourcemod
checkout alliedmodders/metamod-source "$METAMOD_SRC_COMMIT" metamod
checkout alliedmodders/ambuild "$AMBUILD_COMMIT" ambuild
checkout TF2-DMB/CBaseNPC "$CBASENPC_COMMIT" cbasenpc
checkout Vinillia/actions.ext "$ACTIONS_COMMIT" actions
checkout ErikMinekus/sm-ripext "$RIPEXT_COMMIT" ripext

submodules sourcemod
submodules cbasenpc
submodules actions

apply_patches sourcemod
apply_patches cbasenpc
apply_patches actions
apply_patches ripext

# The tf2 manifest points the linker at lib/linux (lib/public on Windows); the
# 32-bit libraries it actually links against live in lib/public/linux
# (lib/public/x86). CBaseNPC resolves the real path itself, Actions does not.
if [ "$windows" = 1 ]; then
	cp "$src"/hl2sdk-tf2/lib/public/x86/*.lib "$src/hl2sdk-tf2/lib/public/"
else
	ln -sfn public/linux "$src/hl2sdk-tf2/lib/linux"
fi

# AMBuild runs from its checkout rather than a pip install: nothing touches
# the host's Python and nothing unpinned comes from PyPI.
if [ "$windows" = 1 ]; then
	export PYTHONPATH=$(cygpath -w "$src/ambuild")
else
	export PYTHONPATH="$src/ambuild" PATH="$src/ambuild/scripts:$PATH"
fi

# clang, not gcc: Actions declares a __cdecl function-pointer alias in a
# template that gcc rejects outright.
[ "$windows" = 1 ] || export CC=clang CXX=clang++

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
	# SourceMod is the tree the others are built against, and has no --sm-path.
	sm_path="--sm-path $src/sourcemod"
	[ "$name" = sourcemod ] && sm_path=
	# One && chain: set -e does not apply inside the left side of an ||.
	(cd "$build" &&
		"$python" ../configure.py --hl2sdk-root "$src" --mms-path "$src/metamod" \
			$sm_path --sdks tf2 --targets x86 "$@" &&
		"$python" -c 'from ambuild2.run import cli_run; cli_run()') >"$src/$name.log" 2>&1 || {
		cat "$src/$name.log" >&2
		exit 1
	}
}

# --extension-only: the bundled example plugin is not shipped, and its plugin
# build needs a compiler this one does not.
build cbasenpc --extension-only
build actions
build ripext --enable-optimize
# SourceMod 1.13.0-git7481 leaves DHooksInfo::int64_address unset on a virtual
# hook, so a callback's Address `this` is sometimes an int64 passed by reference
# (a stack address in the plugin) and sometimes a cell. Patched and built here
# until a drop carries the fix; patch 0002 makes the build produce only dhooks.
build sourcemod --enable-optimize --no-mysql --disable-auto-versioning

if [ "$windows" = 1 ]; then
	suffix=dll
else
	suffix=so
fi
mkdir -p "$out/addons/sourcemod/extensions" "$out/addons/sourcemod/gamedata"
cp "$src/cbasenpc/build/package/addons/sourcemod/extensions/cbasenpc.ext.2.tf2.$suffix" \
	"$src/actions/build/package/addons/sourcemod/extensions/actions.ext.2.tf2.$suffix" \
	"$src/ripext/build/package/addons/sourcemod/extensions/rip.ext.$suffix" \
	"$out/addons/sourcemod/extensions/"
cp "$(find "$src/sourcemod/build/extensions/dhooks" -name "dhooks.ext.$suffix" | head -n 1)" \
	"$out/addons/sourcemod/extensions/"
cp "$src/cbasenpc/build/package/addons/sourcemod/gamedata/cbasenpc.txt" \
	"$src/actions/build/package/addons/sourcemod/gamedata/actions.games.txt" \
	"$out/addons/sourcemod/gamedata/"

echo "built the extensions from source into $out"
