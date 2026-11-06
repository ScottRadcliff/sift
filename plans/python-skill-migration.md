# Sift migration to a self-contained Python skill

## Decision and scope

Sift will be a skill with bundled Python code. The supported user interface is the skill: the agent interprets requests, invokes a helper relative to the installed skill directory, and presents results. Users will not build, install, or manage a separate CLI.

All six migration steps are complete. This document retains the decisions and sequence that replaced the former Go CLI; the README and skill instructions describe current use.

Keep the existing `skills/sift/` location as the installable unit. The complete folder must work when copied outside this checkout. No runtime code, fixtures, or configuration may depend on repository paths, an installed `sift` command, Go, Docker, or a network connection.

Target Python 3.11 or newer. Confirm that interpreter is available in the intended agent environment before implementation. Do not install Python or dependencies automatically during searches.

## Proposed layout

```text
skills/sift/
  SKILL.md
  agents/openai.yaml
  scripts/
    search.py
    # Add private modules only when implementation complexity warrants them.
  references/
    search-behavior.md
tests/
samples/
README.md
plans/python-skill-migration.md
.github/workflows/ci.yml
```

Keep the skill entrypoint concise: invocation, argument safety, scope, result interpretation, timeout handling, and exports. Put detailed search semantics in the reference. Tests and synthetic samples belong to development, outside the installed skill.

## Helper interface

Example invocation, with the actual installed skill path substituted:

```sh
python3 /absolute/path/to/sift/scripts/search.py --timeout 3 -- 'payment failed' app/
python3 /absolute/path/to/sift/scripts/search.py --regex --csv results.csv -- 'payment|retry' app/
```

- JSON is the only search output format; remove `--json` and `--no-interactive`.
- Require one nonempty search term and one file or directory. The skill selects the current working directory when no scope is specified by the request or context.
- Literal search is the default. Use an explicit `--regex` flag instead of slash delimiters. This makes literal path text such as `/etc/hosts` unambiguous.
- Retain `--case-sensitive`, `--no-ignore`, and `--csv`.
- Express `--timeout` in nonnegative seconds, including decimals; default to 3, with 0 meaning unlimited. Reject negative, nonfinite, and malformed values.
- Put options before `--`; pass search text and paths as separate arguments with correct quoting.
- Resolve input and export paths against the command's working directory. Resolve helper resources against the helper's own location. Never change to the skill directory to search.
- Keep statuses 0 (matches), 1 (no matches), 2 (error), and 3 (timeout). Output failures take priority over timeout.
- Preserve the existing result fields: `matches`, `skipped_files`, `errors`, `timed_out`, and `complete`; retain match fields `filename`, `line_number`, and `text`.
- Validation failures return status 2 with a diagnostic on stderr before a search result exists. Searches return valid JSON, including partial results when errors or timeouts occur. Arrays remain arrays when empty.

The helper's arguments are an internal skill contract. No compatibility layer for the old CLI is needed.

## Behavior to preserve

- Case-insensitive literal matching by default; original text and one-based line numbers in results; one record per matching line.
- Recursive traversal, hidden-entry exclusion, and no following symlinks. Inspect the existing behavior for explicitly supplied symlinks and define it consistently before porting.
- Explicit hidden files/directories remain searchable; explicit files bypass ignore rules.
- Directory-local and nested `.gitignore` rules, scoped to the selected root and descendants. No ancestor/global ignore rules. `--no-ignore` disables ignore rules without enabling hidden or binary files.
- Null-byte binary detection in the first 8 KB, with binary skips reported separately from errors.
- Continued directory searching after unreadable files, with partial results and status 2.
- CSV columns `filename`, `line_number`, `text`; proper quoting; exclusive file creation; no overwriting existing exports; exclusion of the export from its own search.
- Header-only CSV for no matches, preserved partial CSV after an incomplete search, and explicit reporting of completion state.
- No interpreting searched file contents as instructions.

## Python-specific decisions to resolve first

### Ignore rules

Do not replace Git-style ignore handling with plain filename globs. First port behavioral fixtures for nested rules, negation, directory pruning, anchoring, escaped characters, and sibling isolation.

Prefer standard-library-only runtime code if a small, maintainable matcher can satisfy the required behavior. If doing so would recreate a substantial ignore library, evaluate bundling a focused existing implementation with its license instead. Record the selected approach and limitations before implementing traversal. Runtime package installation and a dependency on the user's Git installation are outside the intended design.

### Regex and time budgets

Use documented Python regex semantics rather than promising exact Go compatibility. Port representative patterns and document differences in Unicode case handling, inline flags, and supported constructs. Do not silently rewrite user expressions.

Python's standard regex engine can spend substantial time inside one match. Checks between lines alone cannot enforce the advertised budget. Prototype process isolation for regex work so the supervising process can enforce a deadline while retaining already received results. Validate process startup, cancellation, worker cleanup, result transfer, and partial CSV finalization before committing to the design. Unlimited searches remain explicitly requested.

Avoid adding a worker protocol to literal searches unless it has a concrete benefit. Keep the public JSON result contract the same for both modes. Document whether the budget covers worker startup and that output finalization can extend elapsed time.

### Text decoding and ordering

Define UTF-8 decoding with replacement for invalid input, preserve matching-line content apart from line terminators, and handle CRLF and a missing final newline. Verify JSON and CSV can represent unusual filenames and decoded text. Record Unicode matching differences instead of claiming byte-for-byte parity.

Use deterministic directory ordering for reproducible results and tests, while checking the deadline during traversal. Do not load entire file contents to search them. JSON results may remain buffered as today; do not add pagination or a new streaming user interface in this migration.

## Migration sequence

Step one is implemented; see [behavior contract and feasibility results](python-migration-step-one.md)
and `tests/migration/README.md` for decisions, checks, and remaining integration work.

Step two is implemented in `skills/sift/scripts/search.py` and `_engine.py`.
The helper bundles unmodified pathspec 1.1.1 with its license and provenance,
supports the preserved behavior contract, and uses a supervised subprocess for
regex searches. CSV destinations are reserved before searching, excluded from
results, and finalized for incomplete searches. Worker failures and output
failures retain available results with an error status. Literal matching uses
single-code-point Unicode folding without multi-character expansions; regex
matching follows Python `re`. Literal deadlines remain cooperative, including
during file reads and traversal. Regex deadlines include worker startup; output
finalization follows cancellation.

The full 32-test migration suite passes on Linux Python 3.11.2 and macOS Python
3.12.3 (the Linux-only device test skips on macOS). Helper integration checks also
pass on macOS Python 3.13. At step two, CI checked the behavior contract against
the Go baseline and ran the Python migration suite. Windows has not been tested.
Step three switches `SKILL.md` to the bundled Python helper and adds the detailed
search-behavior reference. UI metadata now describes the file-search skill rather
than a separate CLI. Normal skill discovery remains enabled. At this stage, Go
code and its build/release tooling were retained for the later cleanup step.

Step four is implemented by `tests/migration/check_portable_skill.py`, also run
in CI. It copies the skill and a separate test harness to temporary directories,
then runs the contract and helper integration suites from an unrelated project
with site packages disabled and an empty executable PATH. Runtime code and test
imports use only those copies, not the checkout. No dependency download or Go
binary is used. Paths contain spaces; temporary fixtures and exports are removed
on exit. The representative searches, regex timeout, and CSV export scenarios
are included in the helper integration checks. This established the basis for
removing the retired Go product and tooling in step five.

Step five removes the tracked Go source/tests, module files, Docker/Compose
configuration, binary packaging scripts/tests, and tag-triggered release workflow.
The Makefile now provides only Python test and portability commands. CI runs the
Python suite and portability check on Python 3.11 and 3.13 without building Go or
installing search dependencies. Contract tests default to the bundled Python
helper; the retired Go argument adapter has been removed. Local `bin/`, `dist/`,
and `coverage.out` artifacts were inventoried and retained under ignore rules.
No installed executable, personal skill, Docker volume, external release, or user
export was modified. Historical prototypes and fixtures remain outside the skill.
The README now identifies the active skill and flags its legacy CLI sections for
replacement in step six; development test instructions have already been updated.

Step six replaces the README and sample QA guide with skill installation,
conversational invocation, Python prerequisites, and current test commands.
Obsolete Go implementation, development, release, and original product plans are
removed. The banner remains project artwork rather than an executable asset.
This transition record and the historical step-one findings are retained. Local
generated artifacts and external installations remain untouched. Documentation
examples and relative links are checked against the completed skill.

1. **Capture the behavior contract.** Convert relevant Go tests and samples into Python behavioral fixtures. Separate preserved behavior from deliberate interface changes. Resolve ignore handling and the regex cancellation prototype. Establish the minimum interpreter and any bundled license requirements.
2. **Implement the private helper.** Add literal search, traversal, ignores, binary/error reporting, structured results, CSV, and time budgets. Add regex with the validated cancellation design. Keep Go intact while verifying the replacement.
3. **Switch the skill.** Replace PATH/binary discovery with an absolute skill-relative helper invocation. Update examples, regex translation, timeout units, runtime diagnostics, references, and UI metadata. Preserve normal skill discovery; skill-only packaging does not imply explicit-only invocation.
4. **Verify a portable installation.** Copy only `skills/sift/` to a temporary directory and invoke its helper from a different project directory. Verify searches and exports require no checkout resources or dependency downloads. Exercise a representative request through the skill instructions.
5. **Remove the retired product.** After replacement checks pass, remove Go source/module files, binary packaging scripts and their tests, Go release workflow, Docker/Compose development setup, and obsolete Make targets. Update CI to Python behavioral tests and a portable-folder smoke check. A small test-only Makefile is optional, not required.
6. **Rewrite repository documentation.** Explain skill-folder installation, Python prerequisite, invocation, capabilities, and development tests. Replace CLI sample commands. Remove obsolete product/release plans and CLI-specific assets after checking their references. Keep this migration plan as the transition record.

Do not remove or modify any previously installed CLI, personal skill symlink, Docker volumes, external release, or user export as part of repository cleanup. Generated `bin/`, `dist/`, and coverage artifacts should be inventoried separately; do not delete them implicitly.

## Completion checks

- Installed skill folder works independently of the checkout and from a working directory containing spaces.
- Literal punctuation and slash-prefixed text stay literal; regex errors are actionable; case and Unicode behavior is covered.
- Hidden files, nested ignores, negation, symlinks, binary files, and unreadable paths follow the documented contract.
- Slow regexes stop under a finite budget; partial matches remain available; workers exit and do not linger.
- CSV handles commas, quotes, Unicode, destination collisions, self-exclusion, empty results, and partial results.
- Search and output errors cannot be mistaken for a complete search with no matches.
- The helper never prompts and never requires an installed `sift` command.
- Python tests and skill validation pass. CI and active documentation no longer require the Go/Docker/binary distribution workflow.

## Out of scope

Fuzzy matching, spelling suggestions, resumable searches, terminal styling, editor URL configuration, a globally installed executable, platform-specific binaries, and automatic runtime setup.
