# Sift

> Sift now runs as a self-contained [Python skill](skills/sift/SKILL.md), requiring Python 3.11+. Use `make test` and `make portable` for development checks. The Go CLI and Docker/release tooling have been removed; the legacy CLI documentation below is pending replacement in the final migration step.

![Sift — a metal sifter filtering code fragments into matching results](assets/sift-banner.png)

Sift is a small Go CLI for searching text files, with readable results and a companion skill for agents.

The project is in early development. It currently supports literal and regex searches in a single file or recursively through a directory.

## Contents

- [Features](#features)
- [Build locally](#build-locally)
- [Install locally](#install-locally)
- [Build version](#build-version)
- [Usage](#usage)
- [Regex search](#regex-search)
- [Clickable editor links](#clickable-editor-links)
- [CSV export](#csv-export)
- [JSON output](#json-output)
- [Search time budget](#search-time-budget)
- [Results and diagnostics](#results-and-diagnostics)
- [Development](#development)
- [Continuous integration](#continuous-integration)
- [Release packaging](#release-packaging)
- [Agent skill](#agent-skill)
- [Planned work](#planned-work)

## Features

- Case-insensitive literal and regex search by default, with optional exact case matching.
- Interactive timeout continuation with an explicit `--no-interactive` option.
- Three-second search budget, configurable with `--timeout` (use `0` for unlimited).
- JSON results for agents, including matches, skipped files, errors, and completion status.
- CSV export with filename, line number, and original matching-line text.
- Matching occurrences highlighted yellow in terminals; redirected results stay plain.
- Clickable file locations that open the first match in a configurable editor.
- Results formatted as `line_number:filename matching-line`.
- Recursive directory search that skips hidden entries, symlinks, and non-regular files.
- `.gitignore` rules within the search directory, with a `--no-ignore` override.
- Binary-file detection: files containing a null byte in the first 8 KB are skipped and reported on stderr.
- Yellow skipped-file messages in terminals; plain diagnostics when redirected.
- Continued directory searching after unreadable files, with an error exit status.
- Companion agent skill for conversational searches and exports.
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

Rebuild with `make binary` after changing the code. A tag-triggered workflow prepares downloadable packages as a draft release; see [release packaging](#release-packaging). No release is published automatically.

### Install locally

```sh
make install
```

This builds a host-native binary using Docker, creates `~/.local/bin` if needed, and installs `sift` there with executable permissions. An existing `sift` in that destination is replaced, allowing updates. The installed binary runs without Docker or Go.

If that directory is not on your PATH:

```sh
export PATH="$HOME/.local/bin:$PATH"
sift --version
```

Add the PATH setting to your shell configuration to keep it across terminal sessions. Use a version label or a different destination when needed:

```sh
make install VERSION=v0.1.0-local
make install INSTALL_DIR="$HOME/bin"
```

Installation defaults to the `dev` version label. `make install` installs only the CLI; the agent skill is installed separately.

### Build version

```sh
./bin/sift --version
```

Development builds print `sift dev`. Supply a version label when building:

```sh
make binary VERSION=v0.1.0-local
./bin/sift --version
```

This prints `sift v0.1.0-local`. The Makefile passes the label through Go linker flags (`-X main.version=...`); plain `go build` and builds without an override use `dev`. Version labels should be single tokens, such as semantic versions or versions with a commit suffix. `make run ARGS=--version VERSION=v0.1.0-local` uses the same metadata.

`--version` exits successfully without searching files. Like `--help`, it produces plain text even alongside `--json`; help takes precedence when both are specified. Unknown options still produce an error.

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

### Clickable editor links

File locations are clickable in terminals that support [OSC 8 hyperlinks](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda):

```sh
./bin/sift 'payment' 'samples/links/match locations.txt'
```

The default target is VS Code, using its [file, line, and column URL handler](https://code.visualstudio.com/docs/configure/command-line#_opening-vs-code-with-urls). Click the `line_number:filename` portion (your terminal may require Cmd-click or Ctrl-click). It opens the first match on that line. Matching text remains highlighted as before.

Choose a different registered editor URL handler with `SIFT_EDITOR_URL`:

```sh
SIFT_EDITOR_URL='vscode-insiders://file{path}:{line}:{column}' ./bin/sift 'payment' samples/payments.txt
```

Templates require `{path}` and may include `{line}` and `{column}`. Sift substitutes an absolute, URL-encoded path, a one-based line number, and a one-based UTF-16 column; tabs count as one character and non-BMP characters count as two. No shell command is executed by Sift. The chosen editor must be installed and registered to handle that URL scheme.

To open the system's default file application, use `SIFT_EDITOR_URL='file://{path}'`. Standard file URLs do not reliably carry cursor positions, so this option opens the file without promising a line/column jump. `$EDITOR` and `$VISUAL` are not automatically executed when a link is clicked.

`--hyperlinks auto` is the default: links are emitted only when stdout is a terminal and `TERM` is not `dumb`. Redirected output stays plain. Use `--hyperlinks never` to disable links, or `--hyperlinks always` to force them when capturing/testing terminal sequences. Auto mode does not guarantee the terminal supports OSC 8; unsupported terminals may show only the readable location. JSON and CSV contain no hyperlink sequences, even with `--hyperlinks always`. `NO_COLOR` controls color independently and does not disable links.

Use the host binary to test editor links; paths inside Docker refer to the container's filesystem.

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
  "timed_out": false,
  "complete": true
}
```

Matches preserve original text and contain no added terminal formatting. Arrays are empty arrays rather than `null` when there are no entries. `skipped_files` records binary files with reason `binary`, and paths that could not be searched with reason `search_error` and an `error` detail. Hidden entries and ignored paths are intentionally excluded and are not listed.

Search errors return partial results with `complete: false` and error messages. A completed search can have no matches (exit status `1`) or binary skips. Timeouts return `timed_out: true`, `complete: false`, and status `3`. Other exit statuses remain `0`, `1`, and `2`; diagnostics also go to stderr. Argument/regex validation failures produce only stderr and status `2`, before a search result exists; help requests still print help text.

Combine `--json` and `--csv filename` to write JSON to stdout and export the same matches to CSV. JSON results are collected in memory and written after the search finishes.

### Search time budget

Searches default to a three-second budget after argument validation. Override it with a Go-style duration, or use `0` for an unlimited search:

```sh
./bin/sift 'payment' . --timeout 10s
./bin/sift 'payment' . --timeout 0
```

When the budget expires, Sift retains matches already emitted, flushes any partial CSV export, and returns status `3`. JSON includes `complete: false` and `timed_out: true`, plus an explanatory error. Output-write failures return status `2` even if a timeout also occurred. CSV/JSON finalization runs after search cancellation so partial results can be saved.

Checks occur between directory entries, buffered reads, and line matches. This is a cooperative search budget, not a hard wall-clock limit: an OS operation or regex match already in progress, and output finalization, can extend elapsed time beyond it. When stdin, stdout, and stderr are terminals (and `TERM` is not `dumb`), Sift asks whether to restart a timed-out search without a time limit. Enter `y` or `yes` to accept; Enter, `n`, or EOF declines. The prompt explains that a restart begins from the first file, so terminal matches may repeat. After accepting, the exit status reflects the restarted search; declining retains status `3`.

JSON output, redirected or piped streams, and `--no-interactive` never prompt:

```sh
./bin/sift 'payment' . --timeout 10s --no-interactive
```

For interactive CSV searches, accepting preserves the partial export and creates a new sibling such as `matches.continued.csv`, then `matches.continued-2.csv` if that name exists. Both the partial export and the new export are excluded from the restarted search. Sift reports both paths and never overwrites existing files. Manual reruns still need a new CSV filename.

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
| `3` | Search time limit exceeded; results may be partial |

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

## Continuous integration

[Go CI](.github/workflows/ci.yml) runs on pushes and pull requests and can also be started manually from GitHub Actions. It uses the Go version in `go.mod` and caches dependencies and build output.

The Linux job checks formatting, verifies downloaded modules, runs `go vet`, runs tests with race detection and coverage, and builds the CLI. Formatting is checked without modifying source files. Coverage is reported without enforcing a minimum percentage.

Local development remains Docker-based. To run the CI test command locally:

```sh
docker compose run --rm dev go test -race -count=1 -timeout=60s -coverprofile=coverage.out ./...
```

## Release packaging

[Release](.github/workflows/release.yml) runs when a version tag such as `v1.0.0` is pushed. It tests the project, cross-compiles five targets, generates SHA-256 checksums, and attaches the packages to a **draft** GitHub release for review. Prerelease tags such as `v1.0.0-rc.1` are supported.

| Platform | Archive suffix |
| --- | --- |
| macOS Intel | `darwin_amd64.tar.gz` |
| macOS Apple Silicon | `darwin_arm64.tar.gz` |
| Linux Intel/AMD | `linux_amd64.tar.gz` |
| Linux ARM64 | `linux_arm64.tar.gz` |
| Windows x64 | `windows_amd64.zip` |

Each archive is named `sift_VERSION_PLATFORM_ARCH` with the suffix above. It contains `sift` (or `sift.exe`), this README and its banner, and `skills/sift/` for installing the companion skill separately. The version tag is embedded in the executable. Checksums are listed in `checksums.txt`.

To build packages locally without creating a tag or contacting GitHub:

```sh
make release VERSION=v1.0.0-rc.1
```

Packages appear under `dist/v1.0.0-rc.1/`. The output directory must be new; use `DIST_DIR` to select another location. Docker provides Go and Python 3; the packaging helper uses only Python's standard library. Native packaging with Go and Python 3 is also supported through `python3 scripts/package_release.py --version v1.0.0-rc.1 --output dist/local-review`.

Once a reviewed release has been published, download the archive for your platform and `checksums.txt` from GitHub Releases. Compare the archive's SHA-256 digest with its entry in `checksums.txt` (`shasum -a 256 ARCHIVE` on macOS, `sha256sum ARCHIVE` on Linux, or `Get-FileHash ARCHIVE -Algorithm SHA256` in PowerShell).

For example, after downloading a macOS Apple Silicon release into an empty directory:

```sh
tar -xzf sift_v1.0.0_darwin_arm64.tar.gz
mkdir -p "$HOME/.local/bin"
install -m 0755 sift "$HOME/.local/bin/sift"
"$HOME/.local/bin/sift" --version
```

On Windows, extract the ZIP with `Expand-Archive` and run `sift.exe`; add its directory to PATH if desired. Prebuilt binaries require neither Go nor Docker. The current workflow cross-compiles all targets and smoke-tests the build platform's matching executable; it does not run the full suite on every target OS. Builds are not code-signed or notarized.

See the [release guide](plans/releases.md) for local QA and the tag/draft review process.

## Agent skill

The self-contained skill lives in [skills/sift/SKILL.md](skills/sift/SKILL.md). It uses a bundled Python helper to search files, interpret JSON results, report skipped files and incomplete searches, and export CSV. Python 3.11 or newer is required; the skill does not use the Go CLI documented above. Migration progress is tracked in the [Python skill migration plan](plans/python-skill-migration.md).

To try the repository copy directly, tell the agent:

> Use the Sift skill at `skills/sift/SKILL.md` to find payment failures in `samples/`.

For Codex discovery across projects, link the skill into the documented personal skills location. From the repository root:

```sh
mkdir -p "$HOME/.agents/skills"
ln -s "$PWD/skills/sift" "$HOME/.agents/skills/sift"
```

The link command refuses to replace an existing skill. Keep the checkout at that path. Codex supports symlinked skill folders; see the [official skill discovery documentation](https://learn.chatgpt.com/docs/build-skills). If the new skill does not appear, restart Codex.

Keep the entire skill folder, including `scripts/`, its bundled dependency, and `references/`. It can also be copied into a supported skills directory independently of this checkout. No separate executable installation, Go build, Docker, or pip setup is needed for skill searches.

Once discovered, invoke it naturally or explicitly:

> Use $sift to find payment failures in this project and export them to a new CSV.

## Planned work

Spelling suggestions remain deferred beyond v1. See the [implementation plan](plans/implementation-plan.md).
