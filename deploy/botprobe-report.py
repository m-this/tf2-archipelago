#!/usr/bin/env python3
"""Summarize a defender bot run: who never left spawn, and who stood still.

    botprobe-report.py <run-dir> [--gate]

With --gate it exits 1 unless every planned wave has a defender record and no
bot stayed in spawn or stood idle. See docs/waveprobe.md.
"""

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


def jsonl(path):
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def rows(run_dir):
    """Every wave the run recorded: defenders.jsonl, or one file per shard."""
    paths = sorted(Path(run_dir).glob("defenders*.jsonl"))
    return [row for path in paths for row in jsonl(path)]


def sharded(run_dir):
    return len(list(Path(run_dir).glob("defenders-*.jsonl"))) > 1


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


def window(wave):
    if not wave.get("started_utc") or not wave.get("ended_utc"):
        return None
    return datetime.fromisoformat(wave["started_utc"]), datetime.fromisoformat(wave["ended_utc"])


def in_wave(wave, events):
    span = window(wave)
    if span is None:
        return []
    return [event for event in events if span[0] <= event[0] <= span[1]]


def faults(bot):
    kind = CLASSES.get(bot["class"], str(bot["class"]))
    found = []
    if bot["left_spawn_seconds"] < 0:
        found.append("never left spawn")
    elif bot.get("left_spawn_max_seconds", bot["left_spawn_seconds"]) > LEFT_LIMIT:
        found.append(f"a spawn exit took {bot.get('left_spawn_max_seconds', 0):.0f}s over {bot.get('lives', 1)} lives")
    if bot.get("in_spawn_this_life_seconds", 0) > LEFT_LIMIT:
        found.append(f"in spawn for {bot['in_spawn_this_life_seconds']:.0f}s at the end")
    # Runs before idle_max_seconds existed only have the raw stillness.
    if "idle_max_seconds" in bot:
        if kind not in PARKED and bot["idle_max_seconds"] >= STILL_LIMIT:
            at = ",".join(f"{v:.0f}" for v in bot["idle_max_at"])
            found.append(f"idle {bot['idle_max_seconds']:.0f}s with a robot in reach at {at}")
    elif kind not in PARKED and bot["still_max_seconds"] >= STILL_LIMIT and not bot["still_max_in_spawn"]:
        at = ",".join(f"{v:.0f}" for v in bot["still_max_at"])
        found.append(f"still {bot['still_max_seconds']:.0f}s at {at}")
    if bot["teleports"]:
        found.append(f"teleported {bot['teleports']}x")
    return kind, found


def blocking(bot):
    """What fails the release gate: a bot that stays in spawn, or stands idle."""
    kind = CLASSES.get(bot["class"], str(bot["class"]))
    found = []
    if bot["left_spawn_seconds"] < 0 and bot["seen_seconds"] > LEFT_LIMIT:
        found.append(f"never left spawn in {bot['seen_seconds']:.0f}s")
    elif bot.get("in_spawn_this_life_seconds", 0) > LEFT_LIMIT:
        found.append(f"in spawn for {bot['in_spawn_this_life_seconds']:.0f}s at the end")
    if "idle_max_seconds" not in bot:
        found.append("no idle record")
    elif kind not in PARKED and bot["idle_max_seconds"] >= STILL_LIMIT:
        at = ",".join(f"{v:.0f}" for v in bot["idle_max_at"])
        found.append(f"idle {bot['idle_max_seconds']:.0f}s with a robot in reach at {at}")
    return kind, found


def case(row):
    return row["mission"], row.get("mode", "normal"), row.get("wave", 0)


def gate(run_dir, waves):
    """The reasons the run fails the release gate; none means it passes."""
    plan = Path(run_dir) / "plan.jsonl"
    if not plan.exists():
        return ["no plan.jsonl, so nothing says which waves the run had to cover"]
    planned = [case(row) for row in jsonl(plan) if row.get("state") == "planned"]
    if not planned:
        return ["the plan is empty"]
    recorded = {case(wave): wave for wave in waves}
    reasons = []
    for key in planned:
        wave = recorded.get(key)
        where = f"{key[0]} {key[1]} wave {key[2]}"
        if wave is None:
            reasons.append(f"{where}: not played")
        elif not wave.get("defenders"):
            reasons.append(f"{where}: no defender record ({wave.get('outcome', '')} {wave.get('error', '')})".rstrip())
        else:
            for bot in wave["defenders"]:
                kind, found = blocking(bot)
                if found:
                    reasons.append(f"{where}: {bot['name']} ({kind}) {'; '.join(found)}")
    return reasons


def print_gate(run_dir, waves):
    reasons = gate(run_dir, waves)
    print("## Release gate\n")
    if not reasons:
        print("Pass: every planned wave has a defender record, and no bot stayed in spawn or stood idle.\n")
        return True
    print(f"Fail: {len(reasons)} findings.\n")
    for reason in reasons:
        print(f"- {reason}")
    print()
    return False


def main(run_dir):
    """Print the report, and say whether the run passes the release gate."""
    waves = rows(run_dir)
    events = rescues(run_dir)
    bots = [bot for wave in waves for bot in wave.get("defenders") or []]
    print("# Defender bot run\n")
    config = Path(run_dir) / "config.txt"
    if config.exists():
        print("```\n" + config.read_text().strip() + "\n```\n")
    passed = print_gate(run_dir, waves)
    if not bots:
        print("No defender records. See defenders.err.")
        return passed
    left = [b["left_spawn_seconds"] for b in bots if b["left_spawn_seconds"] >= 0]
    never = sum(1 for b in bots if b["left_spawn_seconds"] < 0)
    stuck = sum(1 for b in bots if faults(b)[1])
    print(f"{len(waves)} waves, {len(bots)} bot records. "
          f"Never left spawn: {never}. Bots with a fault: {stuck}. "
          f"Median time to leave spawn: {statistics.median(left) if left else 0:.1f}s.\n")
    kinds = {}
    # Shards play their waves at the same time, so one rescue can fall in two.
    spans = [span for span in map(window, waves) if span]
    for when, kind, text in events:
        if any(start <= when <= end for start, end in spans):
            key = "spawn recovery" if kind == "SpawnNavRecovery" else "wedge teleport" if "moved to" in text else kind.lower()
            kinds[key] = kinds.get(key, 0) + 1
    print("Rescues in the mod's log during the waves: "
          + (", ".join(f"{count} {kind}" for kind, count in sorted(kinds.items())) or "none") + ".\n")
    if sharded(run_dir):
        # Every shard writes the same SourceMod log, so a rescue's time cannot
        # say which server it came from.
        print("The shards share one SourceMod log, so the rescues are not matched to missions.\n")
    else:
        print_rescues(waves, events)
    print_faults(waves)
    return passed


def print_rescues(waves, events):
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


def print_faults(waves):
    print("| Mission | Mode | Wave | Outcome | Bot | Class | Fault |")
    print("| --- | --- | --- | --- | --- | --- | --- |")
    for wave in waves:
        for bot in wave.get("defenders") or []:
            kind, found = faults(bot)
            if found:
                print(f"| {wave['mission']} | {wave.get('mode', 'normal')} | {wave.get('wave', 0)} "
                      f"| {wave.get('outcome', wave.get('state'))} "
                      f"| {bot['name']} | {kind} | {'; '.join(found)} |")
    missing = [w for w in waves if not w.get("defenders")]
    if missing:
        print("\nWaves without a defender record:\n")
        for wave in missing:
            print(f"- {wave['mission']} {wave.get('mode', 'normal')} wave {wave.get('wave', 0)}: "
                  f"{wave.get('outcome', '')} {wave.get('error', '')}")


if __name__ == "__main__":
    passed = main(sys.argv[1])
    sys.exit(0 if passed or "--gate" not in sys.argv[2:] else 1)
