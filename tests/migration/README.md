# Migration checks

Requires Python 3.11+. These checks do not switch the installed skill or replace
the Go implementation.

Build the current Go source with `make binary`, then run from the repository root:

```sh
SIFT_CONTRACT_COMMAND='["/absolute/path/to/sift/bin/sift"]' \
  python3 -m unittest discover -s tests/migration -v
```

An explicit command is required; the suite does not silently use a potentially
stale binary. The argument is a JSON array, not shell code. The baseline must be
a host-native build. The bundled Python helper runs the same preserved-behavior
checks using:

```sh
SIFT_CONTRACT_ENGINE=python \
SIFT_CONTRACT_COMMAND='["python3", "/absolute/path/to/skill/scripts/search.py"]' \
PYTHONPATH=skills/sift/scripts/_vendor \
  python3 -m unittest discover -s tests/migration -v
```

The adapter changes only invocation syntax; it does not relax assertions for the
Python implementation. JSON fixture cases assert ordered matches and skips.
Additional checks cover exports, permissions, symlinks, binary-prefix detection,
validation failures, and incomplete searches. Fixtures are created in temporary
directories whose names contain spaces. No test writes exports into this checkout.

Run only the standalone regex prototype tests without a Go binary:

```sh
python3 -m unittest discover -s tests/migration -p test_regex_deadline.py -v
```

The regex tests exercise a pathological backtracking expression, preservation of
earlier matches, CSV finalization, startup cancellation, worker cleanup, invalid
arguments, worker errors, and destination collisions. The single-file prototype
is intentionally outside the installed skill. It is not a production search
engine and does not implement ignores, traversal, or binary filtering.

The prototype starts its budget before process creation and finalizes output
after stopping/reaping the worker. The pathological-pattern test allows one
second for startup and matching, with a five-second elapsed-time assertion to
catch ineffective cancellation without treating small scheduling delays as bugs.
Permission checks skip when run as root. The Linux-only `/dev/full` test skips
when that device is absent; a portable fault-injection test also checks JSON
write failures. `test_python_helper.py` covers the actual helper, including its
worker cleanup, timeout handling, CSV finalization failures, worker crashes, and
operation from a copied skill folder with site packages disabled.

To run the ignore-library evaluation alone using the bundled source:

```sh
PYTHONPATH=skills/sift/scripts/_vendor \
  python3 -m unittest discover -s tests/migration -p test_ignore_candidate.py -v
```

Without `pathspec` on the test runner's module path, the optional candidate test
explicitly skips. The helper itself always resolves its bundled copy and needs
no `PYTHONPATH` setup or dependency installation.
