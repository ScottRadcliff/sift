package main

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readExport(t *testing.T, filename string) [][]string {
	t.Helper()
	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestCSVExport(t *testing.T) {
	for _, placement := range []string{"before", "after"} {
		t.Run(placement, func(t *testing.T) {
			root := t.TempDir()
			filename := writeSample(t, root, "payments, quoted.txt", "Payment, \"failed\"\nignore\nPAYMENT café\n")
			export := filepath.Join(root, "matches.csv")
			args := []string{"payment", filename, "--csv", export}
			if placement == "before" {
				args = []string{"--csv", export, "payment", filename}
			}
			var stdout, stderr bytes.Buffer
			status := run(args, &stdout, &stderr)
			wantOutput := "1:" + filename + " Payment, \"failed\"\n3:" + filename + " PAYMENT café\n"
			if status != 0 || stdout.String() != wantOutput || stderr.Len() != 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
			want := [][]string{{"filename", "line_number", "text"}, {filename, "1", "Payment, \"failed\""}, {filename, "3", "PAYMENT café"}}
			if rows := readExport(t, export); !reflect.DeepEqual(rows, want) {
				t.Fatalf("rows=%q", rows)
			}
		})
	}
}

func TestCSVDoesNotOverwrite(t *testing.T) {
	root := t.TempDir()
	filename := writeSample(t, root, "input.txt", "payment\n")
	existing := writeSample(t, root, "existing.csv", "original contents")
	for _, export := range []string{existing, filename} {
		var stdout, stderr bytes.Buffer
		if status := run([]string{"payment", filename, "--csv", export}, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "create CSV export") {
			t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
		}
	}
	data, err := os.ReadFile(existing)
	if err != nil || string(data) != "original contents" {
		t.Fatalf("existing export changed: %q %v", data, err)
	}
	data, err = os.ReadFile(filename)
	if err != nil || string(data) != "payment\n" {
		t.Fatalf("input changed: %q %v", data, err)
	}
}

func TestCSVSkipsItsOwnOutput(t *testing.T) {
	root := t.TempDir()
	filename := writeSample(t, root, "a.txt", "payment\n")
	export := filepath.Join(root, "z.csv")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", root, "--csv", export}, &stdout, &stderr); status != 0 || stdout.String() != "1:"+filename+" payment\n" || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if rows := readExport(t, export); len(rows) != 2 || rows[1][0] != filename {
		t.Fatalf("rows=%q", rows)
	}
}

func TestCSVNoMatchesAndSearchErrors(t *testing.T) {
	root := t.TempDir()
	filename := writeSample(t, root, "input.txt", "payment\n")
	for _, tt := range []struct {
		name, term, path string
		status           int
	}{
		{"no-matches", "missing", filename, 1},
		{"missing-file", "payment", filepath.Join(root, "missing"), 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			export := filepath.Join(t.TempDir(), "result.csv")
			var stdout, stderr bytes.Buffer
			if status := run([]string{tt.term, tt.path, "--csv", export}, &stdout, &stderr); status != tt.status {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
			if rows := readExport(t, export); !reflect.DeepEqual(rows, [][]string{{"filename", "line_number", "text"}}) {
				t.Fatalf("rows=%q", rows)
			}
		})
	}
}

func TestCSVPartialResultsAreFlushed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	filename := writeSample(t, root, "a.txt", "payment\n")
	blocked := writeSample(t, root, "b.txt", "payment\n")
	if err := os.Chmod(blocked, 0000); err != nil {
		t.Fatal(err)
	}
	export := filepath.Join(t.TempDir(), "result.csv")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", root, "--csv", export}, &stdout, &stderr); status != 2 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	if rows := readExport(t, export); len(rows) != 2 || rows[1][0] != filename {
		t.Fatalf("rows=%q", rows)
	}
}

func TestCSVFlushFailure(t *testing.T) {
	output, err := openMatchOutput(&bytes.Buffer{}, filepath.Join(t.TempDir(), "result.csv"))
	if err != nil {
		t.Fatal(err)
	}
	output.csv = csv.NewWriter(failingIO{})
	if err := output.writeCSV("input.txt", 1, "payment"); err != nil {
		t.Fatal(err)
	}
	if err := output.finish(); err == nil || !strings.Contains(err.Error(), "finish CSV export") {
		t.Fatalf("err=%v", err)
	}
}

func TestCSVArgumentErrors(t *testing.T) {
	for _, args := range [][]string{
		{"payment", "input.txt", "--csv"},
		{"payment", "input.txt", "--csv", ""},
		{"payment", "input.txt", "--csv", "--case-sensitive"},
		{"payment", "input.txt", "--csv", "first.csv", "--csv", "second.csv"},
	} {
		var stdout, stderr bytes.Buffer
		if status := run(args, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "--csv") {
			t.Fatalf("args=%q status=%d stderr=%q", args, status, stderr.String())
		}
	}
}
