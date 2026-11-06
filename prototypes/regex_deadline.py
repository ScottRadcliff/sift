"""Migration experiment, not the skill helper. Single-file UTF-8 regex search.

Worker records go to a temporary spool outside the searched tree. The parent
never blocks reading a partially written pipe message. Only complete JSONL
records are retained after cancellation. Production integration is step two.
"""

import csv
import json
import math
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time


def _worker(pattern, filename):
    compiled = re.compile(pattern, re.IGNORECASE)
    with open(filename, encoding="utf-8", errors="replace", newline="") as stream:
        for number, line in enumerate(stream, 1):
            text = line.removesuffix("\n").removesuffix("\r")
            if compiled.search(text):
                print(json.dumps({"filename": filename, "line_number": number, "text": text}), flush=True)


def search(pattern, filename, timeout=3, csv_path=None):
    """Return (exit status, result, reaped worker pid) for prototype tests.

    Budget includes process startup, not validation or output finalization.
    A zero budget is unlimited. CSV uses exclusive creation even on timeout.
    """
    if not math.isfinite(timeout) or timeout < 0:
        raise ValueError("timeout must be finite and nonnegative")
    re.compile(pattern, re.IGNORECASE)
    result = {"matches": [], "skipped_files": [], "errors": [],
              "timed_out": False, "complete": True}
    started = time.monotonic()
    with tempfile.TemporaryFile() as spool, tempfile.TemporaryFile() as diagnostics:
        worker = subprocess.Popen([sys.executable, str(Path(__file__).resolve()),
                                   "--worker", pattern, str(filename)],
                                  stdin=subprocess.DEVNULL, stdout=spool, stderr=diagnostics)
        try:
            remaining = None if timeout == 0 else max(0, timeout - (time.monotonic() - started))
            try:
                worker.wait(timeout=remaining)
            except subprocess.TimeoutExpired:
                result.update(timed_out=True, complete=False)
                result["errors"].append("Search time limit exceeded; results may be partial")
        finally:
            if worker.poll() is None:
                worker.kill()
            worker.wait()
        spool.seek(0)
        for record in spool:
            # A killed worker may have been writing a long record.
            if not record.endswith(b"\n"):
                if not result["timed_out"]:
                    result["errors"].append("Incomplete worker record")
                    result["complete"] = False
                break
            result["matches"].append(json.loads(record))
        if worker.returncode and not result["timed_out"]:
            diagnostics.seek(0)
            result["errors"].append(diagnostics.read().decode("utf-8", errors="replace"))
            result["complete"] = False
    status = 3 if result["timed_out"] else (2 if result["errors"] else (0 if result["matches"] else 1))
    if csv_path is not None:
        try:
            with open(csv_path, "x", encoding="utf-8", newline="") as stream:
                writer = csv.DictWriter(stream, fieldnames=["filename", "line_number", "text"])
                writer.writeheader()
                writer.writerows(result["matches"])
        except OSError as error:
            result["errors"].append(str(error))
            result["complete"] = False
            status = 2
    return status, result, worker.pid


if __name__ == "__main__":
    if len(sys.argv) == 4 and sys.argv[1] == "--worker":
        _worker(sys.argv[2], sys.argv[3])
    else:
        raise SystemExit("Experimental module: run through tests/migration/test_regex_deadline.py")
