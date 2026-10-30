#!/usr/bin/env python3
"""Retest nonpassing waveprobe cases against the same wall-time limit."""
import concurrent.futures
import collections
import json
import pathlib
import os
import queue
import subprocess
import sys
import threading
import re
import zlib
from waveprobe_partition import assignment


def rows(path):
    with path.open(encoding="utf-8") as stream:
        for line in stream:
            if line.strip():
                yield json.loads(line)


def rcon_broken(row):
    error = row.get("error", "").lower()
    return any(marker in error for marker in (
        "connection reset", "connection refused", "cannot read the reply",
        "broken pipe", "timed out while waiting for rcon"))


def interleave_maps(cases):
    """Avoid loading the same map on every worker at once."""
    by_map = collections.defaultdict(collections.deque)
    for case in cases:
        by_map[case[1]].append(case)
    while by_map:
        for map_name in list(by_map):
            yield by_map[map_name].popleft()
            if not by_map[map_name]:
                del by_map[map_name]


def main(run_dir, binary, worker_ids, source_shards=None):
    settings = dict(line.split("=", 1) for line in
                    (run_dir / "config.txt").read_text().splitlines() if "=" in line)
    phase = os.environ.get("WAVEPROBE_PHASE", "retest")
    if phase not in ("screen", "retest"):
        raise ValueError(f"unknown waveprobe phase: {phase}")
    wall_limit = settings.get("first_pass_timeout", "180s") if phase == "screen" else settings.get("wave_timeout", "900s")
    match = re.fullmatch(r"(\d+)(s|m)", wall_limit)
    if not match:
        raise ValueError(f"unsupported wave timeout: {wall_limit}")
    subprocess_limit = int(match[1]) * (60 if match[2] == "m" else 1) + 180
    shards = int(settings["shards"])
    if not worker_ids or any(worker < 0 or worker >= shards for worker in worker_ids):
        raise ValueError(f"worker ids must be in [0, {shards})")
    if source_shards is None:
        source_shards = worker_ids
    if any(shard < 0 or shard >= shards for shard in source_shards):
        raise ValueError(f"source shard ids must be in [0, {shards})")
    plan = {(row["mission"], row["mode"], row["wave"]): row
            for row in rows(run_dir / "plan.jsonl")}
    latest = {}
    completed = set()
    files = (sorted(run_dir.glob("shard-*.jsonl")) +
             sorted(run_dir.glob("screen-*.jsonl")) +
             sorted(run_dir.glob("retest-*.jsonl")))
    for path in files:
        for row in rows(path):
            if row["wave"]:
                key = row["mission"], row["mode"], row["wave"]
                if key not in latest or latest[key]["state"] != "passed":
                    latest[key] = row
                if path.name.startswith(phase + "-"):
                    completed.add(key)
    partition_path = run_dir / "partition.json"
    manifest = json.loads(partition_path.read_text()) if partition_path.exists() else None
    candidates = []
    for key, planned in plan.items():
        if planned["state"] != "planned":
            continue
        if (assignment(manifest, key[0], shards) if manifest else zlib.crc32(key[0].encode()) % shards) not in source_shards:
            continue
        if key in completed and latest[key]["state"] != "inconclusive":
            continue
        if key in latest and latest[key]["state"] == "passed":
            continue
        if phase == "screen" and key in latest and latest[key].get("outcome") in ("wave timed out", "wave failed"):
            continue
        candidates.append((key, planned["map"]))
    pending = queue.Queue()
    for case in interleave_maps(candidates):
        pending.put(case)
    total = pending.qsize()
    print(f"{phase.title()}ing {total} incomplete waves with {len(worker_ids)} isolated servers.", flush=True)
    lock = threading.Lock()
    map_gates = {map_name: threading.Semaphore(2) for _, map_name in candidates}
    finished = 0
    projects_file = run_dir / "projects.txt"
    projects = projects_file.read_text().splitlines() if projects_file.exists() else []

    def restart_worker(index):
        if index >= len(projects):
            return
        name = projects[index] + "-srcds-1"
        subprocess.run(["docker", "restart", name], capture_output=True,
                       text=True, timeout=120, check=True)
        print(f"Restarted isolated retry server {index}.", flush=True)

    def worker(index):
        nonlocal finished
        port = int(settings.get("base_port", "27035")) + index
        path = run_dir / f"{phase}-{index}.jsonl"
        restart_worker(index)
        with path.open("a", encoding="utf-8") as stream:
            while True:
                try:
                    (mission, mode, wave), map_name = pending.get_nowait()
                except queue.Empty:
                    return
                command = [str(binary), "-rcon", f"{settings.get('rcon_host', '127.0.0.1')}:{port}",
                           "-mission", mission, "-mode", mode,
                           "-start-wave", str(wave), "-end-wave", str(wave),
                           "-speed", settings.get("speed", "20"), "-timeout", wall_limit,
                           "-load-timeout", "90s"]
                if settings.get("reverse_objectives") == "1":
                    command += ["-reverse-objectives"]
                elif settings.get("reverse_bombs") == "1":
                    command += ["-reverse-bombs"]
                if settings.get("experimental_reverse") == "1":
                    command += ["-experimental-reverse"]
                with map_gates[map_name]:
                    for attempt in range(2):
                        try:
                            result = subprocess.run(command, capture_output=True, text=True,
                                                    timeout=subprocess_limit, check=False)
                            output = [json.loads(line) for line in result.stdout.splitlines()
                                      if line.startswith("{")]
                        except (subprocess.TimeoutExpired, json.JSONDecodeError) as error:
                            output = []
                            result = None
                            reason = str(error)
                        else:
                            reason = result.stderr.strip()
                        wave_rows = [item for item in output if item.get("wave") == wave]
                        if wave_rows:
                            row = wave_rows[-1]
                        else:
                            load_failures = [item for item in output if item.get("state") == "load_failed"]
                            if load_failures:
                                row = {**load_failures[-1], "wave": wave, "seed": 1}
                            else:
                                warmup_failures = [item for item in output
                                                   if 0 < item.get("wave", 0) < wave
                                                   and item.get("state") != "passed"]
                                if warmup_failures:
                                    failed = warmup_failures[-1]
                                    reason = (f"required earlier wave {failed['wave']} "
                                              f"{failed.get('outcome', failed['state'])}: "
                                              f"{failed.get('error', reason)}")
                                elif output:
                                    reason = output[-1].get("error", reason)
                                prefix = ("wave setup could not complete" if warmup_failures
                                          else "retest runner produced no wave result")
                                row = {"mission": mission, "map": map_name, "mode": mode,
                                       "wave": wave, "seed": 1, "state": "inconclusive", "outcome": "inconclusive",
                                       "retest_no_wave_result": phase == "retest",
                                       "error": f"{prefix}: {reason[:300]}"}
                        if attempt == 0 and rcon_broken(row):
                            restart_worker(index)
                            continue
                        break
                stream.write(json.dumps(row) + "\n")
                stream.flush()
                with lock:
                    finished += 1
                    if finished % 25 == 0 or finished == total:
                        print(f"{phase.title()}ed {finished}/{total} waves.", flush=True)
                pending.task_done()

    with concurrent.futures.ThreadPoolExecutor(max_workers=len(worker_ids)) as pool:
        list(pool.map(worker, worker_ids))


if __name__ == "__main__":
    main(pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2]),
         [int(value) for value in sys.argv[3].split(",")],
         [int(value) for value in sys.argv[4].split(",")]
         if len(sys.argv) > 4 else None)
