# Python checks

Requires Python 3.11+. Run the behavior, integration, and migration feasibility
checks from the repository root:

```sh
make test
make portable
```

Equivalent commands without Make:

```sh
PYTHONPATH=skills/sift/scripts/_vendor \
  python3 -m unittest discover -s tests/migration -v
python3 tests/migration/check_portable_skill.py
```

No Go binary, Docker, or package installation is required. `PYTHONPATH` lets the
historical ignore-candidate experiment use the bundled dependency; the actual
helper always resolves its own bundled copy and needs no environment setup.
Tests use the current interpreter and helper by default. An optional
`SIFT_CONTRACT_COMMAND` JSON argv array selects another helper installation;
`SIFT_PYTHON_HELPER` selects the helper used by integration tests. There is no
longer a Go engine adapter.

The behavior fixtures assert ordered matches and skips. Additional checks cover
exports, permissions, symlinks, binary-prefix detection, validation failures,
Unicode, incomplete searches, regex cancellation, worker cleanup and crashes,
CSV finalization failures, and JSON output failures. Tests create fixtures and
exports in temporary directories, including paths with spaces. Permission checks
skip when run as root; `/dev/full` skips where unavailable, with a portable
fault-injection test also covering JSON write errors.

The portability command copies only `skills/sift/` and the contract/integration
test harness into a temporary directory. Tests run from an unrelated project
with site packages disabled, Python environment overrides removed, and an empty
executable PATH. All helper calls and imports use those copies. No checkout
runtime code, installed Sift command, or dependency download is used. Temporary
installations, fixtures, and exports are cleaned up afterward.

The earlier regex and ignore prototypes remain as migration experiments outside
the installable skill. CI runs the full Python suite and the copied-installation
check on Python 3.11 and 3.13.
