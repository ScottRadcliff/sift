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

Other checks:

- `./bin/sift 'payment.failed' samples/payments.txt`: only line 5; punctuation is literal.
- `./bin/sift 'CAFÉ' samples/payments.txt`: only line 6; Unicode case matching.
- `./bin/sift 'refunded' samples/payments.txt`: no output, exit status 1.
- `./bin/sift 'payment' samples/missing.txt`: error on stderr, exit status 2.
- `./bin/sift 'payment' samples/`: directory error, exit status 2; recursion comes later.

Container equivalent: `make run ARGS="'payment failed' samples/payments.txt"`.
