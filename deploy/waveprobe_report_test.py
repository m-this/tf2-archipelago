import contextlib
import importlib.util
import io
import json
import pathlib
import tempfile
import unittest

module_path = pathlib.Path(__file__).with_name("waveprobe-report.py")
spec = importlib.util.spec_from_file_location("waveprobe_report", module_path)
report = importlib.util.module_from_spec(spec)
spec.loader.exec_module(report)


class ReportOutcomeTest(unittest.TestCase):
    def test_native_objective_witnesses_are_bound_to_mission_wave_and_victory(self):
        fixtures = [
            ("mvm_robotfactory_b30_rev_adv_short_circuit_execution", 1,
             {"method": "native-repair", "repairs": 4, "repair_completed": True, "blu_won": True}),
            ("mvm_teien_rc6_rev_adv_shogun_slaughter", 1,
             {"method": "native-route-objective", "point_captures": 4, "blu_won": True}),
            ("mvm_robotfactory_b30_rev_adv_short_circuit_execution", 1,
             {"method": "native-repair", "repairs": 5, "blu_won": True}),
            ("mvm_robotfactory_b30_rev_adv_short_circuit_execution", 2,
             {"method": "native-escort", "escort_stages": 3, "blu_won": True}),
            ("mvm_hideout_b3_rev_adv_galois", 4,
             {"tank_deliveries": 1, "blu_won": True}),
            ("mvm_hideout_b3_rev_adv_galois", 5,
             {"method": "native-combat-victory", "blu_won": True}),
        ]
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            for mission, wave, objective in fixtures:
                for target, evidence, expected in [
                    (mission, objective, 1),
                    (mission, {**objective, "blu_won": False}, 0),
                    ("mvm_coaltown_rev_adv_computer_crash", objective, 0),
                    (mission, {}, 0),
                ]:
                    with self.subTest(mission=mission, target=target, evidence=evidence):
                        case = {"mission": target, "wave": wave, "mode": "normal", "map": "fixture"}
                        (root / "plan.jsonl").write_text(json.dumps({**case, "state": "planned"}) + "\n")
                        row = {**case, "state": "passed", "outcome": "passed", "bots": 4,
                               "bot_spawns": 4, "objective": evidence}
                        (root / "shard-0.jsonl").write_text(json.dumps(row) + "\n")
                        with contextlib.redirect_stdout(io.StringIO()):
                            report.main(root)
                        summary = json.loads((root / "SUMMARY.json").read_text())
                        self.assertEqual(summary["eligible_cases"], 1)
                        self.assertEqual(summary["passed_cases"], expected)

    def test_capture_cannot_replace_a_different_native_objective_or_blu_victory(self):
        self.assertTrue(report.reverse_completion_witness(
            ("mvm_hideout_b3_rev_adv_galois", "surge", 4),
            {"objective": {"captures": 1, "blu_won": True}}))
        for mission, wave in [
            ("mvm_teien_rc6_rev_adv_shogun_slaughter", 1),
            ("mvm_robotfactory_b30_rev_adv_short_circuit_execution", 1),
            ("mvm_robotfactory_b30_rev_adv_short_circuit_execution", 2),
            ("mvm_coaltown_rev_adv_computer_crash", 6),
            ("mvm_autumnull_rc2_rev_exp_codename_omega", 6),
            ("mvm_hideout_b3_rev_adv_galois", 5),
        ]:
            with self.subTest(mission=mission):
                self.assertFalse(report.reverse_completion_witness(
                    (mission, "normal", wave),
                    {"objective": {"captures": 1, "blu_won": True}}))
        for victory in (False, None):
            self.assertFalse(report.reverse_completion_witness(
                ("mvm_rottenburg_rev_int_breaking_walls", "normal", 1),
                {"objective": {"captures": 1, "blu_won": victory}}))

    def test_arctic_tank_stage_requires_all_five_enemy_tank_deaths(self):
        key = ("mvm_cyberia_rc6a_rev_arctic_arrangement", "normal", 4)
        for tanks, victory, want in [(5, True, True), (4, True, False), (0, True, False), (5, False, False)]:
            self.assertEqual(report.reverse_completion_witness(key, {
                "bots": 8, "tanks": tanks,
                "objective": {"method": "native-combat-victory", "blu_won": victory}}), want)
        for wave in [1, 2, 3, 5]:
            self.assertTrue(report.reverse_completion_witness((key[0], "normal", wave), {
                "objective": {"method": "native-route-objective", "blu_won": True, "point_captures": 1}}))
            self.assertFalse(report.reverse_completion_witness((key[0], "normal", wave), {
                "objective": {"captures": 1, "blu_won": True}}))

    def test_supported_reverse_counts_in_score_but_missing_capture_evidence_does_not(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            ordinary = {"mission": "mvm_decoy", "map": "mvm_decoy", "mode": "normal", "wave": 1}
            reverse = {**ordinary, "mission": "mvm_rottenburg_rev_int_breaking_walls"}
            unsupported = {**ordinary, "mission": "mvm_teien_rc6_rev_adv_shogun_slaughter"}
            (root/"plan.jsonl").write_text("".join(json.dumps({**row, "state": state}) + "\n"
                for row, state in [(ordinary, "planned"), (reverse, "planned"), (unsupported, "unsupported_reverse")]))
            for captures in [1, 0]:
                rows = [{**ordinary, "state": "passed", "outcome": "passed"},
                        {**reverse, "state": "passed", "outcome": "passed", "objective": {"captures": captures, "blu_won": True}}]
                (root/"shard-0.jsonl").write_text("".join(json.dumps(row) + "\n" for row in rows))
                with contextlib.redirect_stdout(io.StringIO()):
                    report.main(root)
                summary = json.loads((root/"SUMMARY.json").read_text())
                self.assertEqual(summary["eligible_cases"], 2)
                self.assertEqual(summary["passed_cases"], 1 + captures)
                self.assertEqual(summary["reverse_cases"], {"eligible": 1, "tested": 1, "passed": captures})
                self.assertEqual(summary["outcomes"]["reverse objective"], 1)
                self.assertEqual(summary["outcomes"]["probe error"], 1 - captures)

    def test_combat_exception_needs_the_authored_wave_and_target_witness(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            for wave, kills, expected in [(6, 1, 1), (6, 0, 0), (5, 1, 0)]:
                case = {"mission": "mvm_coaltown_rev_adv_computer_crash", "map": "mvm_coaltown", "mode": "normal", "wave": wave}
                (root/"plan.jsonl").write_text(json.dumps({**case, "state": "planned"}) + "\n")
                (root/"shard-0.jsonl").write_text(json.dumps({**case, "state": "passed", "outcome": "passed", "bots": 7,
                    "objective": {"method": "native-combat-clear", "target_kills": kills}}) + "\n")
                with contextlib.redirect_stdout(io.StringIO()): report.main(root)
                summary = json.loads((root/"SUMMARY.json").read_text())
                self.assertEqual(summary["passed_cases"], expected)

    def test_changelevel_failure_is_counted_and_shows_map_evidence(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            case = {"mission": "mission", "map": "mvm_deathpour_rc1", "mode": "normal", "wave": 5}
            (root / "plan.jsonl").write_text(json.dumps({**case, "state": "planned"}) + "\n")
            (root / "shard-0.jsonl").write_text(json.dumps({**case, "state": "failed",
                "outcome": "wave timed out", "bot_spawns": 82}) + "\n")
            evidence = {"requested_map": "mvm_deathpour_rc1", "before_map": "mvm_decoy",
                        "before_pop": "mvm_decoy_advanced", "after_map": "mvm_decoy",
                        "after_pop": "mvm_decoy_advanced", "after_state": "idle",
                        "command_reply": "rejected"}
            (root / "retest-0.jsonl").write_text(json.dumps({**case, "state": "load_failed",
                "outcome": "changelevel failure", "error": "map stayed on Decoy",
                "changelevel": evidence}) + "\n")
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                report.main(root)
            summary = json.loads((root / "SUMMARY.json").read_text())
            self.assertEqual(summary["outcomes"]["changelevel failure"], 1)
            self.assertIn("Changelevel evidence: requested `mvm_deathpour_rc1`", output.getvalue())
            self.assertIn("Reply: `rejected`", output.getvalue())

    def test_old_zero_kill_rows_do_not_claim_no_spawns(self):
        self.assertEqual(report.classify({"state": "failed", "outcome": "no enemies observed"}),
                         "inconclusive")

    def test_missing_retest_wave_inherits_load_failure(self):
        row = {"state": "inconclusive", "outcome": "inconclusive",
               "retest_no_wave_result": True}
        prior = {"state": "load_failed", "outcome": "wave 0"}
        self.assertEqual(report.classify(row, prior), "wave 0")

    def test_reset_cannot_erase_observed_spawns(self):
        row = {"state": "failed", "outcome": "no enemies spawned",
               "bot_spawns": 0, "tank_spawns": 0,
               "timeline": [{"spawned": 3}, {"spawned": 0}]}
        self.assertEqual(report.classify(row), "probe error")

    def test_resumed_screen_pass_counts_as_verified(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            case = {"mission": "mission", "map": "mvm_map", "mode": "normal", "wave": 1}
            (root / "plan.jsonl").write_text(json.dumps({**case, "state": "planned"}) + "\n")
            (root / "shard-0.jsonl").write_text(json.dumps({**case, "state": "failed", "outcome": "probe error"}) + "\n")
            (root / "screen-0.jsonl").write_text(json.dumps({**case, "state": "passed", "outcome": "passed",
                                                               "bot_spawns": 1}) + "\n")
            with contextlib.redirect_stdout(io.StringIO()):
                report.main(root)
            summary = json.loads((root / "SUMMARY.json").read_text())
            self.assertEqual(summary["passed_cases"], 1)

    def test_report_scores_modes_and_shows_timeline(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)

            def write_rows(name, rows):
                (root / name).write_text("".join(json.dumps(row) + "\n" for row in rows))

            plan = []
            for mode in ("normal", "surge"):
                for wave in (1, 2, 3):
                    plan.append({"mission": "mission", "map": "mvm_map", "mode": mode,
                                 "wave": wave, "state": "planned"})
            write_rows("plan.jsonl", plan)
            write_rows("shard-0.jsonl", [
                {"mission": "mission", "map": "mvm_map", "mode": "normal", "wave": 1,
                 "state": "passed", "outcome": "passed", "bot_spawns": 4, "bots": 4},
                {"mission": "mission", "map": "mvm_map", "mode": "surge", "wave": 1,
                 "state": "failed", "outcome": "wave failed", "bot_spawns": 4,
                 "bots": 3, "alive_at_end": 1, "progress_percent": 75,
                 "timeline": [{"wall_seconds": 10, "game_seconds": 200,
                               "spawned": 4, "killed": 3, "alive": 1,
                               "remaining": 1, "progress_percent": 75}]},
                {"mission": "mission", "map": "mvm_map", "mode": "normal", "wave": 2,
                 "state": "load_failed", "outcome": "wave 0"},
                {"mission": "mission", "map": "mvm_map", "mode": "surge", "wave": 2,
                 "state": "failed", "outcome": "no enemies spawned"},
                {"mission": "mission", "map": "mvm_map", "mode": "normal", "wave": 3,
                 "state": "failed", "outcome": "wave timed out", "bot_spawns": 3,
                 "bots": 2, "alive_at_end": 1, "progress_percent": 66,
                 "wall_seconds": 900, "game_seconds": 18000},
            ])
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                report.main(root)
            summary = json.loads((root / "SUMMARY.json").read_text())
            self.assertEqual(summary["eligible_cases"], 6)
            self.assertEqual(summary["tested_cases"], 5)
            self.assertEqual(summary["passed_cases"], 1)
            self.assertEqual(summary["stability_percent"], 16.67)
            self.assertEqual(summary["by_mode"]["normal"]["tested"], 3)
            self.assertEqual(summary["by_mode"]["surge"]["tested"], 2)
            text = output.getvalue()
            self.assertIn("Stability score: **16.7%**", text)
            self.assertIn("| Wave Failed | 0 | 1 |", text)
            self.assertIn("| Wave 0 | 1 | 0 |", text)
            self.assertIn("| No Enemies Spawned | 0 | 1 |", text)
            self.assertIn("| 10 | 200 | 4 | 3 | 1 | 1 | 75% |", text)


if __name__ == "__main__":
    unittest.main()
