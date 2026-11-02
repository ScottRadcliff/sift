package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

type matchOutput struct {
	terminal  io.Writer
	csv       *csv.Writer
	file      *os.File
	fileInfo  os.FileInfo
	result    *jsonResult
	locations locationFormatter
}

func executeSearch(opts options, stdout, stderr io.Writer) (bool, error) {
	ctx := context.Background()
	if opts.timeout == 0 {
		return executeSearchContext(ctx, opts, stdout, stderr)
	}
	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	return executeSearchContext(ctx, opts, stdout, stderr)
}

func executeSearchContext(ctx context.Context, opts options, stdout, stderr io.Writer) (bool, error) {
	output, err := openSearchOutput(opts, stdout)
	if err != nil {
		if opts.json {
			result := newJSONResult()
			result.Errors = append(result.Errors, err.Error())
			err = errors.Join(err, writeJSON(stdout, result))
		}
		return false, err
	}
	if opts.json {
		output.result = newJSONResult()
	}
	matched, searchErr := searchPath(ctx, opts.path, output, stderr, opts.pattern, opts.noIgnore)
	if searchErr == nil {
		searchErr = ctx.Err()
	}
	timedOut := errors.Is(searchErr, context.DeadlineExceeded)
	if timedOut {
		searchErr = fmt.Errorf("search time limit exceeded; partial results retained; rerun with --timeout 10s or --timeout 0: %w", searchErr)
	}
	err = errors.Join(searchErr, output.finish())
	if output.result != nil {
		output.result.Complete = err == nil
		output.result.TimedOut = timedOut
		if err != nil {
			output.result.Errors = append(output.result.Errors, err.Error())
		}
		err = errors.Join(err, writeJSON(stdout, output.result))
	}
	return matched, err
}

func openMatchOutput(terminal io.Writer, csvPath string) (*matchOutput, error) {
	output := &matchOutput{terminal: terminal}
	if csvPath == "" {
		return output, nil
	}
	file, err := os.OpenFile(csvPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("create CSV export: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	output.file, output.fileInfo = file, info
	output.csv = csv.NewWriter(file)
	if err := output.csv.Write([]string{"filename", "line_number", "text"}); err != nil {
		file.Close()
		return nil, fmt.Errorf("write CSV header: %w", err)
	}
	return output, nil
}

func (o *matchOutput) writeCSV(filename string, lineNumber int, text string) error {
	if o.csv == nil {
		return nil
	}
	if err := o.csv.Write([]string{filename, strconv.Itoa(lineNumber), text}); err != nil {
		return fmt.Errorf("write CSV: %w", err)
	}
	return nil
}

func (o *matchOutput) finish() error {
	if o.csv == nil {
		return nil
	}
	o.csv.Flush()
	if err := errors.Join(o.csv.Error(), o.file.Close()); err != nil {
		return fmt.Errorf("%w: finish CSV export: %w", errOutput, err)
	}
	return nil
}

// A CSV created inside the search tree must not become one of its own inputs.
func (o *matchOutput) isExport(filename string) bool {
	if o.fileInfo == nil {
		return false
	}
	info, err := os.Stat(filename)
	return err == nil && os.SameFile(o.fileInfo, info)
}

func openSearchOutput(opts options, stdout io.Writer) (*matchOutput, error) {
	mode := opts.hyperlinks
	if opts.json {
		mode = "never"
	}
	locations, err := newLocationFormatter(stdout, mode, os.Getenv("SIFT_EDITOR_URL"))
	if err != nil {
		return nil, err
	}
	output, err := openMatchOutput(stdout, opts.csvPath)
	if err != nil {
		return nil, err
	}
	output.locations = locations
	return output, nil
}
