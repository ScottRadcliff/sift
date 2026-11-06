# Step one: behavior contract and feasibility

Historical record of the first migration step. The migration is now complete;
see the [completed transition record](python-skill-migration.md) for the final state.

## Completed

Added reusable JSON behavior fixtures and a Python unittest adapter for the
current Go engine and the future Python helper. Rebuilt the Go binary from the
current source before evaluating it. The fixtures cover literal punctuation,
repeated occurrences, case, Unicode, line endings, common regex patterns, empty
results, nested ignore rules, escaped patterns, directory-only rules, negation,
pruning, sibling isolation, explicit scope, and binary skips.

Additional tests cover CSV quoting/collisions/self-exclusion, empty and partial
exports, missing paths, unreadable files and ignore files, symlinks, validation
errors, and the 8 KB binary-prefix boundary. The adapter requires an explicit
command so an old installed binary cannot accidentally become the baseline.

## Ignore decision

Bundle the unmodified Python source of **pathspec 1.1.1**, including its MPL-2.0
license and attribution, inside the skill during step two. Use `GitIgnoreSpec`
with the pure Python `simple` backend, with rules evaluated relative to their
own directories. Later directory scopes override earlier decisions only when
they actually match. Prune ignored directories before visiting descendants.

This avoids maintaining a second implementation of Git-style pattern parsing.
The scoped traversal prototype passes the relevant behavior fixtures, including
negation, anchoring, escapes, and sibling isolation. This is evidence for the
selected contract, not a claim of exhaustive Git compatibility.

The evaluation downloaded a wheel to `/tmp` and imported it directly; nothing
was installed globally or added to the skill yet. The wheel has no required
third-party dependencies and declares Python >=3.9. Its optional native matching
backends are unnecessary. Keep vendor code unmodified and retain source/license
information; verify distribution contents when vendoring rather than copying
only fragments with undocumented provenance.

Evaluation artifact:

```text
pathspec-1.1.1-py3-none-any.whl
SHA-256: a00ce642f577bf7f473932318056212bc4f8bfdf53128c78bbd5af0b9b20b189
```

Upstream sources: [documentation and licensing](https://python-path-specification.readthedocs.io/en/latest/readme.html),
[source repository](https://github.com/cpburnz/python-pathspec).

## Regex and timeout decision

Use Python `re` semantics with explicit regex mode. A separate process executes
regex searches under a finite deadline. The prototype starts the budget before
process creation, includes startup in that budget, kills a timed-out worker, and
waits for it before returning. Zero remains unlimited. Literal matching should
remain a direct, cooperative traversal unless integration demonstrates a need
for isolation there as well.

The worker flushes each match as a JSONL record to a temporary spool outside the
search tree. The parent reads complete records after worker exit and discards an
unfinished final record on timeout. Unlike reading a pipe directly, this cannot
block the parent halfway through receiving a large worker message. Completed
records survive a stalled later match. The parent then finalizes CSV, including
partial output. Export errors take precedence over timeout in the exit status.

The tests demonstrate that `^(a+)+$` stops within a one-second budget plus process
cleanup allowance, keeps an earlier match, finalizes CSV, and leaves the child
reaped. They also cover cancellation during startup, unlimited search, worker
failure, argument validation, export collision, and Python-specific patterns.

This is a single-file feasibility prototype, not the production worker protocol.
During step two, integrate skip/error events, exclusive export reservation before
traversal, export self-exclusion, and errors during output finalization. Spooling
uses temporary disk proportional to matches, in addition to the existing buffered
JSON result; surface disk errors and clean up files on all exit paths. Deadline
cancellation is distinct from arbitrary process crashes: an unexpected worker
exit must remain an error with incomplete results.

## Explicit compatibility decisions

- Replace slash-delimited patterns with `--regex`; slash-prefixed literal text
  no longer requires regex escaping. Replace Go duration strings with seconds.
- Python lookaround and backreferences are supported. Python inline flag syntax
  applies. Some patterns accepted by Go will require user correction; return the
  diagnostic rather than translating them silently.
- Python case-insensitive regex ranges have Unicode behavior that differs from
  Go. Tests explicitly capture the special matches for İ, ı, ſ, and K in `[a-z]`.
  Define literal case matching separately during implementation; do not assume
  unrestricted Unicode `casefold()` preserves the old behavior.
- UTF-8 decoding with replacement is the proposed Python text contract. The
  prototype strips line terminators and retains final unterminated lines.
- Preserve explicit symlinks to regular files, which the Go engine follows.
  Recursive traversal skips symlinks. Explicit symlinks to directories should
  return a clear search error in Python: the Go traversal does not descend into
  them and can misleadingly produce a complete empty result.

## Validation and remaining boundaries

Run commands are in `tests/migration/README.md`. Validation was performed on
macOS with Python 3.12.3 and a freshly rebuilt host-native Go binary. The full
evaluation includes the optional ignore-library test; running without the
temporary wheel reports that test as skipped.

Python 3.11 is the supported floor; testing that interpreter and Windows/Linux
process behavior remains required during implementation. The current skill,
Go source, development setup, and release workflows remain active and unchanged.
Proceed next to step two: implementing the private helper against these checks.
