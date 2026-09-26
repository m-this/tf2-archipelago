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


class CommunityMissions(Toggle):
    """Whether the run draws community missions at all.

    Every community mission the server can play is in the pool by default, and
    the Options Creator ticks them all. Off keeps the run to Valve's missions
    without naming each community one in excluded_missions.
    """

    display_name = "Community Missions"
    default = 1


class MissionModifiers(Toggle):
    """Give each drawn mission a persistent set of gameplay modifiers.

    The seed chooses modifiers once, per mission. Returning to a mission keeps
    the same combination, and multiple compatible modifiers may stack.
    """

    display_name = "Mission Modifiers"
    default = 0


class MinimumMissionModifiers(Range):
    """Fewest modifiers assigned to each mission when Mission Modifiers is on."""

    display_name = "Minimum Mission Modifiers"
    range_start = 0
    range_end = 3
    default = 1


class MaximumMissionModifiers(Range):
    """Most modifiers assigned to each mission when Mission Modifiers is on."""

    display_name = "Maximum Mission Modifiers"
    range_start = 0
    range_end = 3
    default = 2


class ServerMods(OptionSet):
    """Server-side mods the game server loads, by key.

    Some community missions need a mod the stock server does not have. The
    run draws those only when the mod is named here, so a seed never asks a
    server for a mission it cannot run. sigsegv-mvm is SigMod: upstream ships
    a Linux server build, and the Windows launcher installs this project's
    port of it.
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


class ClassWeaponSlots(Choice):
    """Open loadout slots class by class rather than for everybody at once.

    Off is one Progressive Weapon Slot for everybody, three copies.

    Progressive gives each class an item of its own, two copies, and it earns
    its second and third slots with them in its own order. Its first slot
    comes free with the class, and it is the slot that class is played with:
    the Medigun for a Medic, the Knife for a Spy, the Wrench for an Engineer.

    Any order is the same eighteen items, each naming the slot it opens, found
    in whatever order the multiworld puts them in: a Scout can find his melee
    before his secondary. Safe because the free first slot is the one that
    class is played with, so no class is ever left holding only its worst
    weapon, and the items read as what they are in somebody else's spoiler log.

    Either way it is eighteen items in the pool where there were three, and a
    longer run for it. A run with few missions may not have the checks to hold
    them, and generation says so. A mission at a tier still needs that tier's
    count of classes with that tier's count of slots.
    """

    display_name = "Weapon Slots per Class"
    option_off = 0
    option_progressive = 1
    option_any_order = 2
    alias_false = 0
    alias_true = 1
    default = 0


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
    every spare check awards a weapon buff and this option is ignored. It is
    also ignored when weapon buffs are disabled.
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
    """Whether this reward gates access, is optional, or is disabled."""

    option_useful = 0
    option_progression = 1
    option_disabled = 2


class MissionTicketImportance(RewardImportance):
    """Progression tickets are required to deploy to their missions.

    Useful tickets do not gate deployment; all missions drawn by the seed are
    available from the start. Disabled tickets are all unlocked at the start
    and are not placed as rewards.
    """

    display_name = "Mission Ticket Importance"
    default = 1


class ClassUnlockImportance(RewardImportance):
    """Progression class unlocks satisfy each mission tier's class requirement.

    Useful class unlocks still expand the roster but never block deployment.
    Disabled unlocks make every class playable from the start.
    """

    display_name = "Class Unlock Importance"
    default = 1


class WeaponSlotImportance(RewardImportance):
    """Progression weapon slots satisfy each mission tier's loadout requirement.

    Useful slots still expand loadouts but never block deployment. Disabled
    slots unlock every loadout slot from the start, including per-class slots.
    """

    display_name = "Weapon Slot Importance"
    default = 1


class WeaponBuffImportance(RewardImportance):
    """Useful buffs are optional upgrades. Progression buffs are also required
    in increasing numbers for harder mission tiers. Disabled awards no weapon
    buffs; spare checks receive cash filler instead.
    """

    display_name = "Weapon Buff Importance"
    default = 0


class CashRewards(Toggle):
    """Put cash filler in some spare checks.

    Disabled by default because cash is temporary and less satisfying than a
    persistent weapon buff. When disabled, every spare check is a buff unless
    weapon buffs themselves are disabled; then those checks contain cash.
    """

    display_name = "Cash Rewards"
    default = 0


class BotCards(Toggle):
    """Add useful, optional named bot cards to the reward pool. A bot card
    never gates a mission. Off leaves the card system out of the seed.
    """

    display_name = "Unlockable Bot Cards"
    default = 0


class StartingBotCards(Range):
    """Number of distinct random bot cards to start with. Ignored when the
    starting-card mode is One Stock Card per Class. Starting cards do not use
    checks from the maximum below.
    """

    display_name = "Starting Bot Cards"
    range_start = 0
    range_end = len(data.BOT_CARD_BASES)
    default = 0


class StartingBotCardMode(Choice):
    """Random draws the requested number of starting cards. One Stock Card
    per Class starts with nine stock-loadout cards, one for each mercenary;
    the random starting count is ignored. Rarity and form still roll normally.
    """

    display_name = "Starting Bot Card Mode"
    option_draw_random = 0
    option_stock_classes = 1
    default = 0


class ServerSettings(Toggle):
    """Put the server-setting items in the pool.

    One of them exists: the Grappling Hook, which turns on Mannpower's hook for
    everybody on the server for the rest of the run. It is the largest change to
    how a map plays that this world can hand out, and it is off by default
    because a run that has it is a different game from one that does not.

    Never required to beat anything: a wave stays winnable without it.
    """

    display_name = "Server Settings"
    default = 0


class MedalOnClear(Toggle):
    """Lock an Australium Medal onto every mission clear.

    The goal then reads the medals you hold rather than the clears the server
    reported, which is what generation wants: a medal is your own item and no
    !collect can hand you one.

    The cost is one check per mission. A mission clear is one of the better
    rewards this world puts into a multiworld, and locking it takes that many
    of other people's items out of the pool. Off by default for that reason.
    """

    display_name = "Australium Medal on Clear"
    default = 0


class VictoryCaches(Toggle):
    """Make a mission clear worth checks scaled by its tier.

    A normal clear stays one check; intermediate pays two, advanced three,
    expert and haunted five. The extra checks are victory caches, and the
    server hands them over with the clear, so there is nothing new to do in
    the game.

    The cheap answer to a run short on checks, because it adds no mission. Off
    by default so an existing YAML keeps meaning what it meant.
    """

    display_name = "Victory Caches"
    default = 0


class MilestoneChecks(Toggle):
    """Add checks for running totals over the whole run.

    So many robots, giants and tanks destroyed, whatever mission they fell in:
    fifteen checks on a fixed ladder, from a hundred robots to four thousand,
    five tanks to forty, ten giants to two hundred. Any mission the run has
    open progresses them, so they fill a run short on checks and give a team
    stuck on a wave something to grind.

    Off by default so an existing YAML keeps meaning what it meant.
    """

    display_name = "Milestone Checks"


class Giantsanity(Toggle):
    """A check for every giant of every wave, beside the mission's first.

    The counts come out of Valve's population files, so a giant that is there
    to kill is a check and one that is not is not; community missions have
    no count yet and pay only their first. Empire Escalation alone holds
    eighty-two. Off by default: on, a run is buffed out the wazoo.
    """

    display_name = "Giantsanity"
    default = 0


class Tanksanity(Toggle):
    """A check for every tank of every wave, beside the mission's first.

    Read out of Valve's population files like the giants. Cataclysm holds
    eleven. Off by default.
    """

    display_name = "Tanksanity"
    default = 0


class TrapPercentage(Range):
    """How much of the run's spare space is traps, in percent.

    A trap is an item with a negative effect: another player opens a chest and
    your team gets Jarate. They come out of the same space as the weapon buffs
    and the cash, so raising this lowers those rather than adding checks.

    One percent by default, so a run meets a trap now and then; zero leaves
    them out. A trap can cost the team a wave and never costs the run an
    unlock it already holds.
    """

    display_name = "Trap Percentage"
    range_start = 0
    range_end = 100
    default = 1


@dataclass
class TF2MvMOptions(PerGameCommonOptions):
    mission_count: MissionCount
    difficulty_pool: DifficultyPool
    excluded_missions: ExcludedMissions
    community_missions: CommunityMissions
    mission_modifiers: MissionModifiers
    minimum_mission_modifiers: MinimumMissionModifiers
    maximum_mission_modifiers: MaximumMissionModifiers
    server_mods: ServerMods
    start_mission: StartMission
    start_class: StartClass
    class_weapon_slots: ClassWeaponSlots
    goal: Goal
    missionsanity_percentage: MissionsanityPercentage
    mission_ticket_importance: MissionTicketImportance
    class_unlock_importance: ClassUnlockImportance
    weapon_slot_importance: WeaponSlotImportance
    weapon_buff_importance: WeaponBuffImportance
    cash_rewards: CashRewards
    bot_cards: BotCards
    starting_bot_cards: StartingBotCards
    starting_bot_card_mode: StartingBotCardMode
    weapon_buff_percentage: WeaponBuffPercentage
    weapon_buff_stack_chance: WeaponBuffStackChance
    trap_percentage: TrapPercentage
    server_settings: ServerSettings
    medal_on_clear: MedalOnClear
    victory_caches: VictoryCaches
    milestone_checks: MilestoneChecks
    giantsanity: Giantsanity
    tanksanity: Tanksanity
    death_link: DeathLink


option_groups = [
    OptionGroup(
        "Run shape",
        [
            MissionCount,
            DifficultyPool,
            ExcludedMissions,
            CommunityMissions,
            MissionModifiers,
            MinimumMissionModifiers,
            MaximumMissionModifiers,
            ServerMods,
            StartMission,
            StartClass,
            ClassWeaponSlots,
        ],
    ),
    OptionGroup(
        "Goal",
        [
            Goal,
            MissionsanityPercentage,
            MedalOnClear,
            VictoryCaches,
            MilestoneChecks,
            Giantsanity,
            Tanksanity,
        ],
    ),
    OptionGroup(
        "Rewards",
        [
            MissionTicketImportance,
            ClassUnlockImportance,
            WeaponSlotImportance,
            WeaponBuffImportance,
            CashRewards,
            BotCards,
            StartingBotCards,
            StartingBotCardMode,
            WeaponBuffPercentage,
            WeaponBuffStackChance,
            TrapPercentage,
        ],
    ),
]

# Checked rather than trusted: a tier added to the export and not here would silently drop missions.
if tuple(DifficultyPool.options) != data.DIFFICULTIES:
    raise data.DataFormatError(
        f"difficulty_pool offers {tuple(DifficultyPool.options)}, "
        f"the export has {data.DIFFICULTIES}"
    )
