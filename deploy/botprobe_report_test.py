import contextlib
import importlib.util
import io
import json
import pathlib
import tempfile
import unittest

module_path = pathlib.Path(__file__).with_name("botprobe-report.py")
spec = importlib.util.spec_from_file_location("botprobe_report", module_path)
report = importlib.util.module_from_spec(spec)
spec.loader.exec_module(report)


def bot(**fields):
    return {"name": "Kaboom!", "class": 6, "seen_seconds": 120.0, "left_spawn_seconds": 6.0,
            "lives": 1, "left_spawn_max_seconds": 6.0, "in_spawn_this_life_seconds": 0,
            "still_max_seconds": 4.0, "still_max_in_spawn": False, "still_max_at": [0, 0, 0],
            "idle_max_seconds": 3.0, "idle_max_at": [0, 0, 0], "teleports": 0, **fields}


class GateTest(unittest.TestCase):
    def run_gate(self, planned, recorded):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            (root / "plan.jsonl").write_text("".join(json.dumps(row) + "\n" for row in planned))
            for name, rows in recorded.items():
                (root / name).write_text("".join(json.dumps(row) + "\n" for row in rows))
            with contextlib.redirect_stdout(io.StringIO()):
                passed = report.main(root)
            return passed, report.gate(root, report.rows(root))

    def case(self, mode="normal", wave=1, **fields):
        return {"mission": "mvm_decoy_advanced", "map": "mvm_decoy", "mode": mode, "wave": wave, **fields}

    def test_every_wave_recorded_and_no_fault_passes(self):
        plan = [self.case(state="planned"), self.case("surge", state="planned"),
                self.case("normal", 1, state="unsupported_reverse") | {"mission": "mvm_rev"}]
        passed, reasons = self.run_gate(plan, {
            "defenders-0.jsonl": [self.case(outcome="wave failed", defenders=[bot()])],
            "defenders-1.jsonl": [self.case("surge", outcome="passed", defenders=[bot(**{"class": 9})])],
        })
        self.assertTrue(passed, reasons)

    def test_a_bot_that_never_left_spawn_fails(self):
        passed, reasons = self.run_gate([self.case(state="planned")], {
            "defenders.jsonl": [self.case(defenders=[bot(left_spawn_seconds=-1, in_spawn_this_life_seconds=120)])]})
        self.assertFalse(passed)
        self.assertIn("never left spawn in 120s", reasons[0])

    def test_a_bot_seen_too_briefly_to_leave_is_not_a_fault(self):
        passed, reasons = self.run_gate([self.case(state="planned")], {
            "defenders.jsonl": [self.case(defenders=[bot(left_spawn_seconds=-1, seen_seconds=8)])]})
        self.assertTrue(passed, reasons)

    def test_a_respawn_stuck_in_spawn_fails(self):
        passed, reasons = self.run_gate([self.case(state="planned")], {
            "defenders.jsonl": [self.case(defenders=[bot(lives=2, in_spawn_this_life_seconds=45)])]})
        self.assertFalse(passed)
        self.assertIn("in spawn for 45s at the end", reasons[0])

    def test_idle_fails_except_for_a_parked_class(self):
        idle = {"idle_max_seconds": 31.0}
        passed, reasons = self.run_gate([self.case(state="planned")], {
            "defenders.jsonl": [self.case(defenders=[bot(**idle), bot(**idle, **{"class": 2})])]})
        self.assertFalse(passed)
        self.assertEqual(len(reasons), 1)
        self.assertIn("(heavy) idle 31s", reasons[0])

    def test_a_record_without_idle_cannot_pass(self):
        old = bot()
        del old["idle_max_seconds"]
        passed, reasons = self.run_gate([self.case(state="planned")], {
            "defenders.jsonl": [self.case(defenders=[old])]})
        self.assertFalse(passed)
        self.assertIn("no idle record", reasons[0])

    def test_a_wave_not_played_or_without_record_fails(self):
        passed, reasons = self.run_gate(
            [self.case(state="planned"), self.case(wave=2, state="planned")],
            {"defenders.jsonl": [self.case(outcome="probe error", error="rcon closed")]})
        self.assertFalse(passed)
        self.assertEqual(reasons, ["mvm_decoy_advanced normal wave 1: no defender record (probe error rcon closed)",
                                   "mvm_decoy_advanced normal wave 2: not played"])

    def test_no_plan_fails(self):
        with tempfile.TemporaryDirectory() as folder:
            self.assertEqual(len(report.gate(folder, [])), 1)


if __name__ == "__main__":
    unittest.main()
