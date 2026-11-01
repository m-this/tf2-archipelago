import unittest

from tf2_pins import add_changelog_entry, update_versions

VERSIONS = """SIGSEGV_MVM_WINDOWS_VERSION=20261030
SIGSEGV_MVM_WINDOWS_SHA256=aa
SOURCEMOD_TF2_SERVER_VERSION=11185272
# comment
TF2_SERVER_VERSION=11185272
TF2_DEPOT_232255_MANIFEST=1
TF2_DEPOT_232256_MANIFEST=2
"""
MANIFESTS = {"232250": "10", "232255": "11", "232256": "12"}
SHA = "8a" * 32


class VersionsTest(unittest.TestCase):
    def test_moves_every_pin_and_adds_a_missing_depot(self):
        out = update_versions(VERSIONS, "11191711", "20261101", SHA, MANIFESTS)
        self.assertIn("SIGSEGV_MVM_WINDOWS_VERSION=20261101\n", out)
        self.assertIn(f"SIGSEGV_MVM_WINDOWS_SHA256={SHA}\n", out)
        self.assertIn("SOURCEMOD_TF2_SERVER_VERSION=11191711\n", out)
        self.assertIn("TF2_SERVER_VERSION=11191711\nTF2_DEPOT_232250_MANIFEST=10\n", out)
        self.assertIn("TF2_DEPOT_232255_MANIFEST=11\n", out)
        self.assertEqual(out, update_versions(out, "11191711", "20261101", SHA, MANIFESTS))

    def test_refuses_a_bad_checksum(self):
        with self.assertRaises(ValueError):
            update_versions(VERSIONS, "11191711", "20261101", "nope", MANIFESTS)

    def test_refuses_a_missing_pin(self):
        with self.assertRaises(ValueError):
            update_versions("TF2_SERVER_VERSION=1\n", "2", "3", SHA, MANIFESTS)


class ChangelogTest(unittest.TestCase):
    HEAD = "intro\n\n## v1.0.0\n\n### Added\n\n- x\n"

    def test_creates_the_section_once(self):
        out = add_changelog_entry(self.HEAD, "11191711")
        self.assertTrue(out.startswith("intro\n\n## Unreleased\n\n### Fixed\n\n- **Servers work on TF2 build 11191711.**"))
        self.assertIn("\n## v1.0.0\n", out)
        self.assertEqual(out, add_changelog_entry(out, "11191711"))

    def test_adds_to_an_existing_fixed_list(self):
        text = "## Unreleased\n\n### Fixed\n\n- old\n\n## v1.0.0\n"
        out = add_changelog_entry(text, "2")
        self.assertIn("### Fixed\n\n- **Servers work on TF2 build 2.**", out)
        self.assertIn("- old\n\n## v1.0.0", out)

    def test_adds_fixed_to_an_unreleased_without_one(self):
        out = add_changelog_entry("## Unreleased\n\n### Added\n\n- a\n\n## v1.0.0\n", "2")
        self.assertIn("### Added\n\n- a\n\n### Fixed\n\n- **Servers work on TF2 build 2.**", out)


if __name__ == "__main__":
    unittest.main()
