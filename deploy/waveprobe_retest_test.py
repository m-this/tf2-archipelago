import importlib.util
import json
import os
import pathlib
import tempfile
import unittest
from unittest.mock import patch

module_path = pathlib.Path(__file__).with_name("waveprobe-retest.py")
spec = importlib.util.spec_from_file_location("waveprobe_retest", module_path)
retest = importlib.util.module_from_spec(spec)
spec.loader.exec_module(retest)


class RetestRecoveryTest(unittest.TestCase):
    def test_changelevel_failure_keeps_evidence_for_requested_wave(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            (root / "config.txt").write_text("shards=1\nwave_timeout=1s\n")
            case = {"mission": "mvm_deathpour_rc1_int_technical_terror", "map": "mvm_deathpour_rc1",
                    "mode": "normal", "wave": 5}
            (root / "plan.jsonl").write_text(json.dumps({**case, "state": "planned"}) + "\n")
            (root / "shard-0.jsonl").write_text(json.dumps({**case, "state": "failed",
                                                             "outcome": "wave timed out"}) + "\n")
            evidence = {"requested_map": "mvm_deathpour_rc1", "before_map": "mvm_decoy",
                        "before_pop": "mvm_decoy_advanced", "after_map": "mvm_decoy",
                        "after_pop": "mvm_decoy_advanced", "after_state": "idle"}
            load_row = {**case, "wave": 0, "state": "load_failed",
                        "outcome": "changelevel failure", "error": "map remained on Decoy",
                        "changelevel": evidence}
            runner = root / "probe"
            runner.write_text("#!/bin/sh\necho '" + json.dumps(load_row) + "'\n")
            runner.chmod(0o755)
            retest.main(root, runner, [0])
            result = json.loads((root / "retest-0.jsonl").read_text())
            self.assertEqual(result["wave"], 5)
            self.assertEqual(result["outcome"], "changelevel failure")
            self.assertEqual(result["changelevel"], evidence)

    def test_balanced_partition_and_native_rcon_host_survive_retries(self):
        for profile, flag in [("reverse_bombs", "-reverse-bombs"), ("reverse_objectives", "-reverse-objectives")]:
            with tempfile.TemporaryDirectory() as folder:
                root = pathlib.Path(folder)
                (root/"config.txt").write_text(f"shards=2\nspeed=50\nwave_timeout=1s\nrcon_host=172.20.144.1\n{profile}=1\n")
                (root/"partition.json").write_text(json.dumps({"shards":2,"missions":{"first":0,"second":1}}))
                base = {"map":"mvm_decoy", "mode":"normal", "wave":1}
                (root/"plan.jsonl").write_text("".join(json.dumps({**base,"mission":m,"state":"planned"})+"\n" for m in ["first","second"]))
                probe = root/"probe"
                probe.write_text("#!/bin/sh\nprintf '%s\\n' \"$@\" > '"+str(root/"args")+"'\necho '"+json.dumps({**base,"mission":"second","state":"passed","outcome":"passed"})+"'\n")
                probe.chmod(0o755)
                retest.main(root,probe,[0],[1])
                result = json.loads((root/"retest-0.jsonl").read_text())
                self.assertEqual(result["mission"], "second")
                args = (root/"args").read_text().splitlines()
                self.assertEqual(args[args.index("-rcon")+1],"172.20.144.1:27035")
                self.assertEqual(args[args.index("-speed")+1],"50")
                self.assertIn(flag, args)

    def test_queue_spreads_map_loads(self):
        cases = [("a1", "map_a"), ("a2", "map_a"),
                 ("b1", "map_b"), ("b2", "map_b"),
                 ("c1", "map_c"), ("c2", "map_c")]
        self.assertEqual([case[1] for case in retest.interleave_maps(cases)],
                         ["map_a", "map_b", "map_c", "map_a", "map_b", "map_c"])

    def test_screen_only_rechecks_probe_errors(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            (root / "config.txt").write_text("shards=1\nfirst_pass_timeout=3s\nwave_timeout=900s\n")
            base = {"mission": "mvm_decoy", "map": "mvm_decoy", "mode": "normal"}
            (root / "plan.jsonl").write_text("".join(
                json.dumps({**base, "wave": wave, "state": "planned"}) + "\n"
                for wave in (1, 2, 3)))
            (root / "shard-0.jsonl").write_text("".join(
                json.dumps({**base, "wave": wave, "state": state, "outcome": outcome}) + "\n"
                for wave, state, outcome in ((1, "passed", "passed"),
                                             (2, "failed", "wave timed out"),
                                             (3, "failed", "probe error"))))
            runner = root / "probe"
            runner.write_text("#!/bin/sh\n"
                              "echo '{\"mission\":\"mvm_decoy\",\"mode\":\"normal\",\"wave\":3,\"state\":\"passed\",\"outcome\":\"passed\"}'\n")
            runner.chmod(0o755)
            with patch.dict(os.environ, {"WAVEPROBE_PHASE": "screen"}):
                retest.main(root, runner, [0])
            rows = (root / "screen-0.jsonl").read_text().splitlines()
            self.assertEqual(len(rows), 1)
            self.assertEqual(json.loads(rows[0])["wave"], 3)

    def test_rcon_failure_restarts_server_and_retries_case(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            (root / "config.txt").write_text("shards=1\nbase_port=27035\nwave_timeout=1s\n")
            case = {"mission": "mvm_decoy", "map": "mvm_decoy", "mode": "normal", "wave": 1}
            (root / "plan.jsonl").write_text(json.dumps({**case, "state": "planned"}) + "\n")
            (root / "shard-0.jsonl").write_text(json.dumps({**case, "state": "failed"}) + "\n")
            (root / "projects.txt").write_text("probe-test\n")
            bin_dir = root / "bin"
            bin_dir.mkdir()
            docker = bin_dir / "docker"
            docker.write_text("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$TEST_DOCKER_LOG\"\n")
            docker.chmod(0o755)
            runner = bin_dir / "probe"
            runner.write_text("#!/bin/sh\n"
                              "if [ ! -e \"$TEST_FIRST_RUN\" ]; then\n"
                              "  touch \"$TEST_FIRST_RUN\"\n"
                              "  echo '{\"wave\":0,\"error\":\"connection reset by peer\"}'\n"
                              "else\n"
                              "  echo '{\"mission\":\"mvm_decoy\",\"mode\":\"normal\",\"wave\":1,\"state\":\"passed\",\"outcome\":\"passed\"}'\n"
                              "fi\n")
            runner.chmod(0o755)
            env = {"PATH": str(bin_dir) + os.pathsep + os.environ["PATH"],
                   "TEST_DOCKER_LOG": str(root / "docker.log"),
                   "TEST_FIRST_RUN": str(root / "first-run")}
            with patch.dict(os.environ, env):
                retest.main(root, runner, [0])
            result = json.loads((root / "retest-0.jsonl").read_text())
            self.assertEqual(result["outcome"], "passed")
            self.assertEqual((root / "docker.log").read_text().splitlines(),
                             ["restart probe-test-srcds-1", "restart probe-test-srcds-1"])


if __name__ == "__main__":
    unittest.main()
