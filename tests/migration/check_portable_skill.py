"""Run behavioral checks with only a copied skill and an isolated test harness.

Usage from the repository root: python3 tests/migration/check_portable_skill.py
No Go binary, dependency downloads, or site packages are used.
"""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile


def main():
    repository = Path(__file__).resolve().parents[2]
    with tempfile.TemporaryDirectory(prefix="sift portable installation ") as directory:
        root = Path(directory)
        installed = root / "installed skills" / "sift"
        shutil.copytree(repository / "skills/sift", installed,
                        ignore=shutil.ignore_patterns("__pycache__", "*.pyc"))
        # Copy the test harness too: the subprocess must not load checkout files.
        harness = root / "verification harness"
        harness.mkdir()
        for name in ("test_search_contract.py", "test_python_helper.py"):
            shutil.copy2(repository / "tests/migration" / name, harness / name)
        shutil.copytree(repository / "tests/migration/fixtures", harness / "fixtures")
        project = root / "unrelated project"
        project.mkdir()
        temporary_files = root / "temporary files"
        temporary_files.mkdir()
        empty_path = root / "empty executable path"
        empty_path.mkdir()

        helper = installed / "scripts/search.py"
        environment = dict(os.environ)
        for key in ("PYTHONPATH", "PYTHONHOME", "PYTHONSTARTUP", "PYTHONUSERBASE"):
            environment.pop(key, None)
        environment.update(
            PATH=str(empty_path),
            PYTHONNOUSERSITE="1",
            PYTHONDONTWRITEBYTECODE="1",
            TMPDIR=str(temporary_files), TEMP=str(temporary_files), TMP=str(temporary_files),
            SIFT_CONTRACT_ENGINE="python",
            SIFT_CONTRACT_COMMAND=json.dumps([sys.executable, "-S", str(helper)]),
            SIFT_PYTHON_HELPER=str(helper),
        )
        print("Checking copied skill with site packages disabled and an empty executable PATH", flush=True)
        process = subprocess.run(
            [sys.executable, "-S", "-m", "unittest", "discover", "-s", str(harness), "-v"],
            cwd=project, env=environment, timeout=120)
        if process.returncode:
            return process.returncode
        print("Portable installation checks passed; temporary skill, harness, and exports cleaned up", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
