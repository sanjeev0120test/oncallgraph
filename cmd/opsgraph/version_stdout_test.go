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

func TestGraphMermaidWritesStdoutNotStderr(t *testing.T) {
	fx := fixtureDir(t)
	out, errOut, code := runRoot(t, "graph", "--fixture", fx, "--format", "mermaid")
	if code != 0 {
		t.Fatalf("graph exit = %d err=%q", code, errOut)
	}
	if strings.TrimSpace(errOut) != "" {
		t.Fatalf("graph wrote to stderr: %q", errOut)
	}
	if !strings.Contains(out, "flowchart") {
		t.Fatalf("graph stdout missing flowchart: %q", out)
	}
}

func TestAskRejectsNegativeSince(t *testing.T) {
	fx := fixtureDir(t)
	_, errOut, code := runRoot(t, "ask", "checkout", "--fixture", fx, "--since", "-1h")
	if code != 2 {
		t.Fatalf("exit=%d want 2 err=%q", code, errOut)
	}
}
