#!/usr/bin/env bash
# Run catalog missions on disposable servers with the defender bots on, and
# record how each bot moved. See docs/waveprobe.md.
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
: "${WAVEPROBE_RCONPW:?set a unique password for the disposable server}"
: "${WAVEPROBE_GAME_VOLUME:?set an existing game volume; the server writes to it}"
export WAVEPROBE_BOTS_SMX=${WAVEPROBE_BOTS_SMX:-$root/deploy/bots/build/package/addons/sourcemod/plugins/tf2_defenderbots.smx}
base_port=${WAVEPROBE_RCON_PORT:-27045}
shards=${WAVEPROBE_SHARDS:-1}
speed=${WAVEPROBE_SPEED:-4}
waves=${WAVEPROBE_WAVES:-1}
mode=${WAVEPROBE_MODE:-normal}
mission=${WAVEPROBE_MISSION:-all}
timeout=${WAVEPROBE_TIMEOUT:-240s}
gate=${WAVEPROBE_GATE:-}
run_dir=${WAVEPROBE_RUN_DIR:-$root/docs/audits/botprobe-$(date -u +%Y%m%d-%H%M%S)}
project=${WAVEPROBE_PROJECT:-tf2ap-botprobe}
[[ $shards =~ ^[1-9][0-9]*$ ]] || { echo 'WAVEPROBE_SHARDS must be positive' >&2; exit 2; }
[[ -f $WAVEPROBE_BOTS_SMX ]] || { echo "no bots build at $WAVEPROBE_BOTS_SMX" >&2; exit 2; }
docker volume inspect "$WAVEPROBE_GAME_VOLUME" >/dev/null
mkdir -p "$run_dir"
declare -a pids=()
compose() {
    WAVEPROBE_RCON_PORT=$((base_port + $1)) docker compose -p "$project-$1" \
        -f "$root/deploy/compose.waveprobe.yml" -f "$root/deploy/compose.botprobe.yml" "${@:2}"
}
finish() {
    result=$?
    trap - EXIT INT TERM
    for pid in "${pids[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    for ((i=0; i<shards; i++)); do
        compose "$i" stop > /dev/null 2>&1 || true
    done
    # The mod's own rescue lines, which the report matches to waves by time.
    docker run --rm -v "$WAVEPROBE_GAME_VOLUME:/g:ro" debian:bookworm-slim \
        sh -c 'cat /g/tf/addons/sourcemod/logs/L*.log' 2>/dev/null \
        | grep -E 'SpawnNavRecovery|Stuck:|Stalled:' > "$run_dir/rescues.txt" || true
    python3 "$root/deploy/botprobe-report.py" "$run_dir" ${gate:+--gate} > "$run_dir/REPORT.md" || result=1
    echo "Report: $run_dir/REPORT.md" >&2
    exit "$result"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
printf 'speed=%s\nwaves=%s\nmode=%s\nmission=%s\nshards=%s\ntimeout=%s\nbots_smx=%s\nbots_sha256=%s\nimage=%s\nstarted_utc=%s\nsource_commit=%s\n' \
    "$speed" "$waves" "$mode" "$mission" "$shards" "$timeout" "$WAVEPROBE_BOTS_SMX" \
    "$(sha256sum < "$WAVEPROBE_BOTS_SMX" | cut -d' ' -f1)" \
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
go build -o "$run_dir/rcon" ./launcher/cmd/rcon

# Only the maps this volume holds: a missing map costs a full load timeout.
# The catalog already leaves out the missions whose map has no nav.
maps=${WAVEPROBE_MAPS:-$(docker run --rm -v "$WAVEPROBE_GAME_VOLUME:/g:ro" debian:bookworm-slim \
    sh -c 'cd /g/tf/maps && ls mvm_*.bsp' | sed 's/\.bsp$//' | paste -sd, -)}
# Community popfiles only: the runner never filters Valve missions, whose
# popfiles live in the VPK. A volume without the community packs has none.
docker run --rm -v "$WAVEPROBE_GAME_VOLUME:/g:ro" debian:bookworm-slim \
    sh -c 'ls /g/tf/scripts/population/ /g/tf/custom/*/scripts/population/ 2>/dev/null || true' \
    | { grep '\.pop$' || true; } > "$run_dir/pops.txt"
cases=(-mission "$mission" -maps "$maps" -pops "$run_dir/pops.txt" ${WAVEPROBE_ONE_PER_MAP:+-one-per-map}
    -waves "$waves" -mode "$mode")
"$run_dir/waveprobe" -plan "${cases[@]}" > "$run_dir/plan.jsonl"

# The shards share the game volume, and every server start runs SteamCMD over
# it, so each server answers before the next one starts.
for ((i=0; i<shards; i++)); do
    compose "$i" up -d --force-recreate > "$run_dir/docker-$i.log" 2>&1
    for ((try=0; ; try++)); do
        ((try < 360)) || { echo "shard $i never answered rcon" >&2; exit 1; }
        reply=$(SRCDS_RCONPW=$WAVEPROBE_RCONPW SRCDS_PORT=$((base_port + i)) \
            "$run_dir/rcon" sm_waveprobe_status 2>/dev/null) || true
        [[ $reply == *WAVEPROBE* ]] && break
        sleep 5
    done
    echo "shard $i/$shards on 127.0.0.1:$((base_port + i))" >&2
    out=$run_dir/defenders.jsonl
    ((shards == 1)) || out=$run_dir/defenders-$i.jsonl
    "$run_dir/waveprobe" -rcon "127.0.0.1:$((base_port + i))" -defenders -shards "$shards" -shard "$i" \
        "${cases[@]}" -speed "$speed" -timeout "$timeout" > "$out" 2> "${out%.jsonl}.err" &
    pids+=("$!")
done
failed=0
for pid in "${pids[@]}"; do
    wait "$pid" || failed=1
done
# A lost wave is the mission's or the probe's; the gate judges the bots, and a
# wave that was never recorded fails it there.
[[ -n $gate ]] || exit "$failed"
