package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const usage = "Usage: sift [--case-sensitive] 'search' path\n\nSearch a text file or directory recursively for a case-insensitive literal string.\nBinary files (a null byte in the first 8 KB) are skipped and reported on stderr.\nDirectory searches skip hidden entries, symlinks, and other non-regular files.\nOptions may appear before or after arguments. Use -- before literal arguments starting with -.\n  --case-sensitive  Match case exactly\n  -h, --help        Show help\nExit status: 0 matches, 1 no matches, 2 error."

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "sift: %v\n", err)
		if errors.Is(err, errArguments) {
			fmt.Fprintln(stderr, usage)
		}
		return 2
	}
	if opts.help {
		fmt.Fprintln(stdout, usage)
		return 0
	}

	matched, err := searchPath(opts.path, stdout, stderr, opts.term, opts.caseSensitive)
	if err != nil {
		fmt.Fprintf(stderr, "sift: %v\n", err)
		return 2
	}
	if !matched {
		return 1
	}
	return 0
}

// A failed output stream stops traversal rather than reporting an error for each file.
var errOutput = errors.New("cannot write results")

func searchPath(path string, stdout, stderr io.Writer, term string, caseSensitive bool) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return searchDirectory(path, stdout, stderr, term, caseSensitive)
	}
	return searchFile(path, stdout, stderr, term, caseSensitive)
}

func searchDirectory(path string, stdout, stderr io.Writer, term string, caseSensitive bool) (bool, error) {
	matched := false
	hadFileErrors := false
	err := filepath.WalkDir(path, func(filename string, entry fs.DirEntry, walkErr error) error {
		// The explicitly requested root is searched even if its name starts with a dot.
		if entry != nil && filename != path && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if walkErr == nil && entry.Type().IsRegular() {
			var found bool
			found, walkErr = searchFile(filename, stdout, stderr, term, caseSensitive)
			matched = matched || found
		}
		if errors.Is(walkErr, errOutput) {
			return walkErr
		}
		if walkErr != nil {
			fmt.Fprintf(stderr, "sift: %v\n", walkErr)
			hadFileErrors = true
		}
		return nil
	})
	if err != nil {
		return matched, err
	}
	if hadFileErrors {
		return matched, errors.New("directory search completed with errors")
	}
	return matched, nil
}

func searchFile(filename string, output, diagnostics io.Writer, term string, caseSensitive bool) (bool, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s: expected a regular file", filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()
	reader := bufio.NewReaderSize(file, 8192)
	prefix, err := reader.Peek(8192)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("%s: %w", filename, err)
	}
	if bytes.IndexByte(prefix, 0) >= 0 {
		reportSkippedBinary(diagnostics, filename)
		return false, nil
	}
	matched, err := search(reader, output, term, filename, caseSensitive)
	if err != nil {
		return matched, fmt.Errorf("%s: %w", filename, err)
	}
	return matched, nil
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
					return matched, fmt.Errorf("%w: %w", errOutput, writeErr)
				}
				matched = true
			}
		}
		if err == io.EOF {
			return matched, nil
		}
	}
}
