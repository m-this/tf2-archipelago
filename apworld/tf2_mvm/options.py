"""The YAML options, hand-written.

Option classes are not a table: their docstrings are what the Archipelago
website shows the player, so they are code and they stay code, even though the
values they range over come from the generated data. Where an option enumerates
something the Go side owns, the enumeration is checked against the export at
import time rather than trusted.
"""

from dataclasses import dataclass

from Options import Choice, DeathLink, OptionGroup, PerGameCommonOptions, Range

from . import data


class MissionCount(Range):
    """How many missions the run spans.

    Missions are drawn from the tiers the difficulty pool allows. Asking for
    more than that pool holds gets you the whole pool.
    """

    display_name = "Mission Count"
    range_start = 1
    range_end = len(data.MISSIONS)
    default = 8


class DifficultyPool(Choice):
    """The easiest tier that may appear. Every harder tier is included too.

    Picking haunted leaves exactly one mission in the pool, which is a very
    short run.
    """

    display_name = "Difficulty Pool"
    option_normal = 0
    option_intermediate = 1
    option_advanced = 2
    option_expert = 3
    option_haunted = 4
    default = 1


class Goal(Choice):
    """What ends the run.

    Final Boss flags the hardest mission drawn; clearing it wins. Missionsanity
    wants a share of every mission in the run instead, in any order.
    """

    display_name = "Goal"
    option_final_boss = 0
    option_missionsanity = 1
    default = 0


class MissionsanityPercentage(Range):
    """How much of the run Missionsanity wants, as a percentage of the missions
    drawn, rounded up. Ignored by the Final Boss goal.
    """

    display_name = "Missionsanity Percentage"
    range_start = 10
    range_end = 100
    default = 80


@dataclass
class TF2MvMOptions(PerGameCommonOptions):
    mission_count: MissionCount
    difficulty_pool: DifficultyPool
    goal: Goal
    missionsanity_percentage: MissionsanityPercentage
    death_link: DeathLink


option_groups = [
    OptionGroup("Run shape", [MissionCount, DifficultyPool]),
    OptionGroup("Goal", [Goal, MissionsanityPercentage]),
]

# DifficultyPool enumerates a table the Go side owns, so it is checked rather
# than trusted: a tier added there and not here would silently drop missions.
if tuple(DifficultyPool.options) != data.DIFFICULTIES:
    raise data.DataFormatError(
        f"difficulty_pool offers {tuple(DifficultyPool.options)}, "
        f"the export has {data.DIFFICULTIES}"
    )
