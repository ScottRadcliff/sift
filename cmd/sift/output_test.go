package main

import (
	"regexp"
	"testing"
)

func TestHighlightMatches(t *testing.T) {
	for _, tt := range []struct {
		name, expression, line, want string
	}{
		{"repeated mixed case", "(?i)payment", "Payment and PAYMENT", "\x1b[33mPayment\x1b[0m and \x1b[33mPAYMENT\x1b[0m"},
		{"unicode", "(?i)café", "CAFÉ café", "\x1b[33mCAFÉ\x1b[0m \x1b[33mcafé\x1b[0m"},
		{"literal punctuation", regexp.QuoteMeta("a.$1"), "a.$1 ax$1", "\x1b[33ma.$1\x1b[0m ax$1"},
		{"case sensitive", "Payment", "Payment PAYMENT", "\x1b[33mPayment\x1b[0m PAYMENT"},
		{"adjacent matches", "a", "aa", "\x1b[33ma\x1b[0m\x1b[33ma\x1b[0m"},
		{"regex alternation", "(?i)pay(?:ment)?|retry", "Payment RETRY", "\x1b[33mPayment\x1b[0m \x1b[33mRETRY\x1b[0m"},
		{"zero width", "^", "unchanged", "unchanged"},
		{"no match", "missing", "unchanged", "unchanged"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := highlightMatches(tt.line, regexp.MustCompile(tt.expression)); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
