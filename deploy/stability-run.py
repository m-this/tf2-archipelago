#!/usr/bin/env python3
"""Run the same full wave plan on a disposable native or production Docker bed."""
import argparse
import datetime as dt
import json
import os
from pathlib import Path
import platform
import re
import shutil
import signal
import socket
import subprocess
import sys
import time
import urllib.request
import uuid

from stability_errors import install_error, steam_network_timeout, transient_install_error


def live_version():
    url = "https://api.steampowered.com/ISteamApps/UpToDateCheck/v1/?appid=440&version=1"
    for attempt in range(3):
        try:
            with urllib.request.urlopen(url, timeout=20) as response:
                value = json.load(response)["response"]
            version = str(value["required_version"])
            if value.get("success") and version.isdigit() and int(version) > 0:
                return version
            raise ValueError("Valve returned no live ServerVersion")
        except (OSError, ValueError, KeyError, TypeError):
            if attempt == 2:
                raise
            time.sleep(5)


def installed_version(game):
    match = re.search(r"^ServerVersion=(\d+)\s*$",
                      (game / "tf/steam.inf").read_text(), re.MULTILINE)
    if not match:
        raise ValueError("steam.inf has no ServerVersion")
    return match[1]


def running_version(game, evidence=""):
    # Prefer the live engine's RCON reply. On Linux the Breakpad startup line
    # is in stdout and can be absent from -condebug's console.log.
    current = re.findall(r"^\s*Server version:\s*(\d+)\s*$", evidence, re.MULTILINE | re.IGNORECASE)
    if current:
        return current[-1]
    # Older engine output: the startup line used by the debug bundle.
    console = (game / "tf/console.log").read_text(encoding="utf-8", errors="replace")
    versions = re.findall(r"Using Breakpad minidump system\. Version: (\d+)", console)
    if not versions:
        raise ValueError("engine did not report its running ServerVersion")
    return versions[-1]


def stop_process(process):
    if process.poll() is not None:
        return
    if os.name == "nt":
        subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"],
                       capture_output=True, check=False, timeout=30)
    else:
        # winbed handles Ctrl-C by cancelling RunServer, which also stops the
        # separate process group srcds_run owns. SIGTERM would bypass that
        # handler and leave its game server running after the bed exited.
        try:
            os.killpg(process.pid, signal.SIGINT)
        except ProcessLookupError:
            pass
        try:
            process.wait(timeout=15)
        except subprocess.TimeoutExpired:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
    process.wait(timeout=30)


class NativeServer:
    """Restart only this disposable server; never erase an unexpected exit."""
    def __init__(self, args, logfile, max_restarts):
        self.args, self.logfile, self.max_restarts = args, logfile, max_restarts
        self.events = []
        self.closed = False
        self.process = self.spawn("w")

    def spawn(self, mode):
        with self.logfile.open(mode, encoding="utf-8") as log:
            return subprocess.Popen(self.args, stdout=log, stderr=log,
                                    start_new_session=os.name != "nt",
                                    creationflags=subprocess.CREATE_NEW_CONSOLE if os.name == "nt" else 0)

    def recover(self, phase):
        if self.closed or self.process.poll() is None:
            return self.process
        event = {"exit_code": self.process.returncode, "phase": phase,
                 "utc": dt.datetime.now(dt.timezone.utc).isoformat(), "monotonic": time.monotonic()}
        self.events.append(event)
        with self.logfile.open("a", encoding="utf-8") as log:
            log.write("\nstability unexpected server exit: " + json.dumps(event) + "\n")
        recent = sum(event["monotonic"] - prior["monotonic"] < 60 for prior in self.events)
        if len(self.events) > self.max_restarts or recent > 3:
            self.closed = True
            raise RuntimeError("isolated server repeatedly exited; stopping recovery, crash evidence retained")
        print(f"stability: native server exited ({event['exit_code']}); restarting isolated server", flush=True)
        self.process = self.spawn("a")
        return self.process

    def stop(self):
        self.closed = True
        stop_process(self.process)


def startup_ready(evidence):
    return ("TF2 MvM Archipelago" in evidence and "TF2 MvM wave probe" in evidence
            and not re.search(r"<Failed>|<Error>|<Bad Load>|failed to load|Unknown command",
                              evidence, re.IGNORECASE))


def wait_for_health(rcon, output, deadline):
    """Allow an authored final-map transition to settle, without hiding crashes."""
    last_error = "map configs or mods remained incomplete"
    while time.monotonic() < deadline:
        try:
            evidence = "\n".join(rcon(command) for command in
                                 ["version", "sm plugins list", "sm exts list", "sm_waveprobe_status"])
            output.write_text(evidence, encoding="utf-8")
            if startup_ready(evidence) and re.search(r"\bconfigsready=1\b", evidence):
                return evidence
        except (subprocess.SubprocessError, OSError) as error:
            last_error = str(error)
        time.sleep(2)
    raise RuntimeError("server did not regain healthy mod/probe readiness after the final mission: " + last_error)


def robot_limiter_enabled(reply):
    # Fail closed on an unknown cvar, changed output or a disabled production mod.
    return bool(re.search(r'^\s*"sig_mvm_robot_limit"\s*=\s*"1(?:\.0+)?"',
                          reply, re.MULTILINE))


def check_robot_limiter(rcon, output):
    reply = rcon("sig_mvm_robot_limit")
    output.write_text(reply, encoding="utf-8")
    if not robot_limiter_enabled(reply):
        raise RuntimeError("production robot limiter is disabled or unverified")


def crash_evidence(output):
    # srcds_run restarts a crashed engine. A later healthy RCON reply must not
    # erase a fatal error from startup or an earlier wave in this measurement.
    failures = []
    for name in ["server.log", "docker.log", "console.log",
                 *(p.name for p in sorted(output.glob("server-attempt-*.log")))]:
        path = output / name
        if path.exists():
            for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
                if re.search(r"Segmentation fault|FATAL ERROR|EXCEPTION_ACCESS_VIOLATION|stability unexpected server exit|0x(?:c0000005|c0000409|c0000374|c000001d|c0000094|c00000fd)", line, re.IGNORECASE):
                    failures.append(f"{name}: {line}")
    return failures


def rcon_hosts(environment):
    hosts = ["127.0.0.1"]
    if environment == "docker":
        return hosts
    try:
        hosts += sorted({row[4][0] for row in socket.getaddrinfo(socket.gethostname(), None, socket.AF_INET)})
    except OSError as exc:
        print(f"stability: hostname address lookup unavailable: {exc}", flush=True)
    # Get-NetIPAddress initializes CIM on Windows and can take longer than the
    # address lookup budget on a busy hosted runner. Enumerate via .NET instead.
    # This supplements loopback/DNS; discovery failure must not abort readiness.
    if os.name == "nt":
        command = ["powershell", "-NoProfile", "-NonInteractive", "-Command",
                   "@([System.Net.NetworkInformation.NetworkInterface]::GetAllNetworkInterfaces() | "
                   "ForEach-Object { $_.GetIPProperties().UnicastAddresses } | "
                   "Where-Object { $_.Address.AddressFamily -eq [System.Net.Sockets.AddressFamily]::InterNetwork } | "
                   "ForEach-Object { $_.Address.IPAddressToString }) | ConvertTo-Json -Compress"]
    else:
        command = ["hostname", "-I"]
    try:
        result = subprocess.run(command, capture_output=True, text=True, timeout=60, check=True)
        if os.name == "nt":
            addresses = json.loads(result.stdout)
            hosts += [addresses] if isinstance(addresses, str) else (addresses or [])
        else:
            hosts += [address for address in result.stdout.split() if ":" not in address]
    except (subprocess.SubprocessError, OSError, ValueError) as exc:
        print(f"stability: supplemental interface lookup unavailable; trying loopback and hostname addresses: {exc}", flush=True)
    return list(dict.fromkeys(hosts))


class PhaseFailure(RuntimeError):
    def __init__(self, phase, status, detail):
        self.phase = phase
        super().__init__(f"{phase} exited {status}: {detail or 'see captured logs'}")


def install_with_retries(run, args, output, deadline, shard):
    # Preserve caches and failed logs. Never retry bad pins, missing libraries,
    # corrupt assets or game failures as if they were Steam network failures.
    deadline = min(deadline, time.monotonic() + 7200)
    for attempt in range(1, 4):
        filename = "install.log" if attempt == 1 else f"install-attempt-{attempt}.log"
        try:
            run(args, filename, max(1, deadline - time.monotonic()))
            return attempt
        except PhaseFailure:
            delay = (30 if attempt == 1 else 90) + shard * 3
            if (attempt == 3 or not transient_install_error(output / filename)
                    or time.monotonic() + delay >= deadline):
                raise
            print(f"stability: transient Steam install failure; retry {attempt + 1}/3 in {delay}s; logs retained", flush=True)
            time.sleep(delay)


def main(opt):
    out = opt.output.resolve()
    out.mkdir(parents=True, exist_ok=True)
    if any(out.iterdir()):
        raise ValueError("use an empty --output; previous passes must not leak into this measurement")
    bed = opt.bed.resolve()
    root = opt.root.resolve()
    if root.exists():
        raise ValueError("use a fresh disposable --root; existing roots are refused")
    suffix = ".exe" if os.name == "nt" else ""
    binary = lambda name: str(bed / (name + suffix))
    game = root / "tf-dedicated"
    port = "27035"
    os.environ["SRCDS_RCONPW"] = os.environ["WAVEPROBE_RCONPW"]
    os.environ["SRCDS_PORT"] = port
    shard = getattr(opt, "shard", 0)
    shards = getattr(opt, "shards", 1)
    if shards < 1 or not 0 <= shard < shards:
        raise ValueError("shard must be in [0, shards)")
    mission = getattr(opt, "mission", "all")
    speed = getattr(opt, "speed", 50)
    if not 1 <= speed <= 50:
        raise ValueError("speed must be between 1 and 50")
    reverse_bombs = getattr(opt, "reverse_bombs", True)
    reverse_objectives = getattr(opt, "reverse_objectives", True)
    reverse_flags = ["-reverse-objectives"] if reverse_objectives else (["-reverse-bombs"] if reverse_bombs else [])
    partition = ["-shards", str(shards), "-shard", str(shard)]
    manifest = getattr(opt, "partition", None)
    if manifest:
        shutil.copy2(manifest, out / "global-partition.json")
        partition += ["-partition", str((out / "global-partition.json").resolve())]
    meta = {"environment": opt.environment, "shard": shard, "shards": shards, "speed": speed, "mission": mission,
            "started_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
            "commit": os.environ.get("GITHUB_SHA", "local"), "host": platform.platform(),
            "run_url": os.environ.get("STABILITY_RUN_URL", ""), "startup": False,
            "fresh": False, "completed": False, "robot_limiter": False, "crash_free": False}
    process = None
    native_server = None
    container = "tf2ap-stability-" + uuid.uuid4().hex[:12]
    deadline = time.monotonic() + opt.hours * 3600

    def recover_server(phase):
        nonlocal process
        if native_server is not None and meta["startup"]:
            process = native_server.recover(phase)

    def run(args, filename, limit=None, check=True):
        remaining = max(1, deadline - time.monotonic())
        with (out / filename).open("w", encoding="utf-8") as stream:
            with (out / (filename + ".err")).open("w", encoding="utf-8") as errors:
                command = subprocess.Popen(args, stdout=stream, stderr=errors,
                                           start_new_session=os.name != "nt")
                began = time.monotonic()
                next_progress = began + 30
                operation_deadline = began + min(remaining, limit or remaining)
                try:
                    while True:
                        recover_server(filename)
                        left = operation_deadline - time.monotonic()
                        if left <= 0:
                            raise subprocess.TimeoutExpired(args, operation_deadline - began)
                        try:
                            status = command.wait(timeout=min(1, left))
                            break
                        except subprocess.TimeoutExpired:
                            if time.monotonic() < next_progress:
                                continue
                            next_progress = time.monotonic() + 30
                            elapsed = time.monotonic() - began
                            detail = ""
                            if filename.endswith(".jsonl"):
                                rows = []
                                for line in (out / filename).read_text(encoding="utf-8").splitlines():
                                    try:
                                        rows.append(json.loads(line))
                                    except ValueError:
                                        pass  # The child can be writing the next record.
                                detail = f"; {len(rows)} records, {sum(row.get('state') == 'passed' for row in rows)} passed"
                            print(f"stability {filename}: {elapsed:.0f}s{detail}", flush=True)
                except BaseException:
                    stop_process(command)
                    raise
                finally:
                    elapsed = time.monotonic() - began
                    meta.setdefault("phase_seconds", {})[filename] = round(elapsed, 2)
                    print(f"stability {filename}: finished after {elapsed:.1f}s", flush=True)
                recover_server(filename)
                if check and status:
                    raise PhaseFailure(filename, status, install_error(out / filename))
                return status

    def rcon(command):
        recover_server("health")
        result = subprocess.run([binary("rcon"), command], capture_output=True,
                                text=True, timeout=15, check=True)
        return result.stdout

    try:
        shutil.copy2("deploy/env/versions.env", out / "pins.env")
        if opt.environment == "ubuntu24":
            os_release = Path("/etc/os-release").read_text()
            if not re.search(r'^ID=ubuntu$', os_release, re.MULTILINE) or 'VERSION_ID="24.04"' not in os_release:
                raise ValueError("ubuntu24 requires native Ubuntu 24.04")
        if opt.environment == "windows" and os.name != "nt":
            raise ValueError("Windows must run natively")
        run([binary("waveprobe"), "-plan", "-mode", "both", "-mission", mission, *reverse_flags, *partition], "plan.jsonl", 60)
        (out / "config.txt").write_text(
            f"shards=1\nreverse_objectives={int(reverse_objectives)}\nreverse_bombs={int(reverse_bombs)}\nspeed={speed}\nseed=1\nfirst_pass_timeout=180s\nwave_timeout=900s\n"
            f"base_port={port}\nstarted_utc={meta['started_utc']}\nsource_commit={meta['commit']}\n")
        meta["latest_before"] = live_version()
        meta["install_attempts"] = install_with_retries(run,
            [binary("winbed"), "-root", str(root), "-probe", str(bed / "tf2_waveprobe.smx"), "-install"],
            out, deadline, shard)
        # The player launcher tolerates a failed update; the measurement cannot.
        meta["installed_version"] = installed_version(game)
        if meta["installed_version"] != meta["latest_before"]:
            raise ValueError("installed server is not Valve's current build")
        if opt.environment == "docker":
            # A fresh launcher-prepared tree supplies the same two community packs.
            # Production entrypoint, mods, SteamCMD update and server flags still run.
            run(["docker", "run", "--rm", "--user", "root", "--entrypoint", "chown",
                 "-v", f"{game}:/game", opt.image, "-R", "steam:steam", "/game"], "ownership.log", 300)
            args = ["docker", "run", "--rm", "--name", container,
                    "-p", f"127.0.0.1:{port}:27015/tcp",
                    "-v", f"{game}:/home/steam/tf-dedicated",
                    "-v", f"{bed / 'tf2_waveprobe.smx'}:/opt/tf2-archipelago/addons/sourcemod/plugins/tf2_waveprobe.smx:ro"]
            for value in ["SRCDS_REACH=lan", "SRCDS_LAN=1", "SRCDS_STARTMAP=mvm_decoy",
                          "SRCDS_MAXPLAYERS=32", "SRCDS_PORT=27015", "SRCDS_BOTS=0",
                          "SRCDS_MODS=sigsegv-mvm", "SRCDS_MOD_LOADING=always", "SRCDS_RCONPW", "TF2AP_TEST_MODE=0"]:
                args += ["-e", value]
            args += [opt.image]
        else:
            args = [binary("winbed"), "-root", str(root), "-port", port, "-serve"]
        if opt.environment == "docker":
            with (out / "server.log").open("w", encoding="utf-8") as log:
                process = subprocess.Popen(args, stdout=log, stderr=log, start_new_session=True)
        else:
            missions = {json.loads(line)["mission"] for line in (out / "plan.jsonl").read_text().splitlines() if line}
            native_server = NativeServer(args, out / "server.log", max(3, len(missions) * 2))
            process = native_server.process
            meta["server_exits"] = native_server.events
        startup_log = out / "server.log"
        meta["startup_attempts"] = 1
        hosts = rcon_hosts(opt.environment)
        host_index = 0
        ready_limit = min(deadline, time.monotonic() + 900)
        while time.monotonic() < ready_limit:
            if process.poll() is not None:
                attempt = meta["startup_attempts"]
                delay = (30 if attempt == 1 else 90) + shard * 3
                if (opt.environment == "docker" and attempt < 3 and steam_network_timeout(startup_log)
                        and time.monotonic() + delay + 30 < ready_limit):
                    meta.setdefault("startup_retries", []).append({
                        "attempt": attempt, "error": install_error(startup_log), "log": startup_log.name})
                    print(f"stability: Docker Steam update timed out; startup retry {attempt + 1}/3 in {delay}s; logs retained", flush=True)
                    time.sleep(delay)
                    meta["startup_attempts"] += 1
                    startup_log = out / f"server-attempt-{meta['startup_attempts']}.log"
                    with startup_log.open("w", encoding="utf-8") as log:
                        process = subprocess.Popen(args, stdout=log, stderr=log, start_new_session=True)
                    continue
                raise PhaseFailure(startup_log.name, process.returncode,
                                   install_error(startup_log) or "server exited before readiness")
            os.environ["SRCDS_RCON_HOST"] = hosts[host_index % len(hosts)]
            host_index += 1
            try:
                evidence = "\n".join(rcon(cmd) for cmd in
                                     ["version", "sm plugins list", "sm exts list", "sm_waveprobe_status"])
                (out / "startup.txt").write_text(evidence, encoding="utf-8")
                if startup_ready(evidence) and process.poll() is None:
                    break
            except (subprocess.SubprocessError, OSError):
                pass
            time.sleep(5)
        else:
            raise RuntimeError("startup did not load the mod, extensions and probe within 15 minutes")
        address = os.environ["SRCDS_RCON_HOST"]
        meta["rcon_host"] = address
        with (out / "config.txt").open("a") as config:
            config.write(f"rcon_host={address}\n")
        meta["startup"] = True
        meta["installed_version"] = installed_version(game)
        meta["running_version"] = running_version(game, evidence)
        meta["latest_at_start"] = live_version()
        if not meta["installed_version"] == meta["running_version"] == meta["latest_at_start"]:
            raise ValueError("TF2 changed during setup; rerun against the new build")
        meta["fresh"] = True
        check_robot_limiter(rcon, out / "robot-limiter-before.txt")
        meta["robot_limiter_before"] = True
        run([binary("waveprobe"), "-rcon", f"{address}:{port}", "-mission", mission,
                      "-mode", "both", "-speed", str(speed), "-seed", "1", "-timeout", "180s", *reverse_flags, *partition],
                     "shard-0.jsonl", check=False)
        run([sys.executable, "deploy/waveprobe-retest.py", str(out), binary("waveprobe"), "0"],
            "retest.log")
        final_health = wait_for_health(rcon, out / "final-health.txt", min(deadline, time.monotonic() + 120))
        meta["running_version_after"] = running_version(game, final_health)
        meta["installed_version_after"] = installed_version(game)
        check_robot_limiter(rcon, out / "robot-limiter-after.txt")
        meta["robot_limiter"] = True
        meta["latest_after"] = live_version()
        meta["fresh"] = (meta["latest_after"] == meta["installed_version"]
                         == meta["installed_version_after"] == meta["running_version_after"])
        meta["completed"] = True
    except (Exception, KeyboardInterrupt) as error:
        meta["error"] = str(error)
        if isinstance(error, PhaseFailure):
            meta["failure_phase"] = error.phase
        print(f"stability: {error}", file=sys.stderr)
    finally:
        if opt.environment == "docker":
            try:
                with (out / "docker.log").open("w") as logs:
                    subprocess.run(["docker", "logs", container], stdout=logs,
                                   stderr=subprocess.STDOUT, timeout=30, check=False)
                subprocess.run(["docker", "rm", "-f", container], capture_output=True, timeout=30, check=False)
            except (OSError, subprocess.SubprocessError) as error:
                meta["cleanup_error"] = str(error)
        if native_server is not None:
            native_server.stop()
        elif process:
            stop_process(process)
        for relative in ["tf/console.log", "tf/steam.inf", "steamapps/appmanifest_232250.acf"]:
            source = game / relative
            if source.exists():
                shutil.copy2(source, out / source.name)
        if (game / "tf/addons/sourcemod/logs").exists():
            shutil.copytree(game / "tf/addons/sourcemod/logs", out / "sourcemod-logs", dirs_exist_ok=True)
        crashes = crash_evidence(out)
        meta["crash_free"] = not crashes
        if crashes:
            meta["crashes"] = crashes
            meta["error"] = "; ".join(filter(None, [meta.get("error"), "engine crash recorded; automatic restart does not count as recovery"]))
        meta["finished_utc"] = dt.datetime.now(dt.timezone.utc).isoformat()
        (out / "environment.json").write_text(json.dumps(meta, indent=2) + "\n")
        if (out / "plan.jsonl").exists():
            with (out / "REPORT.md").open("w", encoding="utf-8") as report:
                subprocess.run([sys.executable, "deploy/waveprobe-report.py", str(out)],
                               stdout=report, check=True)
    summary = json.loads((out / "SUMMARY.json").read_text()) if (out / "SUMMARY.json").exists() else {}
    print(f"stability result: {summary.get('passed_cases', 0)}/{summary.get('eligible_cases', 0)} "
          f"eligible cases passed after retries; startup={meta['startup']}, fresh={meta['fresh']}, "
          f"completed={meta['completed']}, crash_free={meta['crash_free']}, "
          f"robot_limiter={meta['robot_limiter']}", flush=True)
    nonpasses = {name: count for name, count in summary.get("outcomes", {}).items()
                 if count and name not in ("passed", "reverse objective")}
    if nonpasses:
        print("stability nonpassing outcomes: " + json.dumps(nonpasses, sort_keys=True), flush=True)
    if meta.get("error"):
        print("stability measurement error: " + meta["error"], flush=True)
    return 0 if (meta["startup"] and meta["fresh"] and meta["completed"] and meta["crash_free"]
                 and summary.get("eligible_cases", 0) > 0
                 and summary.get("passed_cases") == summary.get("eligible_cases")) else 1


if __name__ == "__main__":
    def interrupted(signum, frame):
        raise KeyboardInterrupt(f"interrupted by signal {signum}")

    signal.signal(signal.SIGTERM, interrupted)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--environment", choices=["ubuntu24", "docker", "windows"], required=True)
    parser.add_argument("--bed", type=Path, required=True)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--image", default="tf2ap-stability:current")
    parser.add_argument("--hours", type=float, default=20)
    parser.add_argument("--shards", type=int, default=1)
    parser.add_argument("--shard", type=int, default=0)
    parser.add_argument("--speed", type=int, default=50)
    parser.add_argument("--reverse-objectives", action=argparse.BooleanOptionalAction, default=True,
                        help="include every playable reverse mission in coverage")
    parser.add_argument("--reverse-bombs", action=argparse.BooleanOptionalAction, default=True,
                        help="include validated reverse bomb/carrier missions")
    parser.add_argument("--partition", type=Path)
    parser.add_argument("--mission", default="all")
    sys.exit(main(parser.parse_args()))
