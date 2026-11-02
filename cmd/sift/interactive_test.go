package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContinuationPrompt(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  bool
	}{
		{"y\n", true}, {"YES\n", true}, {" yes \n", true},
		{"n\n", false}, {"\n", false}, {"anything\n", false}, {"", false}, {"yes", false},
	} {
		var output bytes.Buffer
		got, err := promptContinuation(strings.NewReader(tt.input), &output, "")
		if err != nil || got != tt.want || !strings.Contains(output.String(), "from the beginning") || !strings.Contains(output.String(), "[y/N]") {
			t.Fatalf("input=%q accepted=%v err=%v prompt=%q", tt.input, got, err, output.String())
		}
	}
	var output bytes.Buffer
	if _, err := promptContinuation(failingIO{}, &output, "partial.csv"); err == nil {
		t.Fatal("expected read error")
	}
	if !strings.Contains(output.String(), "partial export is preserved") {
		t.Fatalf("prompt=%q", output.String())
	}
	if _, err := promptContinuation(strings.NewReader("yes\n"), failingIO{}, ""); err == nil {
		t.Fatal("expected write error")
	}
}

func TestNoninteractiveTimeoutDoesNotReadInput(t *testing.T) {
	filename := writeSample(t, t.TempDir(), "input.txt", "payment\n")
	for _, flags := range [][]string{nil, {"--no-interactive"}, {"--json"}} {
		var stdout, stderr bytes.Buffer
		args := append([]string{"payment", filename, "--timeout", "1ns"}, flags...)
		status := runWithInput(args, failingIO{}, &stdout, &stderr)
		if status != 3 || strings.Contains(stderr.String(), "[y/N]") {
			t.Fatalf("status=%d stderr=%q", status, stderr.String())
		}
	}
}

func TestRestartSearchUsesUnlimitedBudget(t *testing.T) {
	filename := writeSample(t, t.TempDir(), "input.txt", "payment\npayment\n")
	opts, err := parseArgs([]string{"payment", filename, "--timeout", "1ns"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	matched, err := restartSearch(opts, &stdout, &stderr)
	if err != nil || !matched || strings.Count(stdout.String(), "payment\n") != 2 {
		t.Fatalf("matched=%v err=%v stdout=%q", matched, err, stdout.String())
	}
}

func TestRestartPreservesAndExcludesPartialCSV(t *testing.T) {
	root := t.TempDir()
	filename := writeSample(t, root, "input.txt", "payment\n")
	partial := writeSample(t, root, "results.csv", "original payment export\n")
	previousContinuation := writeSample(t, root, "results.continued.csv", "unrelated existing data\n")
	opts, err := parseArgs([]string{"payment", root, "--csv", partial, "--timeout", "1ns"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	matched, err := restartSearch(opts, &stdout, &stderr)
	if err != nil || !matched || stdout.String() != "1:"+filename+" payment\n" {
		t.Fatalf("matched=%v err=%v stdout=%q stderr=%q", matched, err, stdout.String(), stderr.String())
	}
	continued := filepath.Join(root, "results.continued-2.csv")
	if rows := readExport(t, continued); len(rows) != 2 || rows[1][0] != filename {
		t.Fatalf("rows=%q", rows)
	}
	for path, want := range map[string]string{partial: "original payment export\n", previousContinuation: "unrelated existing data\n"} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != want {
			t.Fatalf("file %s changed: %q %v", path, data, err)
		}
	}
	if !strings.Contains(stderr.String(), continued) {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestContinuedCSVPathSkipsDanglingSymlinks(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "results.csv")
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "results.continued.csv")); err != nil {
		t.Fatal(err)
	}
	got, err := continuedCSVPath(original)
	if err != nil || got != filepath.Join(root, "results.continued-2.csv") {
		t.Fatalf("path=%q err=%v", got, err)
	}
}

func TestRestartNoMatches(t *testing.T) {
	filename := writeSample(t, t.TempDir(), "input.txt", "no matches\n")
	opts, err := parseArgs([]string{"payment", filename})
	if err != nil {
		t.Fatal(err)
	}
	matched, err := restartSearch(opts, io.Discard, io.Discard)
	if err != nil || matched {
		t.Fatalf("matched=%v err=%v", matched, err)
	}
}
