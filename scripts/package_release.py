#!/usr/bin/env python3
"""Build Sift release archives locally or in CI using Go and Python's stdlib."""

import argparse
import hashlib
import io
import os
from pathlib import Path
import platform
import re
import subprocess
import tarfile
import tempfile
import zipfile

TARGETS = [("darwin", "amd64"), ("darwin", "arm64"), ("linux", "amd64"), ("linux", "arm64"), ("windows", "amd64")]
SUPPORT_FILES = ["README.md", "assets/sift-banner.png", "skills/sift/SKILL.md", "skills/sift/agents/openai.yaml"]


def validate_version(version):
    if not re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?", version):
        raise ValueError("release version must be vMAJOR.MINOR.PATCH, optionally with a prerelease suffix (e.g. v1.0.0-rc.1)")
    if "-" in version:
        for part in version.split("-", 1)[1].split("."):
            if part.isdigit() and len(part) > 1 and part.startswith("0"):
                raise ValueError("numeric prerelease identifiers must not have leading zeroes")


def write_archive(destination, entries):
    """Entries are (archive path, bytes, Unix mode); timestamps are fixed."""
    with destination.open("xb") as output:
        if destination.suffix == ".zip":
            with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED) as archive:
                for name, data, mode in entries:
                    info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
                    info.create_system = 3
                    info.external_attr = (0o100000 | mode) << 16
                    info.compress_type = zipfile.ZIP_DEFLATED
                    archive.writestr(info, data)
        else:
            # Avoid gzip embedding a local filename or current timestamp.
            import gzip
            with gzip.GzipFile(fileobj=output, mode="wb", filename="", mtime=0) as compressed:
                with tarfile.open(fileobj=compressed, mode="w") as archive:
                    for name, data, mode in entries:
                        info = tarfile.TarInfo(name)
                        info.size, info.mode, info.mtime = len(data), mode, 0
                        archive.addfile(info, io.BytesIO(data))


def write_checksums(output, archives):
    lines = []
    for archive in sorted(archives):
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        lines.append(f"{digest}  {archive.name}\n")
    with (output / "checksums.txt").open("x") as checksums:
        checksums.writelines(lines)


def package_release(version, output, root):
    validate_version(version)
    support = [(name, (root / name).read_bytes(), 0o644) for name in SUPPORT_FILES]
    # A fresh output directory prevents mixing releases or overwriting packages.
    output.mkdir(parents=True, exist_ok=False)
    archives = []
    native = (platform.system().lower(), {"x86_64": "amd64", "aarch64": "arm64"}.get(platform.machine().lower(), platform.machine().lower()))
    with tempfile.TemporaryDirectory(prefix="sift-release-") as temporary:
        for goos, goarch in TARGETS:
            name = "sift.exe" if goos == "windows" else "sift"
            binary = Path(temporary) / name
            env = dict(os.environ, GOOS=goos, GOARCH=goarch, CGO_ENABLED="0")
            subprocess.run(["go", "build", "-trimpath", "-buildvcs=false", "-ldflags", f"-s -w -X main.version={version}", "-o", str(binary), "./cmd/sift"], cwd=root, env=env, check=True)
            if (goos, goarch) == native:
                actual = subprocess.check_output([str(binary), "--version"], text=True).strip()
                if actual != f"sift {version}":
                    raise RuntimeError(f"embedded version mismatch: {actual}")
            extension = "zip" if goos == "windows" else "tar.gz"
            archive = output / f"sift_{version}_{goos}_{goarch}.{extension}"
            write_archive(archive, [(name, binary.read_bytes(), 0o755)] + support)
            archives.append(archive)
            print(f"Built {archive}", flush=True)
    write_checksums(output, archives)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True)
    parser.add_argument("--output", type=Path, required=True, help="new output directory")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    try:
        package_release(args.version, args.output.resolve(), root)
    except (ValueError, OSError, subprocess.CalledProcessError, RuntimeError) as error:
        parser.exit(1, f"release packaging failed: {error}\n")


if __name__ == "__main__":
    main()
