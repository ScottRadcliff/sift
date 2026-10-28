package main

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// scanMatches emits matching lines without adding terminal formatting.
func scanMatches(input io.Reader, pattern *regexp.Regexp, emit func(lineNumber int, text string) error) (bool, error) {
	reader := bufio.NewReader(input)
	matched := false
	for lineNumber := 1; ; lineNumber++ {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return matched, err
		}
		if len(line) > 0 {
			line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			if pattern.MatchString(line) {
				if err := emit(lineNumber, line); err != nil {
					return matched, err
				}
				matched = true
			}
		}
		if err == io.EOF {
			return matched, nil
		}
	}
}
