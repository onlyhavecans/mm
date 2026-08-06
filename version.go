package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// version is stamped by goreleaser at release time.
var version = "dev"

// maybePrintVersion handles `--version` ahead of normal argument parsing.
func maybePrintVersion() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("%s %s\n", filepath.Base(os.Args[0]), version)
		os.Exit(0)
	}
}
