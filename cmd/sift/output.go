package main

import (
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
		return "\x1b[33m" + match + "\x1b[0m"
	})
}
