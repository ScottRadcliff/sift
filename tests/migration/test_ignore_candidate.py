"""Optional dependency evaluation; supply the downloaded wheel on PYTHONPATH."""

import importlib.util
import json
from pathlib import Path
import tempfile
import unittest


@unittest.skipUnless(importlib.util.find_spec("pathspec"), "supply pathspec 1.1.1 wheel on PYTHONPATH")
class IgnoreCandidate(unittest.TestCase):
    def test_scoped_selection_matches_baseline_fixtures(self):
        spec = importlib.util.spec_from_file_location(
            "ignore_paths", Path(__file__).resolve().parents[2] / "prototypes/ignore_paths.py")
        prototype = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(prototype)
        cases = json.loads((Path(__file__).parent / "fixtures/search_cases.json").read_text())
        for case in cases:
            if not any(".gitignore" in name for name in case["files"]) and "hidden" not in case["name"]:
                continue
            with self.subTest(case=case["name"]), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                for name, text in case["files"].items():
                    path = root / name
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_bytes(text.encode())
                selected = prototype.selected_files(root / case.get("path", "."), case.get("no_ignore", False))
                self.assertEqual([p.relative_to(root).as_posix() for p in selected],
                                 [m[0] for m in case["matches"]])


if __name__ == "__main__":
    unittest.main()
