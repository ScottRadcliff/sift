package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func reportSkippedBinary(output io.Writer, filename string) {
	message := fmt.Sprintf("sift: skipped binary file: %s", filename)
	file, isFile := output.(*os.File)
	if isFile && term.IsTerminal(int(file.Fd())) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" {
		message = "\x1b[33m" + message + "\x1b[0m"
	}
	fmt.Fprintln(output, message)
}
