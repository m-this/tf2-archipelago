#!/bin/sh
# Compile the plugin.
#
# Fetches the pinned SourceMod compiler and the ripext includes into build/ the
# first time, then compiles. Same script by hand and in CI, so a compile that
# works here works there.
set -eu

SOURCEMOD_VERSION="1.12.0-git7246"
RIPEXT_VERSION="1.3.2"

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
build="$root/build"
toolchain="$build/sourcemod-$SOURCEMOD_VERSION"
ripext="$build/ripext-$RIPEXT_VERSION"

mkdir -p "$build"

if [ ! -d "$toolchain" ]; then
	echo "fetching SourceMod $SOURCEMOD_VERSION"
	mkdir -p "$toolchain"
	curl -fsSL "https://sm.alliedmods.net/smdrop/1.12/sourcemod-$SOURCEMOD_VERSION-linux.tar.gz" |
		tar xz -C "$toolchain"
fi

if [ ! -d "$ripext" ]; then
	echo "fetching ripext $RIPEXT_VERSION"
	mkdir -p "$ripext"
	curl -fsSL -o "$build/ripext.zip" \
		"https://github.com/ErikMinekus/sm-ripext/releases/download/$RIPEXT_VERSION/sm-ripext-$RIPEXT_VERSION-linux.zip"
	unzip -oq "$build/ripext.zip" -d "$ripext"
	rm -f "$build/ripext.zip"
fi

spcomp="$toolchain/addons/sourcemod/scripting/spcomp64"
[ -x "$spcomp" ] || spcomp="$toolchain/addons/sourcemod/scripting/spcomp"

cd "$root/scripting"
exec "$spcomp" \
	-i"$toolchain/addons/sourcemod/scripting/include" \
	-i"$ripext/addons/sourcemod/scripting/include" \
	-E \
	-o"$build/tf2_archipelago.smx" \
	tf2_archipelago.sp
