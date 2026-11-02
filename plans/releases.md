# Release preparation

## Build and inspect locally

From the repository root, with Docker running:

```sh
make build
make test
make release VERSION=v1.0.0-rc.1
```

This creates `dist/v1.0.0-rc.1/` with four `.tar.gz` archives, one Windows `.zip`, and `checksums.txt`. The directory must not already exist. To repeat packaging, use a fresh destination:

```sh
make release VERSION=v1.0.0-rc.1 DIST_DIR=dist/second-review
```

`VERSION` must use `vMAJOR.MINOR.PATCH`, optionally followed by a prerelease suffix such as `-rc.1`. The label is embedded with Go linker flags. All binaries use `CGO_ENABLED=0` and contain no dependency on Docker or a Go installation at runtime.

Archives contain the platform executable, README, README banner, and companion skill. Tar archives preserve executable mode 0755; no local paths or current timestamps are embedded in archive headers. The packager checks `--version` on the target matching its own OS and architecture. Other targets are cross-compiled, not executed.

Verify checksums from the directory containing all five archives:

```sh
cd dist/v1.0.0-rc.1
shasum -a 256 -c checksums.txt
```

Linux can use `sha256sum --check checksums.txt`. Extract the archive for your machine into a temporary directory and run its `sift --version`, a sample search, and a JSON search. Check that `skills/sift/SKILL.md` is present. The packaging helper can be tested separately with:

```sh
python3 -m unittest discover -s scripts -p 'test_*.py'
```

The Makefile runs packaging in the development container. Go and Python 3 are required for running the helper directly on the host; no third-party Python modules are used.

## Prepare a draft on GitHub

Commit and push the reviewed source and workflows first. When ready, tag that commit and push the tag (choose the actual release version):

```sh
git tag -a v1.0.0 -m 'Sift v1.0.0'
git push origin v1.0.0
```

The Release workflow runs tests and packaging using the tagged source. Its read-only build job uploads packages as an Actions artifact. A separate job with `contents: write` downloads and verifies the packages and calls `gh release create --draft --verify-tag`.

Review the draft's version, source tag, generated release notes, all five archives, and checksums in GitHub Releases. Publish it manually when ready. Pushing the tag does not publish the release. Drafts are visible to repository collaborators with sufficient access, not general users.

Rerunning the workflow for the same tag replaces assets on an existing draft. It refuses to modify an already published release. If an upload fails partway through, rerun the workflow to replace the draft assets; verify the completed attachment list before publication. Avoid moving or reusing published version tags.

No credentials are needed for local packaging. GitHub Actions uses its repository token for drafting the release; no personal access token is configured. This implementation does not sign/notarize binaries or publish a Homebrew formula.
