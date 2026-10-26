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

Ignore-file support, binary detection, and search time limits are not implemented yet. Use this small sample tree for directory QA.
