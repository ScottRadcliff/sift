package main

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// Rules are scoped to the requested directory and its descendants. Keeping each
// directory's inherited rules separate prevents nested rules affecting siblings.
type ignoreRules struct {
	root     string
	patterns map[string][]gitignore.Pattern
}

func newIgnoreRules(root string) *ignoreRules {
	return &ignoreRules{root: filepath.Clean(root), patterns: make(map[string][]gitignore.Pattern)}
}

func (r *ignoreRules) check(filename string, entry fs.DirEntry) (bool, error) {
	filename = filepath.Clean(filename)
	var patterns []gitignore.Pattern
	var domain []string
	if filename != r.root {
		patterns = r.patterns[filepath.Dir(filename)]
		relative, err := filepath.Rel(r.root, filename)
		if err != nil {
			return false, err
		}
		domain = strings.Split(filepath.ToSlash(relative), "/")
		if gitignore.NewMatcher(patterns).Match(domain, entry.IsDir()) {
			return true, nil
		}
	}
	if !entry.IsDir() {
		return false, nil
	}
	// Copy inherited patterns before appending, so siblings cannot overwrite them.
	r.patterns[filename] = append([]gitignore.Pattern(nil), patterns...)
	file, err := os.Open(filepath.Join(filename, ".gitignore"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r.patterns[filename] = append(r.patterns[filename], gitignore.ParsePattern(line, domain))
	}
	if err := scanner.Err(); err != nil {
		return false, &os.PathError{Op: "read", Path: file.Name(), Err: err}
	}
	return false, nil
}
