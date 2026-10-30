#!/usr/bin/env python3
"""Combine independent environment evidence into a dated repository dashboard."""
import argparse
import csv
import html
import datetime as dt
import importlib.util
import json
import re
from pathlib import Path
import shutil
import subprocess
import sys
import zipfile
import zlib
from waveprobe_partition import assignment
from stability_errors import install_error

ENVIRONMENTS = {"ubuntu24": "Ubuntu 24.04", "windows": "Windows", "docker": "Docker"}


def read_json(path):
    return json.loads(path.read_text(encoding="utf-8")) if path.exists() else {}


def case_key(row):
    return row["mission"], row["mode"], row["wave"]


def records(path):
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]


def combine_shards(source, plan_path, shards, commit, partition_path=None):
    manifest = read_json(partition_path) if partition_path else None
    """Rebuild each environment from the full canonical plan, never just returned shards."""
    for environment in ENVIRONMENTS:
        root = source / environment
        root.mkdir(parents=True, exist_ok=True)
        meta = {"environment": environment, "commit": commit, "startup": True,
                "fresh": True, "completed": True, "robot_limiter": True, "crash_free": True, "shards": []}
        errors = []
        if not plan_path.exists():
            errors.append("build did not produce the canonical wave plan")
            meta.update(startup=False, fresh=False, completed=False)
            plan = []
        else:
            shutil.copy2(plan_path, root / "plan.jsonl")
            plan = records(plan_path)
        partition_valid = True
        if partition_path is not None:
            try:
                for row in plan:
                    assignment(manifest, row["mission"], shards)
                if not manifest or manifest.get("shards") != shards:
                    raise ValueError("partition is absent or has the wrong shard count")
            except (ValueError, KeyError, TypeError) as error:
                partition_valid = False
                meta.update(fresh=False, completed=False)
                errors.append("invalid canonical partition: " + str(error))
        for shard in range(shards):
            path = root / f"part-{shard}"
            evidence = read_json(path / "environment.json")
            meta["shards"].append(evidence or {"shard": shard, "error": "no result"})
            valid = evidence.get("commit") == commit and evidence.get("shard") == shard and evidence.get("shards") == shards
            if not valid:
                errors.append(f"shard {shard}: missing result or incorrect source/partition")
            if evidence.get("error"):
                errors.append(f"shard {shard}: {evidence['error']}")
            meta["startup"] &= valid and evidence.get("startup") is True
            meta["fresh"] &= valid and evidence.get("fresh") is True
            meta["completed"] &= valid and evidence.get("completed") is True
            meta["robot_limiter"] &= valid and evidence.get("robot_limiter") is True
            meta["crash_free"] &= valid and evidence.get("crash_free") is True
            if not valid or not partition_valid:
                continue
            expected = {case_key(row) for row in plan if (assignment(manifest, row["mission"], shards) if manifest else zlib.crc32(row["mission"].encode()) % shards) == shard}
            actual = {case_key(row) for row in records(path / "plan.jsonl")} if (path / "plan.jsonl").exists() else set()
            if actual != expected:
                errors.append(f"shard {shard}: wave plan differs from its canonical partition")
                meta.update(fresh=False, completed=False)
                continue
            # Each hosted shard uses one local retry worker. Prefix its output
            # when merging to preserve phase ordering in the existing report.
            for phase in ("shard", "screen", "retest"):
                destination = root / f"{phase}-{shard}.jsonl"
                with destination.open("w", encoding="utf-8") as stream:
                    for file in sorted(path.glob(f"{phase}-*.jsonl")):
                        stream.write(file.read_text(encoding="utf-8"))
        for field in ("installed_version", "running_version", "host", "speed", "mission"):
            values = {part.get(field) for part in meta["shards"] if part.get(field)}
            meta[field] = next(iter(values)) if len(values) == 1 else "mixed or unknown"
            if (field.endswith("version") or field == "speed") and len(values) != 1:
                meta["fresh"] = False
        starts = [part["started_utc"] for part in meta["shards"] if part.get("started_utc")]
        finishes = [part["finished_utc"] for part in meta["shards"] if part.get("finished_utc")]
        meta["started_utc"] = min(starts, default="unknown")
        meta["finished_utc"] = max(finishes, default="unknown")
        if errors:
            meta["error"] = "; ".join(errors)
        (root / "environment.json").write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")
        if plan:
            with (root / "REPORT.md").open("w", encoding="utf-8") as stream:
                subprocess.run([sys.executable, str(Path(__file__).with_name("waveprobe-report.py")), str(root)],
                               stdout=stream, check=True)


def evaluate(meta, summary, commit, latest=None):
    eligible = summary.get("eligible_cases", 0)
    passed = summary.get("passed_cases", 0)
    tested = summary.get("tested_cases", 0)
    counts_valid = (all(type(n) is int for n in (passed, tested, eligible))
                    and eligible > 0 and 0 <= passed <= tested <= eligible)
    valid = (meta.get("commit") == commit and meta.get("startup") is True
             and meta.get("fresh") is True and meta.get("robot_limiter") is True
             and counts_valid
             and (latest is None or meta.get("installed_version") == latest))
    # Missing waves remain in the denominator, including when the runner crashed.
    score = round(100 * passed / eligible, 2) if counts_valid else None
    if not meta:
        status = "no result"
    elif meta.get("commit") != commit:
        status = "wrong commit"
    elif not meta.get("startup"):
        status = "setup failed (no waves tested)" if tested == 0 else "startup failed"
    elif not meta.get("fresh") or (latest is not None and meta.get("installed_version") != latest):
        status = "latest build unverified"
    elif meta.get("crash_free") is False:
        status = "engine crash recorded"
    elif meta.get("crash_free") is not True:
        status = "crash check unverified"
    elif meta.get("robot_limiter") is not True:
        status = "robot limiter unverified"
    elif not valid:
        status = "invalid or empty wave plan"
    elif not meta.get("completed") or tested < eligible:
        status = "incomplete"
    elif passed < eligible:
        status = "wave failures"
    else:
        status = "passed"
    return {"status": status, "score": score, "startup": bool(meta.get("startup")),
            "passed": passed, "tested": tested, "eligible": eligible,
            "reverse_cases": summary.get("reverse_cases", {"eligible": 0, "tested": 0, "passed": 0}),
            "unsupported_reverse": summary.get("outcomes", {}).get("reverse objective", 0),
            "by_mode": summary.get("by_mode", {}), "evidence": meta}


def safe(value):
    return html.escape(str(value), quote=False).replace("|", "&#124;").replace("\r", " ").replace("\n", " ").replace("`", "&#96;")


def server_issues(root, environment, shard):
    """Attach crashes to the last observed case, without claiming it caused them."""
    issues = []
    crash = re.compile(r"Segmentation fault.*|FATAL ERROR.*|EXCEPTION_ACCESS_VIOLATION.*|SigMod: fault 0x[0-9a-f]+.*|winbed: exit status 0x(?:c0000005|c0000409|c0000374|c000001d|c0000094|c00000fd).*|stability unexpected server exit:.*", re.IGNORECASE)
    for path in [root / "server.log", *sorted(root.glob("server-attempt-*.log"))]:
        if not path.exists():
            continue
        context = {}
        with path.open(encoding="utf-8", errors="replace") as stream:
            for line in stream:
                if "WAVEPROBE state=" in line:
                    context = dict(re.findall(r"\b(map|pop|gamewave)=([^\s\"]+)", line))
                match = crash.search(line)
                if match:
                    issues.append(dict(environment=environment, map=context.get("map", "—"),
                                       mission=context.get("pop", "—"), wave=context.get("gamewave", "—"),
                                       mode="—", shard=shard, stage="crash (last observed case)",
                                       error=match[0].rstrip('"')))
    return issues


def collect_issues(root, environment, meta):
    issues = []
    if not meta:
        issues.append(dict(environment=environment, map="—", mission="—", wave="—", mode="—",
                           shard="", stage="server / setup", error="no environment result returned"))
    parts = meta.get("shards") or [meta]
    mission_shards, part_errors = {}, {}
    for path in root.glob("part-*/global-partition.json"):
        mission_shards.update(read_json(path).get("missions", {}))
        break
    for part in parts:
        number = part.get("shard", "")
        path = root / f"part-{number}"
        if (path / "plan.jsonl").exists():
            mission_shards.update({row["mission"]: number for row in records(path / "plan.jsonl")})
        crash_rows = server_issues(path if path.exists() else root, environment, number)
        issues.extend(crash_rows)
        errors = [] if crash_rows else list(dict.fromkeys(part.get("crashes", [])))
        if part.get("error"):
            error = part["error"]
            if not part.get("startup") and ("-install" in error or str(part.get("failure_phase", "")).startswith("install")):
                logs = list(reversed(sorted(path.glob("install-attempt-*.log")))) + [path / "install.log"]
                for logfile in logs:
                    cause = install_error(logfile)
                    if cause:
                        error = cause
                        break
            if error == "server exited before readiness":
                error = install_error(path / "server.log") or error
            if error.startswith("host setup failed:") and "exit status 124" in error:
                error = "Host package installation exceeded its 20-minute limit (apt-get)."
            part_errors[number] = error
            errors.append(error)
        if part.get("install_attempts", 1) > 1:
            for attempt in range(1, part["install_attempts"]):
                logfile = path / ("install.log" if attempt == 1 else f"install-attempt-{attempt}.log")
                cause = install_error(logfile)
                if cause:
                    issues.append(dict(environment=environment, map="—", mission="—", wave="—", mode="—",
                                       shard=number, stage="recovered retry", error=f"install attempt {attempt}: {cause}"))
        for retry in part.get("startup_retries", []):
            issues.append(dict(environment=environment, map="—", mission="—", wave="—", mode="—", shard=number,
                               stage="recovered retry" if part.get("startup") else "server / setup",
                               error=f"startup attempt {retry['attempt']}: {retry.get('error') or 'see startup log'}"))
        for error in errors:
            issues.append(dict(environment=environment, map="—", mission="—", wave="—", mode="—",
                               shard=number, stage="server / setup", error=error))
    if not (root / "plan.jsonl").exists():
        return issues
    spec = importlib.util.spec_from_file_location("wave_report", Path(__file__).with_name("waveprobe-report.py"))
    wave = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(wave)
    plan, observed, loads, files, outcome = wave.load_results(root)
    def finding(key, row, result, stage):
        error = row.get("error") or result
        if result == "probe error" and row.get("state") == "passed":
            error = "reverse completion witness missing or invalid"
        shard = mission_shards.get(key[0], "")
        if result == "not run":
            part = next((p for p in parts if p.get("shard") == shard), {})
            error = part_errors.get(shard, part.get("error", "no case result returned"))
        return dict(environment=environment, map=plan[key]["map"], mission=key[0], wave=key[2],
                    mode=key[1], shard=shard, stage=stage, error=f"{result}: {error}" if error != result else result)
    for key in sorted(plan, key=lambda k: (plan[k]["map"], k[0], k[2], k[1])):
        result = outcome(key)
        if result not in ("passed", "reverse objective"):
            row = observed.get(key) or loads.get(key[:2], {})
            issues.append(finding(key, row, result, "final"))
    # A passing retry never erases its first failure. Deduplicate repeated errors.
    seen = set()
    for path in files:
        for row in wave.rows(path):
            key = row["mission"], row["mode"], row["wave"]
            if key not in plan or outcome(key) != "passed" or wave.classify(row) == "passed":
                continue
            issue = finding(key, row, wave.classify(row), "recovered retry")
            identity = key, issue["error"]
            if identity not in seen:
                seen.add(identity)
                issues.append(issue)
    return issues


def issue_table(issues, title):
    if not issues:
        return []
    lines = [f"<details><summary>{title}</summary>", "",
             "| Environment | Map | Mission | Wave | Mode | Shard | Stage | Error |",
             "| --- | --- | --- | ---: | --- | --- | --- | --- |"]
    for issue in issues:
        # The CSV and JSON retain the complete error; keep Markdown readable.
        error = issue["error"]
        if len(error) > 350:
            error = error[:347] + "…"
        lines.append("| " + " | ".join(safe(issue.get(field, "")) for field in
                     ("environment", "map", "mission", "wave", "mode", "shard", "stage")) + f" | {safe(error)} |")
    return lines + ["", "</details>", ""]


def main(source, output, commit, run_url, check_live=False, scope="full", *, measured_utc=None, latest=None, run_attempt=1):
    if run_attempt < 1:
        raise ValueError("run attempt must be positive")
    output.mkdir(parents=True, exist_ok=True)
    # Reformatting existing evidence must preserve its original age.
    now = dt.datetime.fromisoformat(measured_utc) if measured_utc else dt.datetime.now(dt.timezone.utc)
    if check_live:
        spec = importlib.util.spec_from_file_location("stability_run", Path(__file__).with_name("stability-run.py"))
        runner = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(runner)
        try:
            latest = runner.live_version()
        except (OSError, ValueError, KeyError, TypeError):
            latest = "unverified"
    results = {key: evaluate(read_json(source / key / "environment.json"),
                             read_json(source / key / "SUMMARY.json"), commit, latest)
               for key in ENVIRONMENTS}
    dashboard = {"scope": scope, "measured_utc": now.isoformat(), "commit": commit, "run_url": run_url,
                 "run_attempt": run_attempt, "stale_after_hours": 36, "latest_at_report": latest, "environments": results}
    lines = ["# TF2-AP stability" if scope == "full" else "# TF2-AP smoke check", "",
             f"Measured **{now:%Y-%m-%d %H:%M UTC}** · [Actions run]({run_url}) · commit `{commit[:12]}`", "",
             "Percentage = passed / planned; missing cases count against it. Red means a check failed. Results expire after 36 hours.", "",
             "| Environment | Startup | TF2 build | Passed / planned | Tested | Stability | Result |",
             "| --- | --- | --- | ---: | ---: | ---: | --- |"]
    if scope != "full":
        lines.insert(2, "**Smoke subset; does not update full-catalog badges.**\n")
    if all(result["tested"] == 0 for result in results.values()):
        lines.insert(2, "**No waves ran: setup did not complete.**\n")
    all_issues = []
    for key, label in ENVIRONMENTS.items():
        result = results[key]
        score = f"{result['score']:.2f}%" if result["score"] is not None else "n/a"
        meta = result["evidence"]
        version = meta.get("installed_version", "unknown")
        parts = meta.get("shards", [])
        startup = f"{sum(part.get('startup') is True for part in parts)}/{len(parts)}" if parts else ("pass" if result["startup"] else "failed")
        lines.append(f"| {label} | {startup} | {safe(version)} | {result['passed']} / {result['eligible']} | "
                     f"{result['tested']} | {score} | {result['status']} |")
        badge = {"schemaVersion": 1, "label": label, "message": score,
                 "color": "brightgreen" if result["status"] == "passed" else "red", "cacheSeconds": 300}
        (output / f"{key}.json").write_text(json.dumps(badge) + "\n")
        all_issues.extend(collect_issues(source / key, label, meta))
    # A release per run AND attempt keeps old summaries and downloaded reports
    # tied to their evidence after the rolling dashboard advances or a run retries.
    run = re.fullmatch(r"(https?://[^/]+/[^/]+/[^/]+)/actions/runs/(\d+)(?:/attempts/\d+)?/?", run_url)
    asset_base = (f"{run[1]}/releases/download/stability-{run[2]}-{run_attempt}/"
                  if run and scope == "full" else "")
    release_lines = lines + ["", f"[Full report]({asset_base}REPORT.md) · [All issues (CSV)]({asset_base}issues.csv) · [Raw evidence]({asset_base}stability-evidence.zip)", ""]
    # Release notes have a size limit; full case tables are downloadable assets.
    (output / "RELEASE.md").write_text("\n".join(release_lines), encoding="utf-8")
    lines += ["", f"[All issues (CSV)]({asset_base}issues.csv) · Raw logs, retries and wave timelines: `stability-evidence.zip`.", ""]
    fields = ["environment", "map", "mission", "wave", "mode", "shard", "stage", "error"]
    with (output / "issues.csv").open("w", newline="", encoding="utf-8") as stream:
        writer = csv.DictWriter(stream, fieldnames=fields)
        writer.writeheader()
        writer.writerows({field: issue.get(field, "") for field in fields} for issue in all_issues)
    dashboard["issues"] = all_issues
    (output / "stability.json").write_text(json.dumps(dashboard, indent=2) + "\n")
    if any(issue["stage"].startswith("crash") for issue in all_issues):
        lines += ["Crash context is the last observed case, not proof of its cause.", ""]
    for key, label in ENVIRONMENTS.items():
        issues = [issue for issue in all_issues if issue["environment"] == label]
        final = [issue for issue in issues if issue["stage"] != "recovered retry"]
        recovered = [issue for issue in issues if issue["stage"] == "recovered retry"]
        lines += [f"## {label}", ""]
        if not issues:
            lines += ["No issues recorded.", ""]
            continue
        lines += issue_table(final, f"{len(final)} issues")
        if recovered:
            lines += issue_table(recovered, f"{len(recovered)} earlier failures recovered on retry")
    report = "\n".join(lines)
    (output / "REPORT.md").write_text(report, encoding="utf-8")
    with zipfile.ZipFile(output / "stability-evidence.zip", "w", zipfile.ZIP_DEFLATED) as archive:
        archive.write(output / "REPORT.md", "REPORT.md")
        archive.write(output / "stability.json", "stability.json")
        archive.write(output / "issues.csv", "issues.csv")
        for path in sorted(source.rglob("*")):
            if path.is_file() and path.relative_to(source).as_posix() not in {
                    "REPORT.md", "stability.json", "issues.csv", "RELEASE.md", "stability-evidence.zip"}:
                archive.write(path, path.relative_to(source))
    return all(value["status"] == "passed" for value in results.values())


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--run-url", required=True)
    parser.add_argument("--run-attempt", type=int, default=1)
    parser.add_argument("--check-live", action="store_true", help="verify Valve's live build again at publication")
    parser.add_argument("--shards", type=int, default=1)
    parser.add_argument("--plan", type=Path, help="canonical full plan emitted by the build job")
    parser.add_argument("--partition", type=Path, help="balanced mission manifest from the same build")
    parser.add_argument("--scope", choices=["full", "smoke"], default="full")
    args = parser.parse_args()
    if args.plan is not None:
        combine_shards(args.source, args.plan, args.shards, args.commit, args.partition)
    elif args.shards > 1:
        parser.error("--plan is required with --shards")
    raise SystemExit(0 if main(args.source, args.output, args.commit, args.run_url, args.check_live, args.scope, run_attempt=args.run_attempt) else 1)
