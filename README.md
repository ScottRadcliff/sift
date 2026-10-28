# Sift

![Sift — a metal sifter filtering code fragments into matching results](assets/sift-banner.png)

Sift is a small Go CLI for searching text files, with readable results and a planned companion skill for agents.

The project is in early development. It currently supports literal and regex searches in a single file or recursively through a directory.

## Features

- Case-insensitive literal and regex search by default, with optional exact case matching.
- JSON results for agents, including matches, skipped files, errors, and completion status.
- CSV export with filename, line number, and original matching-line text.
- Matching occurrences highlighted yellow in terminals; redirected results stay plain.
- Results formatted as `line_number:filename matching-line`.
- Recursive directory search that skips hidden entries, symlinks, and non-regular files.
- `.gitignore` rules within the search directory, with a `--no-ignore` override.
- Binary-file detection: files containing a null byte in the first 8 KB are skipped and reported on stderr.
- Yellow skipped-file messages in terminals; plain diagnostics when redirected.
- Continued directory searching after unreadable files, with an error exit status.
- Sample files and automated tests for development and QA.

## Build locally

The Docker workflow requires Docker with Docker Compose v2 and Make. The Makefile supports macOS and Linux hosts; Go does not need to be installed on your host. Start Docker before running these commands.

From the repository root:

```sh
make build
make binary
./bin/sift --help
```

`make binary` writes `bin/sift` for your host OS and architecture, even when Docker runs Linux. Once built, the executable runs directly on your computer without Docker.

There is no installer or release package yet. Rebuild with `make binary` after changing the code.

## Usage

Search a single file:

```sh
./bin/sift 'payment failed' samples/payments.txt
```

Output:

```text
2:samples/payments.txt Payment failed for order 1002.
4:samples/payments.txt PAYMENT FAILED for order 1003.
```

Search nested directories:

```sh
./bin/sift 'payment failed' samples/tree/
```

Match case exactly:

```sh
./bin/sift 'Payment' samples/payments.txt --case-sensitive
```

Search ignored paths too:

```sh
./bin/sift 'payment' samples/ignore/ --no-ignore
```

Directory searches read `.gitignore` files in the requested directory and its descendants. Ancestor ignore files, global Git settings, and `.git/info/exclude` are not read. Explicitly named files are searched regardless of ignore rules. `--no-ignore` does not enable hidden files or binary files.

Flags can appear before or after positional arguments. Use `--` to stop option parsing when a literal search string or path starts with a hyphen:

```sh
./bin/sift -- '--help' samples/payments.txt
```

Use `-h` or `--help` to display usage. Search strings are literal unless they start with `/`: punctuation such as `.` has no regex meaning in a literal search.

### Regex search

Wrap an expression in slashes:

```sh
./bin/sift '/payment|retry/' samples/payments.txt
./bin/sift '/^payment failed/i' samples/payments.txt
./bin/sift '/Payment/' samples/payments.txt --case-sensitive
```

Both `/pattern/` and `/pattern/i` are case-insensitive by default. Use `/pattern/ --case-sensitive` for exact case; combining `/pattern/i` with that flag returns an error. Only the trailing `i` modifier is supported. Go's inline regex flags are also honored.

Regexes use [Go's regex syntax](https://pkg.go.dev/regexp/syntax), with matching performed independently on each line. Lookaround and backreferences are unsupported. Patterns that match empty text select matching lines but have no visible highlight for the zero-length match. Empty `//` expressions are rejected.

A search starting with `/` opts into regex parsing. To search for a literal slash-containing path, use an expression such as `'/\/etc\/hosts/'`. Quote expressions in your shell to preserve backslashes and special characters. Invalid expressions or modifiers produce an error with exit status `2` before searching any files.

### CSV export

```sh
./bin/sift 'payment' samples/export/ --csv matches.csv
```

Terminal results still appear, with highlighting when supported. The CSV contains a header and one row per matching line, using columns `filename`, `line_number`, and `text`. Values preserve the original case and contain no added terminal color codes; commas and quotes are escaped as CSV.

The output file must not already exist. Choose a new filename for each export. If the export is inside the searched directory, Sift excludes it from the search. No matches produce a header-only CSV and exit status `1`. Search errors retain any partial export and return status `2`; check stderr before treating an export as complete.

### JSON output

```sh
./bin/sift 'payment failed' samples/payments.txt --json
```

Stdout contains one JSON object:

```json
{
  "matches": [
    {"filename": "samples/payments.txt", "line_number": 2, "text": "Payment failed for order 1002."},
    {"filename": "samples/payments.txt", "line_number": 4, "text": "PAYMENT FAILED for order 1003."}
  ],
  "skipped_files": [],
  "errors": [],
  "complete": true
}
```

Matches preserve original text and contain no added terminal formatting. Arrays are empty arrays rather than `null` when there are no entries. `skipped_files` records binary files with reason `binary`, and paths that could not be searched with reason `search_error` and an `error` detail. Hidden entries and ignored paths are intentionally excluded and are not listed.

Search errors return partial results with `complete: false` and error messages. A completed search can have no matches (exit status `1`) or binary skips. Exit statuses remain `0`, `1`, and `2`; diagnostics also go to stderr. Argument/regex validation failures produce only stderr and status `2`, before a search result exists; help requests still print help text.

Combine `--json` and `--csv filename` to write JSON to stdout and export the same matches to CSV. JSON results are collected in memory and written after the search finishes.

### Results and diagnostics

Matches go to stdout (as a JSON object when `--json` is used). Errors and skipped binary-file messages go to stderr, so they can be redirected independently:

```sh
./bin/sift 'payment failed' samples/tree/ > matches.txt 2> diagnostics.txt
```

Set `NO_COLOR=1` to disable color for both matches and diagnostics. Color is also disabled for `TERM=dumb`.

| Exit status | Meaning |
| --- | --- |
| `0` | At least one matching line, with no search errors |
| `1` | No matching lines, with no search errors |
| `2` | Invalid arguments or a search/output error, including partial directory results |

Skipping a binary file is not an error. A search containing only skipped binary files returns `1`.

Hidden entries are skipped during traversal, but an explicitly named hidden file or directory can be searched. The binary check is a prefix heuristic; null bytes after the first 8 KB are not detected. UTF-16 files commonly contain null bytes and are skipped; encoding conversion is not supported.

## Development

Run the CLI inside Docker against files in the repository:

```sh
make run ARGS="'payment failed' samples/payments.txt"
```

The repository is mounted at `/workspace`. Use the host executable for paths elsewhere on your computer.

```sh
make fmt
make vet
make test
```

These commands use disposable containers and do not require a running workspace container. Go dependency and build caches persist in named volumes.

For a persistent workspace, run `make up` in one terminal and `make shell` in another. `make down` removes the workspace container and network while retaining caches. No database, ports, or environment-file setup is required.

Run `make help` for all targets. See the [development guide](plans/development.md) for build overrides and the [sample QA guide](samples/README.md) for expected results.

## Planned work

Search time limits, clickable editor links, and the companion agent skill are not implemented yet. See the [implementation plan](plans/implementation-plan.md).
