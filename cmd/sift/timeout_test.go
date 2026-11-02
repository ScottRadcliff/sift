package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Expiry is controlled by observed search progress, not sleeps or machine speed.
type manualDeadline struct {
	context.Context
	expired atomic.Bool
}

func (c *manualDeadline) Err() error {
	if c.expired.Load() {
		return context.DeadlineExceeded
	}
	return nil
}

type expireWriter struct {
	io.Writer
	ctx *manualDeadline
}

func (w expireWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.ctx.expired.Store(true)
	return n, err
}

func TestTimeoutArguments(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want time.Duration
	}{
		{[]string{"payment", "input"}, 3 * time.Second},
		{[]string{"payment", "input", "--timeout", "10s"}, 10 * time.Second},
		{[]string{"--timeout", "500ms", "payment", "input"}, 500 * time.Millisecond},
		{[]string{"payment", "input", "--timeout", "0"}, 0},
	} {
		opts, err := parseArgs(tt.args)
		if err != nil || opts.timeout != tt.want {
			t.Fatalf("args=%q timeout=%v err=%v", tt.args, opts.timeout, err)
		}
	}
	for _, value := range []string{"-1s", "bad", "", "10", "999999999999999999999h", "--json"} {
		if _, err := parseArgs([]string{"payment", "input", "--timeout", value}); err == nil {
			t.Fatalf("accepted timeout %q", value)
		}
	}
	if _, err := parseArgs([]string{"payment", "input", "--timeout"}); err == nil {
		t.Fatal("accepted missing duration")
	}
}

func TestExpiredSearchReturnsJSONAndHeaderOnlyCSV(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	root := t.TempDir()
	filename := writeSample(t, root, "input.txt", "payment\n")
	export := filepath.Join(root, "result.csv")
	opts, err := parseArgs([]string{"payment", filename, "--json", "--csv", export})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	matched, err := executeSearchContext(ctx, opts, &stdout, &stderr)
	if matched || !errors.Is(err, context.DeadlineExceeded) || searchExitCode(err) != 3 {
		t.Fatalf("matched=%v err=%v", matched, err)
	}
	result := decodeResult(t, stdout.Bytes())
	if result.Complete || !result.TimedOut || len(result.Matches) != 0 || len(result.Errors) == 0 || len(result.SkippedFiles) != 0 {
		t.Fatalf("result=%+v", result)
	}
	if rows := readExport(t, export); len(rows) != 1 {
		t.Fatalf("rows=%q", rows)
	}
}

func TestTimeoutPreservesPartialJSONAndCSV(t *testing.T) {
	root := t.TempDir()
	first := writeSample(t, root, "a.txt", "payment\n")
	writeSample(t, root, "b.dat", "payment\x00\n")
	writeSample(t, root, "c.txt", "payment\n")
	export := filepath.Join(t.TempDir(), "result.csv")
	opts, err := parseArgs([]string{"payment", root, "--json", "--csv", export})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &manualDeadline{Context: context.Background()}
	var stdout, stderr bytes.Buffer
	matched, err := executeSearchContext(ctx, opts, &stdout, expireWriter{&stderr, ctx})
	if !matched || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("matched=%v err=%v", matched, err)
	}
	result := decodeResult(t, stdout.Bytes())
	if result.Complete || !result.TimedOut || len(result.Matches) != 1 || result.Matches[0].Filename != first || len(result.SkippedFiles) != 1 || result.SkippedFiles[0].Reason != "binary" {
		t.Fatalf("result=%+v", result)
	}
	if rows := readExport(t, export); len(rows) != 2 || rows[1][0] != first {
		t.Fatalf("rows=%q", rows)
	}
}

func TestTimeoutPreservesPartialTerminalResults(t *testing.T) {
	filename := writeSample(t, t.TempDir(), "input.txt", "payment\npayment\n")
	opts, err := parseArgs([]string{"payment", filename})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &manualDeadline{Context: context.Background()}
	var stdout, stderr bytes.Buffer
	matched, err := executeSearchContext(ctx, opts, expireWriter{&stdout, ctx}, &stderr)
	if !matched || !errors.Is(err, context.DeadlineExceeded) || stdout.String() != "1:"+filename+" payment\n" {
		t.Fatalf("matched=%v err=%v stdout=%q", matched, err, stdout.String())
	}
}

func TestTimeoutDuringBufferedLongLine(t *testing.T) {
	ctx := &manualDeadline{Context: context.Background()}
	reader := &expiringReader{ctx: ctx}
	matched, err := scanMatches(ctx, reader, testPattern(t, "payment", false), func(_ int, _ string) error { t.Fatal("incomplete line emitted"); return nil })
	if matched || !errors.Is(err, context.DeadlineExceeded) || reader.reads != 1 {
		t.Fatalf("matched=%v err=%v reads=%d", matched, err, reader.reads)
	}
}

type expiringReader struct {
	ctx   *manualDeadline
	reads int
}

func (r *expiringReader) Read(p []byte) (int, error) {
	r.reads++
	r.ctx.expired.Store(true)
	return copy(p, strings.Repeat("x", len(p))), nil
}

func TestTimeoutOutputErrorHasPriority(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	opts, err := parseArgs([]string{"payment", "unused", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executeSearchContext(ctx, opts, failingIO{}, io.Discard)
	if !errors.Is(err, errOutput) || searchExitCode(err) != 2 {
		t.Fatalf("err=%v", err)
	}
}

func TestTimeoutDisabled(t *testing.T) {
	filename := writeSample(t, t.TempDir(), "input.txt", "payment\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", filename, "--timeout", "0", "--json"}, &stdout, &stderr); status != 0 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	result := decodeResult(t, stdout.Bytes())
	if !result.Complete || result.TimedOut || len(result.Matches) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunTimeoutExitStatus(t *testing.T) {
	filename := writeSample(t, t.TempDir(), "input.txt", strings.Repeat("payment\n", 1000))
	var stdout, stderr bytes.Buffer
	status := run([]string{"payment", filename, "--timeout", "1ns", "--json"}, &stdout, &stderr)
	if status != 3 || !strings.Contains(stderr.String(), "search time limit exceeded") {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	result := decodeResult(t, stdout.Bytes())
	if result.Complete || !result.TimedOut {
		t.Fatalf("result=%+v", result)
	}
}
