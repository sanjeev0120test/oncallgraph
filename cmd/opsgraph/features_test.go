package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportCommand(t *testing.T) {
	fx := fixtureDir(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "checkout.md")
	_, _, code := runRoot(t, "export", "checkout", "--fixture", fx, "--format", "markdown", "--out", out)
	if code != 0 {
		t.Fatalf("export exit=%d", code)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "# Incident report") {
		t.Fatalf("bad export: %s", b)
	}
}

func TestNewFeatureCommands(t *testing.T) {
	fx := fixtureDir(t)
	cases := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"services", []string{"services", "--fixture", fx, "--format", "json"}, 0, "checkout"},
		{"owners", []string{"owners", "--fixture", fx, "--format", "json"}, 0, "payments"},
		{"graph", []string{"graph", "--fixture", fx}, 0, "checkout"},
		{"graph-mermaid", []string{"graph", "--fixture", fx, "--format", "mermaid"}, 0, "flowchart"},
		{"evidence", []string{"evidence", "ev-change-1", "--fixture", fx}, 0, "ev-change-1"},
		{"explain", []string{"explain", "checkout", "--fixture", fx}, 0, "Prime suspect"},
		{"report", []string{"report", "checkout", "--fixture", fx}, 0, "# Incident report"},
		{"score", []string{"score", "checkout", "--fixture", fx, "--format", "json"}, 0, "score"},
		{"who", []string{"who", "checkout", "--fixture", fx}, 0, "Payments"},
		{"compare", []string{"compare", "checkout", "auth", "--fixture", fx}, 0, "score"},
		{"timeline", []string{"timeline", "checkout", "--fixture", fx}, 0, "health"},
		{"path", []string{"path", "order", "auth", "--fixture", fx}, 0, "checkout"},
		{"blast", []string{"blast", "checkout", "--fixture", fx}, 0, "UPSTREAM"},
		{"watch-timeout", []string{"watch", "checkout", "--fixture", fx, "--interval", "1ms", "--timeout", "5ms"}, 1, "degraded"},
		{"doctor", []string{"doctor"}, 0, "summary:"},
		{"validate", []string{"validate-fixture", fx}, 0, "valid"},
		{"completion", []string{"completion", "bash"}, 0, "opsgraph"},
		{"health", []string{"health", "--fixture", fx, "--format", "json"}, 0, "degraded"},
		{"top", []string{"top", "--fixture", fx, "--format", "json"}, 0, "checkout"},
		{"resolve", []string{"resolve", "checkout-api", "--fixture", fx}, 0, "checkout"},
		{"changes", []string{"changes", "--fixture", fx, "--service", "checkout"}, 0, "deploy"},
		{"alerts", []string{"alerts", "--fixture", fx, "--firing"}, 0, "CheckoutErrorRateHigh"},
		{"impact", []string{"impact", "auth", "--fixture", fx}, 0, "checkout"},
		{"fingerprint", []string{"fingerprint", "checkout", "--fixture", fx}, 0, "inc_"},
		{"why", []string{"why", "checkout", "--fixture", fx}, 0, "prime suspect"},
		{"handoff", []string{"handoff", "checkout", "--fixture", fx}, 0, "# Handoff"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut, code := runRoot(t, tc.args...)
			if code != tc.code {
				t.Fatalf("exit=%d want %d\nout=%s\nerr=%s", code, tc.code, out, errOut)
			}
			if tc.want == "" {
				return
			}
			// Success-path machine output must land on stdout (redirects/CI).
			haystack := out
			if tc.code != 0 {
				haystack = out + errOut
			}
			if !strings.Contains(haystack, tc.want) {
				t.Fatalf("output missing %q:\nstdout=%s\nstderr=%s", tc.want, out, errOut)
			}
		})
	}
}
