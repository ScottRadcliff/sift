package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestScanMatchesCollectsPlainLines(t *testing.T) {
	type result struct {
		line int
		text string
	}
	var results []result
	matched, err := scanMatches(strings.NewReader("skip\r\nPayment failed\r\nPAYMENT"), testPattern(t, "payment", false), func(line int, text string) error {
		results = append(results, result{line, text})
		return nil
	})
	want := []result{{2, "Payment failed"}, {3, "PAYMENT"}}
	if err != nil || !matched || !reflect.DeepEqual(results, want) {
		t.Fatalf("matched=%v err=%v results=%#v", matched, err, results)
	}
}

func TestScanMatchesStopsWhenConsumerFails(t *testing.T) {
	stop := errors.New("consumer stopped")
	calls := 0
	matched, err := scanMatches(strings.NewReader("payment\npayment\npayment\n"), testPattern(t, "payment", false), func(_ int, _ string) error {
		calls++
		if calls == 2 {
			return stop
		}
		return nil
	})
	if !matched || !errors.Is(err, stop) || calls != 2 {
		t.Fatalf("matched=%v err=%v calls=%d", matched, err, calls)
	}
}
