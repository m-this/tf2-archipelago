"""The YAML options, hand-written.

These stay code rather than generated data: their docstrings are what the
Archipelago website shows the player.
"""

from dataclasses import dataclass

from Options import (
    Choice,
    DeathLink,
    FreeText,
    OptionError,
    OptionGroup,
    OptionSet,
    PerGameCommonOptions,
    Range,
    Toggle,
)

from . import data


class MissionCount(Range):
    """How many missions the run uses.

    The run draws them from the tiers that the difficulty pool allows. A
    request for more missions than the pool holds gives you the whole pool.
    """

    display_name = "Mission Count"
    range_start = 1
    range_end = len(data.MISSION_NAMES)
    default = 8


class DifficultyPool(Choice):
    """The easiest tier that the run can draw. The run also draws every tier
    above it.

    The haunted tier holds only Caliginous Caper. Its wave, tank, giant and
    completion checks provide exactly enough locations for its remaining
    class unlocks, but it has no mission-ticket progression.
    """

    display_name = "Difficulty Pool"
    option_normal = 0
    option_intermediate = 1
    option_advanced = 2
    option_expert = 3
    option_haunted = 4
    default = 1


class ExcludedMissions(OptionSet):
    """Missions the run never draws, by name.

    Use it to keep out a mission that is too long for the evening. Caliginous
    Caper is one mission of 666 robots and takes an hour on its own.
    """

    display_name = "Excluded Missions"
    valid_keys = data.MISSION_NAMES


class ServerMods(OptionSet):
    """Server-side mods the game server loads, by key.

    Some community missions need a mod the stock server does not have. The
    run draws those only when the mod is named here, so a seed never asks a
    server for a mission it cannot run. sigsegv-mvm is SigMod, which upstream
    ships for Linux servers only.
    """

    display_name = "Server Mods"
    valid_keys = data.SERVER_MOD_KEYS


# FreeText and not Choice: a Choice needs one class attribute per value, and
# both of these draw their values from the export rather than from this file.
RANDOM = "random"


class StartMission(FreeText):
    """The mission the run starts on.

    `random` starts the run on the easiest mission it drew. Name a mission
    instead and the run always draws that one and starts there. The difficulty
    pool has to hold it, and Excluded Missions must not.
    """

    display_name = "Start Mission"
    default = RANDOM

    def verify(self, _world, player_name, _plando_options) -> None:
        if self.value == RANDOM or self.value in data.MISSION_NAMES:
            return
        raise OptionError(
            f"{player_name}: start_mission is {self.value!r}, which is not the name of a mission."
        )


class StartClass(FreeText):
    """The mercenary the run starts with.

    `random` draws every starting class at random. Name one instead and the run
    always starts with it. The tier of the start mission decides how many
    classes the run starts with, and this option names one of them.
    """

    display_name = "Start Class"
    default = RANDOM

    def verify(self, _world, player_name, _plando_options) -> None:
        if self.value == RANDOM or self.value in data.CLASS_ITEM_BY_MERC:
            return
        raise OptionError(
            f"{player_name}: start_class is {self.value!r}. "
            f"Name one of {', '.join(sorted(data.CLASS_ITEM_BY_MERC))}."
        )


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


class WeaponBuffPercentage(Range):
    """Percentage of spare checks that award a weapon buff when cash rewards
    are enabled.

    The remaining space contains cash filler. With cash rewards disabled,
    every spare check awards a weapon buff and this option is ignored.
    """

    display_name = "Weapon Buff Percentage"
    range_start = 0
    range_end = 100
    default = 75


class WeaponBuffStackChance(Range):
    """Chance that a weapon-buff reward adds another level to a numeric buff
    already drawn for this seed.

    On/off effects are never repeated. At zero every buff reward is a distinct
    weapon/effect permutation.
    """

    display_name = "Weapon Buff Stack Chance"
    range_start = 0
    range_end = 100
    default = 25


class RewardImportance(Choice):
    """Whether this reward can gate access or is an optional power-up."""

    option_useful = 0
    option_progression = 1


class MissionTicketImportance(RewardImportance):
    """Progression tickets are required to deploy to their missions.

    Useful tickets do not gate deployment; all missions drawn by the seed are
    available from the start.
    """

    display_name = "Mission Ticket Importance"
    default = 1


class ClassUnlockImportance(RewardImportance):
    """Progression class unlocks satisfy each mission tier's class requirement.

    Useful class unlocks still expand the roster but never block deployment.
    """

    display_name = "Class Unlock Importance"
    default = 1


class WeaponSlotImportance(RewardImportance):
    """Progression weapon slots satisfy each mission tier's loadout requirement.

    Useful slots still expand loadouts but never block deployment.
    """

    display_name = "Weapon Slot Importance"
    default = 1


class WeaponBuffImportance(RewardImportance):
    """Useful buffs are optional upgrades. Progression buffs are also required
    in increasing numbers for harder mission tiers.
    """

    display_name = "Weapon Buff Importance"
    default = 0


class CashRewards(Toggle):
    """Put cash filler in some spare checks.

    Disabled by default because cash is temporary and less satisfying than a
    persistent weapon buff. When disabled, every spare check is a buff.
    """

    display_name = "Cash Rewards"
    default = 0


@dataclass
class TF2MvMOptions(PerGameCommonOptions):
    mission_count: MissionCount
    difficulty_pool: DifficultyPool
    excluded_missions: ExcludedMissions
    server_mods: ServerMods
    start_mission: StartMission
    start_class: StartClass
    goal: Goal
    missionsanity_percentage: MissionsanityPercentage
    mission_ticket_importance: MissionTicketImportance
    class_unlock_importance: ClassUnlockImportance
    weapon_slot_importance: WeaponSlotImportance
    weapon_buff_importance: WeaponBuffImportance
    cash_rewards: CashRewards
    weapon_buff_percentage: WeaponBuffPercentage
    weapon_buff_stack_chance: WeaponBuffStackChance
    death_link: DeathLink


option_groups = [
    OptionGroup(
        "Run shape",
        [MissionCount, DifficultyPool, ExcludedMissions, ServerMods, StartMission, StartClass],
    ),
    OptionGroup("Goal", [Goal, MissionsanityPercentage]),
    OptionGroup(
        "Rewards",
        [
            MissionTicketImportance,
            ClassUnlockImportance,
            WeaponSlotImportance,
            WeaponBuffImportance,
            CashRewards,
            WeaponBuffPercentage,
            WeaponBuffStackChance,
        ],
    ),
]

# Checked rather than trusted: a tier added to the export and not here would silently drop missions.
if tuple(DifficultyPool.options) != data.DIFFICULTIES:
    raise data.DataFormatError(
        f"difficulty_pool offers {tuple(DifficultyPool.options)}, "
        f"the export has {data.DIFFICULTIES}"
    )
