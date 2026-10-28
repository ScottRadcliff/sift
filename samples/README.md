# Sample searches

These synthetic files support development and manual QA. Run from the repository root after `make binary`:

```sh
./bin/sift 'payment failed' samples/payments.txt
```

Expected output (exit status 0):

```text
2:samples/payments.txt Payment failed for order 1002.
4:samples/payments.txt PAYMENT FAILED for order 1003.
```

For exact case matching, place `--case-sensitive` before or after the arguments:

```sh
./bin/sift 'Payment' samples/payments.txt --case-sensitive
```

This returns only lines 1 and 2. Without the flag, the same search returns lines 1, 2, 4, and 5.

Use `--` to end option parsing when the search text or filename begins with a hyphen, for example `./bin/sift -- '--help' samples/payments.txt`.

Other checks:

- `./bin/sift 'payment.failed' samples/payments.txt`: only line 5; punctuation is literal.
- `./bin/sift 'CAFÉ' samples/payments.txt`: only line 6; Unicode case matching.
- `./bin/sift 'refunded' samples/payments.txt`: no output, exit status 1.
- `./bin/sift 'payment' samples/missing.txt`: error on stderr, exit status 2.

Container equivalent: `make run ARGS="'payment failed' samples/payments.txt"`.

## Directory search

```sh
./bin/sift 'payment failed' samples/tree/
```

Expected output (exit status 0, files visited in lexical order):

```text
2:samples/tree/nested/retry notes.txt payment failed for order 2002.
2:samples/tree/orders.txt Payment failed for order 2001.
```

Add `--case-sensitive` to return only the lowercase match in the nested file.
Hidden files and directories, symlinks, and other non-regular entries are skipped during traversal. An explicitly named hidden file or directory can still be searched. Unreadable files or directories are reported on stderr, remaining files are searched, and the final exit status is 2 even if matches were found.

Search time limits are not implemented yet. Use this small sample tree for directory QA.

## Binary files

Files with a null byte in the first 8 KB are skipped. The directory search above still produces the same two matches, plus this stderr message:

```text
sift: skipped binary file: samples/tree/binary.dat
```

`./bin/sift 'payment failed' samples/tree/binary.dat` produces only that diagnostic and exits with status 1 (no matches). A binary skip is not an error, so a directory search with text matches still exits with status 0.

This is a prefix heuristic: files without null bytes in their first 8 KB are treated as text, even if binary bytes occur later. UTF-16 files commonly contain null bytes and will be skipped; encoding conversion is not supported yet.

Skipped-file messages appear yellow when stderr is a terminal. Redirected diagnostics remain plain text. Set `NO_COLOR=1` to disable color.

## Ignore rules

Create the two ignored QA files locally (Git intentionally excludes them):

```sh
printf 'payment ignored result\n' > samples/ignore/skipped.tmp
printf 'payment nested ignored result\n' > samples/ignore/nested/local.txt
```

Then search the sample directory:

```sh
./bin/sift 'payment' samples/ignore/
```

Expected output (exit status 0):

```text
1:samples/ignore/keep.tmp payment explicitly included result
1:samples/ignore/visible.txt payment visible result
```

`./bin/sift 'payment' samples/ignore/ --no-ignore` includes all four sample text files, including `skipped.tmp` and `nested/local.txt`. Explicitly naming `samples/ignore/skipped.tmp` also searches it without requiring the flag.

Rules come from `.gitignore` in the requested directory and visited subdirectories; parent directories, global Git settings, and `.git/info/exclude` are not read. Patterns support wildcards, directory rules, and `!` exceptions. An excluded directory is not traversed, so an exception for a file inside it cannot re-include that file. Rules apply to matching paths regardless of whether Git tracks them. Ignored paths are silently skipped. Hidden entries remain skipped with `--no-ignore`.

## Match highlighting

In a terminal, matching occurrences appear yellow while preserving their original case. Line numbers, filenames, and surrounding text stay uncolored.

```sh
./bin/sift 'payment' samples/payments.txt
NO_COLOR=1 ./bin/sift 'payment' samples/payments.txt
./bin/sift 'payment' samples/payments.txt > /tmp/sift-matches.txt
```

The first command highlights each match; the second disables color. The redirected file contains plain text without color escapes. `TERM=dumb` also disables color.

## Regex searches

```sh
./bin/sift '/payment|retry/' samples/payments.txt
```

Expected output (exit status 0):

```text
1:samples/payments.txt Payment received for order 1001.
2:samples/payments.txt Payment failed for order 1002.
3:samples/payments.txt Retry scheduled for order 1002.
4:samples/payments.txt PAYMENT FAILED for order 1003.
5:samples/payments.txt Literal identifier: payment.failed
```

Matching portions are yellow in terminals. Additional checks:

- `./bin/sift '/^payment failed/i' samples/payments.txt`: lines 2 and 4.
- `./bin/sift '/Payment/' samples/payments.txt --case-sensitive`: lines 1 and 2.
- `./bin/sift '/payment\.failed/' samples/payments.txt`: only line 5.
- `./bin/sift 'payment|retry' samples/payments.txt`: literal search, no matches, status 1.
- `./bin/sift '/[/' samples/payments.txt`: invalid regex diagnostic, status 2.
- `./bin/sift '/payment/g' samples/payments.txt`: unsupported modifier diagnostic, status 2.
- `./bin/sift '/payment/i' samples/payments.txt --case-sensitive`: conflicting case options, status 2.

Expressions are matched per line using Go regex syntax; only trailing `i` is supported. `/pattern/` follows the default case-insensitive behavior unless `--case-sensitive` is used. Empty `//` patterns are rejected. A leading slash reserves regex syntax; use a regex with escaped slashes when searching literal paths.

## CSV export

Use a fresh temporary directory so repeated QA runs never overwrite an export:

```sh
qa_dir=$(mktemp -d)
./bin/sift 'payment' samples/export/payments.txt --csv "$qa_dir/matches.csv"
cat "$qa_dir/matches.csv"
```

Expected CSV:

```csv
filename,line_number,text
samples/export/payments.txt,1,"Payment failed, retry requested."
samples/export/payments.txt,2,"Payment note: ""call support""."
samples/export/payments.txt,3,PAYMENT café receipt requested.
```

Terminal results still appear, with yellow matching text when color is enabled. CSV values preserve original text without adding color codes.

Repeat the search with the same output filename: it should report an error and exit with status 2, leaving the existing CSV unchanged. To check regex export, choose another new file and run `./bin/sift '/failed|support/' samples/export/ --csv "$qa_dir/regex.csv"`; expect two data rows. No matches produce a header-only CSV and status 1. Search errors produce status 2 and may leave a partial CSV.

## JSON output

```sh
./bin/sift 'payment failed' samples/payments.txt --json
./bin/sift 'payment failed' samples/tree/ --json
./bin/sift 'missing' samples/payments.txt --json
./bin/sift 'payment' samples/missing.txt --json
```

- The first command returns two matches, empty `skipped_files` and `errors` arrays, and `complete: true` (status 0).
- The directory search returns two matches and a `skipped_files` entry for `samples/tree/binary.dat` with reason `binary`; completion stays true (status 0). Hidden entries are not listed.
- No matches return empty arrays and `complete: true` (status 1).
- The missing file returns no matches, a `search_error` skip, error details, and `complete: false` (status 2).

Stdout is a single JSON object with no added color escapes or human-readable result lines. Diagnostics remain on stderr. `--json` can appear before or after the search arguments and can be combined with `--csv` using a new export filename. Invalid arguments or regexes fail before searching and produce only stderr; `--help` prints ordinary help text.
