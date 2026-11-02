package main

import (
	"bufio"
	"context"
	"io"
	"regexp"
	"strings"
)

// scanMatches emits matching lines without adding terminal formatting.
func scanMatches(ctx context.Context, input io.Reader, pattern *regexp.Regexp, emit func(lineNumber int, text string) error) (bool, error) {
	reader := bufio.NewReader(contextReader{ctx, input})
	matched := false
	for lineNumber := 1; ; lineNumber++ {
		if err := ctx.Err(); err != nil {
			return matched, err
		}
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return matched, err
		}
		if err := ctx.Err(); err != nil {
			return matched, err
		}
		if len(line) > 0 {
			line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			matches := pattern.MatchString(line)
			if err := ctx.Err(); err != nil {
				return matched, err
			}
			if matches {
				if err := emit(lineNumber, line); err != nil {
					return matched, err
				}
				matched = true
			}
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return matched, ctxErr
		}
		if err == io.EOF {
			return matched, nil
		}
	}
}
