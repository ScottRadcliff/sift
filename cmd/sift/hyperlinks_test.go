package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestFirstMatchColumn(t *testing.T) {
	for _, tt := range []struct {
		text, pattern string
		want          int
	}{
		{"prefix payment payment", "payment", 8},
		{"café payment", "payment", 6},
		{"😀 payment", "payment", 4},
		{"\tpayment", "payment", 2},
		{"prefix payment", "^", 1},
	} {
		if got := firstMatchColumn(tt.text, regexp.MustCompile(tt.pattern)); got != tt.want {
			t.Fatalf("%q: got %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestEditorLinksEncodePaths(t *testing.T) {
	formatter, err := newLocationFormatter(&bytes.Buffer{}, "always", "editor://open?path={path}&line={line}&column={column}")
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(t.TempDir(), "a b#&?%.txt")
	got := formatter.format(filename, 12, 4)
	if !strings.Contains(got, "a%20b%23%26%3F%25.txt&line=12&column=4") || !strings.Contains(got, "\x1b\\12:"+filename+"\x1b]8;;\x1b\\") {
		t.Fatalf("link=%q", got)
	}
}

func TestEditorLinksUseAbsolutePaths(t *testing.T) {
	formatter, err := newLocationFormatter(&bytes.Buffer{}, "always", "")
	if err != nil {
		t.Fatal(err)
	}
	got := formatter.format("samples/payments.txt", 2, 1)
	absolute, err := filepath.Abs("samples/payments.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "vscode://file"+filepath.ToSlash(absolute)+":2:1") {
		t.Fatalf("link=%q", got)
	}
}

func TestHyperlinkOutputKeepsMatchTextAndCSVPlain(t *testing.T) {
	t.Setenv("SIFT_EDITOR_URL", "editor://file{path}:{line}:{column}")
	filename := writeSample(t, t.TempDir(), "input.txt", "😀 Payment payment\n")
	export := filepath.Join(t.TempDir(), "matches.csv")
	var stdout, stderr bytes.Buffer
	status := run([]string{"payment", filename, "--hyperlinks", "always", "--csv", export}, &stdout, &stderr)
	if status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), ":1:4\x1b\\") || !strings.HasSuffix(stdout.String(), "\x1b]8;;\x1b\\ 😀 Payment payment\n") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	rows := readExport(t, export)
	if len(rows) != 2 || rows[1][2] != "😀 Payment payment" {
		t.Fatalf("rows=%q", rows)
	}
}

func TestJSONNeverContainsEditorLinks(t *testing.T) {
	t.Setenv("SIFT_EDITOR_URL", "invalid template")
	filename := writeSample(t, t.TempDir(), "input.txt", "payment\n")
	var stdout, stderr bytes.Buffer
	status := run([]string{"payment", filename, "--json", "--hyperlinks", "always"}, &stdout, &stderr)
	if status != 0 || !json.Valid(stdout.Bytes()) || strings.Contains(stdout.String(), "\\u001b") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRedirectedOutputHasNoLinksByDefault(t *testing.T) {
	t.Setenv("SIFT_EDITOR_URL", "editor://file{path}:{line}:{column}")
	filename := writeSample(t, t.TempDir(), "input.txt", "payment\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"payment", filename}, &stdout, &stderr); status != 0 || stdout.String() != "1:"+filename+" payment\n" {
		t.Fatalf("status=%d stdout=%q", status, stdout.String())
	}
}

func TestInvalidLinkConfiguration(t *testing.T) {
	for _, template := range []string{"missing path", "editor://file{path}:{unknown}", "editor://file{path}\x1b", "editor://file{path}\n", "{path}"} {
		if _, err := newLocationFormatter(&bytes.Buffer{}, "always", template); err == nil {
			t.Fatalf("accepted template %q", template)
		}
	}
	for _, args := range [][]string{{"payment", "input", "--hyperlinks"}, {"payment", "input", "--hyperlinks", "bad"}} {
		if _, err := parseArgs(args); err == nil {
			t.Fatalf("accepted args=%q", args)
		}
	}
}

func TestDisabledLinksIgnoreEditorConfiguration(t *testing.T) {
	formatter, err := newLocationFormatter(os.Stdout, "never", "invalid template")
	if err != nil || formatter.enabled {
		t.Fatalf("formatter=%+v err=%v", formatter, err)
	}
}

func TestLinkLabelCannotBreakEscapeSequence(t *testing.T) {
	formatter := locationFormatter{enabled: true, template: defaultEditorURL}
	got := formatter.format("escape\x1b\n.txt", 1, 1)
	if strings.Contains(got, "escape\x1b\n") || !strings.Contains(got, "escape%1B%0A.txt") || !strings.Contains(got, "escape??.txt") {
		t.Fatalf("link=%q", got)
	}
}

func TestRegexLinkUsesFirstMatch(t *testing.T) {
	var stdout bytes.Buffer
	formatter, err := newLocationFormatter(&stdout, "always", "")
	if err != nil {
		t.Fatal(err)
	}
	output := &matchOutput{terminal: &stdout, locations: formatter}
	_, err = search(context.Background(), strings.NewReader("skip retry payment"), output, testPattern(t, "/retry|payment/", false), "sample.txt")
	if err != nil || !strings.Contains(stdout.String(), ":1:"+strconv.Itoa(6)+"\x1b\\") {
		t.Fatalf("err=%v output=%q", err, stdout.String())
	}
}
