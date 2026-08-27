#!/usr/bin/env python3
"""Read TF2's item schema and print candidate functional weapons as JSON.

This is a maintainer tool, not part of a build.  Its output is deliberately not
applied automatically: Archipelago item ids are permanent, so newly discovered
weapons must be reviewed and appended to gamedata's catalog.
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any


TOKEN = re.compile(r'"((?:\\.|[^"\\])*)"|([{}])')
WEAPON_SLOTS = {"primary", "secondary", "melee", "building", "pda", "pda2"}
VARIANT_NAMES = ("botkiller", "festive")


def parse_keyvalues(path: Path) -> dict[str, Any]:
    tokens: list[str] = []
    for match in TOKEN.finditer(path.read_text(encoding="utf-8-sig", errors="replace")):
        quoted, brace = match.groups()
        tokens.append(bytes(quoted, "utf-8").decode("unicode_escape") if quoted is not None else brace)

    def object_at(index: int) -> tuple[dict[str, Any], int]:
        result: dict[str, Any] = {}
        while index < len(tokens) and tokens[index] != "}":
            key = tokens[index]
            value = tokens[index + 1]
            index += 2
            if value == "{":
                value, index = object_at(index)
            result[key] = value
        return result, index + 1

    if len(tokens) < 2 or tokens[1] != "{":
        raise ValueError(f"{path}: expected a root KeyValues object")
    root, end = object_at(2)
    if end != len(tokens):
        raise ValueError(f"{path}: trailing KeyValues tokens")
    return {tokens[0]: root}


def localized_names(path: Path) -> dict[str, str]:
    names: dict[str, str] = {}
    for line in path.read_text(encoding="utf-16", errors="replace").splitlines():
        match = re.match(r'\s*"([^"\\]+)"\s+"((?:\\.|[^"\\])*)"', line)
        if match:
            names[match.group(1).casefold()] = match.group(2).replace(r'\"', '"')
    return names


def merged_item(item: dict[str, Any], prefabs: dict[str, Any]) -> dict[str, Any]:
    merged: dict[str, Any] = {}
    seen: set[str] = set()

    def add_prefab(name: str) -> None:
        if name in seen or name not in prefabs:
            return
        seen.add(name)
        prefab = prefabs[name]
        if not isinstance(prefab, dict):
            return
        for parent in str(prefab.get("prefab", "")).split():
            add_prefab(parent)
        merged.update(prefab)

    for name in str(item.get("prefab", "")).split():
        add_prefab(name)
    merged.update(item)
    return merged


def candidates(schema: Path, language: Path) -> list[dict[str, Any]]:
    game = parse_keyvalues(schema)["items_game"]
    items = game["items"]
    prefabs = game["prefabs"]
    names = localized_names(language)
    found: list[dict[str, Any]] = []
    for raw_index, item in items.items():
        if not raw_index.isdecimal() or not isinstance(item, dict):
            continue
        index = int(raw_index)
        merged = merged_item(item, prefabs)
        # `enabled` is not a usability flag here: stock and promotional weapon
        # prefabs use zero even though their concrete item definitions work.
        if merged.get("item_slot") not in WEAPON_SLOTS:
            continue
        token = str(merged.get("item_name", "")).removeprefix("#").casefold()
        name = names.get(token, str(item.get("name", "")).removeprefix("The "))
        if not name:
            continue
        static = item.get("static_attrs", {})
        variant_of = int(static.get("paintkit_proto_def_index", 0)) if isinstance(static, dict) else 0
        found.append({
            "def_index": index,
            "name": name,
            "slot": merged["item_slot"],
            "item_class": merged.get("item_class", ""),
            "variant_of": variant_of,
        })
    return sorted(found, key=lambda entry: (entry["name"].casefold(), entry["def_index"]))


def grouped_candidates(entries: list[dict[str, Any]]) -> list[dict[str, Any]]:
    grouped: dict[str, dict[str, Any]] = {}
    for entry in entries:
        if (entry["item_class"] == "slot_token" or entry["variant_of"]
                or any(marker in entry["name"].casefold() for marker in VARIANT_NAMES)):
            continue
        weapon = grouped.setdefault(entry["name"], {
            "name": entry["name"], "slot": entry["slot"],
            "item_class": entry["item_class"], "def_indexes": [],
        })
        weapon["def_indexes"].append(entry["def_index"])

    by_definition = {
        definition: weapon for weapon in grouped.values() for definition in weapon["def_indexes"]
    }
    by_name = {name.casefold(): weapon for name, weapon in grouped.items()}
    for entry in entries:
        weapon = by_definition.get(entry["variant_of"]) if entry["variant_of"] else None
        name = entry["name"]
        if weapon is None and name.casefold().startswith("festive "):
            weapon = by_name.get(name[len("Festive "):].casefold())
        if weapon is None and "botkiller" in name.casefold():
            matches = [candidate for key, candidate in by_name.items() if key in name.casefold()]
            weapon = max(matches, key=lambda candidate: len(candidate["name"]), default=None)
        if weapon is not None and entry["def_index"] not in weapon["def_indexes"]:
            weapon["def_indexes"].append(entry["def_index"])
            weapon["def_indexes"].sort()
    return sorted(grouped.values(), key=lambda entry: (entry["name"].casefold(), entry["def_indexes"][0]))


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("schema", type=Path, help="tf/scripts/items/items_game.txt")
    parser.add_argument("language", type=Path, help="tf/resource/tf_english.txt")
    args = parser.parse_args()
    entries = grouped_candidates(candidates(args.schema, args.language))
    print(json.dumps(entries, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
