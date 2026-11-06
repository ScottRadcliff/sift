"""Private traversal and matching engine. No agent or terminal presentation."""

import os
from pathlib import Path
import re
import stat
import sys
import time

# Always use the bundled dependency, independent of the caller's environment.
sys.path.insert(0, str(Path(__file__).resolve().parent / "_vendor"))
from pathspec import GitIgnoreSpec


class DeadlineExceeded(Exception):
    pass


def check_deadline(deadline):
    if deadline is not None and time.monotonic() >= deadline:
        raise DeadlineExceeded


def literal_fold(text):
    """Single-code-point folding; never expand ß into ss or İ into i + dot.

    This retains common Go simple-fold behavior; mappings track Python's Unicode
    tables rather than promising identical results across Unicode versions.
    """
    def fold(char):
        folded = char.casefold()
        if len(folded) == 1:
            return folded
        lowered = char.lower()
        return lowered if len(lowered) == 1 else char
    return "".join(fold(char) for char in text)


def matcher(term, regex, case_sensitive):
    if regex:
        return re.compile(term, 0 if case_sensitive else re.IGNORECASE).search
    if case_sensitive:
        return lambda text: term in text
    folded = literal_fold(term)
    return lambda text: folded in literal_fold(text)


def lines(stream, deadline):
    """Read bounded chunks so long lines still allow cooperative cancellation."""
    parts = []
    while True:
        check_deadline(deadline)
        chunk = stream.read(65536)
        if not chunk:
            if parts:
                yield b"".join(parts)
            return
        chunks = chunk.split(b"\n")
        for piece in chunks[:-1]:
            check_deadline(deadline)
            parts.append(piece)
            yield b"".join(parts)
            parts.clear()
        if chunks[-1]:
            parts.append(chunks[-1])


def scan(config, emit, deadline=None):
    """Emit match/skip/error events, then done. Errors retain available results."""
    root = Path(config["path"])
    excluded = [Path(p).resolve() for p in config.get("excluded", [])]
    matches = matcher(config["term"], config["regex"], config["case_sensitive"])

    def excluded_path(path):
        resolved = path.resolve()
        return any(resolved == item or item in resolved.parents for item in excluded)

    def error(path, exception):
        message = f"{path}: {exception}"
        emit({"type": "skip", "value": {"filename": str(path), "reason": "search_error", "error": message}})
        emit({"type": "error", "value": message})

    def search_file(path):
        check_deadline(deadline)
        try:
            with path.open("rb") as stream:
                if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
                    raise OSError("expected a regular file")
                if b"\x00" in stream.read(8192):
                    emit({"type": "skip", "value": {"filename": str(path), "reason": "binary"}})
                    return
                stream.seek(0)
                for number, raw in enumerate(lines(stream, deadline), 1):
                    text = raw.removesuffix(b"\r").decode("utf-8", errors="replace")
                    check_deadline(deadline)
                    found = matches(text)
                    check_deadline(deadline)
                    if found:
                        emit({"type": "match", "value": {
                            "filename": str(path), "line_number": number, "text": text}})
        except OSError as exception:
            error(path, exception)

    def visit(directory, inherited):
        check_deadline(deadline)
        rules = list(inherited)
        if not config["no_ignore"]:
            ignore = directory / ".gitignore"
            try:
                # read one line at a time and check between rules.
                with ignore.open(encoding="utf-8", errors="replace") as stream:
                    patterns = []
                    for line in stream:
                        check_deadline(deadline)
                        patterns.append(line.removesuffix("\n").removesuffix("\r"))
                rules.append((directory, GitIgnoreSpec.from_lines(patterns, backend="simple")))
            except FileNotFoundError:
                pass
            except (OSError, ValueError) as exception:
                error(ignore, exception)
        try:
            with os.scandir(directory) as entries:
                children = []
                for entry in entries:
                    check_deadline(deadline)
                    children.append(entry)
            children.sort(key=lambda entry: entry.name)
        except OSError as exception:
            error(directory, exception)
            return
        for entry in children:
            check_deadline(deadline)
            if entry.name.startswith(".") or entry.is_symlink():
                continue
            path = directory / entry.name
            try:
                if excluded_path(path):
                    continue
                is_directory = entry.is_dir(follow_symlinks=False)
                ignored = False
                for base, spec in rules:
                    check_deadline(deadline)
                    relative = path.relative_to(base).as_posix() + ("/" if is_directory else "")
                    decision = spec.check_file(relative).include
                    if decision is not None:
                        ignored = decision
                if ignored:
                    continue
                if is_directory:
                    visit(path, rules)
                elif entry.is_file(follow_symlinks=False):
                    search_file(path)
            except (OSError, ValueError) as exception:
                error(path, exception)

    try:
        check_deadline(deadline)
        info = root.stat()
        if excluded_path(root):
            pass
        elif stat.S_ISDIR(info.st_mode):
            if root.is_symlink():
                raise OSError("explicit directory symlinks are not followed")
            visit(root, [])
        elif stat.S_ISREG(info.st_mode):
            search_file(root)
        else:
            raise OSError("expected a regular file or directory")
    except (OSError, ValueError, RecursionError) as exception:
        error(root, exception)
    emit({"type": "done"})
