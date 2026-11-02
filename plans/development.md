# Development

Install Docker with Docker Compose and Make. Host Go is not required.

```sh
make build
make run
make binary
./bin/sift --help
```

Sift currently searches a regular text file or recursively searches a directory for a case-insensitive literal string or slash-delimited regex, with optional `--case-sensitive` matching. See [sample searches](../samples/README.md) for manual QA. Searches default to a three-second budget; `--timeout 0` disables it. Editor links use the VS Code URL handler by default; set `SIFT_EDITOR_URL` to choose another handler. Interactive terminal timeouts offer an unlimited restart; use `--no-interactive` for scripts.

Use `make fmt`, `make vet`, and `make test` for checks. Tests cover literal matching, case handling, line endings, long lines, exit statuses, and I/O errors.

Commands use disposable containers and do not require `make up`. Module and build caches persist in named Docker volumes. On Linux, commands use your host user ID and group ID so generated source and binaries remain editable.

`make run ARGS="'payment failed' samples/payments.txt"` passes arguments to Sift inside Docker. Only mounted paths are visible there; the repository is mounted at `/workspace`. Use the host executable for host paths and terminal editor links.

`make binary` builds for the host OS and architecture, including macOS, rather than Docker's Linux platform. Override with `HOST_OS=linux HOST_ARCH=arm64`, for example. Use `make install` to build and install into `~/.local/bin`, or override `INSTALL_DIR`. Use `make release VERSION=v1.0.0-rc.1` to package downloadable binaries locally; see [the release guide](releases.md).

For an optional persistent workspace, run `make up` in one terminal, then `make shell` in another. Go has no standard REPL, so there is no console target. `make down` removes containers and networks while retaining caches. No ports, database, or other services are needed.

The image and module pin Go 1.26.9; update them together. The temporary module name is `sift`; replace it with the repository's published module path before distribution.

## Versioned development builds

`make binary` defaults to the version label `dev`. Use `make binary VERSION=v0.1.0-local` to embed a different single-token label through Go linker flags, then check it with `./bin/sift --version`. `make run ARGS=--version VERSION=v0.1.0-local` supports the same override. Local installation is available through `make install VERSION=...`; downloadable binaries can be prepared with `make release VERSION=...`.

## Verify installation in isolation

```sh
install_qa=$(mktemp -d)
make install INSTALL_DIR="$install_qa/bin" VERSION=v0.1.0-install-qa
"$install_qa/bin/sift" --version
"$install_qa/bin/sift" 'payment failed' samples/payments.txt
```

Expect `sift v0.1.0-install-qa` and two matching lines. The target creates the destination directory, sets mode 0755, and replaces an existing executable on subsequent installations. It does not edit PATH or install the agent skill. Run `make binary` afterward to restore the checkout binary's default `dev` label.
