#!/usr/bin/env python3
"""Reclaim optional SDKs on an ephemeral GitHub-hosted measurement runner."""
import os
import json
import re
from pathlib import Path
import shutil
import subprocess
import sys


def prefer_ubuntu_archive(path=Path("/etc/apt/apt-mirrors.txt")):
    # GitHub's image ranks Azure's HTTP mirror first. It can trickle downloads
    # indefinitely without hitting APT's idle timeout. Keep the same signed
    # Ubuntu repositories, preferring their HTTPS archive and retaining failover.
    if not path.exists():
        return
    original = path.read_text()
    if "https://archive.ubuntu.com/ubuntu/" not in original or "http://azure.archive.ubuntu.com/ubuntu/" not in original:
        return
    lines = []
    for line in original.splitlines():
        if line.startswith("https://archive.ubuntu.com/ubuntu/"):
            line = re.sub(r"priority:\d+", "priority:1", line)
        elif line.startswith("http://azure.archive.ubuntu.com/ubuntu/"):
            line = re.sub(r"priority:\d+", "priority:2", line)
        lines.append(line)
    subprocess.run(["sudo", "tee", str(path)], input="\n".join(lines) + "\n", text=True,
                   stdout=subprocess.DEVNULL, check=True, timeout=30)
    print("APT: prefer Ubuntu HTTPS archive; Azure and security mirrors retained as fallback", flush=True)


def main():
    if os.environ.get("STABILITY_RUNNER_ENVIRONMENT") != "github-hosted":
        raise RuntimeError("SDK cleanup is allowed only on an ephemeral GitHub-hosted runner")
    candidates = [Path(os.environ["RUNNER_TEMP"])]
    if os.name == "nt":
        candidates.append(Path("C:/tf2ap-stability"))
    for path in candidates:
        path.mkdir(parents=True, exist_ok=True)
    required = 45 if os.environ["STABILITY_ENV"] == "docker" else 35
    target = (required + 5) * 1024**3
    if os.name == "nt":
        # Keep Python, Node and system VC runtimes. Stop before expensive SDK
        # trees when an earlier deletion has already reclaimed enough space.
        tools = Path(os.environ["RUNNER_TOOL_CACHE"])
        paths = [Path("C:/Android"), Path("C:/Program Files/Android"),
                 Path("C:/Program Files/dotnet"), tools/"CodeQL",
                 Path("C:/Program Files/Java"), tools/"Go",
                 tools/"Java_Temurin-Hotspot_jdk", tools/"Ruby", tools/"PyPy",
                 Path("C:/Program Files/Microsoft Visual Studio"),
                 Path("C:/Program Files (x86)/Microsoft Visual Studio")]
    else:
        paths = [Path(value) for value in (
            "/usr/share/dotnet", "/usr/local/lib/android", "/opt/ghc",
            "/usr/local/.ghcup", "/usr/share/swift", "/usr/lib/jvm",
            "/opt/hostedtoolcache/CodeQL")]
    for path in paths:
        available = max(shutil.disk_usage(candidate).free for candidate in candidates)
        if available >= target:
            break
        if not path.exists():
            continue
        print(f"Reclaiming optional SDK {path}; {available/1024**3:.1f} GiB free", flush=True)
        if os.name == "nt":
            shutil.rmtree(path, ignore_errors=True)
        else:
            subprocess.run(["sudo", "rm", "-rf", str(path)], check=True, timeout=180)
    if os.name != "nt":
        prefer_ubuntu_archive()
        packages = ["lib32gcc-s1", "lib32stdc++6", "libc6-i386", "lib32z1"]
        if os.environ["STABILITY_ENV"] == "ubuntu24":
            subprocess.run(["sudo", "dpkg", "--add-architecture", "i386"], check=True, timeout=60)
            packages += ["libtinfo6:i386", "libncurses6:i386", "libbz2-1.0:i386",
                         "libcurl4t64:i386", "libcurl3t64-gnutls:i386", "libsdl2-2.0-0:i386"]
        # Docker's engine libraries are inside its measured production image.
        # The host runs only the Go installer and 32-bit SteamCMD.
        cache_value = os.environ.get("STABILITY_APT_CACHE")
        cache = Path(cache_value).resolve() if cache_value else None
        if cache:
            if not cache.is_relative_to(Path(os.environ["RUNNER_TEMP"]).resolve()):
                raise ValueError("APT cache must be inside this job's temporary directory")
            cache.mkdir(parents=True, exist_ok=True)
        subprocess.run(["sudo", "apt-get", "update", "-qq"], check=True, timeout=300)
        # Bound the whole apt process group: a Python timeout on sudo alone
        # left apt/dpkg running after the step had already reported failure.
        cache_options = ["-o", f"Dir::Cache::archives={cache}",
                         "-o", "APT::Keep-Downloaded-Packages=true"] if cache else []
        try:
            subprocess.run(["sudo", "timeout", "--signal=TERM", "--kill-after=30s", "20m",
                            "env", "DEBIAN_FRONTEND=noninteractive", "apt-get",
                            "-o", "Acquire::Retries=3", "-o", "Acquire::http::Timeout=30",
                            "-o", "Acquire::https::Timeout=30", *cache_options,
                            "install", "-y", "--no-install-recommends", *packages], check=True)
        finally:
            if cache:
                # Even a timed-out install may have downloaded useful complete
                # packages. APT still checks repository versions and hashes.
                try:
                    subprocess.run(["sudo", "chmod", "-R", "a+rX", str(cache)], check=False, timeout=60)
                except (OSError, subprocess.SubprocessError) as error:
                    print(f"APT cache export unavailable: {error}", flush=True)
                output = os.environ.get("GITHUB_OUTPUT")
                if output:
                    with open(output, "a", encoding="utf-8") as stream:
                        stream.write(f"apt_cache_files={str(any(cache.glob('*.deb'))).lower()}\n")

    root = max(candidates, key=lambda path: shutil.disk_usage(path).free)
    free = shutil.disk_usage(root).free / 1024**3
    print(f"Game volume: {root.anchor}; free {free:.1f} GiB; required {required} GiB", flush=True)
    if free < required:
        raise RuntimeError(f"insufficient disk: {free:.1f} GiB available, {required} GiB required")
    with open(os.environ["GITHUB_ENV"], "a", encoding="utf-8") as env:
        env.write(f"STABILITY_GAME_PARENT={root}\n")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        output = Path("stability-result")
        output.mkdir(exist_ok=True)
        (output / "environment.json").write_text(json.dumps({
            "environment": os.environ.get("STABILITY_ENV"),
            "shard": int(os.environ.get("STABILITY_SHARD", "0")), "shards": int(os.environ.get("STABILITY_SHARDS", "12")),
            "commit": os.environ.get("GITHUB_SHA"), "startup": False,
            "fresh": False, "completed": False,
            "error": "host setup failed: " + str(error)}, indent=2) + "\n", encoding="utf-8")
        print(f"stability host setup: {error}", file=sys.stderr)
        raise SystemExit(1)
