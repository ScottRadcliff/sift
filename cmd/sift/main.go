package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const usage = "Usage: sift [--case-sensitive] [--no-ignore] [--csv filename] 'search' path\n\nSearch a text file or directory recursively for a case-insensitive literal string or /pattern/ regex.\nRegex modifier i explicitly enables case-insensitive matching and conflicts with --case-sensitive.\nBinary files (a null byte in the first 8 KB) are skipped and reported on stderr.\nDirectory searches skip hidden entries, symlinks, and other non-regular files.\nOptions may appear before or after arguments. Use -- before literal arguments starting with -.\n  --case-sensitive  Match case exactly\n  --no-ignore       Disable .gitignore rules (hidden entries still skipped)\n  --csv filename    Export matches to a new CSV file\n  -h, --help        Show help\nExit status: 0 matches, 1 no matches, 2 error."

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

	matched, err := executeSearch(opts, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "sift: %v\n", err)
		return 2
	}
	if !matched {
		return 1
	}
	return 0
}
