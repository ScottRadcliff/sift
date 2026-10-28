package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type jsonMatch struct {
	Filename   string `json:"filename"`
	LineNumber int    `json:"line_number"`
	Text       string `json:"text"`
}

type jsonSkip struct {
	Filename string `json:"filename"`
	Reason   string `json:"reason"`
	Error    string `json:"error,omitempty"`
}

type jsonResult struct {
	Matches      []jsonMatch `json:"matches"`
	SkippedFiles []jsonSkip  `json:"skipped_files"`
	Errors       []string    `json:"errors"`
	Complete     bool        `json:"complete"`
}

func newJSONResult() *jsonResult {
	return &jsonResult{Matches: []jsonMatch{}, SkippedFiles: []jsonSkip{}, Errors: []string{}}
}

func writeJSON(output io.Writer, result *jsonResult) error {
	if err := json.NewEncoder(output).Encode(result); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}

func (o *matchOutput) recordSkip(filename, reason, detail string) {
	if o.result != nil {
		o.result.SkippedFiles = append(o.result.SkippedFiles, jsonSkip{filename, reason, detail})
	}
}

func (o *matchOutput) recordError(err error) {
	if o.result != nil {
		o.result.Errors = append(o.result.Errors, err.Error())
	}
}
