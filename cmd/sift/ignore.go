package main

import (
	"bufio"
	"context"
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
	ctx      context.Context
	root     string
	patterns map[string][]gitignore.Pattern
}

func newIgnoreRules(ctx context.Context, root string) *ignoreRules {
	return &ignoreRules{ctx: ctx, root: filepath.Clean(root), patterns: make(map[string][]gitignore.Pattern)}
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
	loaded, err := readIgnoreFile(r.ctx, filepath.Join(filename, ".gitignore"), domain)
	r.patterns[filename] = append(r.patterns[filename], loaded...)
	return false, err
}

func readIgnoreFile(ctx context.Context, filename string, domain []string) ([]gitignore.Pattern, error) {
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var patterns []gitignore.Pattern
	scanner := bufio.NewScanner(contextReader{ctx, file})
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return patterns, err
		}
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, gitignore.ParsePattern(line, domain))
	}
	if err := scanner.Err(); err != nil {
		return patterns, &os.PathError{Op: "read", Path: file.Name(), Err: err}
	}
	return patterns, nil
}
