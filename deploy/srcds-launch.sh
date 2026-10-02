#!/bin/bash
# Prepare the base TF2 installation, then start SRCDS the same way as the
# native launcher. In particular, -condebug is the reliable source for the
# FakeIP allocation line; 32-bit SRCDS block-buffers a redirected stdout pipe.
#
# Not -autoupdate, which the base image's entry.sh passes: with it, srcds_run
# reruns SteamCMD over the game files on every crash restart, under the
# SourceMod tree the entrypoint's loop is syncing. The one update below, at
# container start, is the update.
set -eu

mkdir -p "${STEAMAPPDIR}"

steamcmd() {
	bash "${STEAMCMDDIR}/steamcmd.sh" "$@"
}

update_log=$(mktemp)
update_game() {
	steamcmd \
		+force_install_dir "${STEAMAPPDIR}" \
		+login anonymous \
		+app_update "${STEAMAPPID}" \
		+quit 2>&1 | tee "$update_log"
	return "${PIPESTATUS[0]}"
}

# A failed update can leave StateFlags 6 in the app manifest, and every later
# app_update, validate included, then stops at "state is 0x6 after update job".
# Moving the manifest aside clears it, but only after SteamCMD has run once
# bare and once logged in, as a fresh SteamCMD needs: app_update straight after
# the move fails with "Missing configuration". One recovery per start, no more.
manifest="${STEAMAPPDIR}/steamapps/appmanifest_${STEAMAPPID}.acf"
status=0
update_game || status=$?
if grep -q "state is 0x6" "$update_log"; then
	aside="${manifest}.0x6-$(date -u +%Y%m%dT%H%M%SZ)"
	echo "[AP] the TF2 update is stuck at state 0x6: moving $manifest to $aside and updating again" >&2
	if [ -f "$manifest" ]; then
		mv "$manifest" "$aside"
	fi
	steamcmd +quit || echo "[AP] SteamCMD exited $? on its bare run, carrying on" >&2
	steamcmd +login anonymous +quit || echo "[AP] SteamCMD exited $? on its warm-up login, carrying on" >&2
	status=0
	update_game || status=$?
	if grep -q "state is 0x6" "$update_log"; then
		echo "[AP] the TF2 update is still stuck at state 0x6 after moving $manifest aside; the old manifest is kept as $aside" >&2
		exit 8
	fi
fi
rm -f "$update_log"
if [ "$status" != 0 ]; then
	echo "[AP] the TF2 update failed, SteamCMD exited $status" >&2
	exit "$status"
fi

# Match the base image's first-run setup. The game directory is a volume, so
# these disappear with a fresh volume even though the image itself has them.
if [ -n "${METAMOD_VERSION:-}" ] && [ ! -d "${STEAMAPPDIR}/${STEAMAPP}/addons/metamod" ]; then
	latest_mm=$(wget -qO- "https://mms.alliedmods.net/mmsdrop/${METAMOD_VERSION}/mmsource-latest-linux")
	wget -qO- "https://mms.alliedmods.net/mmsdrop/${METAMOD_VERSION}/${latest_mm}" |
		tar xvzf - -C "${STEAMAPPDIR}/${STEAMAPP}"
fi
if [ -n "${SOURCEMOD_VERSION:-}" ] && [ ! -d "${STEAMAPPDIR}/${STEAMAPP}/addons/sourcemod" ]; then
	latest_sm=$(wget -qO- "https://sm.alliedmods.net/smdrop/${SOURCEMOD_VERSION}/sourcemod-latest-linux")
	wget -qO- "https://sm.alliedmods.net/smdrop/${SOURCEMOD_VERSION}/${latest_sm}" |
		tar xvzf - -C "${STEAMAPPDIR}/${STEAMAPP}"
fi

cd "${STEAMAPPDIR}"

flags=(
	-game "${STEAMAPP}"
	-usercon
	-console
	-condebug
	-debug
	-maxplayers "${SRCDS_MAXPLAYERS}"
	-nowatchdog
)
commands=(
	+map "${SRCDS_STARTMAP}"
	+hostport "${SRCDS_PORT}"
	+rcon_password "${SRCDS_RCONPW}"
	+sv_lan "${SRCDS_LAN}"
)

if [ "${SRCDS_LAN}" = 0 ]; then
	flags+=( -ip 0.0.0.0 )
fi
if [ "${SRCDS_SECURED:-1}" = 0 ]; then
	flags+=( -insecure )
fi
if [ "${SRCDS_SDR_FAKEIP}" = 1 ]; then
	flags+=( -enablefakeip )
	commands+=( +sv_use_steam_networking 1 )
fi
case "${SRCDS_TOKEN:-0}" in
"" | 0) ;;
*)
	if [ "${SRCDS_LAN}" = 0 ]; then
		commands+=( +sv_setsteamaccount "${SRCDS_TOKEN}" )
	fi
	;;
esac
if [ -n "${SRCDS_PW:-}" ]; then
	commands+=( +sv_password "${SRCDS_PW}" )
fi
if [ -n "${SRCDS_WORKSHOP_AUTHKEY:-}" ]; then
	flags+=( -authkey "${SRCDS_WORKSHOP_AUTHKEY}" )
fi
if [ "${SRCDS_REPLAY:-0}" = 1 ]; then
	flags+=( -replay )
fi

exec bash "${STEAMAPPDIR}/srcds_run" "${flags[@]}" "${commands[@]}"
