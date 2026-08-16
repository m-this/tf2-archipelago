"""The YAML options, hand-written.

These stay code rather than generated data: their docstrings are what the
Archipelago website shows the player.
"""

from dataclasses import dataclass

from Options import Choice, DeathLink, OptionGroup, PerGameCommonOptions, Range

from . import data


class MissionCount(Range):
    """How many missions the run uses.

    The run draws them from the tiers that the difficulty pool allows. A
    request for more missions than the pool holds gives you the whole pool.
    """

    display_name = "Mission Count"
    range_start = 1
    range_end = len(data.MISSIONS)
    default = 8


class DifficultyPool(Choice):
    """The easiest tier that the run can draw. The run also draws every tier
    above it.

    The haunted tier holds one mission. One mission gives too few checks for
    the items of a run, and generation stops with an error.
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

    Final Boss marks the most difficult mission that the run drew. Clear that
    mission to win. Missionsanity asks for a part of the missions instead, in
    any sequence.
    """

    display_name = "Goal"
    option_final_boss = 0
    option_missionsanity = 1
    default = 0


class MissionsanityPercentage(Range):
    """How much of the run Missionsanity asks for.

    The percentage applies to the missions that the run drew, and it rounds up.
    The Final Boss goal ignores this option.
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

# Checked rather than trusted: a tier added to the export and not here would silently drop missions.
if tuple(DifficultyPool.options) != data.DIFFICULTIES:
    raise data.DataFormatError(
        f"difficulty_pool offers {tuple(DifficultyPool.options)}, "
        f"the export has {data.DIFFICULTIES}"
    )
