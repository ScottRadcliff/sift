package main

import (
	"errors"
	"fmt"
	"strings"
)

type options struct {
	term          string
	path          string
	caseSensitive bool
	help          bool
}

var errArguments = errors.New("provide a non-empty search string and one file or directory")

func parseArgs(args []string) (options, error) {
	opts := options{help: len(args) == 0}
	var positional []string
	parseOptions := true
	for _, arg := range args {
		if parseOptions {
			switch arg {
			case "--":
				parseOptions = false
				continue
			case "--case-sensitive":
				opts.caseSensitive = true
				continue
			case "--help", "-h":
				opts.help = true
				continue
			}
			if strings.HasPrefix(arg, "-") {
				return options{}, fmt.Errorf("unknown option %q; use -- before literal arguments starting with -", arg)
			}
		}
		positional = append(positional, arg)
	}
	if opts.help {
		return opts, nil
	}
	if len(positional) != 2 || positional[0] == "" {
		return options{}, errArguments
	}
	opts.term, opts.path = positional[0], positional[1]
	return opts, nil
}
