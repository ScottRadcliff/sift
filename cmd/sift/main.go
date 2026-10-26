package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const usage = "Usage: sift [--case-sensitive] 'search' filename\n\nSearch a single text file for a case-insensitive literal string.\nOptions may appear before or after arguments. Use -- before literal arguments starting with -.\n  --case-sensitive  Match case exactly\n  -h, --help        Show help\nExit status: 0 matches, 1 no matches, 2 error."

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	caseSensitive := false
	help := len(args) == 0
	var positional []string
	options := true
	for _, arg := range args {
		if options {
			switch arg {
			case "--":
				options = false
				continue
			case "--case-sensitive":
				caseSensitive = true
				continue
			case "--help", "-h":
				help = true
				continue
			}
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "sift: unknown option %q; use -- before literal arguments starting with -\n", arg)
				return 2
			}
		}
		positional = append(positional, arg)
	}
	if help {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	if len(positional) != 2 || positional[0] == "" {
		fmt.Fprintln(stderr, "sift: provide a non-empty search string and one filename")
		fmt.Fprintln(stderr, usage)
		return 2
	}
	args = positional

	file, err := os.Open(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "sift: %v\n", err)
		return 2
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		fmt.Fprintf(stderr, "sift: %v\n", err)
		return 2
	}
	if !info.Mode().IsRegular() {
		fmt.Fprintf(stderr, "sift: %s: expected a regular file\n", args[1])
		return 2
	}

	matched, err := search(file, stdout, args[0], args[1], caseSensitive)
	if err != nil {
		fmt.Fprintf(stderr, "sift: %s: %v\n", args[1], err)
		return 2
	}
	if !matched {
		return 1
	}
	return 0
}

func search(input io.Reader, output io.Writer, term, filename string, caseSensitive bool) (bool, error) {
	// QuoteMeta keeps punctuation literal while Go's case folding handles Unicode.
	expression := regexp.QuoteMeta(term)
	if !caseSensitive {
		expression = "(?i)" + expression
	}
	pattern := regexp.MustCompile(expression)
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
				if _, writeErr := fmt.Fprintf(output, "%d:%s %s\n", lineNumber, filename, line); writeErr != nil {
					return matched, writeErr
				}
				matched = true
			}
		}
		if err == io.EOF {
			return matched, nil
		}
	}
}
