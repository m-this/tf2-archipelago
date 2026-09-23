#!/usr/bin/env bash
# Run the first waves of every catalog mission on one disposable server with
# the defender bots on, and record how each bot moved. See docs/waveprobe.md.
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
: "${WAVEPROBE_RCONPW:?set a unique password for the disposable server}"
: "${WAVEPROBE_GAME_VOLUME:?set an existing game volume; the server writes to it}"
export WAVEPROBE_BOTS_SMX=${WAVEPROBE_BOTS_SMX:-$root/deploy/bots/build/package/addons/sourcemod/plugins/tf2_defenderbots.smx}
export WAVEPROBE_RCON_PORT=${WAVEPROBE_RCON_PORT:-27045}
speed=${WAVEPROBE_SPEED:-4}
waves=${WAVEPROBE_WAVES:-1}
mission=${WAVEPROBE_MISSION:-all}
timeout=${WAVEPROBE_TIMEOUT:-240s}
run_dir=${WAVEPROBE_RUN_DIR:-$root/docs/audits/botprobe-$(date -u +%Y%m%d-%H%M%S)}
project=${WAVEPROBE_PROJECT:-tf2ap-botprobe}
[[ -f $WAVEPROBE_BOTS_SMX ]] || { echo "no bots build at $WAVEPROBE_BOTS_SMX" >&2; exit 2; }
docker volume inspect "$WAVEPROBE_GAME_VOLUME" >/dev/null
mkdir -p "$run_dir"
compose=(docker compose -p "$project" -f "$root/deploy/compose.waveprobe.yml" -f "$root/deploy/compose.botprobe.yml")
finish() {
    result=$?
    "${compose[@]}" stop > /dev/null 2>&1 || true
    # The mod's own rescue lines, which the report matches to waves by time.
    docker run --rm -v "$WAVEPROBE_GAME_VOLUME:/g:ro" debian:bookworm-slim \
        sh -c 'cat /g/tf/addons/sourcemod/logs/L*.log' 2>/dev/null \
        | grep -E 'SpawnNavRecovery|Stuck:|Stalled:' > "$run_dir/rescues.txt" || true
    python3 "$root/deploy/botprobe-report.py" "$run_dir" > "$run_dir/REPORT.md" || result=1
    echo "Report: $run_dir/REPORT.md" >&2
    exit "$result"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
printf 'speed=%s\nwaves=%s\nmission=%s\ntimeout=%s\nbots_smx=%s\nbots_sha256=%s\nimage=%s\nstarted_utc=%s\nsource_commit=%s\n' \
    "$speed" "$waves" "$mission" "$timeout" "$WAVEPROBE_BOTS_SMX" "$(sha256sum < "$WAVEPROBE_BOTS_SMX" | cut -d' ' -f1)" \
    "${WAVEPROBE_SRCDS_IMAGE:-tf2-archipelago-srcds:latest}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$(git rev-parse HEAD)" \
    > "$run_dir/config.txt"

"$root/plugin/build.sh" > "$run_dir/plugin-build.log" 2>&1
# shellcheck source=/dev/null
source "$root/deploy/env/versions.env"
compiler="$root/plugin/build/sourcemod-$SOURCEMOD_VERSION/addons/sourcemod/scripting/spcomp64"
[[ -x $compiler ]] || compiler="${compiler%64}"
"$compiler" -E -o"$root/plugin/build/tf2_waveprobe.smx" \
    "$root/plugin/scripting/tf2_waveprobe.sp" > "$run_dir/probe-build.log" 2>&1
go build -o "$run_dir/waveprobe" ./launcher/cmd/waveprobe

# Only the maps this volume holds: a missing map costs a full load timeout.
# The catalog already leaves out the missions whose map has no nav.
maps=${WAVEPROBE_MAPS:-$(docker run --rm -v "$WAVEPROBE_GAME_VOLUME:/g:ro" debian:bookworm-slim \
    sh -c 'cd /g/tf/maps && ls mvm_*.bsp' | sed 's/\.bsp$//' | paste -sd, -)}
# Community popfiles only: the runner never filters Valve missions, whose
# popfiles live in the VPK.
docker run --rm -v "$WAVEPROBE_GAME_VOLUME:/g:ro" debian:bookworm-slim \
    sh -c 'ls /g/tf/scripts/population/ /g/tf/custom/*/scripts/population/ 2>/dev/null' \
    | grep '\.pop$' > "$run_dir/pops.txt"
"${compose[@]}" up -d --force-recreate > "$run_dir/docker.log" 2>&1
"$run_dir/waveprobe" -rcon "127.0.0.1:$WAVEPROBE_RCON_PORT" -defenders -mode normal \
    -mission "$mission" -maps "$maps" -pops "$run_dir/pops.txt" ${WAVEPROBE_ONE_PER_MAP:+-one-per-map} -waves "$waves" -speed "$speed" -timeout "$timeout" \
    > "$run_dir/defenders.jsonl" 2> "$run_dir/defenders.err"
