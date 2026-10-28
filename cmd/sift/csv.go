package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

type matchOutput struct {
	terminal io.Writer
	csv      *csv.Writer
	file     *os.File
	fileInfo os.FileInfo
}

func executeSearch(opts options, stdout, stderr io.Writer) (bool, error) {
	output, err := openMatchOutput(stdout, opts.csvPath)
	if err != nil {
		return false, err
	}
	matched, searchErr := searchPath(opts.path, output, stderr, opts.pattern, opts.noIgnore)
	return matched, errors.Join(searchErr, output.finish())
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
		return fmt.Errorf("finish CSV export: %w", err)
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
