package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBinaryDetection(t *testing.T) {
	for _, tt := range []struct {
		name, text string
		binary     bool
	}{
		{"short binary", "payment\x00\n", true},
		{"last sampled byte", strings.Repeat("x", 8191) + "\x00payment", true},
		{"beyond sample", strings.Repeat("x", 8192) + "\x00payment", false},
		{"short text", "payment\n", false},
		{"unicode text", "café payment\n", false},
		{"long text", strings.Repeat("x", 8192) + "payment", false},
		{"empty", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			filename := writeSample(t, t.TempDir(), "sample.dat", tt.text)
			var stdout, stderr bytes.Buffer
			status := run([]string{"payment", filename}, &stdout, &stderr)
			wantStatus := 0
			if tt.binary || tt.text == "" {
				wantStatus = 1
			}
			wantDiagnostic := ""
			if tt.binary {
				wantDiagnostic = "sift: skipped binary file: " + filename + "\n"
			}
			if status != wantStatus || stderr.String() != wantDiagnostic {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
			wantOutput := ""
			if !tt.binary && tt.text != "" {
				wantOutput = "1:" + filename + " " + strings.TrimSuffix(tt.text, "\n") + "\n"
			}
			if stdout.String() != wantOutput {
				t.Fatal("output differs from expected text (or contains skipped binary data)")
			}
		})
	}
}

func TestDirectoryContinuesAfterBinary(t *testing.T) {
	root := t.TempDir()
	binary := writeSample(t, root, "a.dat", "payment\x00\n")
	text := writeSample(t, root, "b.txt", "payment\n")
	for _, term := range []string{"payment", "missing"} {
		var stdout, stderr bytes.Buffer
		status := run([]string{term, root}, &stdout, &stderr)
		wantStatus, wantOutput := 1, ""
		if term == "payment" {
			wantStatus, wantOutput = 0, "1:"+text+" payment\n"
		}
		if status != wantStatus || stdout.String() != wantOutput || stderr.String() != "sift: skipped binary file: "+binary+"\n" {
			t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
		}
	}
}
