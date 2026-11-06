"""Python helper integration checks beyond preserved Go behavior."""

import csv
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock


HELPER = Path(__file__).resolve().parents[2] / "skills/sift/scripts/search.py"


class PythonHelper(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        sys.path.insert(0, str(HELPER.parent))
        try:
            spec = importlib.util.spec_from_file_location("sift_search_test", HELPER)
            cls.module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(cls.module)
        finally:
            sys.path.pop(0)

    def setUp(self):
        directory = tempfile.TemporaryDirectory(prefix="sift helper ")
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)

    def run_search(self, *args, helper=HELPER, cwd=None, env=None):
        return subprocess.run([sys.executable, "-S", str(helper), *map(str, args)],
                              cwd=cwd or self.root, env=env, capture_output=True,
                              text=True, timeout=10)

    def data(self, result, status=0):
        self.assertEqual(result.returncode, status, result.stderr)
        return json.loads(result.stdout)

    def test_regex_timeout_keeps_matches_skips_errors_and_partial_export(self):
        (self.root / "a-binary").write_bytes(b"needle\0")
        (self.root / "b-file").write_text("aaaa\n" + "a" * 100 + "!\n")
        started = time.monotonic()
        result = self.run_search("--regex", "--timeout", "1", "--csv", "partial.csv", "--", "^(a+)+$", ".")
        self.assertLess(time.monotonic() - started, 5)
        data = self.data(result, 3)
        self.assertTrue(data["timed_out"])
        self.assertFalse(data["complete"])
        self.assertEqual([m["text"] for m in data["matches"]], ["aaaa"])
        self.assertEqual(data["skipped_files"][0]["reason"], "binary")
        with (self.root / "partial.csv").open(newline="") as stream:
            self.assertEqual([r["text"] for r in csv.DictReader(stream)], ["aaaa"])

    def test_tiny_and_unlimited_budgets(self):
        (self.root / "input").write_text("needle")
        for options in ([], ["--regex"]):
            with self.subTest(options=options):
                data = self.data(self.run_search(*options, "--timeout", "0.000000001", "needle", "."), 3)
                self.assertTrue(data["timed_out"])
                data = self.data(self.run_search(*options, "--timeout", "0", "needle", "."))
                self.assertTrue(data["complete"])
                self.assertEqual(len(data["matches"]), 1)

    def test_validation_precedes_export_creation(self):
        for options in (["--regex", "["], ["--timeout", "nan", "needle"],
                        ["--timeout", "inf", "needle"], ["--timeout", "-1", "needle"],
                        ["--timeout", "abc", "needle"], [""]):
            with self.subTest(options=options):
                result = self.run_search("--csv", "must-not-exist.csv", *options, ".")
                self.assertEqual(result.returncode, 2)
                self.assertEqual(result.stdout, "")
                self.assertTrue(result.stderr)
                self.assertFalse((self.root / "must-not-exist.csv").exists())
        result = self.run_search("--regex", "--", "(" * 1000, ".")
        self.assertEqual(result.returncode, 2)
        self.assertEqual(result.stdout, "")
        self.assertIn("invalid regex", result.stderr)

    def test_python_regex_and_case_sensitive_mode(self):
        (self.root / "input").write_text("abab\nPAYMENT\npayment\n")
        for pattern, expected, flags in ((r"^(ab)\1$", ["abab"], []),
                                         ("pay(?=ment)", ["PAYMENT", "payment"], []),
                                         ("payment", ["payment"], ["--case-sensitive"])):
            with self.subTest(pattern=pattern, flags=flags):
                data = self.data(self.run_search("--regex", *flags, "--", pattern, "."))
                self.assertEqual([m["text"] for m in data["matches"]], expected)

    def test_literal_slash_hyphen_and_simple_unicode_folding(self):
        (self.root / "input").write_text("/etc/hosts\n--help\nß ẞ\nss\nİ ı\ni\nΣ σ ς\n")
        for term, expected in (("/etc/hosts", ["/etc/hosts"]), ("--help", ["--help"]),
                               ("ß", ["ß ẞ"]), ("ss", ["ss"]), ("i", ["i"]),
                               ("σ", ["Σ σ ς"])):
            with self.subTest(term=term):
                data = self.data(self.run_search("--", term, "."))
                self.assertEqual([m["text"] for m in data["matches"]], expected)

    def test_utf8_replacement_long_line_crlf_and_final_newline(self):
        text = "a" * 140000 + "needle"
        (self.root / "input").write_bytes(b"\xffneedle\r\n" + text.encode() + b"\nlast needle")
        data = self.data(self.run_search("needle", "."))
        self.assertEqual([(m["line_number"], m["text"]) for m in data["matches"]],
                         [(1, "\ufffdneedle"), (2, text), (3, "last needle")])

    def test_explicit_directory_symlink_is_an_error(self):
        directory = self.root / "directory"
        directory.mkdir()
        (directory / "input").write_text("needle")
        (self.root / "link").symlink_to(directory, target_is_directory=True)
        data = self.data(self.run_search("needle", "link"), 2)
        self.assertFalse(data["complete"])
        self.assertIn("symlink", data["errors"][0])

    def test_regex_export_exclusion_and_collision_before_matching(self):
        (self.root / "input").write_text("filename needle")
        data = self.data(self.run_search("--regex", "--csv", "results.csv", "filename", "."))
        self.assertEqual([m["filename"] for m in data["matches"]], ["input"])
        before = (self.root / "results.csv").read_bytes()
        (self.root / "input").write_text("a" * 100 + "!")
        started = time.monotonic()
        data = self.data(self.run_search("--regex", "--timeout", "0", "--csv", "results.csv", "^(a+)+$", "."), 2)
        self.assertLess(time.monotonic() - started, 3)
        self.assertFalse(data["timed_out"])
        self.assertEqual((self.root / "results.csv").read_bytes(), before)

    def test_regex_permission_errors_are_transferred(self):
        if hasattr(os, "geteuid") and os.geteuid() == 0:
            self.skipTest("root bypasses permissions")
        blocked = self.root / "a-blocked"
        blocked.write_text("needle")
        blocked.chmod(0)
        self.addCleanup(blocked.chmod, 0o600)
        (self.root / "visible").write_text("needle")
        data = self.data(self.run_search("--regex", "needle", "."), 2)
        self.assertFalse(data["complete"])
        self.assertEqual([m["filename"] for m in data["matches"]], ["visible"])
        self.assertEqual(data["skipped_files"][0]["reason"], "search_error")

    def test_copied_skill_works_without_site_packages_or_checkout(self):
        installed = self.root / "installed skill"
        shutil.copytree(HELPER.parents[1], installed, ignore=shutil.ignore_patterns("__pycache__"))
        project = self.root / "other project"
        project.mkdir()
        (project / ".gitignore").write_text("ignored\n")
        (project / "ignored").write_text("needle")
        (project / "input").write_text("needle")
        env = dict(os.environ, PYTHONPATH="", PYTHONNOUSERSITE="1")
        for flags in ([], ["--regex"]):
            with self.subTest(flags=flags):
                result = self.run_search(*flags, "needle", ".", helper=installed / "scripts/search.py", cwd=project, env=env)
                data = self.data(result)
                self.assertEqual([m["filename"] for m in data["matches"]], ["input"])

    def test_integrated_worker_is_reaped_on_timeout(self):
        source = self.root / "input"
        source.write_text("a" * 100 + "!")
        args = self.module.arguments(["--regex", "--timeout", "0.1", "^(a+)+$", str(source)])
        processes = []
        original = subprocess.Popen

        def track(*args, **kwargs):
            process = original(*args, **kwargs)
            processes.append(process)
            return process

        with mock.patch.object(self.module.subprocess, "Popen", side_effect=track):
            status, data = self.module.execute(args)
        self.assertEqual(status, 3)
        self.assertTrue(data["timed_out"])
        self.assertEqual(len(processes), 1)
        self.assertIsNotNone(processes[0].returncode)
        if os.name == "posix":
            with self.assertRaises(ChildProcessError):
                os.waitpid(processes[0].pid, os.WNOHANG)

    def test_unexpected_worker_exit_retains_partial_records_as_error(self):
        args = self.module.arguments(["--regex", "needle", str(self.root)])
        original = subprocess.Popen
        event = json.dumps({"type": "match", "value": {
            "filename": "input", "line_number": 1, "text": "needle"}})

        def fail_worker(command, **kwargs):
            return original([sys.executable, "-c", f"print({event!r}, flush=True); raise SystemExit(7)"], **kwargs)

        with mock.patch.object(self.module.subprocess, "Popen", side_effect=fail_worker):
            status, data = self.module.execute(args)
        self.assertEqual(status, 2)
        self.assertFalse(data["complete"])
        self.assertFalse(data["timed_out"])
        self.assertEqual(data["matches"][0]["text"], "needle")
        self.assertIn("status 7", data["errors"][0])

    def test_csv_finalization_failure_takes_priority_over_timeout(self):
        class FailingClose(io.StringIO):
            def close(self):
                super().close()
                raise OSError("simulated full disk on finalization")

        args = self.module.arguments(["--regex", "--csv", str(self.root / "export"), "needle", str(self.root)])
        with mock.patch("builtins.open", return_value=FailingClose()), mock.patch.object(
                self.module, "isolated_scan", return_value=True):
            status, data = self.module.execute(args)
        self.assertEqual(status, 2)
        self.assertTrue(data["timed_out"])
        self.assertFalse(data["complete"])
        self.assertIn("finalize CSV", data["errors"][-1])

    def test_json_write_failure_returns_error_without_relying_on_dev_full(self):
        class FailingWrite(io.StringIO):
            def write(self, value):
                raise OSError("simulated output failure")

        source = self.root / "input"
        source.write_text("needle")
        diagnostics = io.StringIO()
        with mock.patch.object(self.module.sys, "stdout", FailingWrite()), mock.patch.object(
                self.module.sys, "stderr", diagnostics):
            status = self.module.main(["needle", str(source)])
        self.assertEqual(status, 2)
        self.assertIn("write JSON", diagnostics.getvalue())

    @unittest.skipUnless(Path("/dev/full").exists(), "requires /dev/full")
    def test_json_output_failure_returns_error(self):
        (self.root / "input").write_text("needle")
        with open("/dev/full", "w") as full:
            result = subprocess.run([sys.executable, str(HELPER), "needle", str(self.root)],
                                    stdout=full, stderr=subprocess.PIPE, text=True, timeout=10)
        self.assertEqual(result.returncode, 2)
        self.assertIn("write JSON", result.stderr)


if __name__ == "__main__":
    unittest.main()
