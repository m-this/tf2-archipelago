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
MODS=/opt/tf2-mods
COMMUNITY=/opt/tf2-community-pack/tf
GAME="${STEAMAPPDIR}/${STEAMAPP}"
INTERVAL=30
RCON=${TF2AP_RCON:-/usr/local/bin/rcon}
ADMIN_RELOAD_ATTEMPTS=${TF2AP_ADMIN_RELOAD_ATTEMPTS:-120}
ADMIN_RELOAD_INTERVAL=${TF2AP_ADMIN_RELOAD_INTERVAL:-1}

tailscale_fastdl_url() {
	url_file=/run/tf2ap-fastdl/url
	# No upper bound, on purpose: the URL appears once a person has signed the
	# sidecar in from the admin UI, and nothing here can hurry them. The line
	# repeats so `docker compose logs srcds` reads as a wait, not a hang.
	waited=0
	while [ ! -s "$url_file" ]; do
		if [ $((waited % 60)) -eq 0 ]; then
			echo "[AP] waiting for Tailscale sign-in and Funnel setup in the admin UI" >&2
		fi
		sleep 2
		waited=$((waited + 2))
	done
	url="$(sed -n '1p' "$url_file")"
	case "$url" in
	https://*.ts.net/tf)
		host="${url#https://}"
		host="${host%/tf}"
		;;
	*)
		echo "[AP] refusing the invalid Tailscale FastDL URL in $url_file" >&2
		return 1
		;;
	esac
	case "$host" in
	"" | *[!A-Za-z0-9.-]*)
		echo "[AP] refusing the invalid Tailscale FastDL hostname in $url_file" >&2
		return 1
		;;
	esac
	printf 'https://%s/tf\n' "$host"
}

# SourceMod wants STEAM_0:X:Y. What a player actually has to hand is the 17
# digit id from their profile URL or steamid.io, so both are accepted and the
# long one is converted here rather than in the operator's head.
steam_id_for_sourcemod() {
	value="$1"
	case "$value" in
	*[!0-9]*) ;;
	"") ;;
	*)
		if [ "${#value}" -ge 17 ]; then
			account=$((value - 76561197960265728))
			printf 'STEAM_0:%d:%d' "$((account % 2))" "$((account / 2))"
			return 0
		fi
		;;
	esac
	printf '%s' "$value"
}

# SourceMod reads admins_simple.ini into a cache. On a fresh volume its own
# runtime installer creates SourceMod immediately before starting srcds, so our
# supervisor can only write the admin file after that cache was built. The same
# race happens when a changed .env meets an existing file on restart. Retry the
# reload in the background: the first attempts normally precede the RCON socket.
reload_admin_cache() {
	attempt=1
	while [ "$attempt" -le "$ADMIN_RELOAD_ATTEMPTS" ]; do
		"$RCON" sm_reloadadmins >/dev/null 2>&1
		case $? in
		0)
			echo "[AP] refreshed the SourceMod admin cache"
			return 0
			;;
		2)
			# The client refuses before it dials: no password to dial with,
			# and no amount of waiting grows one.
			echo "[AP] wrote the admin list, but SRCDS_RCONPW is unset, so the cache was not refreshed" >&2
			return 1
			;;
		esac
		attempt=$((attempt + 1))
		sleep "$ADMIN_RELOAD_INTERVAL"
	done
	echo "[AP] wrote the admin list, but SourceMod did not answer sm_reloadadmins" >&2
}

# BOT_FILES renders the bots' loadout file. PROVISION decides server.cfg, the
# command line and the mods loaded, with the launcher's own Go: the image and
# the launcher used to write each their own, and the two drifted both ways.
BOT_FILES=${TF2AP_BOT_FILES:-/usr/local/bin/tf2ap-botfiles}
PROVISION=${TF2AP_PROVISION:-/usr/local/bin/tf2ap-provision}

# install_bot_files writes what the bots carry, from the SRCDS_BOT_* variables.
#
# Into the staged tree rather than the game: sync_tree copies a staged file over
# whenever its size differs, so a file written straight into the game would be
# replaced by the one the image shipped within thirty seconds. Staging it keeps
# one delivery path for every file the image owns.
#
# The rendering is a Go command sharing the launcher's own packages. A shell
# version would be a second copy of the weapon catalogue, and the first TF2
# update to move an index would move only one of the two.
install_bot_files() {
	[ -x "$BOT_FILES" ] || return 0
	"$BOT_FILES" -root "$STAGE" >/dev/null || echo "[AP] could not write the bots' loadout file" >&2
}

# SourceMod identifies an admin by Steam id, so the operator's list is the whole
# configuration. Written rather than shipped: an admin list committed to the
# image would be one more place a Steam id lives.
#
# Separated by commas, spaces or newlines, so a growing list stays readable in
# an .env file. Rewritten only when it differs, because this runs inside the
# supervisor loop and the file belongs to whoever is playing once it is right.
install_admin() {
	target="$GAME/addons/sourcemod/configs/admins_simple.ini"
	[ -n "${SRCDS_ADMIN_STEAMIDS:-}" ] || return 0
	[ -d "$(dirname "$target")" ] || return 0

	# Staged next to the target rather than in /tmp: those two are separate
	# filesystems under rootless podman, and the mv that follows falls back to
	# a cross-device copy that needs to unlink the target, which the uid
	# mapping there refuses. Staged beside it, the mv is a same-filesystem
	# rename instead.
	staged=$(mktemp "$(dirname "$target")/.admins_simple.XXXXXX")
	{
		echo "// Managed by the tf2-archipelago image, from SRCDS_ADMIN_STEAMIDS."
		echo "// Edits here are replaced the next time the container starts."
		count=0
		for raw in $(printf '%s' "${SRCDS_ADMIN_STEAMIDS}" | tr ',\n\t' '   '); do
			admin=$(steam_id_for_sourcemod "$raw")
			[ -n "$admin" ] || continue
			echo "\"${admin}\" \"99:z\""
			count=$((count + 1))
		done
		echo "// ${count} admin(s)"
	} >"$staged"

	if cmp -s "$staged" "$target"; then
		rm -f "$staged"
		return 0
	fi
	mv "$staged" "$target"
	chmod 0644 "$target"
	echo "[AP] installed $(grep -c '^"' "$target") admin(s)"
	reload_admin_cache &
}

# Copies a staged tree over the game's, file by file, and only the files whose
# size differs or whose source is newer. cp -u decided on the timestamp alone,
# and a copy cut short by a full disk is newer than its source, so it stayed
# truncated and the server died loading it on every map. The size tells the two
# apart. Nothing that matches is rewritten: cp truncates a file the running
# server has mapped, and that is a SIGBUS thirty seconds later.
sync_tree() {
	src=$1
	dst=$2
	(cd "$src" && find . -type f) | while IFS= read -r file; do
		if [ ! -e "$dst/$file" ] || [ "$(stat -c %s "$src/$file")" != "$(stat -c %s "$dst/$file")" ] || [ "$src/$file" -nt "$dst/$file" ]; then
			mkdir -p "$(dirname "$dst/$file")"
			cp -f "$src/$file" "$dst/$file"
		fi
	done
}

# The server mods this run loads, as provision decides them from SRCDS_MODS,
# SRCDS_MOD_LOADING and the mission pool. Each one the image stages is a tree
# shaped like tf/ under $MODS/<key>.
install_mods() {
	# SourceMod loads an extension because a file named after it sits beside
	# it, and sync_tree only ever adds. So a mod taken out of the run went on
	# loading from the volume it was once synced into. Take the marker away
	# first, then put it back for the mods this run loads.
	rm -f "$GAME/addons/sourcemod/extensions/sigsegv.autoload"

	mods=$("$PROVISION" mods) || return 1
	for key in $mods; do
		if [ ! -d "$MODS/$key" ]; then
			echo "[AP] the run loads $key, which this image does not carry"
			continue
		fi
		sync_tree "$MODS/$key" "$GAME"
	done
}

# The game ships a sample server.cfg that sets "rcon_password changeme", and
# server.cfg runs on map load, after the command line. So the password the
# operator set would be replaced by a published default on the first map. That
# is the whole reason the file is generated rather than left alone. provision
# also turns SigMod's RED robot limit off in the convar file SigMod ships, so
# it runs after install_mods has synced that file.
install_server_cfg() {
	[ -d "$GAME/cfg" ] || return 0
	said=$("$PROVISION" install "$GAME") || return 1
	[ "$said" = changed ] || return 0
	echo "[AP] wrote server.cfg, rcon password from the environment"
	if [ "${TAILSCALE_FASTDL:-0}" = 1 ]; then
		echo "[AP] public Tailscale Funnel FastDL ready at ${SRCDS_DOWNLOADURL:-}"
	fi
}

install_plugin() {
	installed=0
	stockpop_attempts=0
	while true; do
		if [ -d "$GAME/addons/sourcemod/plugins" ]; then
			if [ "$installed" -eq 0 ]; then
				# Repair the _666 file's tank paths and zombie event setting,
				# then give it a selectable name without shipping Valve's
				# popfile in our image. The plugin maps the runtime name back
				# to the seed's stable _666 ID.
				if ! tf2ap-stockpop "$GAME" "$GAME/scripts/population/mvm_ghost_town_ap_caliginous_caper.pop"; then
					stockpop_attempts=$((stockpop_attempts + 1))
					if [ "$stockpop_attempts" -lt 5 ]; then
						echo "[AP] Caliginous Caper's stock mission is unavailable; retrying ($stockpop_attempts/5)" >&2
						sleep "$INTERVAL"
						continue
					fi
					echo "[AP] Caliginous Caper is unavailable after 5 attempts; installing the rest of the server" >&2
				fi
			fi
			# Before the sync, because what it writes is one of the files the
			# sync carries over, and server.cfg below reads what it decided.
			install_bot_files
			sync_tree "$STAGE/addons" "$GAME/addons"
			# -n for the config: it belongs to whoever runs the server once it
			# exists, and an operator who turns on tf2ap_debug should not find
			# it turned off again thirty seconds later.
			cp -rn "$STAGE/cfg/." "$GAME/cfg/" 2>/dev/null || true
			# Community packs use TF2's own directory layout, and the bind mount
			# stays editable between restarts.
			if [ -d "$COMMUNITY" ]; then
				sync_tree "$COMMUNITY" "$GAME"
			fi
			install_mods
			install_server_cfg
			install_admin
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

# Tests source the functions above without starting SteamCMD or srcds.
if [ "${TF2AP_ENTRYPOINT_LIBRARY:-0}" = 1 ]; then
	return 0
fi

# Funnel setup can happen after the stack starts. Waiting here keeps the admin
# UI reachable while ensuring SRCDS never starts with an empty download URL.
if [ "${TAILSCALE_FASTDL:-0}" = 1 ]; then
	SRCDS_DOWNLOADURL="$(tailscale_fastdl_url)"
	export SRCDS_DOWNLOADURL
	echo "[AP] using Tailscale Funnel FastDL at $SRCDS_DOWNLOADURL"
fi

install_plugin &

# TF2 writes its console to this file through -condebug. It is more reliable
# than a stdout pipe for the 32-bit server and is the same source the native
# launcher uses for the FakeIP allocation line. Rotate once per container
# start so the admin sidecar sees only this run.
console_log="${GAME}/console.log"
console_previous="${GAME}/console-previous.log"
mkdir -p "${GAME}"
if [ -f "${console_log}" ]; then
	mv -f "${console_log}" "${console_previous}"
fi

exec bash /usr/local/bin/tf2ap-srcds-launch.sh
