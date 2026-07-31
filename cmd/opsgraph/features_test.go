package main

import (
	"strings"
	"testing"
)

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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut, code := runRoot(t, tc.args...)
			if code != tc.code {
				t.Fatalf("exit=%d want %d\nout=%s\nerr=%s", code, tc.code, out, errOut)
			}
			combined := out + errOut
			if tc.want != "" && !strings.Contains(combined, tc.want) {
				t.Fatalf("output missing %q:\n%s", tc.want, combined)
			}
		})
	}
}
