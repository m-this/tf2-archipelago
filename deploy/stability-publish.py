#!/usr/bin/env python3
"""Publish all badge endpoints atomically on a branch served by raw GitHub."""
import argparse
import json
from pathlib import Path
import re
import subprocess


def api(repository, path, method="GET", payload=None, missing_ok=False):
    command = ["gh", "api", f"repos/{repository}/{path}", "--method", method]
    if payload is not None:
        command += ["--input", "-"]
    result = subprocess.run(command, input=json.dumps(payload) if payload is not None else None,
                            text=True, capture_output=True)
    if result.returncode:
        if missing_ok and "HTTP 404" in result.stderr:
            return None
        raise RuntimeError(result.stderr.strip())
    return json.loads(result.stdout)


def publish(directory, repository):
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
        raise ValueError("expected owner/repository")
    files = []
    for environment in ("ubuntu24", "docker", "windows"):
        content = (directory / f"{environment}.json").read_text(encoding="utf-8")
        badge = json.loads(content)
        if badge.get("schemaVersion") != 1 or not isinstance(badge.get("message"), str):
            raise ValueError(f"invalid badge endpoint: {environment}")
        files.append({"path": f"{environment}.json", "mode": "100644", "type": "blob", "content": content})
    current = api(repository, "git/ref/heads/stability-badges", missing_ok=True)
    parents = [current["object"]["sha"]] if current else []
    tree = api(repository, "git/trees", "POST", {"tree": files})
    commit = api(repository, "git/commits", "POST", {
        "message": "Update daily stability badges [skip ci]", "tree": tree["sha"], "parents": parents})
    if current:
        api(repository, "git/refs/heads/stability-badges", "PATCH", {"sha": commit["sha"], "force": False})
    else:
        api(repository, "git/refs", "POST", {"ref": "refs/heads/stability-badges", "sha": commit["sha"]})
    print(f"Published badge endpoints to {repository}@stability-badges ({commit['sha']})")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--repository", required=True)
    args = parser.parse_args()
    publish(args.directory, args.repository)
