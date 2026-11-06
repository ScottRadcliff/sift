import csv
import importlib.util
import os
from pathlib import Path
import re
import tempfile
import time
import unittest


spec = importlib.util.spec_from_file_location(
    "regex_deadline", Path(__file__).resolve().parents[2] / "prototypes/regex_deadline.py")
prototype = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prototype)


class RegexDeadline(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(prefix="sift regex ")
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.input = self.root / "input.txt"

    def assert_reaped(self, pid):
        if os.name == "posix":
            with self.assertRaises(ChildProcessError):
                os.waitpid(pid, os.WNOHANG)

    def test_catastrophic_regex_retains_matches_and_finalizes_csv(self):
        # First record is easy; the second would backtrack for far longer than
        # the budget. A subprocess is necessary: a thread cannot stop re.search.
        self.input.write_text("aaaa\n" + "a" * 100 + "!\n")
        export = self.root / "partial.csv"
        started = time.monotonic()
        status, data, pid = prototype.search("^(a+)+$", self.input, timeout=1, csv_path=export)
        self.assertLess(time.monotonic() - started, 5)
        self.assertEqual(status, 3)
        self.assertTrue(data["timed_out"])
        self.assertFalse(data["complete"])
        self.assertEqual([m["text"] for m in data["matches"]], ["aaaa"])
        with export.open(newline="") as stream:
            self.assertEqual([r["text"] for r in csv.DictReader(stream)], ["aaaa"])
        self.assert_reaped(pid)

    def test_unlimited_search_and_utf8_line_handling(self):
        self.input.write_bytes('CAFÉ\r\ncafé'.encode())
        status, data, pid = prototype.search("café", self.input, timeout=0)
        self.assertEqual(status, 0)
        self.assertEqual([m["text"] for m in data["matches"]], ["CAFÉ", "café"])
        self.assertTrue(data["complete"])
        self.assert_reaped(pid)

    def test_tiny_budget_cancels_startup_and_reaps(self):
        self.input.write_text("needle")
        status, data, pid = prototype.search("needle", self.input, timeout=0.000001)
        self.assertEqual(status, 3)
        self.assertFalse(data["complete"])
        self.assert_reaped(pid)

    def test_worker_failure_is_not_no_matches(self):
        status, data, pid = prototype.search("needle", self.input)
        self.assertEqual(status, 2)
        self.assertFalse(data["complete"])
        self.assertTrue(data["errors"])
        self.assert_reaped(pid)

    def test_export_collision_preserves_destination(self):
        self.input.write_text("needle")
        export = self.root / "existing.csv"
        export.write_text("existing")
        status, data, pid = prototype.search("needle", self.input, csv_path=export)
        self.assertEqual(status, 2)
        self.assertFalse(data["complete"])
        self.assertEqual(export.read_text(), "existing")
        self.assert_reaped(pid)

    def test_invalid_pattern_and_budgets_fail_before_worker(self):
        with self.assertRaises(re.error):
            prototype.search("[", self.input)
        for budget in (-1, float("nan"), float("inf")):
            with self.subTest(budget=budget), self.assertRaises(ValueError):
                prototype.search("needle", self.input, timeout=budget)

    def test_python_regex_semantics_are_explicit(self):
        self.input.write_text("abab\npayment\nİ ı ſ K\n")
        for pattern, expected in ((r"^(ab)\1$", ["abab"]),
                                  (r"pay(?=ment)", ["payment"]),
                                  (r"(?i)^PAYMENT$", ["payment"]),
                                  (r"^[a-z] [a-z] [a-z] [a-z]$", ["İ ı ſ K"])):
            with self.subTest(pattern=pattern):
                status, data, pid = prototype.search(pattern, self.input)
                self.assertEqual(status, 0)
                self.assertEqual([m["text"] for m in data["matches"]], expected)
                self.assert_reaped(pid)


if __name__ == "__main__":
    unittest.main()
