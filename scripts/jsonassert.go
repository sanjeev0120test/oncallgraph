//go:build ignore

// jsonassert validates JSON from stdin against simple key/path assertions.
// Usage:
//
//	go run scripts/jsonassert.go has:ai_summary startswith:ai_summary=AI unavailable
//	go run scripts/jsonassert.go islist
//	go run scripts/jsonassert.go has:version has:commit has:goos
//	go run scripts/jsonassert.go eq:service=checkout eqnum:total=4 gte:total=1
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
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
			m, ok := asObject(v)
			if !ok {
				fatal("has:%s requires object, got %T", key, v)
			}
			if _, ok := m[key]; !ok {
				fatal("missing key %q", key)
			}
		case strings.HasPrefix(arg, "true:"):
			key := strings.TrimPrefix(arg, "true:")
			m, ok := asObject(v)
			if !ok {
				fatal("true:%s requires object", key)
			}
			b, ok := m[key].(bool)
			if !ok || !b {
				fatal("key %q want true, got %#v", key, m[key])
			}
		case strings.HasPrefix(arg, "false:"):
			key := strings.TrimPrefix(arg, "false:")
			m, ok := asObject(v)
			if !ok {
				fatal("false:%s requires object", key)
			}
			b, ok := m[key].(bool)
			if !ok || b {
				fatal("key %q want false, got %#v", key, m[key])
			}
		case strings.HasPrefix(arg, "eq:"):
			rest := strings.TrimPrefix(arg, "eq:")
			key, want, ok := strings.Cut(rest, "=")
			if !ok {
				fatal("eq wants key=value")
			}
			m, ok := asObject(v)
			if !ok {
				fatal("eq requires object")
			}
			s, ok := m[key].(string)
			if !ok || s != want {
				fatal("key %q want %q, got %#v", key, want, m[key])
			}
		case strings.HasPrefix(arg, "eqnum:"):
			rest := strings.TrimPrefix(arg, "eqnum:")
			key, wantStr, ok := strings.Cut(rest, "=")
			if !ok {
				fatal("eqnum wants key=number")
			}
			want, err := strconv.ParseFloat(wantStr, 64)
			if err != nil {
				fatal("eqnum invalid number %q", wantStr)
			}
			m, ok := asObject(v)
			if !ok {
				fatal("eqnum requires object")
			}
			got, ok := asFloat(m[key])
			if !ok || got != want {
				fatal("key %q want %v, got %#v", key, want, m[key])
			}
		case strings.HasPrefix(arg, "gte:"):
			rest := strings.TrimPrefix(arg, "gte:")
			key, wantStr, ok := strings.Cut(rest, "=")
			if !ok {
				fatal("gte wants key=number")
			}
			want, err := strconv.ParseFloat(wantStr, 64)
			if err != nil {
				fatal("gte invalid number %q", wantStr)
			}
			m, ok := asObject(v)
			if !ok {
				fatal("gte requires object")
			}
			got, ok := asFloat(m[key])
			if !ok || got < want {
				fatal("key %q want >= %v, got %#v", key, want, m[key])
			}
		case strings.HasPrefix(arg, "startswith:"):
			rest := strings.TrimPrefix(arg, "startswith:")
			key, prefix, ok := strings.Cut(rest, "=")
			if !ok {
				fatal("startswith wants key=prefix")
			}
			m, ok := asObject(v)
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

func asObject(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "jsonassert: "+format+"\n", args...)
	os.Exit(1)
}
