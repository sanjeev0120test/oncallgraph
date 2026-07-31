package main

import (
	"strings"
	"testing"
)

func TestVersionWritesStdoutNotStderr(t *testing.T) {
	out, errOut, code := runRoot(t, "version")
	if code != 0 {
		t.Fatalf("version exit = %d", code)
	}
	if strings.TrimSpace(errOut) != "" {
		t.Fatalf("version wrote to stderr: %q", errOut)
	}
	if !strings.Contains(out, "opsgraph") {
		t.Fatalf("version stdout missing opsgraph: %q", out)
	}
}
