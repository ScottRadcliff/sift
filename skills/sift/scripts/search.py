#!/usr/bin/env python3
"""Private JSON search helper for the Sift skill (Python 3.11+)."""

import argparse
import csv
import json
import math
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time

from _engine import DeadlineExceeded, scan


class OutputFailure(Exception):
    pass


def timeout_seconds(value):
    try:
        seconds = float(value)
    except ValueError:
        raise argparse.ArgumentTypeError("timeout must be finite, nonnegative seconds") from None
    if not math.isfinite(seconds) or seconds < 0:
        raise argparse.ArgumentTypeError("timeout must be finite, nonnegative seconds")
    return seconds


def arguments(argv):
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    parser.add_argument("--regex", action="store_true")
    parser.add_argument("--case-sensitive", action="store_true")
    parser.add_argument("--no-ignore", action="store_true")
    parser.add_argument("--csv", metavar="FILE")
    parser.add_argument("--timeout", type=timeout_seconds, default=3.0, metavar="SECONDS")
    parser.add_argument("term")
    parser.add_argument("path")
    args = parser.parse_args(argv)
    if not args.term:
        parser.error("search term must not be empty")
    if args.regex:
        try:
            re.compile(args.term, 0 if args.case_sensitive else re.IGNORECASE)
        except (re.error, RecursionError, OverflowError) as exception:
            parser.error(f"invalid regex: {exception}; correct the expression or omit --regex for literal search")
    return args


def worker():
    config = json.load(sys.stdin)

    def emit(event):
        # ASCII escaping safely represents surrogate-containing filenames, too.
        try:
            sys.stdout.write(json.dumps(event, ensure_ascii=True) + "\n")
            sys.stdout.flush()
        except OSError as exception:
            raise OutputFailure(str(exception)) from exception

    scan(config, emit)


def isolated_scan(config, timeout, emit):
    """Kill and reap stalled regex workers; return whether the budget expired.

    Spooling to disk avoids blocking the supervisor on partial pipe messages.
    The temporary directory is excluded even when the user searches its parent.
    """
    started = time.monotonic()
    with tempfile.TemporaryDirectory(prefix="sift-worker-") as directory:
        config = dict(config, excluded=config["excluded"] + [directory])
        with (Path(directory) / "events").open("w+b") as spool, (Path(directory) / "stderr").open("w+b") as diagnostics:
            process = subprocess.Popen(
                [sys.executable, str(Path(__file__).resolve()), "--_worker"],
                stdin=subprocess.PIPE, stdout=spool, stderr=diagnostics)
            timed_out = False
            try:
                remaining = None if timeout == 0 else max(0, timeout - (time.monotonic() - started))
                try:
                    process.communicate(json.dumps(config).encode("utf-8"), timeout=remaining)
                except subprocess.TimeoutExpired:
                    timed_out = True
            finally:
                if process.poll() is None:
                    process.kill()
                process.wait()
                if process.stdin is not None:
                    process.stdin.close()
            spool.seek(0)
            done = False
            for record in spool:
                if not record.endswith(b"\n"):
                    if not timed_out:
                        raise OutputFailure("incomplete regex worker record")
                    break
                try:
                    event = json.loads(record)
                    if not isinstance(event, dict) or event.get("type") not in ("match", "skip", "error", "done"):
                        raise ValueError("unexpected event")
                except (ValueError, UnicodeError) as exception:
                    raise OutputFailure(f"invalid regex worker record: {exception}") from exception
                done = done or event["type"] == "done"
                emit(event)
            if not timed_out and (process.returncode or not done):
                diagnostics.seek(0)
                detail = diagnostics.read(8192).decode("utf-8", errors="replace").strip()
                raise OutputFailure(f"regex worker failed (status {process.returncode}): {detail or 'missing completion record'}")
            return timed_out


def execute(args):
    result = {"matches": [], "skipped_files": [], "errors": [], "timed_out": False, "complete": True}
    output_failed = False
    export = None
    writer = None
    config = {"term": args.term, "path": args.path, "regex": args.regex,
              "case_sensitive": args.case_sensitive, "no_ignore": args.no_ignore,
              "excluded": []}

    def emit(event):
        kind = event["type"]
        if kind == "match":
            result["matches"].append(event["value"])
            if writer is not None:
                try:
                    writer.writerow(event["value"])
                except (OSError, UnicodeError) as exception:
                    raise OutputFailure(f"write CSV: {exception}") from exception
        elif kind == "skip":
            result["skipped_files"].append(event["value"])
        elif kind == "error":
            result["errors"].append(event["value"])
            result["complete"] = False

    try:
        if args.csv:
            # Reserve before searching, and never replace an existing export.
            export = open(args.csv, "x", encoding="utf-8", errors="backslashreplace", newline="")
            writer = csv.DictWriter(export, fieldnames=["filename", "line_number", "text"])
            writer.writeheader()
            export.flush()
            config["excluded"].append(str(Path(args.csv).resolve()))
        if args.regex:
            result["timed_out"] = isolated_scan(config, args.timeout, emit)
        else:
            deadline = time.monotonic() + args.timeout if args.timeout else None
            try:
                scan(config, emit, deadline)
            except DeadlineExceeded:
                result["timed_out"] = True
        if result["timed_out"]:
            result["complete"] = False
            result["errors"].append("Search time limit exceeded; results may be partial")
    except (OSError, ValueError, OutputFailure) as exception:
        result["errors"].append(str(exception))
        result["complete"] = False
        output_failed = True
    finally:
        if export is not None:
            try:
                export.close()
            except OSError as exception:
                result["errors"].append(f"finalize CSV: {exception}")
                result["complete"] = False
                output_failed = True
    status = (2 if output_failed else 3 if result["timed_out"] else
              2 if result["errors"] else 0 if result["matches"] else 1)
    return status, result


def main(argv=None):
    args = arguments(sys.argv[1:] if argv is None else argv)
    status, result = execute(args)
    for error in result["errors"]:
        print(f"sift: {error}", file=sys.stderr)
    try:
        json.dump(result, sys.stdout, ensure_ascii=True)
        sys.stdout.write("\n")
        sys.stdout.flush()
    except (OSError, UnicodeError) as exception:
        print(f"sift: write JSON: {exception}", file=sys.stderr)
        # Avoid a second flush error at interpreter shutdown.
        sys.stdout = None
        return 2
    return status


if __name__ == "__main__":
    if sys.argv[1:] == ["--_worker"]:
        worker()
    else:
        raise SystemExit(main())
