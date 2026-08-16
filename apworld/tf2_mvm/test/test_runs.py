"""Each class here is one YAML the world has to survive.

WorldTestBase brings three tests of its own to every subclass: the seed fills,
every location is reachable with everything collected, and empty state reaches
at least one location. That last one is the sphere 0 guarantee, which is the
single rule this world cannot get wrong, so the option sets below are chosen to
attack it from the corners: the shortest run, the longest, and the tier where
the starting requirement is highest.
"""

import math

from . import TF2MvMTestBase
from .. import data
from ..rules import REQUIREMENTS


class TestDefaults(TF2MvMTestBase):
    options = {}


class TestWholeRoster(TF2MvMTestBase):
    options = {"mission_count": 29, "difficulty_pool": "normal"}


class TestShortestRun(TF2MvMTestBase):
    options = {"mission_count": 1, "difficulty_pool": "normal"}

    def test_unlocks_fit_the_checks(self) -> None:
        # Asking for one mission can leave fewer checks than the run owes
        # unlock items, in which case the draw widens rather than the fill
        # failing. Either way the two end up equal.
        checks = sum(mission.waves + 1 for mission in self.world.missions)
        self.assertEqual(checks, len(self.multiworld.itempool))


class TestHardestPool(TF2MvMTestBase):
    options = {"mission_count": 4, "difficulty_pool": "expert"}

    def test_start_mission_is_open_from_nothing(self) -> None:
        self.assertTrue(self.can_reach_region(self.world.start_mission.name))

    def test_starting_inventory_covers_the_starting_tier(self) -> None:
        requirement = REQUIREMENTS[self.world.start_mission.difficulty]
        held = self.world.start_items
        self.assertIn(data.TICKET_NAMES[self.world.start_mission.id], held)
        self.assertEqual(
            requirement.slots, held.count(data.PROGRESSIVE_WEAPON_SLOT)
        )
        self.assertEqual(
            requirement.classes, sum(name in data.CLASS_NAMES for name in held)
        )


class TestMissionsanity(TF2MvMTestBase):
    options = {"goal": "missionsanity", "missionsanity_percentage": 100, "mission_count": 6}

    def test_every_mission_is_required(self) -> None:
        self.assertEqual(len(self.world.missions), self.world.missionsanity_target)


class TestMissionsanityPartial(TF2MvMTestBase):
    options = {"goal": "missionsanity", "missionsanity_percentage": 50, "mission_count": 9}

    def test_target_rounds_up(self) -> None:
        expected = math.ceil(len(self.world.missions) * 0.5)
        self.assertEqual(expected, self.world.missionsanity_target)


class TestFinalBoss(TF2MvMTestBase):
    options = {"goal": "final_boss", "mission_count": 8, "difficulty_pool": "normal"}

    def test_goal_is_the_hardest_mission_drawn(self) -> None:
        hardest = max(
            data.DIFFICULTIES.index(mission.difficulty) for mission in self.world.missions
        )
        self.assertEqual(hardest, data.DIFFICULTIES.index(self.world.goal_mission.difficulty))

    def test_goal_needs_more_than_the_starting_inventory(self) -> None:
        self.assertFalse(self.multiworld.completion_condition[self.player](self.multiworld.state))
