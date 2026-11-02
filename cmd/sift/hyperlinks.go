package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/term"
)

const defaultEditorURL = "vscode://file{path}:{line}:{column}"

type locationFormatter struct {
	enabled  bool
	template string
}

func newLocationFormatter(output io.Writer, mode, template string) (locationFormatter, error) {
	enabled := mode == "always" || (mode != "never" && terminalHyperlinks(output))
	if !enabled {
		return locationFormatter{}, nil
	}
	if template == "" {
		template = defaultEditorURL
	}
	if err := validateEditorURL(template); err != nil {
		return locationFormatter{}, err
	}
	return locationFormatter{enabled: true, template: template}, nil
}

func terminalHyperlinks(output io.Writer) bool {
	file, ok := output.(*os.File)
	return ok && term.IsTerminal(int(file.Fd())) && os.Getenv("TERM") != "dumb"
}

func validateEditorURL(template string) error {
	if strings.ContainsFunc(template, unicode.IsControl) || !strings.Contains(template, "{path}") {
		return fmt.Errorf("SIFT_EDITOR_URL must contain {path} and no control characters")
	}
	// Validate after substituting known tokens; raw braces indicate a typo.
	example := expandEditorURL(template, "/example.txt", 1, 1)
	parsed, err := url.Parse(example)
	if err != nil || parsed.Scheme == "" || strings.ContainsAny(example, "{}") {
		return fmt.Errorf("invalid SIFT_EDITOR_URL; use an absolute URL containing {path}, with optional {line} and {column}")
	}
	return nil
}

func expandEditorURL(template, path string, line, column int) string {
	return strings.NewReplacer("{path}", path, "{line}", strconv.Itoa(line), "{column}", strconv.Itoa(column)).Replace(template)
}

func (f locationFormatter) format(filename string, line, column int) string {
	label := fmt.Sprintf("%d:%s", line, filename)
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return label
	}
	// Encode path components rather than the template so query-based handlers
	// work too, and special filename characters cannot change the target URL.
	parts := strings.Split(filepath.ToSlash(absolute), "/")
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(url.QueryEscape(part), "+", "%20")
	}
	encodedPath := strings.Join(parts, "/")
	if !strings.HasPrefix(encodedPath, "/") {
		encodedPath = "/" + encodedPath
	}
	target := expandEditorURL(f.template, encodedPath, line, column)
	label = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, label)
	return "\x1b]8;;" + target + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

// VS Code uses one-based UTF-16 columns, not byte offsets or screen cells.
func firstMatchColumn(text string, pattern *regexp.Regexp) int {
	index := pattern.FindStringIndex(text)
	if index == nil {
		return 1
	}
	column := 1
	for _, r := range text[:index[0]] {
		column++
		if r > 0xffff {
			column++
		}
	}
	return column
}
