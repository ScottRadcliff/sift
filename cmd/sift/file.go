package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
)

func searchFile(filename string, output *matchOutput, diagnostics io.Writer, pattern *regexp.Regexp) (bool, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s: expected a regular file", filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()
	reader := bufio.NewReaderSize(file, 8192)
	prefix, err := reader.Peek(8192)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("%s: %w", filename, err)
	}
	if bytes.IndexByte(prefix, 0) >= 0 {
		output.recordSkip(filename, "binary", "")
		reportSkippedBinary(diagnostics, filename)
		return false, nil
	}
	matched, err := search(reader, output, pattern, filename)
	if err != nil {
		return matched, fmt.Errorf("%s: %w", filename, err)
	}
	return matched, nil
}
