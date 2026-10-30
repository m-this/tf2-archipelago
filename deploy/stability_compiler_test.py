"""Build-number resolution must never drift to another official compiler."""
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('compiler_source', Path(__file__).parent / 'bots/compiler-source.py')
compiler = importlib.util.module_from_spec(spec)
spec.loader.exec_module(compiler)


class CompilerSourceTest(unittest.TestCase):
    def test_build_count_follows_first_parent_history_including_merges(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            def git(*args):
                return subprocess.check_output(['git', '-C', str(repo), *args], text=True).strip()
            git('init', '-q')
            git('config', 'user.name', 'Fixture')
            git('config', 'user.email', 'fixture@example.invalid')
            git('commit', '--allow-empty', '-qm', 'one')
            first = git('rev-parse', 'HEAD')
            git('branch', 'side')
            git('checkout', '-q', 'side')
            git('commit', '--allow-empty', '-qm', 'side two')
            git('checkout', '-q', '-')
            git('commit', '--allow-empty', '-qm', 'main two')
            second = git('rev-parse', 'HEAD')
            git('merge', '--no-ff', '-qm', 'merge', 'side')
            fourth = git('rev-parse', 'HEAD')
            self.assertEqual(compiler.revision_for_build(repo, 1), first)
            self.assertEqual(compiler.revision_for_build(repo, 2), second)
            self.assertEqual(compiler.revision_for_build(repo, 4), fourth)
            with self.assertRaisesRegex(ValueError, 'refusing another compiler'):
                compiler.revision_for_build(repo, 3)
            with self.assertRaises(ValueError):
                compiler.revision_for_build(repo, 99)
