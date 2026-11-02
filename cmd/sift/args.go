package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type options struct {
	pattern       *regexp.Regexp
	path          string
	noIgnore      bool
	csvPath       string
	json          bool
	help          bool
	timeout       time.Duration
	hyperlinks    string
	noInteractive bool
	previousCSV   string
}

var errArguments = errors.New("provide a non-empty search string and one file or directory")

func parseArgs(args []string) (options, error) {
	opts := options{help: len(args) == 0, timeout: 3 * time.Second, hyperlinks: "auto"}
	caseSensitive := false
	var positional []string
	parseOptions := true
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if parseOptions {
			switch arg {
			case "--":
				parseOptions = false
				continue
			case "--case-sensitive":
				caseSensitive = true
				continue
			case "--no-ignore":
				opts.noIgnore = true
				continue
			case "--no-interactive":
				opts.noInteractive = true
				continue
			case "--json":
				opts.json = true
				continue
			case "--hyperlinks":
				if i+1 >= len(args) {
					return options{}, fmt.Errorf("--hyperlinks requires auto, always, or never")
				}
				i++
				switch args[i] {
				case "auto", "always", "never":
					opts.hyperlinks = args[i]
				default:
					return options{}, fmt.Errorf("invalid --hyperlinks %q; use auto, always, or never", args[i])
				}
				continue
			case "--timeout":
				if i+1 >= len(args) {
					return options{}, fmt.Errorf("--timeout requires a duration such as 3s, 500ms, or 0")
				}
				i++
				duration, err := time.ParseDuration(args[i])
				if err != nil || duration < 0 {
					return options{}, fmt.Errorf("invalid --timeout %q; use a non-negative duration such as 3s, 500ms, or 0", args[i])
				}
				opts.timeout = duration
				continue
			case "--csv":
				if opts.csvPath != "" {
					return options{}, fmt.Errorf("--csv may only be specified once")
				}
				if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "-") {
					return options{}, fmt.Errorf("--csv requires an output filename; use ./ for a filename starting with -")
				}
				i++
				opts.csvPath = args[i]
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
	pattern, err := compileSearch(positional[0], caseSensitive)
	if err != nil {
		return options{}, err
	}
	opts.pattern = pattern
	return opts, nil
}
