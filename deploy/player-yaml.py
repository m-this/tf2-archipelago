"""Write the Archipelago player file from the MVM_* environment.

The compose stack sets popfile names, because that is what every other part of
this project calls a mission. The apworld's options take the mission's display
name. This does the translation, from the same export the apworld reads, so
the two cannot drift.

It replaced a shell heredoc that quietly dropped MVM_EXCLUDED_MISSIONS: the
variable reached the container, no line of YAML carried it, and generation
drew the excluded mission anyway. A popfile this does not know is an error
here rather than a surprise three hours into an evening.

Usage: player-yaml.py <data directory> <archipelago version>
"""

import json
import os
import pathlib
import sys

RANDOM = "random"


class ConfigError(Exception):
    pass


def yaml_string(value: str) -> str:
    return '"' + value.replace('"', '\\"') + '"'


def read(name: str, fallback: str) -> str:
    value = os.environ.get(name, "").strip()
    return value if value else fallback


def mission_name(popfile: str, by_popfile: dict[str, str], variable: str) -> str:
    name = by_popfile.get(popfile)
    if name is None:
        raise ConfigError(f"{variable}: {popfile!r} is not a mission of this game")
    return name


def build(data: pathlib.Path, archipelago_version: str) -> str:
    meta = json.loads((data / "meta.json").read_text())
    missions = json.loads((data / "missions.json").read_text())["missions"]
    by_popfile = {mission["pop_file"]: mission["name"] for mission in missions}
    merc_names = {entry["name"] for entry in meta["classes"]}

    excluded = [
        mission_name(popfile.strip(), by_popfile, "MVM_EXCLUDED_MISSIONS")
        for popfile in read("MVM_EXCLUDED_MISSIONS", "").split(",")
        if popfile.strip()
    ]

    start_mission = read("MVM_START_MISSION", RANDOM)
    if start_mission != RANDOM:
        start_mission = mission_name(start_mission, by_popfile, "MVM_START_MISSION")

    start_class = read("MVM_START_CLASS", RANDOM)
    if start_class != RANDOM and start_class not in merc_names:
        raise ConfigError(
            f"MVM_START_CLASS: {start_class!r} is not a class. "
            f"Name one of {', '.join(sorted(merc_names))}."
        )

    known_mods = {mod["key"] for mod in meta["server_mods"]}
    server_mods = [key.strip() for key in read("SRCDS_MODS", "").split(",") if key.strip()]
    for key in server_mods:
        if key not in known_mods:
            raise ConfigError(
                f"SRCDS_MODS: {key!r} is not a server mod of this game. "
                f"Name one of {', '.join(sorted(known_mods))}."
            )

    game = meta["game"]
    lines = [
        f"name: {yaml_string(read('AP_SLOT_NAME', 'tf2'))}",
        f"game: {yaml_string(game)}",
        "requires:",
        f"  version: {archipelago_version}",
        f"{yaml_string(game)}:",
        f"  mission_count: {read('MVM_MISSION_COUNT', '8')}",
        f"  difficulty_pool: {read('MVM_DIFFICULTY', 'intermediate')}",
        f"  goal: {read('MVM_GOAL', 'final_boss')}",
        f"  missionsanity_percentage: {read('MVM_MISSIONSANITY_PERCENTAGE', '80')}",
        f"  death_link: {read('MVM_DEATH_LINK', 'false')}",
        f"  start_mission: {yaml_string(start_mission)}",
        f"  start_class: {yaml_string(start_class)}",
    ]
    if excluded:
        lines.append("  excluded_missions:")
        lines += [f"    - {yaml_string(name)}" for name in excluded]
    else:
        lines.append("  excluded_missions: []")
    lines.append(f"  community_missions: {read('MVM_COMMUNITY_MISSIONS', 'true')}")
    if server_mods:
        lines.append("  server_mods:")
        lines += [f"    - {yaml_string(key)}" for key in server_mods]
    else:
        lines.append("  server_mods: []")
    return "\n".join(lines) + "\n"


def main() -> int:
    if len(sys.argv) != 3:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    try:
        sys.stdout.write(build(pathlib.Path(sys.argv[1]), sys.argv[2]))
    except (ConfigError, KeyError, OSError, json.JSONDecodeError) as failure:
        print(f"cannot write the player file: {failure}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
