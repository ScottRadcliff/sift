package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegexSearch(t *testing.T) {
	root := t.TempDir()
	filename := writeSample(t, root, "regex.txt", "Payment failed\nretry RETRY\npayment.failed\nCAFÉ\n/path/to/file\n\n")
	for _, tt := range []struct {
		name, term string
		flags      []string
		want       string
	}{
		{"alternation", "/payment|retry/", nil, "1:FILE Payment failed\n2:FILE retry RETRY\n3:FILE payment.failed\n"},
		{"explicit insensitive", "/PAYMENT/i", nil, "1:FILE Payment failed\n3:FILE payment.failed\n"},
		{"case sensitive", "/Payment/", []string{"--case-sensitive"}, "1:FILE Payment failed\n"},
		{"anchors", "/^payment failed$/", nil, "1:FILE Payment failed\n"},
		{"escaped dot", `/payment\.failed/`, nil, "3:FILE payment.failed\n"},
		{"literal remains literal", "payment|retry", nil, ""},
		{"unicode", "/café/", nil, "4:FILE CAFÉ\n"},
		{"slashes", `/\/path\/to\/file/`, nil, "5:FILE /path/to/file\n"},
		{"unescaped internal slashes", "/path/to/file/", nil, "5:FILE /path/to/file\n"},
		{"empty line", "/^$/", nil, "6:FILE \n"},
		{"zero width", "/^/", nil, "1:FILE Payment failed\n2:FILE retry RETRY\n3:FILE payment.failed\n4:FILE CAFÉ\n5:FILE /path/to/file\n6:FILE \n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := run(append([]string{tt.term, filename}, tt.flags...), &stdout, &stderr)
			wantStatus := 0
			if tt.want == "" {
				wantStatus = 1
			}
			want := strings.ReplaceAll(tt.want, "FILE", filename)
			if status != wantStatus || stdout.String() != want || stderr.Len() != 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestInvalidRegex(t *testing.T) {
	for _, tt := range []struct {
		term, diagnostic string
		flags            []string
	}{
		{"/[/", "invalid regex", nil},
		{"/(/", "invalid regex", nil},
		{"/unterminated", "missing a closing /", nil},
		{`/escaped\/`, "missing a closing /", nil},
		{"//", "must not be empty", nil},
		{"/foo/g", "only i is supported", nil},
		{"/foo/ii", "only i is supported", nil},
		{"/foo/i", "conflicts with --case-sensitive", []string{"--case-sensitive"}},
		{`/(foo)\1/`, "invalid regex", nil},
		{"/foo(?=bar)/", "invalid regex", nil},
	} {
		t.Run(tt.term, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			// A missing path ensures regex validation happens before opening files.
			status := run(append([]string{tt.term, filepath.Join(t.TempDir(), "missing")}, tt.flags...), &stdout, &stderr)
			if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), tt.diagnostic) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRegexDirectorySearch(t *testing.T) {
	root := t.TempDir()
	first := writeSample(t, root, "a.txt", "payment\n")
	second := writeSample(t, root, "nested/b.txt", "retry\n")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"/payment|retry/", root}, &stdout, &stderr); status != 0 || stdout.String() != "1:"+first+" payment\n1:"+second+" retry\n" || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
