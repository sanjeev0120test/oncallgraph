//go:build ignore

// jsonassert validates JSON from stdin against simple key/path assertions.
// Usage:
//
//	go run scripts/jsonassert.go has:ai_summary startswith:ai_summary=AI unavailable
//	go run scripts/jsonassert.go islist
//	go run scripts/jsonassert.go has:version has:commit has:goos
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fatal("read stdin: %v", err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		fatal("invalid json: %v", err)
	}
	for _, arg := range os.Args[1:] {
		switch {
		case arg == "islist":
			if _, ok := v.([]any); !ok {
				fatal("want JSON array, got %T", v)
			}
		case strings.HasPrefix(arg, "has:"):
			key := strings.TrimPrefix(arg, "has:")
			m, ok := v.(map[string]any)
			if !ok {
				fatal("has:%s requires object, got %T", key, v)
			}
			if _, ok := m[key]; !ok {
				fatal("missing key %q", key)
			}
		case strings.HasPrefix(arg, "true:"):
			key := strings.TrimPrefix(arg, "true:")
			m, ok := v.(map[string]any)
			if !ok {
				fatal("true:%s requires object", key)
			}
			b, ok := m[key].(bool)
			if !ok || !b {
				fatal("key %q want true, got %#v", key, m[key])
			}
		case strings.HasPrefix(arg, "startswith:"):
			rest := strings.TrimPrefix(arg, "startswith:")
			key, prefix, ok := strings.Cut(rest, "=")
			if !ok {
				fatal("startswith wants key=prefix")
			}
			m, ok := v.(map[string]any)
			if !ok {
				fatal("startswith requires object")
			}
			s, ok := m[key].(string)
			if !ok || !strings.HasPrefix(s, prefix) {
				fatal("key %q does not start with %q (got %q)", key, prefix, s)
			}
		default:
			fatal("unknown assertion %q", arg)
		}
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "jsonassert: "+format+"\n", args...)
	os.Exit(1)
}
