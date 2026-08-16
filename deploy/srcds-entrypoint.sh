#!/bin/bash
# Install the plugin into the game volume, then hand over to the image's own
# entrypoint.
#
# SourceMod is downloaded by that entrypoint, into the volume, on first start,
# so the install waits for it in the background. It keeps watching for the
# container's life because the game auto-updates and SourceMod can be
# reinstalled under it: a deliberate supervisor loop, not a retry.
set -eu

STAGE=/opt/tf2-archipelago
GAME="${STEAMAPPDIR}/${STEAMAPP}"
INTERVAL=30

install_plugin() {
	installed=0
	while true; do
		if [ -d "$GAME/addons/sourcemod/plugins" ]; then
			# -u so an unchanged file is not rewritten every half minute.
			cp -ru "$STAGE/addons/." "$GAME/addons/"
			# -n for the config: it belongs to whoever runs the server once it
			# exists, and an operator who turns on tf2ap_debug should not find
			# it turned off again thirty seconds later.
			cp -rn "$STAGE/cfg/." "$GAME/cfg/" 2>/dev/null || true
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
