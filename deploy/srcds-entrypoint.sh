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
COMMUNITY=/opt/tf2-community-pack/tf
GAME="${STEAMAPPDIR}/${STEAMAPP}"
INTERVAL=30

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

	staged=$(mktemp)
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

	// A stock server refuses direct downloads larger than 16 MB. Potato maps
	// such as Autumnull fit under Source's 64 MB direct-download cap.
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
}

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
			# Community packs use TF2's own directory layout. -u makes the bind
			# mount editable between restarts without rewriting a live map every
			# thirty seconds when nothing changed.
			if [ -d "$COMMUNITY" ]; then
				cp -ru "$COMMUNITY/." "$GAME/"
			fi
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

install_plugin &

exec bash "${HOMEDIR}/entry.sh"
