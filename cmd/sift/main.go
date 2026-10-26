package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] != "--help" && os.Args[1] != "-h" {
		fmt.Fprintln(os.Stderr, "sift: searching is not implemented yet")
		os.Exit(2)
	}
	fmt.Println("Usage: sift 'search' path\n\nSift searches text files. Search support is coming next.")
}
