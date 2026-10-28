# Development

Install Docker with Docker Compose and Make. Host Go is not required.

```sh
make build
make run
make binary
./bin/sift --help
```

Sift currently searches a regular text file or recursively searches a directory for a case-insensitive literal string, with optional `--case-sensitive` matching. See [sample searches](../samples/README.md) for manual QA. Regex syntax, search time limits, highlighting, and editor links are planned for later steps.

Use `make fmt`, `make vet`, and `make test` for checks. Tests cover literal matching, case handling, line endings, long lines, exit statuses, and I/O errors.

Commands use disposable containers and do not require `make up`. Module and build caches persist in named Docker volumes. On Linux, commands use your host user ID and group ID so generated source and binaries remain editable.

`make run ARGS="'payment failed' samples/payments.txt"` passes arguments to Sift inside Docker. Only mounted paths are visible there; the repository is mounted at `/workspace`. Use the host executable for host paths and, once implemented, terminal editor links.

`make binary` builds for the host OS and architecture, including macOS, rather than Docker's Linux platform. Override with `HOST_OS=linux HOST_ARCH=arm64`, for example. Installation and release packaging will come in a later step.

For an optional persistent workspace, run `make up` in one terminal, then `make shell` in another. Go has no standard REPL, so there is no console target. `make down` removes containers and networks while retaining caches. No ports, database, or other services are needed.

The image and module pin Go 1.26.9; update them together. The temporary module name is `sift`; replace it with the repository's published module path before distribution.
