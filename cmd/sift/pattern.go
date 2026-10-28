package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Slash-delimited expressions opt into regex syntax; all other terms are literal.
func compileSearch(term string, caseSensitive bool) (*regexp.Regexp, error) {
	expression := regexp.QuoteMeta(term)
	if strings.HasPrefix(term, "/") {
		var err error
		expression, err = regexExpression(term, caseSensitive)
		if err != nil {
			return nil, err
		}
	}
	if !caseSensitive {
		expression = "(?i)" + expression
	}
	pattern, err := regexp.Compile(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w; check brackets and escapes, or omit / delimiters for a literal search", err)
	}
	return pattern, nil
}

func regexExpression(term string, caseSensitive bool) (string, error) {
	closing := closingSlash(term)
	if closing < 1 {
		return "", fmt.Errorf("regex is missing a closing /; use /pattern/ or /pattern/i")
	}
	expression := term[1:closing]
	if expression == "" {
		return "", fmt.Errorf("regex pattern must not be empty; use /.+/ to match non-empty lines")
	}
	switch modifiers := term[closing+1:]; modifiers {
	case "":
	case "i":
		if caseSensitive {
			return "", fmt.Errorf("regex modifier i conflicts with --case-sensitive; remove i or the flag")
		}
	default:
		return "", fmt.Errorf("unsupported regex modifiers %q; only i is supported", modifiers)
	}
	return expression, nil
}

// The last unescaped slash closes the expression, allowing slashes in the body.
func closingSlash(term string) int {
	for i := len(term) - 1; i > 0; i-- {
		if term[i] != '/' {
			continue
		}
		backslashes := 0
		for j := i - 1; j >= 0 && term[j] == '\\'; j-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return i
		}
	}
	return -1
}
