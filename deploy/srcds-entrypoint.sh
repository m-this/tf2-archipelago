#!/bin/bash
# Install the plugin into the game volume, then hand over to the image's own
# entrypoint.
#
# The install cannot happen before SourceMod does, and SourceMod is downloaded
# by that same entrypoint, into the volume, on first start. So the install runs
# in the background and waits for SourceMod to appear.
#
# It keeps watching for the container's life rather than installing once: the
# game auto-updates, SourceMod can be reinstalled under it, and a plugin that
# quietly stopped being there is the kind of failure that gets blamed on the
# randomizer. The loop is deliberately unbounded; it is a supervisor, not a
# retry.
set -eu

STAGE=/opt/tf2-archipelago
GAME="${STEAMAPPDIR}/${STEAMAPP}"
INTERVAL=30

install_plugin() {
	installed=0
	while true; do
		if [ -d "$GAME/addons/sourcemod/plugins" ]; then
			cp -r "$STAGE/." "$GAME/"
			if [ "$installed" -eq 0 ]; then
				echo "[AP] installed the plugin and ripext into $GAME"
				installed=1
			fi
		elif [ "$installed" -eq 1 ]; then
			echo "[AP] SourceMod went missing, waiting for it to come back"
			installed=0
		fi
		sleep "$INTERVAL"
	done
}

install_plugin &

exec bash "${HOMEDIR}/entry.sh"
