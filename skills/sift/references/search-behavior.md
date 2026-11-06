# Search behavior

Read this reference when interpreting scope exclusions, regex or Unicode behavior, partial results, and exports. The installed helper is `../scripts/search.py`, resolved relative to this file. It provides structured results without terminal formatting or editor URL handlers.

## Scope and traversal

- A directory is traversed in deterministic filename order. Hidden descendants, symlinks, and non-regular files are excluded.
- An explicitly named hidden file or directory is searchable. Hidden descendants of an explicit directory remain excluded.
- An explicit file bypasses ignore rules. A symlink explicitly naming a regular file is followed; an explicit directory symlink returns a search error.
- `.gitignore` files apply within the selected root and descendants, with nested rules scoped to their own directories. Ancestor rules, global Git configuration, and `.git/info/exclude` are not read.
- Ignored directories are pruned; a negation inside an excluded directory cannot restore its contents. `--no-ignore` disables ignore handling, without enabling hidden or binary files.
- Unreadable files and ignore files produce search errors while traversal continues where possible. The result remains incomplete.
- A null byte within the first 8 KB identifies a binary file and produces a `binary` skip. Null bytes later in the file are not detected by this heuristic. UTF-16 files are commonly skipped.

## Matching and text

Searches select lines rather than individual occurrences. A line appears once even if it contains several matches. Records include original text and one-based line numbers, without match spans or column positions. Newline and trailing carriage-return terminators are removed; other text is preserved. The helper reads UTF-8 with replacement for invalid byte sequences and handles a final line without a newline.

Literal text is never interpreted as a regex, including `/etc/hosts` or `/pattern/i`. Case-insensitive literals use single-code-point Unicode folding: `ß` can match `ẞ`, but does not expand into `ss`; `İ` and `ı` do not become ASCII `i`. Unicode mappings track the Python runtime's Unicode tables.

`--regex` uses Python `re`, evaluated independently on each line. Pass the expression itself, without surrounding slashes. Case-insensitive matching is the default; `--case-sensitive` removes that default, while explicit inline flags still have their normal Python meaning. Lookaround and backreferences are supported. Python case-insensitive regex ranges can include additional Unicode characters: `[a-z]` also matches `İ`, `ı`, `ſ`, and `K`. Patterns matching empty text select applicable lines but produce no extra occurrence records. The search term must be nonempty in either mode.

## Results and exit status

The helper writes one JSON object after searching:

```json
{
  "matches": [{"filename": "app/payments.txt", "line_number": 2, "text": "Payment failed"}],
  "skipped_files": [],
  "errors": [],
  "timed_out": false,
  "complete": true
}
```

All three list fields are arrays, including when empty. Filenames can be relative to the invocation's working directory or absolute, depending on the supplied search path. JSON escapes are representation details; parse JSON before presenting paths or matching text.

Skip records have `filename` and `reason`: `binary` or `search_error`. Search errors also include an `error` string. Hidden and ignored paths are excluded without skip records. `complete: true` therefore means completion within the selected scope and skip policy, not that every filesystem entry was searched.

| Status | Meaning |
| --- | --- |
| 0 | Matching lines, with no errors or timeout |
| 1 | No matching lines, with no errors or timeout; binary-only searches also return 1 |
| 2 | Validation, search, worker, or output error; results may be partial |
| 3 | Search budget expired; results may be partial |

Output failures take precedence over timeout, so inspect `timed_out` even when status is 2. Argument and regex validation occur before export creation and return status 2 with stderr and no JSON. A failed JSON write may leave missing or truncated stdout. `--help` prints ordinary usage text; there is no `--version` contract.

## Budgets and temporary storage

`--timeout` takes finite nonnegative seconds. The default is 3; 0 is unlimited. Literal searches check their deadline cooperatively between reads, lines, and traversal operations; an operation in progress can exceed the budget.

Regex searches run in a supervised subprocess. The budget includes worker startup; the supervisor stops and reaps a worker that exceeds it, retaining completed match records. A temporary directory holds worker records and diagnostics, is excluded from traversal, and is cleaned up on exit. This uses temporary disk proportional to emitted records. Unexpected worker exits are errors, not completed empty searches.

JSON output, worker cleanup, and CSV finalization occur after searching or cancellation, so elapsed wall time can exceed the configured search budget. Results are buffered in memory; choose a relevant subtree rather than assuming unbounded output is inexpensive. Searches do not resume or suggest spellings.

## CSV

`--csv FILE` reserves a new destination before searching and excludes it from results. The parent directory must exist. A destination collision returns an error before searching and leaves the existing file untouched.

CSV uses UTF-8 and the columns `filename`, `line_number`, `text`, with standard quoting for commas, quotes, and newlines. No matches produce a header-only file. Search errors and timeouts preserve available rows in a partial export; output failures can also leave a partial file. Check status and completion fields before describing an export as complete.
