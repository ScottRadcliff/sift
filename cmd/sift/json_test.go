package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func decodeResult(t *testing.T, data []byte) jsonResult {
	t.Helper()
	var result jsonResult
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("invalid JSON %q: %v", data, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("unexpected extra output: %q", data)
	}
	if result.Matches == nil || result.SkippedFiles == nil || result.Errors == nil {
		t.Fatal("expected arrays, not null")
	}
	return result
}

func TestJSONMatches(t *testing.T) {
	root := t.TempDir()
	filename := writeSample(t, root, "quoted file.txt", "Payment, \"failed\"\nPAYMENT café\n")
	for _, flags := range [][]string{{"--json", "payment", filename}, {"/payment/i", filename, "--json"}} {
		var stdout, stderr bytes.Buffer
		if status := run(flags, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
			t.Fatalf("status=%d stderr=%q", status, stderr.String())
		}
		result := decodeResult(t, stdout.Bytes())
		want := []jsonMatch{{filename, 1, "Payment, \"failed\""}, {filename, 2, "PAYMENT café"}}
		if !result.Complete || !reflect.DeepEqual(result.Matches, want) || len(result.SkippedFiles) != 0 || len(result.Errors) != 0 {
			t.Fatalf("result=%+v", result)
		}
	}
}

func TestJSONSkippedFiles(t *testing.T) {
	root := t.TempDir()
	binary := writeSample(t, root, "a.dat", "payment\x00\n")
	text := writeSample(t, root, "b.txt", "payment\n")
	writeSample(t, root, ".hidden", "payment\n")
	writeSample(t, root, ".gitignore", "ignored.txt\n")
	writeSample(t, root, "ignored.txt", "payment\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", root, "--json"}, &stdout, &stderr); status != 0 {
		t.Fatalf("status=%d", status)
	}
	result := decodeResult(t, stdout.Bytes())
	if !result.Complete || !reflect.DeepEqual(result.Matches, []jsonMatch{{text, 1, "payment"}}) || !reflect.DeepEqual(result.SkippedFiles, []jsonSkip{{binary, "binary", ""}}) || len(result.Errors) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestJSONNoMatches(t *testing.T) {
	root := t.TempDir()
	filename := writeSample(t, root, "input.txt", "payment\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"missing", filename, "--json"}, &stdout, &stderr); status != 1 {
		t.Fatalf("status=%d", status)
	}
	result := decodeResult(t, stdout.Bytes())
	if !result.Complete || len(result.Matches) != 0 || len(result.Errors) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestJSONSearchFailure(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "missing")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", filename, "--json"}, &stdout, &stderr); status != 2 || stderr.Len() == 0 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	result := decodeResult(t, stdout.Bytes())
	if result.Complete || len(result.Matches) != 0 || len(result.Errors) == 0 || len(result.SkippedFiles) != 1 || result.SkippedFiles[0].Reason != "search_error" {
		t.Fatalf("result=%+v", result)
	}
}

func TestJSONPartialResults(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	blocked := writeSample(t, root, "a-private.txt", "payment\n")
	if err := os.Chmod(blocked, 0000); err != nil {
		t.Fatal(err)
	}
	visible := writeSample(t, root, "b.txt", "Payment\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", root, "--json"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	result := decodeResult(t, stdout.Bytes())
	if result.Complete || !reflect.DeepEqual(result.Matches, []jsonMatch{{visible, 1, "Payment"}}) || len(result.Errors) == 0 || len(result.SkippedFiles) != 1 || result.SkippedFiles[0].Filename != blocked || result.SkippedFiles[0].Error == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestJSONWithCSV(t *testing.T) {
	root := t.TempDir()
	filename := writeSample(t, root, "a.txt", "payment\n")
	export := filepath.Join(root, "z.csv")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", root, "--json", "--csv", export}, &stdout, &stderr); status != 0 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	result := decodeResult(t, stdout.Bytes())
	if !result.Complete || len(result.Matches) != 1 || result.Matches[0].Filename != filename || len(readExport(t, export)) != 2 {
		t.Fatalf("result=%+v", result)
	}
	stdout.Reset()
	stderr.Reset()
	if status := run([]string{"payment", root, "--json", "--csv", export}, &stdout, &stderr); status != 2 {
		t.Fatalf("status=%d", status)
	}
	result = decodeResult(t, stdout.Bytes())
	if result.Complete || len(result.Errors) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestJSONOutputFailure(t *testing.T) {
	filename := writeSample(t, t.TempDir(), "a.txt", "payment\n")
	var stderr bytes.Buffer
	if status := run([]string{"payment", filename, "--json"}, failingIO{}, &stderr); status != 2 || !strings.Contains(stderr.String(), "write JSON") {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
}
