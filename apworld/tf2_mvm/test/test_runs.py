"""Each class here is one YAML the world has to survive.

WorldTestBase brings three tests of its own to every subclass, one of them being
the sphere 0 guarantee: empty state reaches at least one location, because the
easiest mission drawn is the starting one and exactly what its tier requires is
precollected. That is the rule this world cannot get wrong, so the option sets
below attack it from the corners: shortest run, longest, hardest starting tier.
"""

import math
from typing import Any, ClassVar

from .. import data
from ..rules import REQUIREMENTS
from . import TF2MvMTestBase


class TestDefaults(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {}


class TestWholeRoster(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {"mission_count": 29, "difficulty_pool": "normal"}


class TestShortestRun(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {"mission_count": 1, "difficulty_pool": "normal"}

    def test_unlocks_fit_the_checks(self) -> None:
        # One mission can leave fewer checks than unlocks owed; the draw widens rather than failing.
        checks = sum(mission.waves + 1 for mission in self.world.missions)
        self.assertEqual(checks, len(self.multiworld.itempool))


class TestHardestPool(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {"mission_count": 4, "difficulty_pool": "expert"}

    def test_start_mission_is_open_from_nothing(self) -> None:
        self.assertTrue(self.can_reach_region(self.world.start_mission.name))

    def test_starting_inventory_covers_the_starting_tier(self) -> None:
        requirement = REQUIREMENTS[self.world.start_mission.difficulty]
        held = self.world.start_items
        self.assertIn(data.TICKET_NAMES[self.world.start_mission.id], held)
        self.assertEqual(requirement.slots, held.count(data.PROGRESSIVE_WEAPON_SLOT))
        self.assertEqual(requirement.classes, sum(name in data.CLASS_NAMES for name in held))


class TestMissionsanity(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "goal": "missionsanity",
        "missionsanity_percentage": 100,
        "mission_count": 6,
    }

    def test_every_mission_is_required(self) -> None:
        self.assertEqual(len(self.world.missions), self.world.missionsanity_target)


class TestMissionsanityPartial(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "goal": "missionsanity",
        "missionsanity_percentage": 50,
        "mission_count": 9,
    }

    def test_target_rounds_up(self) -> None:
        expected = math.ceil(len(self.world.missions) * 0.5)
        self.assertEqual(expected, self.world.missionsanity_target)


class TestFinalBoss(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "goal": "final_boss",
        "mission_count": 8,
        "difficulty_pool": "normal",
    }

    def test_goal_is_the_hardest_mission_drawn(self) -> None:
        hardest = max(
            data.DIFFICULTIES.index(mission.difficulty) for mission in self.world.missions
        )
        self.assertEqual(hardest, data.DIFFICULTIES.index(self.world.goal_mission.difficulty))

    def test_goal_needs_more_than_the_starting_inventory(self) -> None:
        self.assertFalse(self.multiworld.completion_condition[self.player](self.multiworld.state))
