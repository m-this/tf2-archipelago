#!/bin/sh
# Generate a seed if there is not one already, then host it.
#
# Regenerating on every start would hand the players a different multiworld each
# evening and invalidate everything the bridge has recorded.
set -eu

AP_PORT="${AP_PORT:-38281}"
AP_SLOT_NAME="${AP_SLOT_NAME:-tf2}"
AP_PASSWORD="${AP_PASSWORD:-}"
AP_VERSION="${ARCHIPELAGO_VERSION:?the image must set ARCHIPELAGO_VERSION}"

MVM_MISSION_COUNT="${MVM_MISSION_COUNT:-8}"
MVM_DIFFICULTY="${MVM_DIFFICULTY:-intermediate}"
MVM_GOAL="${MVM_GOAL:-final_boss}"
MVM_MISSIONSANITY_PERCENTAGE="${MVM_MISSIONSANITY_PERCENTAGE:-80}"
MVM_DEATH_LINK="${MVM_DEATH_LINK:-false}"

output=/ap/output
players=/ap/Players

# gamedata owns the game name and exports it. Reading it here keeps the YAML
# that this script generates from being a fifth place to spell it wrong.
meta=/ap/custom_worlds/meta.json
if [ -f "$meta" ]; then
	game=$(sed -n 's/.*"game": "\([^"]*\)".*/\1/p' "$meta" | head -n 1)
fi
if [ -z "${game:-}" ]; then
	echo "cannot read the game name from $meta" >&2
	exit 1
fi

archive=$(find "$output" -maxdepth 1 -name 'AP_*.zip' | sort | head -n 1)

if [ -z "$archive" ]; then
	echo "no seed in $output, generating one"
	mkdir -p "$players"
	cat > "$players/tf2.yaml" <<-YAML
		name: $AP_SLOT_NAME
		game: $game
		requires:
		  version: $AP_VERSION
		$game:
		  mission_count: $MVM_MISSION_COUNT
		  difficulty_pool: $MVM_DIFFICULTY
		  goal: $MVM_GOAL
		  missionsanity_percentage: $MVM_MISSIONSANITY_PERCENTAGE
		  death_link: $MVM_DEATH_LINK
	YAML

	python Generate.py --player_files_path "$players" --outputpath "$output" < /dev/null

	archive=$(find "$output" -maxdepth 1 -name 'AP_*.zip' | sort | head -n 1)
	if [ -z "$archive" ]; then
		echo "generation produced no archive" >&2
		exit 1
	fi
fi

echo "hosting $archive on port $AP_PORT"
set -- python MultiServer.py --port "$AP_PORT" --host 0.0.0.0
if [ -n "$AP_PASSWORD" ]; then
	set -- "$@" --password "$AP_PASSWORD"
fi
exec "$@" "$archive" < /dev/null
