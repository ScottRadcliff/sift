package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSample(t *testing.T, root, name, text string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDirectorySearch(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".explicit-root")
	first := writeSample(t, root, "a.txt", "Payment\n")
	second := writeSample(t, root, "nested/b file.txt", "skip\npayment\n")
	writeSample(t, root, ".hidden.txt", "payment\n")
	writeSample(t, root, ".hidden/nested.txt", "payment\n")
	writeSample(t, root, "nested/.hidden.txt", "payment\n")
	for _, tt := range []struct {
		term   string
		flags  []string
		status int
		want   string
	}{
		{"payment", nil, 0, "1:" + first + " Payment\n2:" + second + " payment\n"},
		{"Payment", []string{"--case-sensitive"}, 0, "1:" + first + " Payment\n"},
		{"missing", nil, 1, ""},
	} {
		var stdout, stderr bytes.Buffer
		args := append([]string{tt.term, root}, tt.flags...)
		if status := run(args, &stdout, &stderr); status != tt.status || stdout.String() != tt.want || stderr.Len() != 0 {
			t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
		}
	}
}

func TestDirectorySkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	file := writeSample(t, outside, "outside.txt", "payment\n")
	for name, target := range map[string]string{"file-link": file, "dir-link": outside, "cycle": root, "broken": filepath.Join(outside, "missing")} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", root}, &stdout, &stderr); status != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestDirectoryContinuesAfterPermissionErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	blocked := writeSample(t, root, "a-private.txt", "payment\n")
	if err := os.Chmod(blocked, 0000); err != nil {
		t.Fatal(err)
	}
	blockedDir := filepath.Join(root, "b-private")
	if err := os.Mkdir(blockedDir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(blockedDir, 0700) })
	visible := writeSample(t, root, "z-visible.txt", "payment\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", root}, &stdout, &stderr); status != 2 || stdout.String() != "1:"+visible+" payment\n" || !strings.Contains(stderr.String(), blocked) || !strings.Contains(stderr.String(), blockedDir) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestDirectoryEmpty(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", t.TempDir()}, &stdout, &stderr); status != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestDirectoryOutputFailure(t *testing.T) {
	root := t.TempDir()
	writeSample(t, root, "a.txt", "payment\n")
	writeSample(t, root, "b.txt", "payment\n")
	var stderr bytes.Buffer
	if status := run([]string{"payment", root}, failingIO{}, &stderr); status != 2 || strings.Count(stderr.String(), "cannot write results") != 1 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
}
