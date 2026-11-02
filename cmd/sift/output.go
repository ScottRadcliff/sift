package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	"golang.org/x/term"
)

func reportSkippedBinary(output io.Writer, filename string) {
	message := fmt.Sprintf("sift: skipped binary file: %s", filename)
	if supportsColor(output) {
		message = "\x1b[33m" + message + "\x1b[0m"
	}
	fmt.Fprintln(output, message)
}

func supportsColor(output io.Writer) bool {
	file, ok := output.(*os.File)
	return ok && term.IsTerminal(int(file.Fd())) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
}

func highlightMatches(line string, pattern *regexp.Regexp) string {
	return pattern.ReplaceAllStringFunc(line, func(match string) string {
		if match == "" {
			return ""
		}
		return "\x1b[33m" + match + "\x1b[0m"
	})
}

// A failed output stream stops traversal rather than reporting an error for each file.
var errOutput = errors.New("cannot write results")

func search(ctx context.Context, input io.Reader, output *matchOutput, pattern *regexp.Regexp, filename string) (bool, error) {
	color := supportsColor(output.terminal)
	return scanMatches(ctx, input, pattern, func(lineNumber int, text string) error {
		if err := output.writeCSV(filename, lineNumber, text); err != nil {
			return fmt.Errorf("%w: %w", errOutput, err)
		}
		if output.result != nil {
			output.result.Matches = append(output.result.Matches, jsonMatch{filename, lineNumber, text})
			return nil
		}
		location := fmt.Sprintf("%d:%s", lineNumber, filename)
		if output.locations.enabled {
			location = output.locations.format(filename, lineNumber, firstMatchColumn(text, pattern))
		}
		if color {
			text = highlightMatches(text, pattern)
		}
		if _, err := fmt.Fprintf(output.terminal, "%s %s\n", location, text); err != nil {
			return fmt.Errorf("%w: %w", errOutput, err)
		}
		return nil
	})
}
