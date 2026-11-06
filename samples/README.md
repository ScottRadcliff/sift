# Sample searches

These synthetic files support manual QA of the Sift skill. From the repository root, ask the agent to use `skills/sift/SKILL.md` with the sample paths below. Direct helper commands are development checks, not a separate executable to install. They require Python 3.11+ and return JSON on stdout.

## Literal and regex matching

```sh
python3 skills/sift/scripts/search.py -- 'payment failed' samples/payments.txt
python3 skills/sift/scripts/search.py --case-sensitive -- 'Payment' samples/payments.txt
python3 skills/sift/scripts/search.py --regex -- 'payment|retry' samples/payments.txt
python3 skills/sift/scripts/search.py -- 'payment|retry' samples/payments.txt
```

Expected results:

| Search | Matching lines | Status |
| --- | --- | --- |
| Literal `payment failed` | 2, 4 | 0 |
| Case-sensitive `Payment` | 1, 2 | 0 |
| Regex `payment\|retry` | 1, 2, 3, 4, 5 | 0 |
| Literal `payment\|retry` | None | 1 |

Matching text retains its original case. Regexes use Python `re` without slash delimiters. An invalid expression such as `[` with `--regex` returns status 2 and a diagnostic on stderr, without JSON.

## Recursive scope and binary skips

```sh
python3 skills/sift/scripts/search.py -- 'payment failed' samples/tree/
python3 skills/sift/scripts/search.py -- 'payment failed' samples/tree/binary.dat
python3 skills/sift/scripts/search.py -- 'payment' 'samples/links/match locations.txt'
```

The directory search returns two matches: `samples/tree/nested/retry notes.txt` line 2 and `samples/tree/orders.txt` line 2. It also records `samples/tree/binary.dat` with skip reason `binary`; `complete` remains true and status is 0. Hidden descendants are excluded without skip records.

The explicit binary search returns no matches, a binary skip, and status 1. Binary detection checks for null bytes in the first 8 KB.

The filename containing spaces returns three matches at lines 1, 2, and 3, including Unicode text. The agent can present file-and-line links using its host's supported format; the helper emits JSON without terminal links or highlighting.

## Ignore rules

```sh
python3 skills/sift/scripts/search.py -- 'payment' samples/ignore/
```

The committed sample files produce two matches: `keep.tmp` and `visible.txt`. The root ignore file excludes `*.tmp` and restores `keep.tmp`; the nested ignore file excludes `local.txt` within its directory.

The automated fixtures exercise ignored files, nested rules, negation, and `--no-ignore` without creating ignored files in this checkout. Run them with:

```sh
python3 -m unittest discover -s tests/migration -p test_search_contract.py -v
```

Explicit files bypass ignore rules. `--no-ignore` includes ignored paths but still excludes hidden descendants and binary files. Ancestor and global ignore rules are not loaded.

## CSV export

Use a fresh temporary directory for each QA run:

```sh
sift_qa_dir=$(mktemp -d)
python3 skills/sift/scripts/search.py --csv "$sift_qa_dir/matches.csv" -- 'payment' samples/export/payments.txt
cat "$sift_qa_dir/matches.csv"
```

Expected CSV (three data rows, status 0):

```csv
filename,line_number,text
samples/export/payments.txt,1,"Payment failed, retry requested."
samples/export/payments.txt,2,"Payment note: ""call support""."
samples/export/payments.txt,3,PAYMENT café receipt requested.
```

Stdout also contains the JSON result. Repeating with the same destination returns status 2, leaves the CSV unchanged, and does not search. A new export with no matches contains only the header and returns status 1. Report partial exports as incomplete after search errors or timeouts.

## Search budgets and errors

```sh
python3 skills/sift/scripts/search.py --timeout 10 -- 'payment' samples/payments.txt
python3 skills/sift/scripts/search.py --timeout 0 -- 'payment' samples/payments.txt
python3 skills/sift/scripts/search.py --timeout 0.000000001 -- 'payment' samples/payments.txt
python3 skills/sift/scripts/search.py -- 'payment' samples/missing.txt
```

The first two return four matches with `complete: true`. The tiny budget returns status 3 with `timed_out: true` and `complete: false`; the match count can vary with timing. The missing file returns status 2 with a `search_error` skip and diagnostic details. Negative or nonfinite budgets fail validation before searching.

Timeouts never prompt. Literal deadlines are cooperative; regex matching runs in a supervised worker. Worker cleanup and output finalization can extend total wall time beyond the configured search budget. Automated integration tests cover a stalled regex, worker cleanup, and partial CSV finalization.

See the [behavior reference](../skills/sift/references/search-behavior.md) and [test guide](../tests/migration/README.md) for the full contract and portability checks.
