#!/usr/bin/env bash
# Run every catalog wave, with and without Bot Surge, on disposable Docker servers.
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
shards=${WAVEPROBE_SHARDS:-6}
speed=${WAVEPROBE_SPEED:-50}
reverse_bombs=${WAVEPROBE_REVERSE_BOMBS:-1}
[[ $reverse_bombs == 0 || $reverse_bombs == 1 ]] || { echo "WAVEPROBE_REVERSE_BOMBS must be 0 or 1" >&2; exit 2; }
reverse_objectives=${WAVEPROBE_REVERSE_OBJECTIVES:-1}
[[ $reverse_objectives == 0 || $reverse_objectives == 1 ]] || { echo "WAVEPROBE_REVERSE_OBJECTIVES must be 0 or 1" >&2; exit 2; }
reverse_flags=()
if [[ $reverse_objectives == 1 ]]; then reverse_flags=(-reverse-objectives); elif [[ $reverse_bombs == 1 ]]; then reverse_flags=(-reverse-bombs); fi
seed=${WAVEPROBE_SEED:-1}
first_timeout=${WAVEPROBE_FIRST_PASS_TIMEOUT:-180s}
timeout=${WAVEPROBE_TIMEOUT:-900s}
base_port=${WAVEPROBE_BASE_PORT:-27035}
source_volume=${WAVEPROBE_SOURCE_VOLUME:-tf2-archipelago-waveprobe_tf2game_waveprobe}
run_dir=${WAVEPROBE_RUN_DIR:-$root/docs/audits/waveprobe-$(date -u +%Y%m%d-%H%M%S)}
: "${WAVEPROBE_RCONPW:?set a unique password for the disposable servers}"
[[ $shards =~ ^[1-9][0-9]*$ ]] || { echo 'WAVEPROBE_SHARDS must be positive' >&2; exit 2; }
[[ $base_port =~ ^[1-9][0-9]*$ ]] || { echo 'WAVEPROBE_BASE_PORT must be positive' >&2; exit 2; }
[[ ! -f $run_dir/plan.jsonl ]] || { echo 'Use a fresh run directory; resume-waveprobe.sh preserves old runs.' >&2; exit 2; }
mkdir -p "$run_dir"
declare -a pids=() projects=()
compose=(docker compose -f "$root/deploy/compose.waveprobe.yml")
if [[ ${WAVEPROBE_SHARED_IMAGE:-0} == 1 ]]; then
    compose+=(-f "$root/deploy/compose.waveprobe-image.yml")
fi
finish() {
    result=$?
    trap - EXIT INT TERM
    for pid in "${pids[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    if [[ -f $run_dir/plan.jsonl ]]; then
        if ! python3 "$root/deploy/waveprobe-report.py" "$run_dir" > "$run_dir/REPORT.md"; then
            printf '# MvM wave smoke probe\n\nReport generation failed. See the JSONL files in this directory.\n' \
                > "$run_dir/REPORT.md"
            result=1
        fi
    else
        printf '# MvM wave smoke probe\n\nSetup ended before the wave plan was created. See the build and Docker logs in this directory.\n' \
            > "$run_dir/REPORT.md"
    fi
    for ((j=0; j<${#projects[@]}; j++)); do
        WAVEPROBE_RCON_PORT=$((base_port + j)) "${compose[@]}" -p "${projects[j]}" stop > /dev/null 2>&1 || true
    done
    echo "Report: $run_dir/REPORT.md" >&2
    exit "$result"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
if [[ ${WAVEPROBE_SHARED_IMAGE:-0} != 1 ]]; then
    docker volume inspect "$source_volume" >/dev/null
fi
printf 'shards=%s\nspeed=%s\nseed=%s\nfirst_pass_timeout=%s\nwave_timeout=%s\nbase_port=%s\nstarted_utc=%s\nsource_commit=%s\nmain_commit=%s\n' \
    "$shards" "$speed" "$seed" "$first_timeout" "$timeout" "$base_port" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    "$(git rev-parse HEAD)" "$(git rev-parse origin/main)" > "$run_dir/config.txt"
printf 'reverse_bombs=%s\nreverse_objectives=%s\n' "$reverse_bombs" "$reverse_objectives" >> "$run_dir/config.txt"

# Build the two plugins from the current checkout before starting any shard.
"$root/plugin/build.sh" > "$run_dir/plugin-build.log" 2>&1
# shellcheck source=/dev/null
source "$root/deploy/env/versions.env"
compiler="$root/plugin/build/sourcemod-$SOURCEMOD_VERSION/addons/sourcemod/scripting/spcomp64"
[[ -x $compiler ]] || compiler="${compiler%64}"
"$compiler" -E -o"$root/plugin/build/tf2_waveprobe.smx" \
    "$root/plugin/scripting/tf2_waveprobe.sp" > "$run_dir/probe-build.log" 2>&1
(cd "$root" && go build -o "$run_dir/waveprobe" ./launcher/cmd/waveprobe)
"$run_dir/waveprobe" -plan -mode both "${reverse_flags[@]}" > "$run_dir/plan.jsonl"
python3 "$root/deploy/waveprobe_partition.py" "$run_dir/plan.jsonl" "$run_dir/partition.json" --shards "$shards"

run_id=${WAVEPROBE_RUN_ID:-$(date -u +%Y%m%d%H%M%S)-$$}
: > "$run_dir/projects.txt"
for ((i=0; i<shards; i++)); do
    project="tf2ap-waveprobe-${run_id}-${i}"
    projects+=("$project")
    printf '%s\n' "$project" >> "$run_dir/projects.txt"
    volume="${project}_tf2game_waveprobe"
    if [[ ${WAVEPROBE_SHARED_IMAGE:-0} != 1 ]] && ! docker volume inspect "$volume" >/dev/null 2>&1; then
        docker volume create "$volume" >/dev/null
        echo "copying isolated game files to $volume" >&2
        copied=0
        for attempt in 1 2 3; do
            if docker run --rm -v "$source_volume:/src:ro" -v "$volume:/dst" \
                debian:bookworm-slim sh -c 'cp -a /src/. /dst/'; then
                copied=1
                break
            fi
            echo "game copy to $volume failed (attempt $attempt/3)" >&2
            sleep 2
        done
        [[ $copied == 1 ]] || exit 1
    fi
    port=$((base_port + i))
    WAVEPROBE_RCON_PORT=$port "${compose[@]}" -p "$project" up -d --force-recreate > "$run_dir/docker-$i.log" 2>&1
    echo "shard $i/$shards on 127.0.0.1:$port" >&2
    "$run_dir/waveprobe" -rcon "127.0.0.1:$port" -shards "$shards" -shard "$i" \
        -partition "$run_dir/partition.json" -mission all -mode both "${reverse_flags[@]}" -speed "$speed" -seed "$seed" -timeout "$first_timeout" \
        > "$run_dir/shard-$i.jsonl" 2> "$run_dir/shard-$i.err" &
    pids+=("$!")
done

failed=0
for ((i=0; i<shards; i++)); do
    wait "${pids[i]}" || failed=1
done
echo "retesting nonpasses with ${timeout} real-time wave limit" >&2
worker_ids=$(seq -s, 0 "$((shards - 1))")
python3 "$root/deploy/waveprobe-retest.py" "$run_dir" "$run_dir/waveprobe" "$worker_ids" \
    > "$run_dir/retest.log" 2>&1 || failed=1
exit "$failed"
