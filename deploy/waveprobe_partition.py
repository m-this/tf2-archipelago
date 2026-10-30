#!/usr/bin/env python3
"""Balance complete missions by planned wave/mode count; never split a mission."""
import argparse
import collections
import json
from pathlib import Path


def partition(plan, shards):
    if not 1 <= shards <= 64:
        raise ValueError("shards must be between 1 and 64")
    weights = collections.Counter()
    for row in plan:
        weights.setdefault(row["mission"], 0)
        if row["state"] == "planned":
            weights[row["mission"]] += 1
    loads = [0] * shards
    missions = {}
    for mission in sorted(weights, key=lambda name: (-weights[name], name)):
        target = min(range(shards), key=lambda i: (loads[i], i))
        missions[mission] = target
        loads[target] += max(1, weights[mission])
    return {"schema": 1, "shards": shards, "missions": missions,
            "planned_cases": [sum(w for m, w in weights.items() if missions[m] == i) for i in range(shards)]}


def assignment(manifest, mission, shards):
    if manifest["shards"] != shards:
        raise ValueError("partition shard count differs from requested shards")
    value = manifest["missions"].get(mission)
    if not isinstance(value, int) or not 0 <= value < shards:
        raise ValueError("partition missing or invalid for " + mission)
    return value


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("plan", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--shards", type=int, required=True)
    opt = parser.parse_args()
    rows = [json.loads(line) for line in opt.plan.read_text().splitlines() if line.strip()]
    result = partition(rows, opt.shards)
    opt.output.write_text(json.dumps(result, indent=2) + "\n")
    print("Planned wave/mode cases per shard: " + str(result["planned_cases"]))
