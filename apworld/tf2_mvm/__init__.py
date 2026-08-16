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
from .options import TF2MvMOptions, option_groups
from .rules import REQUIREMENTS, Requirement

CLASSIFICATIONS = {
    "progression": ItemClassification.progression,
    "useful": ItemClassification.useful,
    "filler": ItemClassification.filler,
    "trap": ItemClassification.trap,
}


class TF2MvMItem(Item):
    game = data.GAME


class TF2MvMLocation(Location):
    game = data.GAME


class TF2MvMWeb(WebWorld):
    theme = "dirt"
    option_groups = option_groups
    tutorials: ClassVar[list[Tutorial]] = [
        Tutorial(
            "Multiworld Setup Guide",
            "A guide to running a Team Fortress 2 Mann vs Machine server for Archipelago.",
            "English",
            "setup_en.md",
            "setup/en",
            ["mathis"],
        )
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
        wanted = min(self.options.mission_count.value, len(available))
        drawn = self.random.sample(available, wanted)
        spare = [mission for mission in available if mission not in drawn]

        # Widening closes a shortfall: a mission adds more waves as checks than the ticket it costs.
        while self._shortfall(drawn) > 0 and spare:
            drawn.append(spare.pop(self.random.randrange(len(spare))))
        if self._shortfall(drawn) > 0:
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
        self.start_mission = self.missions[0]
        self.goal_mission = self.missions[-1]

        share = self.options.missionsanity_percentage.value / 100
        self.missionsanity_target = max(1, math.ceil(len(self.missions) * share))

        requirement = REQUIREMENTS[self.start_mission.difficulty]
        self.start_items = [
            data.TICKET_NAMES[self.start_mission.id],
            *self.random.sample(data.CLASS_NAMES, requirement.classes),
            *[data.PROGRESSIVE_WEAPON_SLOT] * requirement.slots,
        ]

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
        pool += [self.create_filler() for _ in range(self._check_count(self.missions) - len(pool))]
        self.multiworld.itempool += pool

    def set_rules(self) -> None:
        if self.options.goal.current_key == "missionsanity":
            self.multiworld.completion_condition[self.player] = self._missionsanity_rule()
        else:
            self.multiworld.completion_condition[self.player] = self._final_boss_rule()

    def create_item(self, name: str) -> TF2MvMItem:
        item = data.ITEMS_BY_NAME[name]
        return TF2MvMItem(name, CLASSIFICATIONS[item.classification], item.id, self.player)

    def get_filler_item_name(self) -> str:
        return self.random.choice(data.FILLER_NAMES)

    def fill_slot_data(self) -> dict[str, object]:
        # Only what the bridge cannot work out from gamedata alone.
        return {
            "format_version": data.FORMAT_VERSION,
            "missions": [mission.pop_file for mission in self.missions],
            "goal": self.options.goal.current_key,
            "goal_mission": self.goal_mission.pop_file,
            "missionsanity_target": self.missionsanity_target,
            "death_link": bool(self.options.death_link.value),
        }

    def _available_missions(self) -> list[data.Mission]:
        floor = data.DIFFICULTIES.index(self.options.difficulty_pool.current_key)
        allowed = data.DIFFICULTIES[floor:]
        return [mission for mission in data.MISSIONS if mission.difficulty in allowed]

    @staticmethod
    def _tier_order(mission: data.Mission) -> tuple[int, int]:
        return data.DIFFICULTIES.index(mission.difficulty), mission.id

    @staticmethod
    def _check_count(missions: list[data.Mission]) -> int:
        return sum(mission.waves + 1 for mission in missions)

    def _shortfall(self, missions: list[data.Mission]) -> int:
        """Unlock items owed minus the checks there is room for; filler only closes a surplus."""
        requirement = REQUIREMENTS[min(missions, key=self._tier_order).difficulty]
        unlocks = (
            len(missions)
            - 1
            + len(data.CLASS_NAMES)
            - requirement.classes
            + data.WEAPON_SLOT_COUNT
            - requirement.slots
        )
        return unlocks - self._check_count(missions)

    def _deploy_rule(self, mission: data.Mission) -> Callable[[CollectionState], bool]:
        ticket = data.TICKET_NAMES[mission.id]
        requirement: Requirement = REQUIREMENTS[mission.difficulty]
        player = self.player

        def can_deploy(state: CollectionState) -> bool:
            return (
                state.has(ticket, player)
                and state.has_group("Classes", player, requirement.classes)
                and state.has(data.PROGRESSIVE_WEAPON_SLOT, player, requirement.slots)
            )

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
