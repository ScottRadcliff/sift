package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitignoreRules(t *testing.T) {
	root := t.TempDir()
	writeSample(t, root, ".gitignore", "# comment\n*.tmp\n!keep.tmp\nbuild/\n/root.txt\ndeep/**/secret.txt\n\\#literal\n")
	writeSample(t, root, "nested/.gitignore", "!restore.tmp\nlocal.txt\n")
	for _, name := range []string{"skip.tmp", "keep.tmp", "build/output.txt", "root.txt", "nested/root.txt", "deep/secret.txt", "deep/a/secret.txt", "nested/restore.tmp", "nested/local.txt", "sibling/local.txt", "sibling/restore.tmp", "#literal", "visible.txt"} {
		writeSample(t, root, name, "needle\n")
	}
	var stdout, stderr bytes.Buffer
	status := run([]string{"needle", root}, &stdout, &stderr)
	var want strings.Builder
	for _, name := range []string{"keep.tmp", "nested/restore.tmp", "nested/root.txt", "sibling/local.txt", "visible.txt"} {
		want.WriteString("1:" + filepath.Join(root, name) + " needle\n")
	}
	if status != 0 || stdout.String() != want.String() || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestNoIgnoreAndExplicitFile(t *testing.T) {
	root := t.TempDir()
	writeSample(t, root, ".gitignore", "ignored/\n*.tmp\n")
	filename := writeSample(t, root, "ignored/result.tmp", "needle\n")
	writeSample(t, root, ".hidden", "needle\n")
	for _, args := range [][]string{
		{"needle", root, "--no-ignore"},
		{"--no-ignore", "needle", root},
		{"needle", filename},
		{"needle", filepath.Dir(filename)},
	} {
		var stdout, stderr bytes.Buffer
		if status := run(args, &stdout, &stderr); status != 0 || stdout.String() != "1:"+filename+" needle\n" || stderr.Len() != 0 {
			t.Fatalf("args=%q status=%d stdout=%q stderr=%q", args, status, stdout.String(), stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if status := run([]string{"needle", root}, &stdout, &stderr); status != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestIgnoredDirectoryCannotBeReincluded(t *testing.T) {
	root := t.TempDir()
	writeSample(t, root, ".gitignore", "blocked/\n!blocked/keep.txt\n")
	writeSample(t, root, "blocked/keep.txt", "needle\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"needle", root}, &stdout, &stderr); status != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestUnreadableGitignore(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	ignore := writeSample(t, root, ".gitignore", "*.txt\n")
	if err := os.Chmod(ignore, 0000); err != nil {
		t.Fatal(err)
	}
	visible := writeSample(t, root, "visible.txt", "needle\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"needle", root}, &stdout, &stderr); status != 2 || stdout.String() != "1:"+visible+" needle\n" || !strings.Contains(stderr.String(), ignore) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if status := run([]string{"needle", root, "--no-ignore"}, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
		t.Fatalf("no-ignore status=%d stderr=%q", status, stderr.String())
	}
}
