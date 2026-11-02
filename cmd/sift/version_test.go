package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	original := version
	version = "v1.0.0-test"
	t.Cleanup(func() { version = original })
	for _, args := range [][]string{
		{"--version"},
		{"--version", "--json"},
		{"/invalid[", "missing-file", "--version"},
	} {
		var stdout, stderr bytes.Buffer
		if status := runWithInput(args, failingIO{}, &stdout, &stderr); status != 0 || stdout.String() != "sift v1.0.0-test\n" || stderr.Len() != 0 {
			t.Fatalf("args=%q status=%d stdout=%q stderr=%q", args, status, stdout.String(), stderr.String())
		}
	}
}

func TestVersionOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	if status := run([]string{"--version"}, failingIO{}, &stderr); status != 2 || !strings.Contains(stderr.String(), "write version") {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
}

func TestLiteralVersionText(t *testing.T) {
	filename := writeSample(t, t.TempDir(), "input.txt", "--version\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"--", "--version", filename}, &stdout, &stderr); status != 0 || stdout.String() != "1:"+filename+" --version\n" {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestVersionHelpAndUnknownOptions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"--version", "--help"}, &stdout, &stderr); status != 0 || stdout.String() != usage+"\n" {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if status := run([]string{"--version", "--unknown"}, &stdout, &stderr); status != 2 || stdout.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
