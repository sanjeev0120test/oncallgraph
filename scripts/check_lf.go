//go:build ignore

// check_lf fails if any listed paths contain CR bytes (CRLF drift).
// Usage: go run ./scripts/check_lf.go fixtures/incident_checkout/expected fixtures/fleet_healthy/expected
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fatal("usage: check_lf <dir|file>...")
	}
	var bad []string
	for _, root := range os.Args[1:] {
		info, err := os.Stat(root)
		if err != nil {
			fatal("%s: %v", root, err)
		}
		if !info.IsDir() {
			if hasCR(root) {
				bad = append(bad, root)
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".json", ".yaml", ".yml", ".md", ".go":
			default:
				return nil
			}
			if hasCR(path) {
				bad = append(bad, path)
			}
			return nil
		})
		if err != nil {
			fatal("walk %s: %v", root, err)
		}
	}
	if len(bad) > 0 {
		fmt.Fprintln(os.Stderr, "check_lf: CRLF (CR bytes) found in:")
		for _, p := range bad {
			fmt.Fprintln(os.Stderr, " ", p)
		}
		os.Exit(1)
	}
}

func hasCR(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		fatal("read %s: %v", path, err)
	}
	return bytesIndexByte(b, '\r') >= 0
}

func bytesIndexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "check_lf: "+format+"\n", args...)
	os.Exit(1)
}
