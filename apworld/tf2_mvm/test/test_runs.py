"""Each class here is one YAML the world has to survive.

WorldTestBase brings three tests of its own to every subclass, one of them being
the sphere 0 guarantee: empty state reaches at least one location, because the
easiest mission drawn is the starting one and exactly what its tier requires is
precollected. That is the rule this world cannot get wrong, so the option sets
below attack it from the corners: shortest run, longest, hardest starting tier.
"""

import math
from typing import Any, ClassVar

from BaseClasses import CollectionState, ItemClassification

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

    def test_bot_cards_are_off_by_default(self) -> None:
        self.assertFalse([name for name in self.world.start_items if name.startswith("Bot: ")])
        self.assertFalse(
            [item for item in self.multiworld.itempool if item.name.startswith("Bot: ")]
        )

    def test_only_numeric_weapon_buffs_repeat(self) -> None:
        buffs = [
            item.name for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES
        ]
        repeated = {name for name in buffs if buffs.count(name) > 1}
        self.assertLessEqual(repeated, data.STACKABLE_WEAPON_BUFF_NAMES)

    def test_mission_modifiers_are_off_by_default(self) -> None:
        self.assertEqual({}, self.world.mission_modifiers)
        self.assertEqual({}, self.world.fill_slot_data()["mission_modifiers"])

    def test_victory_caches_stay_out_by_default(self) -> None:
        self.assertFalse(self.world.fill_slot_data()["victory_caches"])
        for mission in self.world.missions:
            for location in mission.locations:
                if location.cache:
                    with self.assertRaises(KeyError, msg=location.name):
                        self.world.get_location(location.name)

    def test_milestones_stay_out_by_default(self) -> None:
        self.assertFalse(self.world.fill_slot_data()["milestone_checks"])
        for milestone in data.MILESTONES:
            with self.assertRaises(KeyError, msg=milestone.name):
                self.world.get_location(milestone.name)

    def test_slots_are_one_item_for_every_class_by_default(self) -> None:
        self.assertFalse(self.world.fill_slot_data()["class_weapon_slots"])
        names = {item.name for item in self.multiworld.itempool} | set(self.world.start_items)
        self.assertIn(data.PROGRESSIVE_WEAPON_SLOT, names)
        self.assertFalse(names & set(data.CLASS_SLOT_ITEMS.values()))

    def test_per_kill_checks_stay_out_by_default(self) -> None:
        slot_data = self.world.fill_slot_data()
        self.assertFalse(slot_data["giantsanity"] or slot_data["tanksanity"])
        for mission in self.world.missions:
            for location in mission.locations:
                if location.index:
                    with self.assertRaises(KeyError, msg=location.name):
                        self.world.get_location(location.name)


class TestRandomBotCards(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "bot_cards": True,
        "mission_count": 20,
        "starting_bot_cards": 3,
    }

    def test_every_remaining_card_can_be_a_useful_reward(self) -> None:
        held = [
            data.ITEMS_BY_NAME[name] for name in self.world.start_items if name.startswith("Bot: ")
        ]
        rewards = [item for item in self.multiworld.itempool if item.name.startswith("Bot: ")]
        self.assertEqual(3, len(held))
        self.assertEqual(len(data.BOT_CARD_BASES) - len(held), len(rewards))
        self.assertEqual(
            len(held) + len(rewards),
            len(
                {data.ITEMS_BY_NAME[item.name].bot_name for item in rewards}
                | {item.bot_name for item in held}
            ),
        )
        self.assertTrue(all(item.classification == ItemClassification.useful for item in rewards))
        self.assertTrue(all(item.bot_tier in {"Common", "Elite", "Legendary"} for item in held))
        self.assertTrue(all(item.bot_form in {"Human", "Robot", "Giant"} for item in held))


class TestStockStartingBotCards(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "bot_cards": True,
        "mission_count": 20,
        "starting_bot_cards": 2,
        "starting_bot_card_mode": "stock_classes",
    }

    def test_one_stock_card_per_class_ignores_random_count(self) -> None:
        held = [
            data.ITEMS_BY_NAME[name] for name in self.world.start_items if name.startswith("Bot: ")
        ]
        self.assertEqual(9, len(held))
        self.assertTrue(all(item.bot_stock for item in held))
        rewards = [item for item in self.multiworld.itempool if item.name.startswith("Bot: ")]
        self.assertEqual(len(data.BOT_CARD_BASES) - len(held), len(rewards))


class TestMissionModifiers(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_modifiers": True,
        "minimum_mission_modifiers": 3,
        "maximum_mission_modifiers": 3,
    }

    def test_every_mission_gets_the_requested_persistent_assignment(self) -> None:
        assignments = self.world.mission_modifiers
        self.assertEqual({mission.pop_file for mission in self.world.missions}, set(assignments))
        self.assertTrue(all(len(modifiers) == 3 for modifiers in assignments.values()))
        self.assertEqual(assignments, self.world.fill_slot_data()["mission_modifiers"])

    def test_gravity_variants_never_stack(self) -> None:
        gravity = {"low_gravity", "high_gravity"}
        for modifiers in self.world.mission_modifiers.values():
            keys = {modifier["key"] for modifier in modifiers}
            self.assertLessEqual(len(keys & gravity), 1)


class TestMissionModifierRangeStartsAtZero(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_modifiers": True,
        "minimum_mission_modifiers": 0,
        "maximum_mission_modifiers": 1,
    }

    def test_no_mission_exceeds_the_maximum(self) -> None:
        for pop_file, modifiers in self.world.mission_modifiers.items():
            self.assertLessEqual(len(modifiers), 1, pop_file)


class TestMissionModifiersDrawnAtZero(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_modifiers": True,
        "minimum_mission_modifiers": 0,
        "maximum_mission_modifiers": 0,
    }

    def test_a_draw_of_zero_assigns_nothing(self) -> None:
        # A draw of zero used to match nothing, and the mission got every
        # modifier the exclusive groups allowed (gh-85).
        assignments = self.world.mission_modifiers
        self.assertEqual({mission.pop_file for mission in self.world.missions}, set(assignments))
        self.assertTrue(all(modifiers == [] for modifiers in assignments.values()))


CACHES_BY_TIER = {"normal": 0, "intermediate": 1, "advanced": 2, "expert": 4, "haunted": 4}


class TestVictoryCaches(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "victory_caches": True,
        "difficulty_pool": "intermediate",
        "mission_count": 6,
    }

    def test_a_clear_pays_its_tier(self) -> None:
        self.assertTrue(self.world.fill_slot_data()["victory_caches"])
        for mission in self.world.missions:
            caches = [location for location in mission.locations if location.cache]
            self.assertEqual(CACHES_BY_TIER[mission.difficulty], len(caches), mission.name)
            for location in caches:
                self.assertEqual(location.id, self.world.get_location(location.name).address)


class TestMilestoneChecks(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {"milestone_checks": True, "mission_count": 8}

    def test_every_milestone_is_a_check(self) -> None:
        self.assertTrue(self.world.fill_slot_data()["milestone_checks"])
        self.assertEqual(15, len(data.MILESTONES))
        for milestone in data.MILESTONES:
            self.assertEqual(milestone.id, self.world.get_location(milestone.name).address)

    def test_the_first_tally_of_each_kind_is_open_from_the_start(self) -> None:
        for kind in {milestone.kind for milestone in data.MILESTONES}:
            first = min(
                (one for one in data.MILESTONES if one.kind == kind),
                key=lambda one: one.threshold,
            )
            self.assertTrue(self.can_reach_location(first.name), first.name)

    def test_the_longest_tally_of_each_kind_waits_for_the_run(self) -> None:
        """The bug this covers cost somebody a multiworld.

        Every tally used to sit in sphere 0, so a progression item could be
        placed behind "40 Tanks Destroyed" on the first move, and the only way
        forward was replaying the one reachable mission with tanks until forty
        tanks had died. Reachable is not the same as reasonable: the long ones
        have to wait for most of the run.
        """
        for kind in {milestone.kind for milestone in data.MILESTONES}:
            last = max(
                (one for one in data.MILESTONES if one.kind == kind),
                key=lambda one: one.threshold,
            )
            self.assertFalse(self.can_reach_location(last.name), last.name)

    def test_every_milestone_is_reachable_with_the_whole_pool(self) -> None:
        """Deeper, and still not out of reach: a gate nothing opens is a lock."""
        for item in self.multiworld.get_items():
            self.collect(item)
        for milestone in data.MILESTONES:
            self.assertTrue(self.can_reach_location(milestone.name), milestone.name)


class TestClassWeaponSlots(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "class_weapon_slots": True,
        "difficulty_pool": "advanced",
        "mission_count": 6,
    }

    def test_each_class_earns_its_own_slots(self) -> None:
        self.assertTrue(self.world.fill_slot_data()["class_weapon_slots"])
        self.assertEqual(2, data.CLASS_SLOT_COUNT)
        # Advanced asks for three classes with two slots: three classes, each
        # holding one copy of its own item on top of the free first slot.
        held = [name for name in self.world.start_items if name in data.CLASS_SLOT_ITEMS.values()]
        self.assertEqual(3, len(held))
        self.assertNotIn(data.PROGRESSIVE_WEAPON_SLOT, self.world.start_items)
        pool = [
            item.name
            for item in self.multiworld.itempool
            if item.name in data.CLASS_SLOT_ITEMS.values()
        ]
        self.assertEqual(len(data.CLASS_SLOT_ITEMS) * data.CLASS_SLOT_COUNT - 3, len(pool))
        self.assertFalse(
            [item for item in self.multiworld.itempool if item.name == data.PROGRESSIVE_WEAPON_SLOT]
        )
        self.assertTrue(self.can_reach_region(self.world.start_mission.name))

    def test_a_class_without_its_slots_does_not_deploy(self) -> None:
        # Take every slot item away: the start mission's three classes fall
        # back to one slot each, short of what advanced asks for.
        self.collect_all_but(list(data.CLASS_SLOT_ITEMS.values()))
        for name in data.CLASS_SLOT_ITEMS.values():
            self.remove(self.get_item_by_name(name))
        self.assertFalse(self.can_reach_region(self.world.goal_mission.name))
        self.collect_by_name(list(data.CLASS_SLOT_ITEMS.values()))
        self.assertTrue(self.can_reach_region(self.world.goal_mission.name))


class TestClassWeaponSlotsInAnyOrder(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "class_weapon_slots": "any_order",
        "difficulty_pool": "advanced",
        "mission_count": 6,
    }

    def test_every_slot_is_its_own_item(self) -> None:
        """The pool holds each class's slots by name, one copy each, and no progressive one."""
        slot_data = self.world.fill_slot_data()
        self.assertTrue(slot_data["class_weapon_slots"])
        self.assertTrue(slot_data["class_weapon_slots_any_order"])

        named = {name for names in data.CLASS_NAMED_SLOT_ITEMS.values() for name in names}
        pool = [item.name for item in self.multiworld.itempool if item.name in named]
        self.assertEqual(sorted(pool), sorted(set(pool)), "a named slot is in the pool twice")

        held = [name for name in self.world.start_items if name in named]
        self.assertEqual(3, len(held), "advanced starts three classes with one earned slot each")
        self.assertEqual(len(named) - len(held), len(pool))

        progressive = set(data.CLASS_SLOT_ITEMS.values()) | {data.PROGRESSIVE_WEAPON_SLOT}
        self.assertFalse([item for item in self.multiworld.itempool if item.name in progressive])
        self.assertFalse([name for name in self.world.start_items if name in progressive])

    def test_a_starting_class_is_given_the_slot_it_is_played_with(self) -> None:
        """The free first slot is the class's own, so nobody starts on their worst weapon."""
        named = {name for names in data.CLASS_NAMED_SLOT_ITEMS.values() for name in names}
        for name in self.world.start_items:
            if name in named:
                owner = next(
                    class_item
                    for class_item, names in data.CLASS_NAMED_SLOT_ITEMS.items()
                    if name in names
                )
                self.assertIn(owner, self.world.start_items)
                self.assertEqual(
                    data.CLASS_NAMED_SLOT_ITEMS[owner][0],
                    name,
                    "a starting class was given a slot out of its own order",
                )

    def test_a_class_can_earn_either_of_its_slots(self) -> None:
        """No slot is the required one: the rule counts how many, not which."""
        mission = self.world.start_mission
        requirement = REQUIREMENTS[mission.difficulty]
        rule = self.world._deploy_rule(mission)

        # The state starts from the run's own precollected inventory, so the
        # claim to make here is that the two slots of a class are worth the
        # same, not that either is worth nothing on its own.
        for index in range(data.CLASS_SLOT_COUNT):
            state = CollectionState(self.multiworld)
            for class_item, names in data.CLASS_NAMED_SLOT_ITEMS.items():
                state.collect(self.world.create_item(class_item), prevent_sweep=True)
                state.collect(self.world.create_item(names[index]), prevent_sweep=True)
            state.collect(self.world.create_item(data.TICKET_NAMES[mission.id]), prevent_sweep=True)
            self.assertTrue(
                rule(state),
                f"a team holding slot {index} of every class could not deploy, "
                f"though {requirement.classes} classes held what the tier asks",
            )


class TestGiantsanityAndTanksanity(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "giantsanity": True,
        "tanksanity": True,
        "excluded_missions": [],
        "community_missions": False,
        "mission_count": 6,
    }

    def test_every_giant_and_tank_of_every_wave_is_a_check(self) -> None:
        slot_data = self.world.fill_slot_data()
        self.assertTrue(slot_data["giantsanity"] and slot_data["tanksanity"])
        kills = 0
        for mission in self.world.missions:
            for location in mission.locations:
                if location.index:
                    kills += 1
                    self.assertEqual(location.id, self.world.get_location(location.name).address)
        # Every Valve mission holds a giant somewhere, and the seed drew six.
        self.assertGreater(kills, 6)


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


class TestOneTrapInAHundredByDefault(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {}

    def test_a_run_that_did_not_ask_still_meets_one(self) -> None:
        traps = [item for item in self.multiworld.itempool if item.name in data.TRAP_NAMES]
        buffs = [item for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES]
        self.assertEqual(math.ceil((len(traps) + len(buffs)) / 100), len(traps))
        self.assertGreater(len(traps), 0)


class TestNoTrapsAtZero(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {"trap_percentage": 0}

    def test_zero_leaves_them_out(self) -> None:
        self.assertFalse(any(item.name in data.TRAP_NAMES for item in self.multiworld.itempool))


class TestTraps(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "trap_percentage": 50,
    }

    def test_traps_take_half_the_spare_checks(self) -> None:
        traps = [item for item in self.multiworld.itempool if item.name in data.TRAP_NAMES]
        buffs = [item for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES]
        self.assertGreater(len(traps), 0)
        self.assertEqual(math.ceil((len(traps) + len(buffs)) / 2), len(traps))

    def test_traps_are_classified_as_traps(self) -> None:
        traps = [item for item in self.multiworld.itempool if item.name in data.TRAP_NAMES]
        self.assertTrue(all(item.classification == ItemClassification.trap for item in traps))


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


class TestDisabledRewards(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_count": 1,
        "difficulty_pool": "normal",
        "mission_ticket_importance": "disabled",
        "class_unlock_importance": "disabled",
        "weapon_slot_importance": "disabled",
        "weapon_buff_importance": "disabled",
    }

    def test_all_unlocks_are_precollected_and_absent_from_the_pool(self) -> None:
        held = self.world.start_items
        self.assertEqual(
            {data.TICKET_NAMES[mission.id] for mission in self.world.missions},
            {name for name in held if name in data.TICKET_NAMES.values()},
        )
        self.assertEqual(set(data.CLASS_NAMES), set(held) & set(data.CLASS_NAMES))
        self.assertEqual(data.WEAPON_SLOT_COUNT, held.count(data.PROGRESSIVE_WEAPON_SLOT))
        unlock_kinds = {"mission_ticket", "class", "weapon_slot", "class_weapon_slot"}
        self.assertFalse(
            [
                item
                for item in self.multiworld.itempool
                if data.ITEMS_BY_NAME[item.name].kind in unlock_kinds
            ]
        )
        self.assertEqual(
            1, len(self.world.missions), "disabled unlocks should not widen a short run"
        )

    def test_no_buffs_are_awarded_even_with_cash_rewards_off(self) -> None:
        self.assertFalse(
            [item for item in self.multiworld.itempool if item.name in data.WEAPON_BUFF_NAMES]
        )
        self.assertTrue(any(item.name in data.FILLER_NAMES for item in self.multiworld.itempool))
        self.assertTrue(all(self.can_reach_region(mission.name) for mission in self.world.missions))


class TestDisabledProgressiveClassSlots(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "class_weapon_slots": "progressive",
        "mission_ticket_importance": "disabled",
        "class_unlock_importance": "disabled",
        "weapon_slot_importance": "disabled",
    }

    def test_every_class_and_its_slots_start_unlocked(self) -> None:
        self.assertLessEqual(set(data.CLASS_NAMES), set(self.world.start_items))
        for name in data.CLASS_SLOT_ITEMS.values():
            self.assertEqual(data.CLASS_SLOT_COUNT, self.world.start_items.count(name))
            self.assertFalse(any(item.name == name for item in self.multiworld.itempool))
        self.assertTrue(all(self.can_reach_region(mission.name) for mission in self.world.missions))


class TestDisabledNamedClassSlots(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "class_weapon_slots": "any_order",
        "mission_ticket_importance": "disabled",
        "class_unlock_importance": "disabled",
        "weapon_slot_importance": "disabled",
    }

    def test_every_named_slot_starts_unlocked(self) -> None:
        names = {name for slots in data.CLASS_NAMED_SLOT_ITEMS.values() for name in slots}
        self.assertLessEqual(names, set(self.world.start_items))
        self.assertFalse(any(item.name in names for item in self.multiworld.itempool))
        self.assertTrue(all(self.can_reach_region(mission.name) for mission in self.world.missions))


class TestDisabledClassesWithProgressionSlots(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "class_weapon_slots": "progressive",
        "class_unlock_importance": "disabled",
        "weapon_slot_importance": "progression",
        "difficulty_pool": "advanced",
    }

    def test_each_class_gets_only_the_starting_tier_slots(self) -> None:
        starting = REQUIREMENTS[self.world.start_mission.difficulty].slots - 1
        for name in data.CLASS_SLOT_ITEMS.values():
            self.assertEqual(starting, self.world.start_items.count(name))
            self.assertEqual(
                data.CLASS_SLOT_COUNT - starting,
                sum(item.name == name for item in self.multiworld.itempool),
            )


class TestDisabledSlotsWithProgressionClasses(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "class_weapon_slots": "any_order",
        "class_unlock_importance": "progression",
        "weapon_slot_importance": "disabled",
    }

    def test_all_slots_are_open_but_classes_remain_rewards(self) -> None:
        names = {name for slots in data.CLASS_NAMED_SLOT_ITEMS.values() for name in slots}
        self.assertLessEqual(names, set(self.world.start_items))
        self.assertFalse(any(item.name in names for item in self.multiworld.itempool))
        self.assertTrue(any(item.name in data.CLASS_NAMES for item in self.multiworld.itempool))


class TestDisabledBuffsWithCashEnabled(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "weapon_buff_importance": "disabled",
        "cash_rewards": True,
        "weapon_buff_percentage": 100,
    }

    def test_buff_percentage_cannot_restore_disabled_buffs(self) -> None:
        self.assertFalse(
            any(item.name in data.WEAPON_BUFF_NAMES for item in self.multiworld.itempool)
        )
        self.assertTrue(any(item.name in data.FILLER_NAMES for item in self.multiworld.itempool))


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
        checks = sum(len(self.world._checks(mission)) for mission in self.world.missions)
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

    def test_tracker_gets_the_final_precollected_inventory(self) -> None:
        expected = [item.name for item in self.multiworld.precollected_items[self.world.player]]
        tracker = self.world.fill_slot_data()["tracker"]
        self.assertEqual(1, tracker["version"])
        self.assertEqual(expected, tracker["starting_items"])
        # The list, rather than a set, preserves progressive copies.
        requirement = REQUIREMENTS[self.world.start_mission.difficulty]
        self.assertEqual(
            requirement.slots,
            tracker["starting_items"].count(data.PROGRESSIVE_WEAPON_SLOT),
        )


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


class TestCommunityMissionsCanBeKeptOut(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "mission_count": len(data.MISSION_NAMES),
        "difficulty_pool": "normal",
        "community_missions": False,
    }

    def test_only_valve_missions_are_drawn(self) -> None:
        for mission in self.world.missions:
            self.assertFalse(mission.community, mission.name)
        valve = [m for m in PLAYABLE_MISSIONS if not m.community]
        self.assertEqual(len(valve), len(self.world.missions))


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
            tanks = [
                loc for loc in mission.locations if loc.kind == "tank_destroyed" and not loc.index
            ]
            self.assertEqual(1 if mission.has_tank else 0, len(tanks))

    def test_every_mission_has_a_giant_check(self) -> None:
        # Every catalogued mission has a giant, and every playable one gets the check.
        for mission in data.MISSIONS:
            giants = [
                loc for loc in mission.locations if loc.kind == "giant_killed" and not loc.index
            ]
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


class TestNoMedalsByDefault(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {}

    def test_a_run_that_did_not_ask_gets_none(self) -> None:
        medals = set(data.MEDAL_NAMES.values())
        self.assertFalse(any(item.name in medals for item in self.multiworld.itempool))
        for mission in self.world.missions:
            location = self.multiworld.get_location(f"{mission.name} Complete", self.player)
            self.assertIsNone(location.item)


class TestMedalOnClear(TF2MvMTestBase):
    """One medal per mission, on that mission's own clear and nowhere else."""

    options: ClassVar[dict[str, Any]] = {
        "medal_on_clear": 1,
        "mission_count": 6,
    }

    def test_every_clear_holds_its_own_medal(self) -> None:
        for mission in self.world.missions:
            location = self.multiworld.get_location(f"{mission.name} Complete", self.player)
            self.assertIsNotNone(location.item)
            self.assertEqual(data.MEDAL_NAMES[mission.id], location.item.name)
            self.assertTrue(location.item.advancement)

    def test_no_medal_reaches_the_multiworld(self) -> None:
        medals = set(data.MEDAL_NAMES.values())
        self.assertFalse(any(item.name in medals for item in self.multiworld.itempool))

    def test_the_pool_shrinks_by_the_locked_clears(self) -> None:
        # Every location the pool may fill holds exactly one item, so a pool
        # that ignored the locked clears would overfill or come up short.
        free = self.world._check_count(self.world.missions) - len(self.world.missions)
        self.assertEqual(free, len(self.multiworld.itempool))


class TestMedalGoalFinalBoss(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "medal_on_clear": 1,
        "goal": "final_boss",
        "mission_count": 6,
    }

    def test_the_goal_reads_the_goal_mission_medal(self) -> None:
        state = self.multiworld.get_all_state(False)
        self.assertTrue(self.multiworld.completion_condition[self.player](state))

        medal = data.MEDAL_NAMES[self.world.goal_mission.id]
        state.remove(self.world.create_item(medal))
        self.assertFalse(self.multiworld.completion_condition[self.player](state))


class TestMedalGoalMissionsanity(TF2MvMTestBase):
    options: ClassVar[dict[str, Any]] = {
        "medal_on_clear": 1,
        "goal": "missionsanity",
        "missionsanity_percentage": 100,
        "mission_count": 6,
    }

    def test_the_goal_counts_the_medals(self) -> None:
        state = self.multiworld.get_all_state(False)
        self.assertTrue(self.multiworld.completion_condition[self.player](state))

        # One short of every mission is one short of the goal.
        medal = data.MEDAL_NAMES[self.world.missions[0].id]
        state.remove(self.world.create_item(medal))
        self.assertFalse(self.multiworld.completion_condition[self.player](state))
