"""Move the pins keyed to a TF2 build in versions.env, and say so in CHANGELOG.md.

Used by .github/workflows/tf2-build-watch.yml once a SigMod release exists for
the new build. Values come from the release (version, the .sha256 asset) and
from Steam (depot manifests); nothing here computes a checksum.
"""
import argparse
import re
import sys
from pathlib import Path

DEPOTS = ("232250", "232255", "232256")


def set_pin(text: str, key: str, value: str) -> str:
    pattern = re.compile(rf"^{key}=.*$", re.MULTILINE)
    if len(pattern.findall(text)) != 1:
        raise ValueError(f"{key} must appear exactly once")
    return pattern.sub(f"{key}={value}", text)


def set_depot_pins(text: str, manifests: dict[str, str]) -> str:
    for depot in DEPOTS:
        key = f"TF2_DEPOT_{depot}_MANIFEST"
        if re.search(rf"^{key}=", text, re.MULTILINE):
            text = set_pin(text, key, manifests[depot])
        else:
            text = set_pin_after(text, "TF2_SERVER_VERSION", f"{key}={manifests[depot]}")
    return text


def set_pin_after(text: str, anchor: str, line: str) -> str:
    return re.sub(rf"^({anchor}=.*)$", rf"\1\n{line}", text, count=1, flags=re.MULTILINE)


def update_versions(text: str, build: str, windows_version: str, windows_sha256: str,
                    manifests: dict[str, str]) -> str:
    if not re.fullmatch(r"[0-9a-f]{64}", windows_sha256):
        raise ValueError(f"not a SHA256: {windows_sha256!r}")
    if not build.isdigit():
        raise ValueError(f"not a TF2 build: {build!r}")
    text = set_pin(text, "SIGSEGV_MVM_WINDOWS_VERSION", windows_version)
    text = set_pin(text, "SIGSEGV_MVM_WINDOWS_SHA256", windows_sha256)
    text = set_pin(text, "SOURCEMOD_TF2_SERVER_VERSION", build)
    text = set_pin(text, "TF2_SERVER_VERSION", build)
    return set_depot_pins(text, manifests)


def add_changelog_entry(text: str, build: str) -> str:
    bullet = (f"- **Servers work on TF2 build {build}.** Windows servers load SigMod\n"
              f"  again on the new game update, so the community missions that need it\n"
              f"  start at their first wave.\n")
    if f"TF2 build {build}." in text:
        return text
    unreleased = re.search(r"^## Unreleased\n", text, re.MULTILINE)
    if not unreleased:
        first_release = re.search(r"^## v", text, re.MULTILINE)
        section = f"## Unreleased\n\n### Fixed\n\n{bullet}\n"
        return text[:first_release.start()] + section + text[first_release.start():]
    end = re.search(r"^## ", text[unreleased.end():], re.MULTILINE)
    stop = unreleased.end() + (end.start() if end else len(text) - unreleased.end())
    body = text[unreleased.end():stop]
    fixed = re.search(r"^### Fixed\n\n", body, re.MULTILINE)
    if fixed:
        body = body[:fixed.end()] + bullet + body[fixed.end():]
    else:
        body = body.rstrip("\n") + f"\n\n### Fixed\n\n{bullet}\n"
    return text[:unreleased.end()] + body + text[stop:]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--versions", type=Path, required=True)
    parser.add_argument("--changelog", type=Path, required=True)
    parser.add_argument("--build", required=True)
    parser.add_argument("--windows-version", required=True)
    parser.add_argument("--windows-sha256", required=True)
    parser.add_argument("--manifest", action="append", required=True, metavar="DEPOT=ID")
    args = parser.parse_args()
    manifests = dict(item.split("=", 1) for item in args.manifest)
    if set(manifests) != set(DEPOTS) or not all(v.isdigit() for v in manifests.values()):
        parser.error(f"--manifest wants exactly {', '.join(DEPOTS)}, each a number")
    args.versions.write_text(update_versions(args.versions.read_text(), args.build,
                                             args.windows_version, args.windows_sha256, manifests))
    args.changelog.write_text(add_changelog_entry(args.changelog.read_text(), args.build))
    return 0


if __name__ == "__main__":
    sys.exit(main())
