package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

func searchWithContinuation(opts options, stdin io.Reader, stdout, stderr io.Writer) (bool, error) {
	matched, err := executeSearch(opts, stdout, stderr)
	if err == nil || searchExitCode(err) != 3 || !canPrompt(opts, stdin, stdout, stderr) {
		return matched, err
	}
	accepted, promptErr := promptContinuation(stdin, stderr, opts.csvPath)
	if promptErr != nil {
		return matched, fmt.Errorf("continuation prompt: %w", promptErr)
	}
	if !accepted {
		return matched, err
	}
	return restartSearch(opts, stdout, stderr)
}

func canPrompt(opts options, stdin io.Reader, stdout, stderr io.Writer) bool {
	return !opts.json && !opts.noInteractive && os.Getenv("TERM") != "dumb" &&
		isTerminal(stdin) && isTerminal(stdout) && isTerminal(stderr)
}

func isTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func promptContinuation(input io.Reader, output io.Writer, csvPath string) (bool, error) {
	message := "Search exceeded its time budget. Partial results shown.\nRestart from the beginning without a time limit? Results may repeat."
	if csvPath != "" {
		message += " A new CSV will be created; the partial export is preserved."
	}
	if _, err := fmt.Fprint(output, message+" [y/N] "); err != nil {
		return false, err
	}
	response, err := bufio.NewReader(input).ReadString('\n')
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(response)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func restartSearch(opts options, stdout, stderr io.Writer) (bool, error) {
	opts.timeout = 0
	opts.noInteractive = true
	if opts.csvPath != "" {
		next, err := continuedCSVPath(opts.csvPath)
		if err != nil {
			return false, err
		}
		opts.previousCSV, opts.csvPath = opts.csvPath, next
		if _, err := fmt.Fprintf(stderr, "Partial CSV preserved at %s\nWriting restarted search to %s\n", opts.previousCSV, next); err != nil {
			return false, err
		}
	}
	return executeSearch(opts, stdout, stderr)
}

func continuedCSVPath(original string) (string, error) {
	ext := filepath.Ext(original)
	stem := strings.TrimSuffix(original, ext)
	for number := 1; ; number++ {
		suffix := ".continued"
		if number > 1 {
			suffix = fmt.Sprintf(".continued-%d", number)
		}
		candidate := stem + suffix + ext
		_, err := os.Lstat(candidate)
		if os.IsNotExist(err) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("choose continued CSV: %w", err)
		}
	}
}
