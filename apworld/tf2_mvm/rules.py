"""What a mission asks for before its waves are considered beatable.

A mission needs its ticket, some number of classes, and some number of loadout
slots. The counts climb with the tier, which is what turns a flat list of 29
missions into a progression: an Expert mission drawn in the first sphere is not
reachable until the run has handed out most of a team.

The numbers are a judgement call about MvM, not a fact about it. They are
deliberately below what a real team would want, because the logic decides what
is *possible*, and a wave that is merely hard is still possible.
"""

from dataclasses import dataclass

from . import data


@dataclass(frozen=True, slots=True)
class Requirement:
    """Items a mission's region asks for. Both counts are always satisfiable:
    the pool holds every class and every weapon slot in every seed.
    """

    classes: int
    slots: int


REQUIREMENTS: dict[str, Requirement] = {
    "normal": Requirement(classes=1, slots=1),
    "intermediate": Requirement(classes=2, slots=1),
    "advanced": Requirement(classes=3, slots=2),
    "expert": Requirement(classes=4, slots=3),
    "haunted": Requirement(classes=5, slots=3),
}

# A tier with no entry here would raise a KeyError deep inside region building,
# so it is caught at import instead.
if tuple(REQUIREMENTS) != data.DIFFICULTIES:
    raise data.DataFormatError(
        f"requirements cover {tuple(REQUIREMENTS)}, the export has {data.DIFFICULTIES}"
    )

if max(requirement.slots for requirement in REQUIREMENTS.values()) > data.WEAPON_SLOT_COUNT:
    raise data.DataFormatError("a tier asks for more weapon slots than the pool holds")

if max(requirement.classes for requirement in REQUIREMENTS.values()) > len(data.CLASS_NAMES):
    raise data.DataFormatError("a tier asks for more classes than the pool holds")
