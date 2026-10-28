package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func searchPath(path string, output *matchOutput, stderr io.Writer, pattern *regexp.Regexp, noIgnore bool) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		output.recordSkip(path, "search_error", err.Error())
		return false, err
	}
	if info.IsDir() {
		return searchDirectory(path, output, stderr, pattern, noIgnore)
	}
	matched, err := searchFile(path, output, stderr, pattern)
	if err != nil && !errors.Is(err, errOutput) {
		output.recordSkip(path, "search_error", err.Error())
	}
	return matched, err
}

func searchDirectory(path string, output *matchOutput, stderr io.Writer, pattern *regexp.Regexp, noIgnore bool) (bool, error) {
	matched := false
	hadFileErrors := false
	ignores := newIgnoreRules(path)
	err := filepath.WalkDir(path, func(filename string, entry fs.DirEntry, walkErr error) error {
		skip, filterErr := skipEntry(path, filename, entry, ignores, noIgnore || walkErr != nil)
		if skip {
			return skipWalkEntry(entry)
		}
		if walkErr == nil {
			walkErr = filterErr
		}
		if walkErr == nil && entry.Type().IsRegular() && !output.isExport(filename) {
			var found bool
			found, walkErr = searchFile(filename, output, stderr, pattern)
			matched = matched || found
		}
		if errors.Is(walkErr, errOutput) {
			return walkErr
		}
		if walkErr != nil {
			output.recordSkip(filename, "search_error", walkErr.Error())
			output.recordError(walkErr)
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

// Explicit roots remain searchable even when hidden. Nil entries are left to
// WalkDir's error handling, and symlinks are never followed by WalkDir.
func skipEntry(root, filename string, entry fs.DirEntry, ignores *ignoreRules, noIgnore bool) (bool, error) {
	if entry == nil {
		return false, nil
	}
	if filename != root && strings.HasPrefix(entry.Name(), ".") {
		return true, nil
	}
	if noIgnore {
		return false, nil
	}
	return ignores.check(filename, entry)
}

func skipWalkEntry(entry fs.DirEntry) error {
	if entry.IsDir() {
		return filepath.SkipDir
	}
	return nil
}
