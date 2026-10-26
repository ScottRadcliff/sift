# Sift implementation plan

## Direction

Build Sift as a Go command-line executable with a companion agent skill. Develop, format, test, and compile Go inside Docker; do not require Go on the host. Docker itself and Docker Compose are host prerequisites.

The executable owns file traversal, matching, output, exports, and exit statuses. The skill translates natural-language requests into CLI arguments, interprets results, and asks follow-up questions. Both live in this repository and use the same search implementation.

## Development environment

Adapt the dev-server-setup skill to a command-line application:

- Add a development Dockerfile based on a supported official Go image. Select and pin the Go version during implementation, recording the same version in go.mod.
- Add compose.yaml with one service, dev, and no published ports or database. Bind-mount the repository at /workspace. Use named volumes for the Go module and build caches.
- Run commands as an appropriate user so generated files remain editable on the host. Avoid requiring privileged containers or mounting the Docker socket.
- Provide .dockerignore and a Makefile with help as the default target.
- Use disposable containers for build, test, formatting, and CLI commands. These must work without a running service.
- Offer an optional foreground development container through make up, allowing make shell to use compose exec. Describe it as a workspace container, not an application server.
- Go has no standard REPL. Omit the skill's generic console target and document make shell and make run instead of adding an unnecessary REPL dependency.
- Keep down non-destructive: preserve named caches. Lifecycle targets can include stop, restart, logs, and ps for the optional workspace container.

Planned commands:

| Command | Purpose |
| --- | --- |
| make build | Build the development Docker image |
| make test | Run go test ./... in a disposable container |
| make fmt | Format Go source in a disposable container |
| make vet | Run go vet ./... in a disposable container |
| make run ARGS="'payment failed' /workspace" | Run Sift against container-visible files |
| make binary | Compile a host-compatible executable into bin/sift |
| make install | Copy the host executable to ~/.local/bin, creating that directory if needed |
| make up | Start the optional workspace container with attached foreground logs |
| make shell | Open a shell in the running workspace container |
| make down | Remove containers and networks while retaining caches |

Build the host executable with an explicit target OS and architecture. Detect common host combinations and permit overrides. Use CGO_ENABLED=0 if the selected dependencies support it. In particular, Docker on macOS builds a Linux executable by default, so a plain container go build is insufficient for host installation.

Container searches see container paths and mounted files only. Validate real editor links and host filesystem behavior with bin/sift on the host. Do not mount the whole home directory just to support searches during development.

## Initial CLI contract

Examples:

```sh
sift 'payment failed' app/
sift 'Payment' app/ --case-sensitive
sift '/payment|refund/i' app/
sift 'payment' app/ --csv results.csv
sift 'payment' app/ --json --no-interactive
```

- Literal matching by default; case insensitive unless explicitly overridden.
- Define supported regex syntax and modifiers around Go's regex engine. Document unsupported constructs and flag precedence. Invalid expressions produce an actionable error; never silently change the expression.
- Search directories recursively. Respect ignore files by default, skip hidden files by default, and do not follow directory symlinks. Provide explicit override flags where needed.
- Define binary detection and report skipped binary and unreadable files separately from matches.
- Display matches as line_number:filename matching-line, with highlighted match spans when color is supported. Store column positions for precise navigation.
- Generate terminal hyperlinks when supported, using configurable editor URL templates. Provide readable locations when links are unavailable.
- Emit stable JSON records for agents, including path, line, column, text, match spans, skipped files, and completion status. Keep diagnostics separate from machine-readable output.
- Export CSV with a documented schema and proper quoting. Never silently overwrite an existing export; require an explicit overwrite option.
- Document exit statuses for matches, no matches, errors, and incomplete searches.

## Long searches and interaction

Use three seconds as an elapsed search budget rather than trying to predict directory size. Return partial results and a clear incomplete status when the budget is reached.

When attached to an interactive terminal, offer to continue. When invoked by an agent or pipeline, return immediately without prompting. Provide explicit timeout and unlimited-search options so the agent can repeat the search with user authorization. Clearly indicate that a repeated search starts over in the first version; resumable searches are out of scope.

Do not add fuzzy matching to the first version. Later, offer spelling suggestions after searches with no exact matches, with an explicit cost and scope limit.

## Repository layout

```text
cmd/sift/             CLI entry point and argument parsing
internal/search/     Traversal, matching, cancellation, result types
internal/output/     Terminal, JSON, CSV, editor links
skills/sift/SKILL.md  Agent usage and result interpretation
samples/             Committed sample files for development and manual QA
plans/               Product notes and implementation plan
Dockerfile.dev
compose.yaml
.dockerignore
Makefile
go.mod
README.md
```

Choose a small, maintained argument parser that accepts flags after positional arguments, as shown in the examples. Review ignore-file handling dependencies before implementing traversal. Keep the executable self-contained rather than requiring an installed rg command.

## Implementation sequence

1. Scaffold the Go module and minimal CLI; add Docker, Compose, Makefile, and development instructions. Verify a fresh checkout can build and run without host Go.
2. Add a committed samples/ directory with documented searches and expected results for development and manual QA. Include nested directories, mixed-case and Unicode text, repeated matches, regex examples, CSV-sensitive text, hidden and ignored files, a small binary file, and files with spaces in their names. Generate large directory trees and permission-sensitive cases on demand rather than committing bulky or platform-dependent fixtures. Implement literal search, recursive traversal, case handling, skip reporting, and cancellation. Test against temporary fixture trees.
3. Add regex parsing, terminal formatting, editor links, JSON, CSV, and explicit interaction controls.
4. Add the companion skill with examples, machine-output instructions, error handling, and long-search follow-up behavior.
5. Add host installation and release documentation. Later automate versioned macOS, Linux, and Windows binaries and publish a Homebrew formula. Check the Sift name for distribution conflicts before publishing.

## Verification

Validate Compose configuration and Make recipes, build the image, and run the tests and vet in Docker. Cover meaningful behavior: literal metacharacters, regex errors, Unicode case behavior, ignored paths, binary/unreadable files, CSV quoting, machine-output validity, cancellation, and noninteractive execution without prompts.

Run the compiled executable on the host against known files. Manually inspect highlighting and an editor link in a supported terminal. Verify installation to an isolated temporary destination before recommending the default install path. Document any checks blocked by unavailable Docker, network access, or host platform.

Use samples/ for repeatable manual QA in both Docker and the host terminal. Include samples/README.md with commands and expected match counts, skipped-file reports, and export behavior. Keep samples synthetic and free of secrets or personal data; write generated exports and stress-test files to a temporary directory.

## Installation model

The executable and skill are separate installations. Initially, developers use make build, make binary, and make install, then add ~/.local/bin to PATH if needed. The host executable requires no Go installation.

Install skills/sift into the chosen agent's supported skill directory. The skill expects sift on PATH and should provide a clear installation hint if it is missing. Document agent-specific installation when implementing the skill rather than assuming one directory works for every agent.
