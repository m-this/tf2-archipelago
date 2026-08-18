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

	// LAN mode skips Steam authentication. The server has no Game Server Login
	// Token by default, so it never logs in to Steam, and a client trying to
	// authenticate against a server with no Steam session is refused. Going
	// online means a real SRCDS_TOKEN and SRCDS_LAN=0.
	sv_lan ${SRCDS_LAN:-1}
	sv_pure 0
	sv_pausable 0
	setpause 0

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

install_plugin &

exec bash "${HOMEDIR}/entry.sh"
