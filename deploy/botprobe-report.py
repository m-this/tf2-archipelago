#!/usr/bin/env python3
"""Summarize a defender bot run: who never left spawn, and who stood still."""

import json
import re
import statistics
import sys
from datetime import datetime, timezone
from pathlib import Path

CLASSES = {1: "scout", 2: "sniper", 3: "soldier", 4: "demoman", 5: "medic",
           6: "heavy", 7: "pyro", 8: "spy", 9: "engineer"}
# An engineer at his nest and a sniper at his spot stand still on purpose.
PARKED = {"engineer", "sniper"}
STILL_LIMIT = 30.0
LEFT_LIMIT = 20.0


def rows(run_dir):
    path = Path(run_dir) / "defenders.jsonl"
    if not path.exists():
        return []
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


RESCUE = re.compile(r"^L (\d\d/\d\d/\d{4} - \d\d:\d\d:\d\d): \[tf2_defenderbots\.smx\] "
                    r"(SpawnNavRecovery|Stuck|Stalled): (.*)$")


def rescues(run_dir):
    """The mod's own spawn and wedge rescues, as (time, kind, text)."""
    path = Path(run_dir) / "rescues.txt"
    if not path.exists():
        return []
    found = []
    for line in path.read_text(errors="replace").splitlines():
        match = RESCUE.match(line)
        if match:
            when = datetime.strptime(match[1], "%m/%d/%Y - %H:%M:%S").replace(tzinfo=timezone.utc)
            found.append((when, match[2], match[3]))
    return found


def in_wave(wave, events):
    if not wave.get("started_utc") or not wave.get("ended_utc"):
        return []
    start = datetime.fromisoformat(wave["started_utc"])
    end = datetime.fromisoformat(wave["ended_utc"])
    return [event for event in events if start <= event[0] <= end]


def faults(bot):
    kind = CLASSES.get(bot["class"], str(bot["class"]))
    found = []
    if bot["left_spawn_seconds"] < 0:
        found.append("never left spawn")
    elif bot.get("left_spawn_max_seconds", bot["left_spawn_seconds"]) > LEFT_LIMIT:
        found.append(f"a spawn exit took {bot.get('left_spawn_max_seconds', 0):.0f}s over {bot.get('lives', 1)} lives")
    if bot.get("in_spawn_this_life_seconds", 0) > LEFT_LIMIT:
        found.append(f"in spawn for {bot['in_spawn_this_life_seconds']:.0f}s at the end")
    if kind not in PARKED and bot["still_max_seconds"] >= STILL_LIMIT and not bot["still_max_in_spawn"]:
        at = ",".join(f"{v:.0f}" for v in bot["still_max_at"])
        found.append(f"still {bot['still_max_seconds']:.0f}s at {at}")
    if bot["teleports"]:
        found.append(f"teleported {bot['teleports']}x")
    return kind, found


def main(run_dir):
    waves = rows(run_dir)
    events = rescues(run_dir)
    bots = [bot for wave in waves for bot in wave.get("defenders") or []]
    print("# Defender bot run\n")
    config = Path(run_dir) / "config.txt"
    if config.exists():
        print("```\n" + config.read_text().strip() + "\n```\n")
    if not bots:
        print("No defender records. See defenders.err.")
        return
    left = [b["left_spawn_seconds"] for b in bots if b["left_spawn_seconds"] >= 0]
    never = sum(1 for b in bots if b["left_spawn_seconds"] < 0)
    stuck = sum(1 for b in bots if faults(b)[1])
    print(f"{len(waves)} waves, {len(bots)} bot records. "
          f"Never left spawn: {never}. Bots with a fault: {stuck}. "
          f"Median time to leave spawn: {statistics.median(left) if left else 0:.1f}s.\n")
    kinds = {}
    for wave in waves:
        for _, kind, text in in_wave(wave, events):
            key = "spawn recovery" if kind == "SpawnNavRecovery" else "wedge teleport" if "moved to" in text else kind.lower()
            kinds[key] = kinds.get(key, 0) + 1
    print("Rescues in the mod's log during the waves: "
          + (", ".join(f"{count} {kind}" for kind, count in sorted(kinds.items())) or "none") + ".\n")
    print("| Mission | Rescues | Spawn recoveries by bot |")
    print("| --- | --- | --- |")
    for wave in waves:
        found = in_wave(wave, events)
        if not found:
            continue
        per_bot = {}
        for _, kind, text in found:
            if kind == "SpawnNavRecovery":
                name = text.split(" exceeded ")[0].split(" made no ")[0].split(" has no ")[0]
                per_bot[name] = per_bot.get(name, 0) + 1
        bots_text = ", ".join(f"{name} {count}" for name, count in sorted(per_bot.items(), key=lambda item: -item[1]))
        print(f"| {wave['mission']} | {len(found)} | {bots_text} |")
    print()
    print("| Mission | Wave | Outcome | Bot | Class | Fault |")
    print("| --- | --- | --- | --- | --- | --- |")
    for wave in waves:
        for bot in wave.get("defenders") or []:
            kind, found = faults(bot)
            if found:
                print(f"| {wave['mission']} | {wave.get('wave', 0)} | {wave.get('outcome', wave.get('state'))} "
                      f"| {bot['name']} | {kind} | {'; '.join(found)} |")
    missing = [w for w in waves if not w.get("defenders")]
    if missing:
        print("\nWaves without a defender record:\n")
        for wave in missing:
            print(f"- {wave['mission']} wave {wave.get('wave', 0)}: {wave.get('outcome', '')} {wave.get('error', '')}")


if __name__ == "__main__":
    main(sys.argv[1])
