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

# BOT_FILES renders the bots' loadout file; bot_custom_loadouts is what it said
# the mod should do with it, read by install_server_cfg below.
BOT_FILES=${TF2AP_BOT_FILES:-/usr/local/bin/tf2ap-botfiles}
bot_custom_loadouts=0

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
	said=$("$BOT_FILES" -root "$STAGE") || {
		echo "[AP] could not write the bots' loadout file" >&2
		return 0
	}
	bot_custom_loadouts=$(printf '%s' "$said" | tail -n 1)
	case $bot_custom_loadouts in
	0 | 1) ;;
	*) bot_custom_loadouts=0 ;;
	esac
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

# Server mods a community mission can require, by the keys community.json
# uses, from SRCDS_MODS. Each one the image stages is a tree shaped like tf/
# under $MODS/<key>. A key nothing was staged for is a line in the log, and
# the seed generated with server_mods naming it will not find its missions.
install_mods() {
	# SourceMod loads an extension because a file named after it sits beside
	# it, and sync_tree only ever adds. So a mod taken out of SRCDS_MODS went
	# on loading from the volume it was once synced into, and there was no way
	# to stop it short of deleting the volume. Take the marker away first, then
	# put it back for the mods that are still named.
	rm -f "$GAME/addons/sourcemod/extensions/sigsegv.autoload"

	[ -n "${SRCDS_MODS:-}" ] || return 0
	for key in $(printf '%s' "${SRCDS_MODS}" | tr ',' ' '); do
		if [ ! -d "$MODS/$key" ]; then
			echo "[AP] SRCDS_MODS names $key, which this image does not carry"
			continue
		fi
		sync_tree "$MODS/$key" "$GAME"
	done
}

# The game ships a sample server.cfg that sets "rcon_password changeme", and
# server.cfg runs on map load, after the command line. So the password the
# operator set is replaced by a published default on the first map, on a port
# that is open to the network. That is the whole reason this file is generated
# rather than left alone.
#
# It also owns the handful of settings the stack really cares about, so nothing
# else has to be true about a file the game wrote.
install_server_cfg() {
	target="$GAME/cfg/server.cfg"
	[ -d "$(dirname "$target")" ] || return 0

	if [ "${SRCDS_BOTS:-1}" = 0 ]; then
		bots_mode=0
	else
		bots_mode=2
	fi

	download_url="${SRCDS_DOWNLOADURL:-}"
	if [ -z "$download_url" ] && [ -n "${FASTDL_HOST:-}" ]; then
		download_url="http://${FASTDL_HOST}:${FASTDL_PORT:-27080}/tf"
	fi
	download_cfg=""
	if [ -n "$download_url" ]; then
		download_cfg="sv_downloadurl \"${download_url}\""
	fi

	staged=$(mktemp)
	cat >"$staged" <<-CFG
	// Managed by the tf2-archipelago image. Edits here are replaced the next
	// time the container starts.
	//
	// Generated because the game's own sample sets rcon_password to a
	// published default, and this file runs after the command line that set
	// the real one.
	hostname "${SRCDS_HOSTNAME:-Mann vs Archipelago}"
	rcon_password "${SRCDS_RCONPW}"
	sv_password "${SRCDS_PW:-}"

	// Mann vs Machine needs 32 slots to host at all and puts six players on
	// RED, which is the number worth advertising.
	sv_visiblemaxplayers 6

	// A wave starts when enough players have readied up, and one is enough
	// here. A private server is a handful of friends, so one of them arriving
	// late is not a reason for the evening to stall, and it is what lets one
	// player start a wave alone.
	tf_mvm_min_players_to_start 1

	// Team Fortress 2 moves an idle player to spectator after
	// mp_idlemaxtime minutes. On a public server that frees a slot; here it
	// takes a friend off RED, the bots fill the seat, and the game then
	// refuses them back on a full team. Nobody wants their own seat given
	// away for stepping out to the kitchen.
	mp_idlemaxtime 0
	mp_idledealmethod 0

	// The defender bots. Valve tunes every wave for six players on RED, so a
	// run with fewer than that is unwinnable without them.
	//
	// Mode 2 is AUTO_BOTS: the mod fills RED when mvm_begin_wave fires and
	// tops it back up every second for the rest of the wave. Mode 0 leaves the
	// bots to an admin's !addbots, which is what SRCDS_BOTS=0 means here: the
	// plugins stay loaded, nothing spawns on its own.
	//
	// min_players -1 disables the mod's own ready-up gate. It defaults to 3
	// and counts RED before the wave, where a solo player has no bots yet, so
	// leaving it on blocks the F4 that would have spawned them.
	sm_redbots_manager_mode ${bots_mode}
	sm_redbots_manager_defender_team_size ${SRCDS_BOT_TEAM_SIZE:-6}
	sm_redbots_manager_min_players -1
	// Keep the bots between waves. The mod's default kicks every bot inside
	// the mvm_wave_complete handler and respawns a fresh set at the next
	// wave: five KickClient calls in the middle of the wave-end sequence.
	// With the default, the game server froze at wave clear and the player
	// lost the connection. With this, it has not. Nothing on a private
	// server wants the reroll anyway.
	sm_redbots_manager_kick_bots 0
	// Classes the bots never play, and the classes they fill RED with, in
	// order. A team named in the second beats the first.
	sm_redbots_manager_class_blacklist "${SRCDS_BOT_CLASS_BLACKLIST:-}"
	sm_redbots_manager_team_composition "${SRCDS_BOT_TEAM_COMP:-}"
	// Whether the bots are handed what configs/defenderbots/loadout.cfg says.
	// The image ships an example file there, so this is the whole of the
	// question: 1 only when the settings named a loadout for a seat or a class.
	sm_redbots_manager_use_custom_loadouts ${bot_custom_loadouts}
	// What the bots look like, none of which changes how they play: a hat
	// each, and an unusual effect on that hat.
	sm_redbots_manager_bot_hats ${SRCDS_BOT_HATS:-1}
	sm_redbots_manager_bot_hat_effects ${SRCDS_BOT_HAT_EFFECTS:-0}

	// The mission the run starts on, how long a cleared mission stays on
	// the scoreboard before the next one loads, and whether the bots'
	// purchases reach the chat.
	tf2ap_start_mission "${SRCDS_START_MISSION:-}"
	tf2ap_next_mission_delay ${TF2AP_NEXT_MISSION_DELAY:-30}
	tf2ap_bot_upgrades_chat ${TF2AP_BOT_UPGRADES_CHAT:-0}

	// LAN mode skips Steam authentication, and refuses everyone who is not on
	// the local network. On is the default, because the default has no
	// SRCDS_TOKEN, and a server with no Steam session refuses every client
	// that tries to join: LAN mode off without a token is the one combination
	// that cannot work, and it fails as a join that hangs rather than as an
	// error anybody can read.
	//
	// Turning it off is a deliberate pair with a real token. Do both.
	sv_lan ${SRCDS_LAN:-1}
	sv_use_steam_networking ${SRCDS_SDR_FAKEIP:-0}
	sv_pure 0
	sv_pausable 0
	setpause 0

	// Maps and other content come from this address over HTTP, when the
	// stack has one to give: the fastdl service at FASTDL_HOST, or the
	// operator's own SRCDS_DOWNLOADURL. The game server's own transfer can
	// reach the end of a large packed BSP without the client accepting it,
	// which restarts the download forever.
	${download_cfg}
	// A stock server refuses direct downloads larger than 16 MB. Potato maps
	// such as Autumnull fit under Source's 64 MB direct-download cap. Kept on
	// so a client that cannot reach sv_downloadurl still gets the map here.
	sv_allowdownload 1
	// Client uploads carry sprays and other player customization.
	sv_allowupload 1
	net_maxfilesize 64

	// Long enough to stop a scan, short enough that fat-fingering the password
	// does not lock the operator out for a day.
	sv_rcon_banpenalty 15
	sv_rcon_maxfailures 10
	sv_rcon_log 1

	exec banned_user.cfg
	exec banned_ip.cfg
	CFG

	if cmp -s "$staged" "$target"; then
		rm -f "$staged"
		return 0
	fi
	mv "$staged" "$target"
	chmod 0644 "$target"
	echo "[AP] wrote server.cfg, rcon password from the environment"
	if [ "${TAILSCALE_FASTDL:-0}" = 1 ]; then
		echo "[AP] public Tailscale Funnel FastDL ready at $download_url"
	fi
}

install_plugin() {
	installed=0
	while true; do
		if [ -d "$GAME/addons/sourcemod/plugins" ]; then
			if [ "$installed" -eq 0 ]; then
				# Repair the _666 file's tank paths and zombie event setting,
				# then give it a selectable name without shipping Valve's
				# popfile in our image. The plugin maps the runtime name back
				# to the seed's stable _666 ID.
				if ! tf2ap-stockpop "$GAME" "$GAME/scripts/population/mvm_ghost_town_ap_caliginous_caper.pop"; then
					echo "[AP] Caliginous Caper's stock mission is unavailable; retrying" >&2
					sleep "$INTERVAL"
					continue
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

# SRCDS_REACH says in one word where players come from. The game understands
# two separate things instead: sv_lan in server.cfg, and -enablefakeip on the
# command line, which the image's own entrypoint adds for SRCDS_SDR_FAKEIP=1.
# Resolving it here keeps the .env file with one answer rather than two that
# can contradict each other.
#
# SRCDS_LAN on its own is the older spelling and still works: leaving
# SRCDS_REACH unset changes nothing.
case "${SRCDS_REACH:-}" in
"") ;;
lan)
	SRCDS_LAN=1
	SRCDS_SDR_FAKEIP=0
	;;
steam)
	SRCDS_LAN=0
	SRCDS_SDR_FAKEIP=1
	;;
port)
	SRCDS_LAN=0
	SRCDS_SDR_FAKEIP=0
	;;
*)
	echo "[AP] SRCDS_REACH=${SRCDS_REACH} is not lan, steam or port: staying on the local network"
	SRCDS_LAN=1
	SRCDS_SDR_FAKEIP=0
	;;
esac

# Every reach but lan logs in to Steam, and a server with no token never gets a
# session: it refuses every player, the ones on the local network included. It
# is worth more on the local network than it is refusing everybody.
case "${SRCDS_TOKEN:-0}" in
"" | 0)
	if [ "${SRCDS_LAN:-0}" = 0 ]; then
		echo "[AP] no SRCDS_TOKEN, so the server stays on the local network: get one at steamcommunity.com/dev/managegameservers for app id 440"
		SRCDS_LAN=1
		SRCDS_SDR_FAKEIP=0
	fi
	;;
esac
export SRCDS_LAN SRCDS_SDR_FAKEIP

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
