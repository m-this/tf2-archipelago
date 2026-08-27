#!/usr/bin/env python3
"""Build the small ZIP artifacts needed by the standalone launcher.

This intentionally uses only Python's standard library. A WSL checkout can
therefore build tf2ap.exe without Docker or the optional `zip` command.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path, PurePosixPath
import zipfile


def files_under(source: Path):
    for path in sorted(source.rglob("*")):
        if path.is_file():
            yield path, path.relative_to(source)


def build_apworld(source: Path, output: Path, container_version: int) -> None:
    manifest_path = source / "archipelago.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest.update(
        {
            "compatible_version": container_version,
            "version": container_version,
        }
    )

    output.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(
        output, "w", zipfile.ZIP_DEFLATED, compresslevel=9, allowZip64=True
    ) as archive:
        for path, relative in files_under(source):
            # Mirrors this world's .apignore plus Archipelago's global Python
            # build-artifact exclusions. The manifest is written below with
            # the APContainer fields stamped into it.
            if relative == Path("archipelago.json") or relative == Path(".apignore"):
                continue
            if "test" in relative.parts or "__pycache__" in relative.parts:
                continue
            if path.suffix in {".pyc", ".pyo"}:
                continue
            archive.write(path, PurePosixPath(source.name, *relative.parts).as_posix())

        archive.writestr(
            PurePosixPath(source.name, "archipelago.json").as_posix(),
            json.dumps(manifest),
        )

    validate_apworld(output, source.name, container_version)


def validate_apworld(output: Path, module: str, container_version: int) -> None:
    with zipfile.ZipFile(output) as archive:
        names = set(archive.namelist())
        required = {f"{module}/__init__.py", f"{module}/archipelago.json"}
        missing = required - names
        if missing:
            raise RuntimeError(f"invalid apworld, missing: {', '.join(sorted(missing))}")
        manifest = json.loads(archive.read(f"{module}/archipelago.json"))
        if manifest.get("version") != container_version:
            raise RuntimeError("invalid apworld container version")


def build_tree(source: Path, output: Path, excluded_suffixes: tuple[str, ...]) -> None:
    output.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(
        output, "w", zipfile.ZIP_DEFLATED, compresslevel=9, allowZip64=True
    ) as archive:
        for path, relative in files_under(source):
            if path.suffix in excluded_suffixes:
                continue
            archive.write(path, PurePosixPath(*relative.parts).as_posix())


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)

    apworld = subparsers.add_parser("apworld")
    apworld.add_argument("source", type=Path)
    apworld.add_argument("output", type=Path)
    apworld.add_argument("--container-version", required=True, type=int)

    tree = subparsers.add_parser("tree")
    tree.add_argument("source", type=Path)
    tree.add_argument("output", type=Path)
    tree.add_argument("--exclude-suffix", action="append", default=[])

    return parser.parse_args()


def main() -> None:
    args = parse_args()
    if args.command == "apworld":
        build_apworld(args.source, args.output, args.container_version)
    else:
        build_tree(args.source, args.output, tuple(args.exclude_suffix))


if __name__ == "__main__":
    main()
