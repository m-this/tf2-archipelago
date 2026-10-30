#!/usr/bin/env python3
"""Summarize a waveprobe sweep with a comparable score and per-case evidence."""

import collections
import json
import pathlib
import sys

RESULTS = ("passed", "wave failed", "wave 0", "no enemies spawned",
           "wave timed out", "probe error", "changelevel failure", "load blocked", "inconclusive",
           "not run", "reverse objective")
MODES = {"normal": "Bot Surge off", "surge": "Bot Surge on"}


def rows(path):
    with path.open(encoding="utf-8") as stream:
        for line in stream:
            if line.strip():
                yield json.loads(line)


def classify(row, prior_load_failure=None):
    if row.get("retest_no_wave_result") and prior_load_failure:
        return classify(prior_load_failure)
    value = row.get("outcome") or row["state"]
    if value == "no enemies spawned" and (
            row.get("bot_spawns", 0) + row.get("tank_spawns", 0) > 0 or
            any(point.get("spawned", 0) > 0 for point in row.get("timeline", []))):
        # A plugin reset can clear its counters after enemies were already
        # observed. Keep that as a probe error, never a missing-spawn claim.
        return "probe error"
    # Old runs recorded kill attempts, not spawns. Do not promote their
    # zero-kill results to the new "No enemies spawned" finding.
    return {"wave lost": "wave failed", "active at limit": "inconclusive",
            "no enemies observed": "inconclusive", "load_failed": "load blocked",
            "failed": "probe error"}.get(value, value)


def pct(n, d):
    return f"{100 * n / d:.1f}%" if d else "n/a"


def progress(row):
    value = row.get("progress_percent", -1)
    return f"{value:.0f}%" if value >= 0 else "unknown"


def safe(value):
    return str(value).replace("|", "\\|").replace("\n", " ")


def samples_to_show(samples, limit=12):
    if len(samples) <= limit:
        return samples
    indices = {round(i * (len(samples) - 1) / (limit - 1)) for i in range(limit)}
    return [samples[i] for i in sorted(indices)]


def load_results(run_dir):
    plan = {(r["mission"], r["mode"], r["wave"]): r for r in rows(run_dir / "plan.jsonl")}
    observed, load_errors = {}, {}
    files = (sorted(run_dir.glob("shard-*.jsonl")) +
             sorted(run_dir.glob("screen-*.jsonl")) +
             sorted(run_dir.glob("retest-*.jsonl")))
    for path in files:
        for row in rows(path):
            key = row["mission"], row["mode"], row["wave"]
            if row["state"] == "load_failed" and row["wave"] == 0:
                load_errors[key[:2]] = row
            elif key not in observed or classify(row) == "passed" or classify(observed[key]) != "passed":
                observed[key] = row

    def outcome(key):
        if plan[key]["state"] == "unsupported_reverse":
            return "reverse objective"
        if key in observed:
            result = classify(observed[key], load_errors.get(key[:2]))
            if "_rev_" in key[0] and result == "passed" and not reverse_completion_witness(key, observed[key]):
                return "probe error"
            return result
        if key[:2] in load_errors:
            return classify(load_errors[key[:2]])
        return "not run"

    return plan, observed, load_errors, files, outcome


def reverse_completion_witness(key, row):
    """Validate native evidence independently of the controller's verdict."""
    mission, _, wave = key
    objective = row.get("objective") or {}
    if not isinstance(objective, dict):
        return False
    def positive(name):
        return type(objective.get(name)) is int and objective[name] > 0
    method = objective.get("method")
    bosses = {"mvm_coaltown_rev_adv_computer_crash", "mvm_autumnull_rc2_rev_exp_codename_omega"}
    if mission in bosses and wave == 6:
        return method == "native-combat-clear" and positive("target_kills")
    if objective.get("blu_won") is not True:
        return False
    arctic = mission == "mvm_cyberia_rc6a_rev_arctic_arrangement"
    if arctic and wave == 4:
        return method == "native-combat-victory" and type(row.get("tanks")) is int and row["tanks"] >= 5 and type(row.get("bots")) is int and row["bots"] > 0
    robotfactory = "mvm_robotfactory_b30_rev_adv_short_circuit_execution"
    route = arctic or mission == "mvm_teien_rc6_rev_adv_shogun_slaughter" or (mission == robotfactory and wave == 3)
    if route:
        return method == "native-route-objective" and positive("point_captures")
    if mission == robotfactory:
        if wave == 1:
            return method == "native-repair" and (objective.get("repairs") == 5 or objective.get("repair_completed") is True)
        if wave == 2:
            return method == "native-escort" and objective.get("escort_stages") == 3
    if mission == "mvm_hideout_b3_rev_adv_galois":
        if wave == 4:
            return positive("tank_deliveries") or positive("captures")
        if wave == 5:
            return method == "native-combat-victory" and type(row.get("bots")) is int and row["bots"] > 0
    return positive("captures")


def main(run_dir):
    plan, observed, load_errors, files, outcome = load_results(run_dir)
    outcomes = {key: outcome(key) for key in plan}
    counts = collections.Counter(outcomes.values())
    eligible = sum(row["state"] != "unsupported_reverse" for row in plan.values())
    tested = eligible - counts["not run"]
    passed = counts["passed"]
    config = run_dir / "config.txt"
    settings = dict(line.split("=", 1) for line in config.read_text().splitlines()
                    if "=" in line) if config.exists() else {}
    modes = [mode for mode in MODES if any(key[1] == mode for key in plan)]
    per_mode = {}
    for mode in modes:
        keys = [key for key in plan if key[1] == mode and plan[key]["state"] != "unsupported_reverse"]
        mode_tested = sum(outcomes[key] != "not run" for key in keys)
        mode_passed = sum(outcomes[key] == "passed" for key in keys)
        per_mode[mode] = {"eligible": len(keys), "tested": mode_tested,
                          "passed": mode_passed,
                          "stability_percent": round(100 * mode_passed / len(keys), 2) if keys else None}
    reverse = [key for key in plan if "_rev_" in key[0] and plan[key]["state"] != "unsupported_reverse"]
    summary = {"reverse_cases": {"eligible": len(reverse),
                                "tested": sum(outcomes[key] != "not run" for key in reverse),
                                "passed": sum(outcomes[key] == "passed" for key in reverse)},
               "planned_cases": len(plan), "eligible_cases": eligible,
               "tested_cases": tested, "passed_cases": passed,
               "stability_percent": round(100 * passed / eligible, 2) if eligible else None,
               "tested_pass_percent": round(100 * passed / tested, 2) if tested else None,
               "verified_coverage_percent": round(100 * passed / eligible, 2) if eligible else None,
               "outcomes": {name: counts[name] for name in RESULTS}, "by_mode": per_mode}
    (run_dir / "SUMMARY.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")

    print("# MvM wave smoke probe\n")
    if settings.get("scope"):
        print(f"Subset: {safe(settings['scope'])}.\n")
    print(f"## Stability score: **{pct(passed, eligible)}** ({passed}/{eligible} planned cases passed)\n")
    print(f"Tested **{tested}/{eligible}** · {len({key[0] for key in plan})} missions · "
          f"{settings.get('speed', '?')}× · seed {settings.get('seed', '1')} · "
          f"screen {settings.get('first_pass_timeout', '?')}, retry {settings.get('wave_timeout', '?')} wall time.\n")
    if settings.get("source_commit") or settings.get("main_commit"):
        print(f"Source commit `{settings.get('source_commit', 'unknown')}` · "
              f"upstream main `{settings.get('main_commit', 'unknown')}`.\n")
    if settings.get("retry_commit"):
        print(f"Retry runner commit `{settings['retry_commit']}`. The first pass used the source commit above.\n")
    print("| Result | Bot Surge off | Bot Surge on | Total |\n| --- | ---: | ---: | ---: |")
    for name in RESULTS:
        if not counts[name]:
            continue
        normal = sum(value == name for key, value in outcomes.items() if key[1] == "normal")
        surge = sum(value == name for key, value in outcomes.items() if key[1] == "surge")
        print(f"| {name.title()} | {normal} | {surge} | **{counts[name]}** |")
    print()
    print("| Mode | Stability | Passed | Tested | Eligible |\n| --- | ---: | ---: | ---: | ---: |")
    for mode in modes:
        part = per_mode[mode]
        print(f"| {MODES[mode]} | {pct(part['passed'], part['eligible'])} | "
              f"{part['passed']} | {part['tested']} | {part['eligible']} |")
    print()

    issue_names = set(RESULTS[1:-2])
    issues = sorted((key for key, value in outcomes.items() if value in issue_names or value == "not run"),
                    key=lambda key: (plan[key]["map"], key[0], key[2], key[1]))
    if issues:
        print("## Waves to inspect\n")
        print("Each row is one wave in one mode. Progress uses the game's non-support enemy "
              "counter relative to its highest observed value; `unknown` means no baseline.\n")
        print("| Map | Mission | Mode | Wave | Result | Spawned | Auto killed | Alive | Progress | Real / game s |")
        print("| --- | --- | --- | ---: | --- | ---: | ---: | ---: | ---: | ---: |")
        for key in issues:
            row = observed.get(key) or load_errors.get(key[:2], {})
            spawns = row.get("bot_spawns", 0) + row.get("tank_spawns", 0)
            kills = row.get("bots", 0) + row.get("tanks", 0)
            print(f"| {safe(plan[key]['map'])} | {safe(key[0])} | {MODES.get(key[1], key[1])} | "
                  f"{key[2]} | {outcomes[key].title()} | {spawns} | {kills} | "
                  f"{row.get('alive_at_end', 0)} | {progress(row)} | "
                  f"{row.get('wall_seconds', 0):.0f} / {row.get('game_seconds', 0):.0f} |")
        print("\n### Timelines\n")
        for key in issues:
            row = observed.get(key) or load_errors.get(key[:2], {})
            samples = row.get("timeline", [])
            print(f"<details><summary>{safe(key[0])} · {MODES.get(key[1], key[1])} · "
                  f"wave {key[2]} · {outcomes[key].title()}</summary>\n")
            if samples:
                print("| Real s | Game s | Spawned | Auto killed | Alive | Game remaining | Progress |")
                print("| ---: | ---: | ---: | ---: | ---: | ---: | ---: |")
                for point in samples_to_show(samples):
                    print(f"| {point['wall_seconds']:.0f} | {point['game_seconds']:.0f} | "
                          f"{point['spawned']} | {point['killed']} | {point['alive']} | "
                          f"{point['remaining']} | {progress(point)} |")
            else:
                print("No wave timeline: the mission did not load or the probe did not start.")
            if row.get("error"):
                print(f"\nReason: {safe(row['error'])[:500]}")
            if row.get("debug_snapshot"):
                print("\nFailure snapshot:\n\n```text")
                print(row["debug_snapshot"][:3000])
                print("```")
            if row.get("changelevel"):
                evidence = row["changelevel"]
                print("\nChangelevel evidence: "
                      f"requested `{safe(evidence['requested_map'])}`, "
                      f"before `{safe(evidence['before_map'])}`/`{safe(evidence['before_pop'])}`, "
                      f"after `{safe(evidence['after_map'])}`/`{safe(evidence['after_pop'])}` "
                      f"(state `{safe(evidence['after_state'])}`). "
                      f"Reply: `{safe(evidence.get('command_reply', ''))[:500]}`; "
                      f"command error: `{safe(evidence.get('command_error', ''))[:500]}`.")
            print("\n</details>\n")

    if counts["not run"]:
        print(f"## Not run\n\n{counts['not run']} eligible cases have no result; see `plan.jsonl`.\n")
    reverse = {key[0] for key, value in outcomes.items() if value == "reverse objective"}
    if reverse:
        print(f"## Reverse objectives\n\n{len(reverse)} missions require objective simulators or validation beyond the enabled bomb/carrier subset "
              "and are excluded from the score.\n")
    print("## Files\n\n- [Machine-readable summary](SUMMARY.json)")
    for path in files:
        print(f"- [{path.name}]({path.name})")
    print("\nA pass is one completion witness at this seed and timing; it does not prove every "
          "authored spawn appeared or every timing will complete.")


if __name__ == "__main__":
    main(pathlib.Path(sys.argv[1]))
