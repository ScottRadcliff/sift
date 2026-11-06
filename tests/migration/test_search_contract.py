"""Behavior contract for the bundled Python helper.

By default use this interpreter and the skill's helper. SIFT_CONTRACT_COMMAND
can supply a JSON argv array to verify a copied installation instead.
"""

import csv
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class SearchContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        raw = os.environ.get("SIFT_CONTRACT_COMMAND")
        cls.command = json.loads(raw) if raw else [
            sys.executable, str(Path(__file__).resolve().parents[2] / "skills/sift/scripts/search.py")]
        if not isinstance(cls.command, list) or not cls.command or not all(isinstance(v, str) for v in cls.command):
            raise ValueError("SIFT_CONTRACT_COMMAND must be a nonempty JSON string array")

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="sift contract ")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def write(self, name, content):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(content.encode("utf-8"))
        return path

    def search(self, term="needle", path=".", regex=False, case_sensitive=False,
               no_ignore=False, export=None):
        args = []
        if regex:
            args.append("--regex")
        if case_sensitive:
            args.append("--case-sensitive")
        if no_ignore:
            args.append("--no-ignore")
        if export:
            args.extend(["--csv", str(self.root / export)])
        result = subprocess.run(self.command + args + ["--", term, str(self.root / path)],
                                cwd=self.root, capture_output=True, text=True, timeout=15)
        data = json.loads(result.stdout)
        for key in ("matches", "skipped_files", "errors"):
            self.assertIsInstance(data[key], list)
        self.assertIsInstance(data["complete"], bool)
        self.assertIsInstance(data["timed_out"], bool)
        return result, data

    def matches(self, data):
        return [(Path(m["filename"]).relative_to(self.root).as_posix(), m["line_number"], m["text"])
                for m in data["matches"]]

    def test_fixtures(self):
        cases = json.loads((Path(__file__).parent / "fixtures/search_cases.json").read_text())
        # Each fixture has an independent root, including the search directory name with spaces.
        for case in cases:
            with self.subTest(case=case["name"]), tempfile.TemporaryDirectory(prefix="sift fixture ") as directory:
                self.root = Path(directory)
                for name, content in case["files"].items():
                    self.write(name, content)
                kwargs = {k: case[k] for k in ("term", "path", "regex", "case_sensitive", "no_ignore") if k in case}
                result, data = self.search(**kwargs)
                self.assertEqual(result.returncode, case.get("status", 0), result.stderr)
                self.assertEqual(self.matches(data), [tuple(m) for m in case["matches"]])
                self.assertEqual([(Path(s["filename"]).relative_to(self.root).as_posix(), s["reason"])
                                  for s in data["skipped_files"]], [tuple(s) for s in case.get("skips", [])])
                self.assertTrue(data["complete"])
                self.assertFalse(data["timed_out"])
                self.assertEqual(data["errors"], [])

    def test_csv_quoting_self_exclusion_and_collision(self):
        self.write("input.txt", 'needle, "café"\n')
        result, data = self.search(export="results.csv")
        self.assertEqual(result.returncode, 0)
        self.assertEqual(len(data["matches"]), 1)
        destination = self.root / "results.csv"
        original = destination.read_bytes()
        with destination.open(newline="") as stream:
            rows = list(csv.reader(stream))
        self.assertEqual(rows, [["filename", "line_number", "text"],
                                [str(self.root / "input.txt"), "1", 'needle, "café"']])
        result, data = self.search(export="results.csv")
        self.assertEqual(result.returncode, 2)
        self.assertFalse(data["complete"])
        self.assertTrue(data["errors"])
        self.assertEqual(destination.read_bytes(), original)

    def test_empty_csv(self):
        self.write("input", "other")
        result, data = self.search(export="empty.csv")
        self.assertEqual(result.returncode, 1)
        self.assertTrue(data["complete"])
        self.assertEqual((self.root / "empty.csv").read_text().splitlines(), ["filename,line_number,text"])

    def test_missing_path(self):
        result, data = self.search(path="missing")
        self.assertEqual(result.returncode, 2)
        self.assertFalse(data["complete"])
        self.assertTrue(data["errors"])
        self.assertEqual(data["skipped_files"][0]["reason"], "search_error")

    def test_unreadable_ignore_reports_error_and_continues(self):
        if hasattr(os, "geteuid") and os.geteuid() == 0:
            self.skipTest("root bypasses permissions")
        ignore = self.write(".gitignore", "*.txt\n")
        ignore.chmod(0)
        self.addCleanup(ignore.chmod, 0o600)
        self.write("visible.txt", "needle")
        result, data = self.search()
        self.assertEqual(result.returncode, 2)
        self.assertFalse(data["complete"])
        self.assertEqual(self.matches(data), [("visible.txt", 1, "needle")])
        self.assertTrue(data["errors"])
        result, data = self.search(no_ignore=True)
        self.assertEqual(result.returncode, 0)
        self.assertTrue(data["complete"])

    def test_binary_detection_is_prefix_only(self):
        self.write("late-null", "x" * 8192 + "\x00needle\n")
        result, data = self.search()
        self.assertEqual(result.returncode, 0)
        self.assertEqual(len(data["matches"]), 1)
        self.assertEqual(data["skipped_files"], [])

    def test_invalid_regex_has_diagnostic_and_no_json(self):
        args = ["--regex"]
        term = "["
        result = subprocess.run(self.command + args + ["--", term, str(self.root)],
                                capture_output=True, text=True, timeout=15)
        self.assertEqual(result.returncode, 2)
        self.assertEqual(result.stdout, "")
        self.assertTrue(result.stderr)

    def test_unreadable_file_retains_matches_and_partial_csv(self):
        if hasattr(os, "geteuid") and os.geteuid() == 0:
            self.skipTest("root bypasses permissions")
        blocked = self.write("a-private", "needle")
        blocked.chmod(0)
        self.addCleanup(blocked.chmod, 0o600)
        self.write("b-visible", "needle")
        result, data = self.search(export="results.csv")
        self.assertEqual(result.returncode, 2)
        self.assertFalse(data["complete"])
        self.assertEqual(self.matches(data), [("b-visible", 1, "needle")])
        self.assertEqual(data["skipped_files"][0]["reason"], "search_error")
        self.assertTrue(data["errors"])
        self.assertIn("b-visible", (self.root / "results.csv").read_text())

    def test_traversal_skips_symlinks_but_explicit_file_symlink_is_followed(self):
        self.write("target", "needle")
        (self.root / "link").symlink_to("target")
        result, data = self.search()
        self.assertEqual(result.returncode, 0)
        self.assertEqual(self.matches(data), [("target", 1, "needle")])
        result, data = self.search(path="link")
        self.assertEqual(result.returncode, 0)
        self.assertEqual(self.matches(data), [("link", 1, "needle")])


if __name__ == "__main__":
    unittest.main()
