package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	for _, tt := range []struct {
		name, text, term, want string
	}{
		{"mixed case", "Ready\nPayment failed\npayment FAILED twice: payment failed\n", "payment failed", "2:sample.txt Payment failed\n3:sample.txt payment FAILED twice: payment failed\n"},
		{"literal punctuation", "a.b\naxb\n", "a.b", "1:sample.txt a.b\n"},
		{"unicode", "CAFÉ\n", "café", "1:sample.txt CAFÉ\n"},
		{"line endings", "skip\r\nMatch\r\nMATCH", "match", "2:sample.txt Match\n3:sample.txt MATCH\n"},
		{"long line", strings.Repeat("x", 70000) + "needle", "needle", "1:sample.txt " + strings.Repeat("x", 70000) + "needle\n"},
		{"no matches", "ready\n", "missing", ""},
		{"empty file", "", "missing", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			matched, err := search(strings.NewReader(tt.text), &matchOutput{terminal: &output}, testPattern(t, tt.term, false), "sample.txt")
			if err != nil || matched != (tt.want != "") || output.String() != tt.want {
				t.Fatalf("matched=%v err=%v; output matches expectation: %v", matched, err, output.String() == tt.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "sample file.txt")
	if err := os.WriteFile(filename, []byte("Payment failed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		args   []string
		status int
		want   string
	}{
		{"match", []string{"payment", filename}, 0, "1:" + filename + " Payment failed\n"},
		{"case sensitive before", []string{"--case-sensitive", "Payment", filename}, 0, "1:" + filename + " Payment failed\n"},
		{"case sensitive after", []string{"Payment", filename, "--case-sensitive"}, 0, "1:" + filename + " Payment failed\n"},
		{"case sensitive between", []string{"Payment", "--case-sensitive", filename}, 0, "1:" + filename + " Payment failed\n"},
		{"case mismatch", []string{"payment", filename, "--case-sensitive"}, 1, ""},
		{"unknown option", []string{"payment", filename, "--unknown"}, 2, ""},
		{"flag without arguments", []string{"--case-sensitive"}, 2, ""},
		{"literal option after separator", []string{"--", "--case-sensitive", filename}, 1, ""},
		{"no match", []string{"missing", filename}, 1, ""},
		{"missing file", []string{"payment", filepath.Join(dir, "missing")}, 2, ""},
		{"directory", []string{"payment", dir}, 0, "1:" + filename + " Payment failed\n"},
		{"missing argument", []string{"payment"}, 2, ""},
		{"extra argument", []string{"payment", filename, filename}, 2, ""},
		{"empty term", []string{"", filename}, 2, ""},
		{"help", []string{"--help"}, 0, usage + "\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := run(tt.args, &stdout, &stderr)
			if status != tt.status || stdout.String() != tt.want || (stderr.Len() > 0) != (tt.status == 2) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

type failingIO struct{}

func TestUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read files without read permissions")
	}
	filename := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(filename, []byte("payment"), 0000); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", filename}, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "permission denied") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func (failingIO) Read([]byte) (int, error)  { return 0, io.ErrUnexpectedEOF }
func (failingIO) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestSearchIOErrors(t *testing.T) {
	if _, err := search(failingIO{}, &matchOutput{terminal: io.Discard}, testPattern(t, "term", false), "sample"); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("read error: %v", err)
	}
	if _, err := search(strings.NewReader("term"), &matchOutput{terminal: failingIO{}}, testPattern(t, "term", false), "sample"); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write error: %v", err)
	}
}

func TestCaseSensitiveSearch(t *testing.T) {
	var output bytes.Buffer
	matched, err := search(strings.NewReader("CAFÉ\ncafé\ncafÉ\n"), &matchOutput{terminal: &output}, testPattern(t, "café", true), "sample")
	if err != nil || !matched || output.String() != "2:sample café\n" {
		t.Fatalf("matched=%v err=%v output=%q", matched, err, output.String())
	}
}

func testPattern(t *testing.T, term string, caseSensitive bool) *regexp.Regexp {
	t.Helper()
	pattern, err := compileSearch(term, caseSensitive)
	if err != nil {
		t.Fatal(err)
	}
	return pattern
}
