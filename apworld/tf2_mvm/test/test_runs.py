"""Each class here is one YAML the world has to survive.

WorldTestBase brings three tests of its own to every subclass, one of them being
the sphere 0 guarantee: empty state reaches at least one location, because the
easiest mission drawn is the starting one and exactly what its tier requires is
precollected. That is the rule this world cannot get wrong, so the option sets
below attack it from the corners: shortest run, longest, hardest starting tier.
"""

import math
from typing import Any, ClassVar

from BaseClasses import ItemClassification

from .. import data
from ..rules import REQUIREMENTS
from . import TF2MvMTestBase

PLAYABLE_MISSIONS = tuple(mission for mission in data.MISSIONS if mission.playable)
PLAYABLE_MISSION_COUNT = len(PLAYABLE_MISSIONS)
MODDED_MISSIONS = tuple(
    mission for mission in data.MISSIONS if mission.requires in data.SERVER_MOD_KEYS
)


class TestDefaults(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {}

    def test_every_spare_reward_is_a_weapon_buff_by_default(self) -> None:
        buffs = [
            item.name for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES
        ]
        cash = [item for item in self.multiworld.itempool if item.name in data.FILLER_NAMES]
        self.assertGreater(len(buffs), 0)
        self.assertFalse(cash)

    def test_only_numeric_weapon_buffs_repeat(self) -> None:
        buffs = [
            item.name for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES
        ]
        repeated = {name for name in buffs if buffs.count(name) > 1}
        self.assertLessEqual(repeated, data.STACKABLE_WEAPON_BUFF_NAMES)


class TestNoWeaponBuffs(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "cash_rewards": True,
        "weapon_buff_percentage": 0,
    }

    def test_every_non_progression_reward_is_cash(self) -> None:
        self.assertFalse(
            any(item.name in data.WEAPON_BUFF_NAMES for item in self.multiworld.itempool)
        )
        self.assertTrue(any(item.name in data.FILLER_NAMES for item in self.multiworld.itempool))


class TestUniqueWeaponBuffs(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "weapon_buff_percentage": 100,
        "weapon_buff_stack_chance": 0,
    }

    def test_every_spare_reward_is_a_distinct_buff(self) -> None:
        buffs = [
            item.name for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES
        ]
        self.assertGreater(len(buffs), 0)
        self.assertEqual(len(buffs), len(set(buffs)))
        self.assertFalse(any(item.name in data.FILLER_NAMES for item in self.multiworld.itempool))


class TestCashRewards(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "cash_rewards": True,
        "weapon_buff_percentage": 75,
    }

    def test_cash_and_buffs_share_spare_checks_when_enabled(self) -> None:
        buffs = [item for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES]
        cash = [item for item in self.multiworld.itempool if item.name in data.FILLER_NAMES]
        self.assertGreater(len(buffs), 0)
        self.assertGreater(len(cash), 0)
        self.assertEqual(math.ceil((len(buffs) + len(cash)) * 0.75), len(buffs))


class TestUsefulUnlockModes(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_ticket_importance": "useful",
        "class_unlock_importance": "useful",
        "weapon_slot_importance": "useful",
    }

    def test_every_mission_is_reachable_without_unlocks(self) -> None:
        for mission in self.world.missions:
            self.assertTrue(self.can_reach_region(mission.name))

    def test_unlock_items_are_useful(self) -> None:
        kinds = {"mission_ticket", "class", "weapon_slot"}
        items = [
            item for item in self.multiworld.itempool if data.ITEMS_BY_NAME[item.name].kind in kinds
        ]
        self.assertTrue(items)
        self.assertTrue(all(item.classification == ItemClassification.useful for item in items))


class TestProgressionWeaponBuffs(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "weapon_buff_importance": "progression",
        "cash_rewards": True,
        "weapon_buff_percentage": 0,
    }

    def test_buffs_are_progression_items(self) -> None:
        buffs = [item for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES]
        self.assertGreaterEqual(len(buffs), 5)
        self.assertTrue(
            all(item.classification == ItemClassification.progression for item in buffs)
        )

    def test_non_start_mission_needs_buffs(self) -> None:
        mission = next(m for m in self.world.missions if m is not self.world.start_mission)
        self.assertFalse(self.can_reach_region(mission.name))


class TestStackedWeaponBuffs(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "weapon_buff_percentage": 100,
        "weapon_buff_stack_chance": 100,
    }

    def test_repeated_rewards_are_numeric_levels(self) -> None:
        buffs = [
            item.name for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES
        ]
        repeated = {name for name in buffs if buffs.count(name) > 1}
        self.assertTrue(repeated)
        self.assertLessEqual(repeated, data.STACKABLE_WEAPON_BUFF_NAMES)


class TestWholeRoster(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_count": PLAYABLE_MISSION_COUNT,
        "difficulty_pool": "normal",
    }


class TestShortestRun(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {"mission_count": 1, "difficulty_pool": "normal"}

    def test_unlocks_fit_the_checks(self) -> None:
        # One mission can leave fewer checks than unlocks owed; the draw widens rather than failing.
        checks = sum(len(mission.locations) for mission in self.world.missions)
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


class TestExcludedMissions(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_count": PLAYABLE_MISSION_COUNT,
        "difficulty_pool": "normal",
        "excluded_missions": {"Caliginous Caper", "Doe's Drill"},
    }

    def test_excluded_missions_are_never_drawn(self) -> None:
        drawn = {mission.name for mission in self.world.missions}
        self.assertNotIn("Caliginous Caper", drawn)
        self.assertNotIn("Doe's Drill", drawn)
        self.assertEqual(PLAYABLE_MISSION_COUNT - 2, len(drawn))


class TestStockServerDrawsNoModdedMission(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_count": len(data.MISSION_NAMES),
        "difficulty_pool": "normal",
    }

    def test_a_mission_that_needs_a_mod_stays_out(self) -> None:
        drawn = {mission.name for mission in self.world.missions}
        for mission in MODDED_MISSIONS:
            self.assertNotIn(mission.name, drawn)
        self.assertEqual(PLAYABLE_MISSION_COUNT, len(drawn))


class TestModdedServerDrawsItsMissions(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_count": len(data.MISSION_NAMES),
        "difficulty_pool": "normal",
        "server_mods": set(data.SERVER_MOD_KEYS),
    }

    def test_every_seedable_mission_is_drawn(self) -> None:
        drawn = {mission.name for mission in self.world.missions}
        self.assertEqual(data.MISSION_NAMES, drawn)
        self.assertEqual(sorted(data.SERVER_MOD_KEYS), self.world.fill_slot_data()["server_mods"])


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


class TestTankChecks(TF2MvMTestBase):
    """Mannhattan's three missions have no tank, so they must have no tank check.

    A location a mission can never satisfy is a run nobody can finish, which is
    why has_tank comes from the wiki's tank health table and not from a guess.
    """

    options: ClassVar[dict[str, Any]] = {
        "mission_count": PLAYABLE_MISSION_COUNT,
        "difficulty_pool": "normal",
    }

    def test_a_mission_with_a_tank_has_the_check(self) -> None:
        quarry = next(m for m in data.MISSIONS if m.name == "Quarry")
        self.assertTrue(quarry.has_tank)
        self.assertIn("Quarry Tank", self.multiworld.regions.location_cache[self.player])

    def test_a_mission_without_a_tank_has_no_check(self) -> None:
        for name in ("Big Apple Barricade", "Empire Escalation", "Metro Malice"):
            mission = next(m for m in data.MISSIONS if m.name == name)
            self.assertFalse(mission.has_tank)
            self.assertNotIn(f"{name} Tank", self.multiworld.regions.location_cache[self.player])

    def test_every_tank_check_belongs_to_a_mission_that_has_one(self) -> None:
        for mission in data.MISSIONS:
            tanks = [loc for loc in mission.locations if loc.kind == "tank_destroyed"]
            self.assertEqual(1 if mission.has_tank else 0, len(tanks))

    def test_every_mission_has_a_giant_check(self) -> None:
        # Every catalogued mission has a giant, and every playable one gets the check.
        for mission in data.MISSIONS:
            giants = [loc for loc in mission.locations if loc.kind == "giant_killed"]
            self.assertTrue(mission.has_giant)
            self.assertEqual(1, len(giants))
            if mission.playable:
                self.assertIn(
                    f"{mission.name} Giant",
                    self.multiworld.regions.location_cache[self.player],
                )


class TestNamedStartMission(TF2MvMTestBase):
    """Quarry is intermediate, so it is not the easiest mission a normal pool draws."""

    options: ClassVar[dict[str, Any]] = {
        "mission_count": 6,
        "difficulty_pool": "normal",
        "start_mission": "Quarry",
        "start_class": "Engineer",
        "goal": "missionsanity",
    }

    def test_the_run_starts_where_it_was_told_to(self) -> None:
        self.assertEqual("Quarry", self.world.start_mission.name)
        self.assertIn(self.world.start_mission, self.world.missions)
        self.assertIn(data.TICKET_NAMES[self.world.start_mission.id], self.world.start_items)

    def test_the_named_start_mission_is_open_from_nothing(self) -> None:
        self.assertTrue(self.can_reach_region("Quarry"))

    def test_the_run_starts_with_the_class_it_was_told_to(self) -> None:
        self.assertIn(data.CLASS_ITEM_BY_MERC["Engineer"], self.world.start_items)

    def test_the_starting_class_is_not_in_the_pool_twice(self) -> None:
        engineer = data.CLASS_ITEM_BY_MERC["Engineer"]
        self.assertEqual(0, sum(item.name == engineer for item in self.multiworld.itempool))


class TestNamedStartBeatsTheSort(TF2MvMTestBase):
    """The whole roster is drawn, so a normal mission is certainly in it.

    Quarry is intermediate. Without start_mission the run would begin on one of
    the four normal missions, which is what makes this prove the override.
    """

    options: ClassVar[dict[str, Any]] = {
        "mission_count": PLAYABLE_MISSION_COUNT,
        "difficulty_pool": "normal",
        "start_mission": "Quarry",
        "goal": "missionsanity",
    }

    def test_the_start_is_not_the_easiest_mission_drawn(self) -> None:
        easiest = min(
            self.world.missions, key=lambda mission: data.DIFFICULTIES.index(mission.difficulty)
        )
        self.assertEqual("normal", easiest.difficulty)
        self.assertEqual("Quarry", self.world.start_mission.name)
        self.assertTrue(self.can_reach_region("Quarry"))


class TestNamedStartCountsAsOneOfTheTier(TF2MvMTestBase):
    """An expert start mission asks for four classes, and start_class names one of them."""

    options: ClassVar[dict[str, Any]] = {
        "mission_count": 5,
        "difficulty_pool": "expert",
        "start_class": "Medic",
    }

    def test_the_named_class_does_not_add_one(self) -> None:
        requirement = REQUIREMENTS[self.world.start_mission.difficulty]
        held = self.world.start_items
        self.assertIn(data.CLASS_ITEM_BY_MERC["Medic"], held)
        self.assertEqual(requirement.classes, sum(name in data.CLASS_NAMES for name in held))


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
