#!/bin/sh
# Compile the plugin.
#
# Fetches the pinned SourceMod compiler and the ripext includes into build/ the
# first time, then compiles. Same script by hand and in CI, so a compile that
# works here works there.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

# Versions live in one file for the whole project, so the compiler this builds
# against and the extension the image installs cannot drift apart.
versions="$root/../deploy/env/versions.env"
if [ ! -f "$versions" ]; then
	echo "missing $versions" >&2
	exit 1
fi
. "$versions"
build="$root/build"
toolchain="$build/sourcemod-$SOURCEMOD_VERSION"
ripext="$build/ripext-$RIPEXT_VERSION"
tf2attributes="$build/tf2attributes-$TF2ATTRIBUTES_VERSION"
tf_econ_data="$build/tf-econ-data-$TFECONDATA_VERSION"
tf2utils="$build/tf2utils-$TF2UTILS_VERSION"

mkdir -p "$build"

if [ ! -d "$toolchain" ]; then
	echo "fetching SourceMod $SOURCEMOD_VERSION"
	mkdir -p "$toolchain"
	curl -fsSL "https://sm.alliedmods.net/smdrop/$SOURCEMOD_BRANCH/sourcemod-$SOURCEMOD_VERSION-linux.tar.gz" |
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

if [ ! -d "$tf2attributes" ]; then
	echo "fetching tf2attributes $TF2ATTRIBUTES_VERSION includes"
	mkdir -p "$tf2attributes"
	curl -fsSL "https://github.com/FlaminSarge/tf2attributes/archive/refs/tags/$TF2ATTRIBUTES_VERSION.tar.gz" |
		tar xz -C "$tf2attributes" --strip-components=1
fi

if [ ! -d "$tf_econ_data" ]; then
	echo "fetching TF2 Econ Data $TFECONDATA_VERSION includes"
	mkdir -p "$tf_econ_data"
	curl -fsSL "https://github.com/nosoop/SM-TFEconData/archive/refs/tags/$TFECONDATA_VERSION.tar.gz" |
		tar xz -C "$tf_econ_data" --strip-components=1
fi

if [ ! -d "$tf2utils" ]; then
	echo "fetching TF2Utils $TF2UTILS_VERSION includes"
	mkdir -p "$tf2utils"
	curl -fsSL "https://github.com/nosoop/SM-TFUtils/archive/refs/heads/$TF2UTILS_VERSION.tar.gz" |
		tar xz -C "$tf2utils" --strip-components=1
fi

spcomp="$toolchain/addons/sourcemod/scripting/spcomp64"
[ -x "$spcomp" ] || spcomp="$toolchain/addons/sourcemod/scripting/spcomp"

cd "$root/scripting"
exec "$spcomp" \
	-i"$toolchain/addons/sourcemod/scripting/include" \
	-i"$ripext/addons/sourcemod/scripting/include" \
	-i"$tf2attributes/scripting/include" \
	-i"$tf_econ_data/scripting/include" \
	-i"$tf2utils/scripting/include" \
	-E \
	-o"$build/tf2_archipelago.smx" \
	tf2_archipelago.sp
