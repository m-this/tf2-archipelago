"""Steam setup failures must stay actionable without erasing their evidence."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from stability_errors import install_error, transient_install_error

spec = importlib.util.spec_from_file_location('setup_runner', Path(__file__).with_name('stability-run.py'))
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


class SetupFailureTest(unittest.TestCase):
    def failure(self, path):
        path.write_text('\n'.join('time=2026-10-26 level=INFO msg=install step=' + json.dumps(line)
                                 for line in ['  \\x1b[0mERROR! Info request for AppId 7 returned error Timeout.',
                                              '  ERROR! Failed to load platform info']))
        Path(str(path)+'.err').write_text('winbed: SteamCMD could not install app 232250: exit status 5. If it keeps failing, open Settings\nopen tf2_waveprobe.smx: no such file or directory\n')

    def test_real_failure_extracts_steam_cause_not_secondary_missing_probe(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)/'install.log'
            self.failure(path)
            self.assertTrue(transient_install_error(path))
            self.assertEqual(install_error(path), 'SteamCMD: Info request for AppId 7 returned error Timeout.; Failed to load platform info')
            self.assertNotIn('tf2_waveprobe', install_error(path))
            # Linux's Go text logger emits a single-backslash x1b escape.
            path.write_text(path.read_text().replace(r"\\x1b", r"\x1b"))
            self.assertEqual(install_error(path), 'SteamCMD: Info request for AppId 7 returned error Timeout.; Failed to load platform info')

    def test_docker_entrypoint_metadata_failure_does_not_need_wrapper_stderr(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)/'server.log'
            path.write_text('Waiting for client config...ERROR! Info request for AppId 7 returned error Timeout.\nERROR! Failed to load platform info\n[AP] the TF2 update failed, SteamCMD exited 5\n')
            self.assertEqual(install_error(path),'SteamCMD: Info request for AppId 7 returned error Timeout.; Failed to load platform info')

    def test_missing_case_reports_steam_cause_from_legacy_installer_logs(self):
        spec = importlib.util.spec_from_file_location('setup_report', Path(__file__).with_name('stability-report.py'))
        report = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(report)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            part = root/'part-0'
            part.mkdir()
            row = dict(mission='fixture',map='mvm_map',mode='normal',wave=1,state='planned')
            for path in (root/'plan.jsonl',part/'plan.jsonl'):
                path.write_text(json.dumps(row)+'\n')
            self.failure(part/'install.log')
            meta = {'shards':[{'shard':0,'startup':False,'error':"Command ['winbed', '-install'] returned non-zero exit status 1"}]}
            issues = report.collect_issues(root,'Windows',meta)
            missing = next(issue for issue in issues if issue['stage']=='final')
            self.assertIn('Info request for AppId 7',missing['error'])
            self.assertNotIn("Command [",missing['error'])
            self.assertEqual((missing['map'],missing['mission'],missing['wave']),('mvm_map','fixture',1))

    def test_transient_retry_preserves_first_logs_and_resumes_same_install(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            calls = []
            def run(args, filename, limit):
                calls.append((args, filename, limit))
                if len(calls) == 1:
                    self.failure(output/filename)
                    raise runner.PhaseFailure(filename, 1, install_error(output/filename))
                (output/filename).write_text('installed latest build')
            with mock.patch.object(runner.time, 'sleep') as sleep, mock.patch.object(runner.time, 'monotonic', return_value=0):
                self.assertEqual(runner.install_with_retries(run, ['bed','-root','same-root'], output, 10000, 2), 2)
            self.assertEqual(calls[0][0], calls[1][0])
            self.assertEqual(calls[0][2], 7200)
            self.assertEqual(calls[1][1], 'install-attempt-2.log')
            self.assertIn('Timeout', (output/'install.log').read_text())
            sleep.assert_called_once_with(36)

    def test_persistent_service_failure_stops_after_three_attempts(self):
        with tempfile.TemporaryDirectory() as directory:
            output=Path(directory)
            calls=[]
            def run(args, filename, limit):
                calls.append(filename)
                self.failure(output/filename)
                raise runner.PhaseFailure(filename, 1, 'Steam unavailable')
            with mock.patch.object(runner.time,'sleep'), mock.patch.object(runner.time,'monotonic',return_value=0):
                with self.assertRaises(runner.PhaseFailure):
                    runner.install_with_retries(run,[],output,10000,0)
            self.assertEqual(len(calls),3)
            self.assertTrue(all((output/name).exists() for name in calls))

    def test_wrong_pins_and_missing_libraries_are_not_network_retries(self):
        with tempfile.TemporaryDirectory() as directory:
            output=Path(directory)
            path=output/'install.log'
            for error in ('checksum mismatch', 'missing 32-bit runtime', 'HTTP 404'):
                with self.subTest(error=error):
                    path.write_text('ERROR! '+error)
                    Path(str(path)+'.err').write_text('winbed: '+error)
                    run=mock.Mock(side_effect=runner.PhaseFailure('install.log',1,error))
                    with mock.patch.object(runner.time,'sleep') as sleep:
                        with self.assertRaises(runner.PhaseFailure):
                            runner.install_with_retries(run,[],output,10**15,0)
                    run.assert_called_once()
                    sleep.assert_not_called()

    def test_retry_does_not_extend_the_setup_deadline(self):
        with tempfile.TemporaryDirectory() as directory:
            output=Path(directory)
            self.failure(output/'install.log')
            run=mock.Mock(side_effect=runner.PhaseFailure('install.log',1,'Steam timeout'))
            with mock.patch.object(runner.time,'monotonic',return_value=100), mock.patch.object(runner.time,'sleep') as sleep:
                with self.assertRaises(runner.PhaseFailure):
                    runner.install_with_retries(run,[],output,110,0)
            sleep.assert_not_called()
