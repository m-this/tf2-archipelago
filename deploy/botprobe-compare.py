#!/usr/bin/env python3
"""Compare two defender bot runs map by map.

    botprobe-compare.py <arm-a-dir>[,<dir>...] <arm-b-dir>[,<dir>...]

An arm may span several run directories, for a run that was resumed; a later
directory's row for a map replaces an earlier one. Only maps where both arms
have bot records are compared.
"""

import importlib.util
import statistics
import sys
from pathlib import Path

spec = importlib.util.spec_from_file_location("report", Path(__file__).with_name("botprobe-report.py"))
report = importlib.util.module_from_spec(spec)
spec.loader.exec_module(report)

PARKED = {2, 8, 9}  # sniper, spy and engineer stand still on purpose


def arm(dirs):
    waves = {}
    for run_dir in dirs.split(","):
        events = report.rescues(run_dir)
        for wave in report.rows(run_dir):
            if wave.get("defenders"):
                waves[wave["map"]] = (wave, report.in_wave(wave, events))
    return waves


def measure(wave, events):
    bots = wave["defenders"]
    recoveries = [text for _, kind, text in events if kind == "SpawnNavRecovery"]
    looped = [text for text in recoveries if " has no route " not in text]
    names = {}
    for text in looped:
        for bot in bots:
            if text.startswith(bot["name"] + " "):
                names[bot["name"]] = names.get(bot["name"], 0) + 1
    return {
        "passed": wave.get("outcome") == "passed",
        "lives": sum(b["lives"] for b in bots),
        "no-route recoveries": sum(1 for text in recoveries if " has no route " in text),
        "timer/progress recoveries": len(looped),
        "bots recovered twice or more": sum(1 for count in names.values() if count >= 2),
        "stuck catches": sum(1 for _, kind, text in events if kind == "Stuck" and ", stuck #" in text),
        "local wedge moves": sum(1 for _, kind, text in events if kind == "Stuck" and "was wedged" in text and "again" not in text),
        "moves to the goal": sum(1 for _, kind, text in events if kind == "Stuck" and " again, moved to its goal" in text),
        "spawn exits over 20s": sum(1 for b in bots if b["left_spawn_max_seconds"] > 20 or b["left_spawn_seconds"] < 0),
        "still 30s+ away from spawn": sum(1 for b in bots if b["class"] not in PARKED
                                          and b["still_max_seconds"] >= 30 and not b["still_max_in_spawn"]),
        "left": [b["left_spawn_seconds"] for b in bots if b["left_spawn_seconds"] >= 0],
    }


def main(a_dirs, b_dirs):
    a, b = arm(a_dirs), arm(b_dirs)
    common = sorted(set(a) & set(b))
    print(f"{len(common)} maps with bot records in both arms.\n")
    keys = ["passed", "lives", "no-route recoveries", "timer/progress recoveries", "bots recovered twice or more",
            "stuck catches", "local wedge moves", "moves to the goal", "spawn exits over 20s",
            "still 30s+ away from spawn"]
    totals = {"A": {k: 0 for k in keys}, "B": {k: 0 for k in keys}}
    left = {"A": [], "B": []}
    rows = []
    for name in common:
        ma, mb = measure(*a[name]), measure(*b[name])
        for label, m in (("A", ma), ("B", mb)):
            for key in keys:
                totals[label][key] += int(m[key])
            left[label] += m["left"]
        changed = [f"{key} {int(ma[key])}->{int(mb[key])}" for key in keys[1:] if int(ma[key]) != int(mb[key])]
        if changed:
            rows.append(f"| {name} | {'; '.join(changed)} |")
    print("| Measure | A | B |")
    print("| --- | ---: | ---: |")
    for key in keys:
        print(f"| {key} | {totals['A'][key]} | {totals['B'][key]} |")
    print(f"| median seconds to leave spawn | {statistics.median(left['A']) if left['A'] else 0:.1f} "
          f"| {statistics.median(left['B']) if left['B'] else 0:.1f} |")
    print("\n| Map | A -> B |")
    print("| --- | --- |")
    print("\n".join(rows))


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
