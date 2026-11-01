"""The known-failure list must tolerate only exact, unexpired matches and never a crash."""
import datetime as dt
import importlib.util
import json
import tempfile
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("stability_report", Path(__file__).with_name("stability-report.py"))
report = importlib.util.module_from_spec(spec)
spec.loader.exec_module(report)

TODAY = dt.date(2026, 11, 1)
ENTRY = {"mission": "mvm_a", "wave": 2, "mode": "normal", "outcome": "wave timed out",
         "bead": "apw-xwe1", "env": "ubuntu24", "expires": "2026-12-01"}


def failure(outcome="wave timed out", environment="Ubuntu 24.04", mission="mvm_a", wave=2, mode="normal", stage="final"):
    return dict(environment=environment, map="mvm_a", mission=mission, wave=wave, mode=mode, shard=0,
                stage=stage, outcome=outcome, error=outcome)


def apply(issues, entries, complete=("ubuntu24",)):
    return report.apply_known(issues, entries, TODAY, set(complete))


class KnownFailureTest(unittest.TestCase):
    def test_exact_match_is_known_and_leaves_the_denominator(self):
        issues, tolerated, notes = apply([failure()], [ENTRY])
        self.assertEqual(issues[0]["stage"], "known (apw-xwe1)")
        self.assertEqual(tolerated["Ubuntu 24.04"], 1)
        self.assertEqual(notes, [])
        meta = {"commit": "c", "startup": True, "fresh": True, "robot_limiter": True, "crash_free": True, "completed": True}
        summary = {"eligible_cases": 10, "passed_cases": 9, "tested_cases": 10}
        result = report.evaluate(meta, summary, "c", tolerated=1)
        self.assertEqual((result["status"], result["passed"], result["eligible"], result["tested"]), ("passed", 9, 9, 9))

    def test_each_field_must_match(self):
        for change in ({"mission": "mvm_b"}, {"wave": 3}, {"mode": "surge"}, {"environment": "Docker"}):
            issues, tolerated, _ = apply([failure(**change)], [ENTRY], complete=())
            self.assertEqual(issues[0]["stage"], "final", change)
            self.assertEqual(sum(tolerated.values()), 0)

    def test_entry_without_env_applies_to_every_environment(self):
        entry = {key: value for key, value in ENTRY.items() if key != "env"}
        issues, tolerated, _ = apply([failure(environment="Docker"), failure(environment="Windows")], [entry])
        self.assertEqual([issue["stage"] for issue in issues], ["known (apw-xwe1)"] * 2)
        self.assertEqual(tolerated["Windows"], 1)

    def test_changed_verdict_stays_red_and_is_reported(self):
        issues, tolerated, notes = apply([failure(outcome="wave failed")], [ENTRY])
        self.assertEqual(issues[0]["stage"], "final")
        self.assertEqual(sum(tolerated.values()), 0)
        self.assertEqual(len(notes), 1)
        self.assertIn("verdict changed", notes[0])

    def test_expired_entry_stays_red_and_is_reported(self):
        entry = {**ENTRY, "expires": "2026-10-31"}
        issues, _, notes = apply([failure()], [entry])
        self.assertEqual(issues[0]["stage"], "final")
        self.assertIn("expired", notes[0])
        issues, _, _ = apply([failure()], [{**ENTRY, "expires": "2026-11-01"}])
        self.assertTrue(issues[0]["stage"].startswith("known"))

    def test_entry_that_matches_nothing_is_flagged_for_removal_without_failing(self):
        issues, tolerated, notes = apply([], [ENTRY])
        self.assertEqual(issues, [])
        self.assertIn("remove it", notes[0])
        _, _, notes = apply([], [ENTRY], complete=())
        self.assertEqual(notes, [])

    def test_crash_is_never_tolerated(self):
        crash = failure(stage="crash (last observed case)", outcome="wave timed out")
        issues, tolerated, _ = apply([crash], [ENTRY])
        self.assertEqual(issues[0]["stage"], crash["stage"])
        self.assertEqual(sum(tolerated.values()), 0)
        meta = {"commit": "c", "startup": True, "fresh": True, "robot_limiter": True, "crash_free": False, "completed": True}
        summary = {"eligible_cases": 10, "passed_cases": 9, "tested_cases": 10}
        self.assertEqual(report.evaluate(meta, summary, "c", tolerated=1)["status"], "engine crash recorded")
        with self.assertRaises(ValueError):
            self.load([{**ENTRY, "outcome": "crash"}])

    def test_unlisted_failure_keeps_the_environment_red(self):
        meta = {"commit": "c", "startup": True, "fresh": True, "robot_limiter": True, "crash_free": True, "completed": True}
        summary = {"eligible_cases": 10, "passed_cases": 8, "tested_cases": 10}
        self.assertEqual(report.evaluate(meta, summary, "c", tolerated=1)["status"], "wave failures")

    def load(self, entries):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "known.json"
            path.write_text(json.dumps(entries), encoding="utf-8")
            return report.load_known(path)

    def test_invalid_files_fail_loudly(self):
        self.assertEqual(self.load([ENTRY]), [ENTRY])
        bad = [{**ENTRY, "bead": "xwe1"}, {**ENTRY, "expires": "soon"}, {**ENTRY, "expires": 20261201},
               {**ENTRY, "wave": "2"}, {**ENTRY, "wave": 0}, {**ENTRY, "mode": "reverse"}, {**ENTRY, "env": "mac"},
               {**ENTRY, "note": "x"}, {key: value for key, value in ENTRY.items() if key != "bead"}, "mvm_a", [ENTRY, ENTRY]]
        for entry in bad:
            with self.assertRaises(ValueError, msg=entry):
                self.load(entry if isinstance(entry, list) else [entry])
        with self.assertRaises(ValueError):
            self.load({"entries": []})

    def test_shipped_list_is_valid(self):
        entries = report.load_known(report.KNOWN_PATH)
        self.assertTrue(entries)
        self.assertFalse([e for e in entries if e["bead"] in ("apw-4he", "apw-7yuq")])


if __name__ == "__main__":
    unittest.main()
