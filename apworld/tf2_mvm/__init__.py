"""Team Fortress 2 Mann vs Machine.

The item and location tables come from ``gamedata/`` in Go and are read out of
``data/`` by :mod:`.data`; see ADR 0001. What lives here is the part that is not
data: which missions a seed draws, how they gate each other, what the goal is,
and what the server needs to know afterwards.
"""

import logging
import math
from collections.abc import Callable
from typing import ClassVar

from BaseClasses import CollectionState, Item, ItemClassification, Location, Region, Tutorial
from Options import OptionError
from worlds.AutoWorld import WebWorld, World

from . import data
from .options import RANDOM, TF2MvMOptions, option_groups
from .rules import REQUIREMENTS, Requirement

CLASSIFICATIONS = {
    "progression": ItemClassification.progression,
    "useful": ItemClassification.useful,
    "filler": ItemClassification.filler,
    "trap": ItemClassification.trap,
}

BUFF_REQUIREMENTS = {
    "normal": 1,
    "intermediate": 2,
    "advanced": 3,
    "expert": 4,
    "haunted": 5,
}

IMPORTANCE_OPTION_BY_KIND = {
    "mission_ticket": "mission_ticket_importance",
    "class": "class_unlock_importance",
    "weapon_slot": "weapon_slot_importance",
    "weapon_buff": "weapon_buff_importance",
}


class TF2MvMItem(Item):
    game = data.GAME


class TF2MvMLocation(Location):
    game = data.GAME


class TF2MvMWeb(WebWorld):
    theme = "dirt"
    option_groups = option_groups
    game_info_languages: ClassVar[list[str]] = ["en", "fr"]
    tutorials: ClassVar[list[Tutorial]] = [
        Tutorial(
            "Multiworld Setup Guide",
            "A guide to running a Team Fortress 2 Mann vs Machine server for Archipelago.",
            "English",
            "setup_en.md",
            "setup/en",
            ["mathis"],
        ),
        Tutorial(
            "Multiworld Setup Guide",
            "Un guide pour héberger un serveur Team Fortress 2 Mann vs Machine pour Archipelago.",
            "Français",
            "setup_fr.md",
            "setup/fr",
            ["mathis"],
        ),
    ]


class TF2MvMWorld(World):
    """Mann vs Machine is the co-operative mode of Team Fortress 2. A run draws
    a set of missions. Each wave that the team clears is a check. The mission
    tickets, the mercenary classes and the loadout slots are the items. The
    whole server plays one slot, so all the players share the unlocks and
    nobody installs a modification.
    """

    game = data.GAME
    options_dataclass = TF2MvMOptions
    options: TF2MvMOptions
    web = TF2MvMWeb()
    topology_present = True

    item_name_to_id = data.ITEM_NAME_TO_ID
    location_name_to_id = data.LOCATION_NAME_TO_ID
    item_name_groups = data.ITEM_NAME_GROUPS

    missions: list[data.Mission]
    start_mission: data.Mission
    goal_mission: data.Mission
    start_items: list[str]
    missionsanity_target: int

    def generate_early(self) -> None:
        available = self._available_missions()
        if not available:
            raise OptionError(
                f"{self.player_name}: excluded_missions leaves nothing in a "
                f"{self.options.difficulty_pool.current_key} pool."
            )
        asked = self._asked_start_mission(available)
        wanted = min(self.options.mission_count.value, len(available))
        drawn = self._draw(available, wanted, asked)
        spare = [mission for mission in available if mission not in drawn]

        # Widening closes a shortfall: a mission adds more waves as checks than the ticket it costs.
        while self._shortfall(drawn, asked) > 0 and spare:
            drawn.append(spare.pop(self.random.randrange(len(spare))))
        if self._shortfall(drawn, asked) > 0:
            raise OptionError(
                f"{self.player_name}: a {self.options.difficulty_pool.current_key} pool holds "
                f"{len(available)} mission(s), too few checks for the unlocks the run needs. "
                f"Lower difficulty_pool."
            )
        if len(drawn) > wanted:
            logging.info(
                "%s: drew %d missions rather than %d, so the run has room for its unlocks",
                self.player_name,
                len(drawn),
                wanted,
            )

        self.missions = sorted(drawn, key=self._tier_order)
        self.start_mission = asked or self.missions[0]
        self.goal_mission = self.missions[-1]
        # Clearing the start mission would win on the spot, which is not a run.
        if (
            self.options.goal.current_key == "final_boss"
            and self.goal_mission is self.start_mission
            and len(self.missions) > 1
        ):
            raise OptionError(
                f"{self.player_name}: start_mission is {self.start_mission.name!r}, the hardest "
                f"mission of the run, and the Final Boss goal is that mission. Name an easier "
                f"start_mission, or set goal to missionsanity."
            )

        share = self.options.missionsanity_percentage.value / 100
        self.missionsanity_target = max(1, math.ceil(len(self.missions) * share))

        requirement = REQUIREMENTS[self.start_mission.difficulty]
        self.start_items = [
            data.TICKET_NAMES[self.start_mission.id],
            *self._start_classes(requirement.classes),
            *[data.PROGRESSIVE_WEAPON_SLOT] * requirement.slots,
        ]

    def _asked_start_mission(self, available: list[data.Mission]) -> data.Mission | None:
        """The mission start_mission names, or None for the easiest one drawn."""
        wanted = self.options.start_mission.value
        if wanted == RANDOM:
            return None
        for mission in available:
            if mission.name == wanted:
                return mission
        raise OptionError(
            f"{self.player_name}: start_mission is {wanted!r}, which a "
            f"{self.options.difficulty_pool.current_key} pool does not hold, or "
            f"excluded_missions keeps out."
        )

    def _draw(
        self, available: list[data.Mission], wanted: int, asked: data.Mission | None
    ) -> list[data.Mission]:
        if asked is None:
            return self.random.sample(available, wanted)
        rest = [mission for mission in available if mission is not asked]
        return [asked, *self.random.sample(rest, wanted - 1)]

    def _start_classes(self, count: int) -> list[str]:
        """The classes the run starts with, one of them named by start_class."""
        wanted = self.options.start_class.value
        if wanted == RANDOM:
            return self.random.sample(data.CLASS_NAMES, count)
        first = data.CLASS_ITEM_BY_MERC[wanted]
        rest = [name for name in data.CLASS_NAMES if name != first]
        return [first, *self.random.sample(rest, count - 1)]

    def create_regions(self) -> None:
        menu = Region(self.origin_region_name, self.player, self.multiworld)
        self.multiworld.regions.append(menu)
        for mission in self.missions:
            region = Region(mission.name, self.player, self.multiworld)
            region.add_locations(
                {location.name: location.id for location in mission.locations}, TF2MvMLocation
            )
            self.multiworld.regions.append(region)
            menu.connect(region, f"Deploy to {mission.name}", self._deploy_rule(mission))

    def create_items(self) -> None:
        for name in self.start_items:
            self.push_precollected(self.create_item(name))

        pool = [
            self.create_item(data.TICKET_NAMES[mission.id])
            for mission in self.missions
            if mission is not self.start_mission
        ]
        pool += [
            self.create_item(name) for name in data.CLASS_NAMES if name not in self.start_items
        ]
        slots_held = self.start_items.count(data.PROGRESSIVE_WEAPON_SLOT)
        pool += [
            self.create_item(data.PROGRESSIVE_WEAPON_SLOT)
            for _ in range(data.WEAPON_SLOT_COUNT - slots_held)
        ]

        # Buffs and cash share the non-progression space. Numeric permutations
        # may repeat as levels; toggle permutations remain unique.
        open_slots = self._check_count(self.missions) - len(pool)
        buff_count = open_slots
        if self.options.cash_rewards.value:
            buff_count = math.ceil(open_slots * self.options.weapon_buff_percentage.value / 100)
        if self.options.weapon_buff_importance.current_key == "progression":
            buff_count = max(buff_count, max(BUFF_REQUIREMENTS.values()))
        pool += [self.create_item(name) for name in self._draw_weapon_buffs(buff_count)]
        pool += [self.create_filler() for _ in range(self._check_count(self.missions) - len(pool))]
        self.multiworld.itempool += pool

    def _draw_weapon_buffs(self, count: int) -> list[str]:
        """Draw reward names, repeating numeric buffs but never toggles."""
        unused = list(data.WEAPON_BUFF_NAMES)
        stackable_drawn: list[str] = []
        drawn: list[str] = []
        stack_chance = self.options.weapon_buff_stack_chance.value
        while len(drawn) < count and (unused or stackable_drawn):
            stack = bool(stackable_drawn) and self.random.randrange(100) < stack_chance
            if stack or not unused:
                drawn.append(self.random.choice(stackable_drawn))
                continue
            name = unused.pop(self.random.randrange(len(unused)))
            drawn.append(name)
            if name in data.STACKABLE_WEAPON_BUFF_NAMES:
                stackable_drawn.append(name)
        return drawn

    def set_rules(self) -> None:
        if self.options.goal.current_key == "missionsanity":
            self.multiworld.completion_condition[self.player] = self._missionsanity_rule()
        else:
            self.multiworld.completion_condition[self.player] = self._final_boss_rule()

    def create_item(self, name: str) -> TF2MvMItem:
        item = data.ITEMS_BY_NAME[name]
        classification = item.classification
        if option_name := IMPORTANCE_OPTION_BY_KIND.get(item.kind):
            classification = getattr(self.options, option_name).current_key
        return TF2MvMItem(name, CLASSIFICATIONS[classification], item.id, self.player)

    def get_filler_item_name(self) -> str:
        return self.random.choice(data.FILLER_NAMES)

    def fill_slot_data(self) -> dict[str, object]:
        # Only what the bridge cannot work out from gamedata alone.
        return {
            "format_version": data.FORMAT_VERSION,
            "missions": [mission.pop_file for mission in self.missions],
            "goal": self.options.goal.current_key,
            "start_mission": self.start_mission.pop_file,
            "goal_mission": self.goal_mission.pop_file,
            "missionsanity_target": self.missionsanity_target,
            "death_link": bool(self.options.death_link.value),
            "mission_ticket_importance": self.options.mission_ticket_importance.current_key,
        }

    def _available_missions(self) -> list[data.Mission]:
        floor = data.DIFFICULTIES.index(self.options.difficulty_pool.current_key)
        allowed = data.DIFFICULTIES[floor:]
        excluded = self.options.excluded_missions.value
        return [
            mission
            for mission in data.MISSIONS
            if mission.playable and mission.difficulty in allowed and mission.name not in excluded
        ]

    @staticmethod
    def _tier_order(mission: data.Mission) -> tuple[int, int]:
        return data.DIFFICULTIES.index(mission.difficulty), mission.id

    @staticmethod
    def _check_count(missions: list[data.Mission]) -> int:
        # The export decides how many checks a mission holds. Counting the
        # waves and adding one for the clear stopped being that the day a
        # mission grew a tank check.
        return sum(len(mission.locations) for mission in missions)

    def _shortfall(self, missions: list[data.Mission], start: data.Mission | None) -> int:
        """Unlock items owed minus the checks there is room for; filler only closes a surplus."""
        if start is None:
            start = min(missions, key=self._tier_order)
        requirement = REQUIREMENTS[start.difficulty]
        unlocks = (
            len(missions)
            - 1
            + len(data.CLASS_NAMES)
            - requirement.classes
            + data.WEAPON_SLOT_COUNT
            - requirement.slots
        )
        if self.options.weapon_buff_importance.current_key == "progression":
            unlocks += max(BUFF_REQUIREMENTS.values())
        return unlocks - self._check_count(missions)

    def _deploy_rule(self, mission: data.Mission) -> Callable[[CollectionState], bool]:
        ticket = data.TICKET_NAMES[mission.id]
        requirement: Requirement = REQUIREMENTS[mission.difficulty]
        player = self.player

        def can_deploy(state: CollectionState) -> bool:
            ticket_ready = (
                self.options.mission_ticket_importance.current_key == "useful"
                or state.has(ticket, player)
            )
            classes_ready = (
                self.options.class_unlock_importance.current_key == "useful"
                or state.has_group("Classes", player, requirement.classes)
            )
            slots_ready = self.options.weapon_slot_importance.current_key == "useful" or state.has(
                data.PROGRESSIVE_WEAPON_SLOT, player, requirement.slots
            )
            buffs_ready = (
                mission is self.start_mission
                or self.options.weapon_buff_importance.current_key == "useful"
                or state.has_group("Weapon Buffs", player, BUFF_REQUIREMENTS[mission.difficulty])
            )
            return ticket_ready and classes_ready and slots_ready and buffs_ready

        return can_deploy

    def _final_boss_rule(self) -> Callable[[CollectionState], bool]:
        goal = f"{self.goal_mission.name} Complete"
        player = self.player
        return lambda state: state.can_reach_location(goal, player)

    def _missionsanity_rule(self) -> Callable[[CollectionState], bool]:
        clears = [f"{mission.name} Complete" for mission in self.missions]
        target = self.missionsanity_target
        player = self.player
        return lambda state: (
            sum(state.can_reach_location(clear, player) for clear in clears) >= target
        )
