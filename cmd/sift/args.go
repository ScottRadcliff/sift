package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type options struct {
	pattern       *regexp.Regexp
	path          string
	caseSensitive bool
	noIgnore      bool
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
			case "--no-ignore":
				opts.noIgnore = true
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
	opts.path = positional[1]
	pattern, err := compileSearch(positional[0], opts.caseSensitive)
	if err != nil {
		return options{}, err
	}
	opts.pattern = pattern
	return opts, nil
}
