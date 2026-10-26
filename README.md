# Sift

![Sift — a metal sifter filtering code fragments into matching results](assets/sift-banner.png)

Sift is a small Go CLI for searching text files, with readable results and a planned companion skill for agents.

The project is in early development. It currently supports literal searches in a single file or recursively through a directory.

## Features

- Case-insensitive search by default, with optional exact case matching.
- Results formatted as `line_number:filename matching-line`.
- Recursive directory search that skips hidden entries, symlinks, and non-regular files.
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

Flags can appear before or after positional arguments. Use `--` to stop option parsing when a literal search string or path starts with a hyphen:

```sh
./bin/sift -- '--help' samples/payments.txt
```

Use `-h` or `--help` to display usage. Search strings are literal: punctuation such as `.` has no regex meaning.

### Results and diagnostics

Matches go to stdout. Errors and skipped binary-file messages go to stderr, so they can be redirected independently:

```sh
./bin/sift 'payment failed' samples/tree/ > matches.txt 2> diagnostics.txt
```

Set `NO_COLOR=1` to disable colored diagnostics. Color is also disabled for `TERM=dumb`.

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

Ignore-file support, regex search, search time limits, match highlighting, clickable editor links, CSV/JSON output, and the companion agent skill are not implemented yet. See the [implementation plan](plans/implementation-plan.md).
