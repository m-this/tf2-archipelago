#!/bin/sh
# Stage the MvM defender bot stack.
#
# Valve tunes every wave for six defenders, so a solo or small-team run needs
# the empty RED slots filled with bots that play. That is seven upstream
# projects. This script produces one staged SourceMod tree in $out that serves
# both a Linux and a Windows server, because SourceMod picks the .so or the
# .dll by platform and ignores the other.
#
# Two kinds of artifact, handled differently:
#
#   - The two compiled extensions (CBaseNPC, Actions) are downloaded from the
#     pinned upstream releases, 32-bit for Linux and Windows. TF2's dedicated
#     server is 32-bit on both. Building them needs a C++ toolchain and several
#     CPU-minutes; nothing we change is in them. BOTS_BUILD_EXTENSIONS=1 builds
#     them from source instead, which is the path to take when a TF2 update
#     moves a signature and the fix has to be ours. See build-extensions.sh.
#
#   - The four SourcePawn plugins are compiled here, always. They are bytecode,
#     so one build serves both platforms, spcomp takes seconds, and compiling
#     is what lets us carry the compile fix TF2Attributes needs. Compiling
#     TF2Attributes rather than shipping its release also keeps us off a
#     binary that has no license on it.
#
#     The defender mod itself is m-this/tf2-mvm-bots-go, which holds both the Go
#     that authors the decisions and the SourcePawn tree they compile into,
#     under plugin/. It is a Go module dependency and nothing else: the go.mod
#     requirement is the pin, the module download is the source, and `go get`
#     is how it moves. There is no tag for it in versions.env, because a second
#     place to say which version this build runs is a second place to be wrong.
#
# Every other version comes from deploy/env/versions.env, source fixes from
# deploy/patches/, whose README explains every patch.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$root/deploy/env/versions.env"

work=${BOTS_WORK:-$root/deploy/bots/build}
out=${BOTS_OUT:-$work/package}
patches="$root/deploy/patches"
src="$work/src"

mkdir -p "$src" "$out/addons/sourcemod/plugins" \
	"$out/addons/sourcemod/extensions" \
	"$out/addons/sourcemod/gamedata" \
	"$out/addons/sourcemod/configs"

git config --global advice.detachedHead false 2>/dev/null || true

# $1 repo, $2 ref, $3 directory
#
# The checkouts survive between runs, here and in CI's cache, and that is worth
# having: a clone of seven repositories is most of this script's time. It is
# only worth having for the ref that is asked for now, though. Keeping whatever
# is on disk meant a version bump built the previous version and said nothing,
# which does not fail: it ships, labelled as the new one.
#
# A stamp beside the checkout rather than asking git what it holds: a --depth 1
# --branch clone knows one ref and cannot say whether some other tag would have
# been the same commit without going back to the network.
fetch() {
	stamp="$src/$3.ref"
	if [ -d "$src/$3" ]; then
		[ "$(cat "$stamp" 2>/dev/null)" = "$2" ] && return 0
		echo "re-fetching $3: held $(cat "$stamp" 2>/dev/null || echo "an unknown ref"), wants $2"
		rm -rf "$src/$3"
	fi
	echo "fetching $1@$2"
	git clone --quiet --depth 1 --branch "$2" "https://github.com/$1" "$src/$3"
	printf '%s\n' "$2" >"$stamp"
}

# A patch that no longer applies is a build failure, never a silent skip: it is
# the signal that upstream moved and the fix needs rebasing or dropping.
#
# Already applied is not that failure. The checkouts survive between runs, here
# and in CI's cache, so a second run finds its own work: --reverse --check asks
# whether this exact patch is already in the tree and moves on if it is.
#
# Every tree this patches is a clone with a .git of its own, and that is load
# bearing: git apply resolves the patch's paths from the repository root rather
# than from the directory it was given, so the same call in a tree with no .git
# looks for them under this repository, finds nothing and skips the patch
# without a word. The defender mod was such a tree, and its patch had never
# applied here.
apply_patches() {
	dir="$src/$1"
	for patch in "$patches/$1"/*.patch; do
		[ -e "$patch" ] || return 0
		git -C "$dir" apply --reverse --check "$patch" 2>/dev/null && continue
		echo "applying $(basename "$patch")"
		git -C "$dir" apply --whitespace=nowarn "$patch"
	done
}

# --- Sources for the plugins ---

fetch OfficerSpy/SM_Stock_OfficerSpy "$SM_STOCK_OFFICERSPY_REF" stocklib
fetch FlaminSarge/tf2attributes "$TF2ATTRIBUTES_VERSION" tf2attributes
fetch nosoop/SM-TFEconData "$TFECONDATA_VERSION" tf_econ_data
fetch nosoop/SM-TFUtils "$TF2UTILS_VERSION" tf2utils
fetch nosoop/stocksoup "$STOCKSOUP_REF" stocksoup-root/stocksoup

# --- The defender mod, from the pinned Go module ---
#
# One pin, and it is the go.mod requirement. The module is the mod: the Go that
# authors the decisions, the generator that turns it into SourcePawn, and the
# plugin tree that compiles. So there is nothing to clone here and no second
# version to hold in step with the one the generator runs from.
#
# Copied out of the module cache rather than compiled in place. The cache is
# read only and shared, and everything below treats $src as a tree it may stage
# and patch.
defenderbots_module=github.com/m-this/tf2-mvm-bots-go

command -v go >/dev/null 2>&1 || {
	echo "the defender mod is a Go module dependency, so this needs a Go toolchain" >&2
	exit 1
}

( cd "$root" && go mod download "$defenderbots_module" )
defenderbots_version=$(cd "$root" && go list -m -f '{{.Version}}' "$defenderbots_module")
defenderbots_dir=$(cd "$root" && go list -m -f '{{.Dir}}' "$defenderbots_module")

# The same stamp rule the checkouts use, and for the same reason: a tree left
# from the previous version builds and says nothing.
defenderbots_stamp="$src/defenderbots.ref"
if [ ! -d "$src/defenderbots" ] || [ "$(cat "$defenderbots_stamp" 2>/dev/null)" != "$defenderbots_version" ]; then
	echo "staging $defenderbots_module@$defenderbots_version"
	rm -rf "$src/defenderbots"
	cp -a "$defenderbots_dir" "$src/defenderbots"
	# The module cache is 0444 all the way down, and a later run has to be
	# able to replace this.
	chmod -R u+w "$src/defenderbots"
	printf '%s\n' "$defenderbots_version" >"$defenderbots_stamp"
fi

# --- The generated SourcePawn, checked against the pinned generator ---
#
# The mod's tree carries a committed copy of the generated files, because its
# own build is a shell script and a compiler. Here we have Go, so the files are
# regenerated and compared. A mismatch means the copy in the mod's tree is not
# what its generator writes, which is the drift the copies exist to risk.
#
# Against the module cache rather than the staged copy, so the question stays
# the one worth asking: does the pinned module agree with its own generator.
# Nothing here patches the staged tree any more, and a patch on a generated
# file is exactly what made this check fail on a build that reused its cache.
generate_from_module() {
	gen=$(mktemp -d)
	# go tool, not go run: the version comes from the tool directive in
	# go.mod, so `go get -tool` moves the mod and nothing else has to be
	# edited to agree with it.
	( cd "$root" && go tool github.com/m-this/tf2-mvm-bots-go/cmd/gen \
		-plugin "$defenderbots_dir/plugin" -out "$gen" ) || {
		echo "the mod's generator failed" >&2
		rm -rf "$gen"
		return 1
	}

	drift=0
	for file in "$gen"/sourcepawn/*.sp; do
		name=$(basename "$file")
		for committed in "$defenderbots_dir/plugin/source/redbots3/generated/$name" \
			"$defenderbots_dir/plugin/testbed/stats/generated/$name"; do
			[ -f "$committed" ] || continue
			cmp -s "$file" "$committed" || {
				echo "drift: $committed is not what the generator writes" >&2
				diff -u "$committed" "$file" | head -40 >&2
				drift=1
			}
		done
	done
	rm -rf "$gen"
	[ "$drift" = 0 ] || return 1

	echo "the generated SourcePawn matches the pinned generator"
}

generate_from_module

# The include roots the plugins compile against. Only the includes are needed,
# so these are checkouts of source we never build.
fetch TF2-DMB/CBaseNPC "$CBASENPC_VERSION" cbasenpc
fetch Vinillia/actions.ext "$ACTIONS_VERSION" actions

apply_patches tf2attributes

# --- The compiler ---
#
# spcomp from SOURCEMOD_VERSION, which plugin/build.sh uses for our own plugin,
# segfaults on the defender mod: upstream issue #5, no diagnostic, exit 139.
# This is the drop the mod's own CI uses, and it compiles it.
if [ ! -d "$work/spcomp" ]; then
	echo "fetching SourceMod $DEFENDERBOTS_SOURCEMOD_VERSION"
	mkdir -p "$work/spcomp"
	curl -fsSL "https://sm.alliedmods.net/smdrop/$SOURCEMOD_BRANCH/sourcemod-$DEFENDERBOTS_SOURCEMOD_VERSION-linux.tar.gz" |
		tar xz -C "$work/spcomp"
fi
sm="$work/spcomp/addons/sourcemod/scripting"

ripext_include="$work/ripext/addons/sourcemod/scripting/include/ripext.inc"
if [ ! -f "$ripext_include" ]; then
	echo "fetching ripext $RIPEXT_VERSION"
	rm -rf "$work/ripext"
	# Download before creating the cache directory. If curl is interrupted, the
	# next build cannot mistake an empty directory for a complete dependency.
	curl -fsSL -o "$work/ripext.zip" \
		"https://github.com/ErikMinekus/sm-ripext/releases/download/$RIPEXT_VERSION/sm-ripext-$RIPEXT_VERSION-linux.zip"
	mkdir -p "$work/ripext"
	unzip -oq "$work/ripext.zip" -d "$work/ripext"
	rm -f "$work/ripext.zip"
fi

# --- The extensions ---

if [ "${BOTS_BUILD_EXTENSIONS:-0}" = 1 ]; then
	BOTS_WORK="$work" BOTS_OUT="$out" sh "$root/deploy/bots/build-extensions.sh"
else
	if [ ! -d "$work/prebuilt" ]; then
		mkdir -p "$work/prebuilt/cbasenpc-linux" "$work/prebuilt/cbasenpc-windows" \
			"$work/prebuilt/actions"

		echo "fetching CBaseNPC $CBASENPC_VERSION"
		curl -fsSL "https://github.com/TF2-DMB/CBaseNPC/releases/download/$CBASENPC_VERSION/cbasenpc${CBASENPC_VERSION}_linux.tar.gz" |
			tar xz -C "$work/prebuilt/cbasenpc-linux"
		curl -fsSL -o "$work/cbasenpc-windows.zip" \
			"https://github.com/TF2-DMB/CBaseNPC/releases/download/$CBASENPC_VERSION/cbasenpc${CBASENPC_VERSION}_windows.zip"
		unzip -oq "$work/cbasenpc-windows.zip" -d "$work/prebuilt/cbasenpc-windows"

		# One zip carries every game and both architectures. TF2 srcds is
		# 32-bit on Linux and on Windows, so the x64 copies are not taken.
		echo "fetching Actions $ACTIONS_VERSION"
		curl -fsSL -o "$work/actions.zip" \
			"https://github.com/Vinillia/actions.ext/releases/download/$ACTIONS_VERSION/actions.ext.zip"
		unzip -oq "$work/actions.zip" -d "$work/prebuilt/actions"
		rm -f "$work/cbasenpc-windows.zip" "$work/actions.zip"
	fi

	cp "$work/prebuilt/cbasenpc-linux/addons/sourcemod/extensions/cbasenpc.ext.2.tf2.so" \
		"$work/prebuilt/cbasenpc-windows/addons/sourcemod/extensions/cbasenpc.ext.2.tf2.dll" \
		"$work/prebuilt/actions/actions.ext/extensions/actions.ext.2.tf2.so" \
		"$work/prebuilt/actions/actions.ext/extensions/actions.ext.2.tf2.dll" \
		"$out/addons/sourcemod/extensions/"

	# The Windows zip's gamedata is the same file as the Linux one; either does.
	cp "$work/prebuilt/cbasenpc-linux/addons/sourcemod/gamedata/cbasenpc.txt" \
		"$out/addons/sourcemod/gamedata/"
	cp "$src/actions/sourcemod/gamedata/actions.games.txt" \
		"$out/addons/sourcemod/gamedata/"
fi

# --- The plugins ---

# One include root per project. Never flatten them into a single directory:
# several projects ship a vector.inc, and the wrong one shadows SourceMod's.
#
# Compile from WSL's native filesystem. SourcePawn's include traversal is
# pathologically slow on a repository mounted through /mnt/<drive>; staging
# these small source trees turns a multi-minute apparent hang into seconds.
# Downloads and final artifacts remain in $work and $out respectively.
compile_work=$(mktemp -d "${TMPDIR:-/tmp}/tf2ap-bots-spcomp.XXXXXX")
trap 'rm -rf "$compile_work"' EXIT HUP INT TERM
cp -a "$sm" "$compile_work/scripting"
cp -a "$src" "$compile_work/src"
mkdir -p "$compile_work/ripext"
cp -a "$work/ripext/addons/sourcemod/scripting/include" "$compile_work/ripext/include"

compile_sm="$compile_work/scripting"
compile_src="$compile_work/src"

compile() {
	name=$(basename "$1" .sp)
	echo "compiling $name"
	"$compile_sm/spcomp64" \
		-i"$compile_sm/include" \
		-i"$compile_src/stocklib" \
		-i"$compile_src/stocksoup-root" \
		-i"$compile_src/cbasenpc/scripting/include" \
		-i"$compile_src/actions/sourcemod/include" \
		-i"$compile_work/ripext/include" \
		-i"$compile_src/tf2attributes/scripting/include" \
		-i"$compile_src/tf_econ_data/scripting/include" \
		-i"$compile_src/tf2utils/scripting/include" \
		-i"$(dirname "$1")" \
		-o"$out/addons/sourcemod/plugins/$name.smx" \
		"$1" >"$work/$name.log" 2>&1 ||
		{ cat "$work/$name.log"; exit 1; }
}

compile "$compile_src/defenderbots/plugin/source/tf2_defenderbots.sp"
compile "$compile_src/tf2attributes/scripting/tf2attributes.sp"
compile "$compile_src/tf_econ_data/scripting/tf_econ_data.sp"
compile "$compile_src/tf2utils/scripting/tf2utils.sp"

cp "$src/defenderbots/plugin/gamedata/tf2.defenderbots.txt" \
	"$src/tf2attributes/gamedata/tf2.attributes.txt" \
	"$src/tf_econ_data/gamedata/tf2.econ_data.txt" \
	"$src/tf2utils/gamedata/tf2.utils.nosoop.txt" \
	"$out/addons/sourcemod/gamedata/"

# Per-map navigation hints and bot names. Without these the bots path badly on
# the Valve maps, which is most of what this stack is for. The names are the
# game's own TFBot names rather than the mod's list, so the team reads like a
# Valve server.
cp -r "$src/defenderbots/plugin/configs/defenderbots" "$out/addons/sourcemod/configs/"
cp "$root/deploy/bots/bot_names.txt" "$out/addons/sourcemod/configs/defenderbots/bot_names.txt"

echo "staged the defender bots into $out"
