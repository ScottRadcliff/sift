package main

import (
	"context"
	"io"
)

// Check the budget between buffered reads, including reads within a long line.
// An OS read already in progress must return before cancellation is observed.
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
