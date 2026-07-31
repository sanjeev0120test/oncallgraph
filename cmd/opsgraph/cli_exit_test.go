package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	// Tests run with CWD = package dir (cmd/opsgraph); fixtures live at repo root.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "fixtures", "incident_checkout")); err != nil {
		t.Fatalf("repo root %q missing fixtures: %v", root, err)
	}
	return root
}

func fixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "fixtures", "incident_checkout")
}

func runRoot(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	root := newRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errBuf.String(), exitCodeFor(err)
}

func TestCLIExitAskCheckoutOK(t *testing.T) {
	_, _, code := runRoot(t, "ask", "checkout", "--fixture", fixtureDir(t), "--format", "json")
	if code != 0 {
		t.Fatalf("ask checkout exit = %d, want 0 (stale runbook must not fail ask)", code)
	}
}

func TestCLIExitAskUnknownService(t *testing.T) {
	_, _, code := runRoot(t, "ask", "nosuch", "--fixture", fixtureDir(t))
	if code != 1 {
		t.Fatalf("ask nosuch exit = %d, want 1", code)
	}
}

func TestCLIExitAskNoSource(t *testing.T) {
	// No --fixture and no .opsgraph.yaml in the test CWD (cmd/opsgraph).
	_, _, code := runRoot(t, "ask", "checkout")
	if code != 2 {
		t.Fatalf("ask without source exit = %d, want 2", code)
	}
}

func TestCLIExitVerifyStale(t *testing.T) {
	_, _, code := runRoot(t, "verify-runbook", "checkout", "--fixture", fixtureDir(t))
	if code != 1 {
		t.Fatalf("verify checkout exit = %d, want 1", code)
	}
}

func TestCLIExitVerifyPass(t *testing.T) {
	_, _, code := runRoot(t, "verify-runbook", "auth", "--fixture", fixtureDir(t))
	if code != 0 {
		t.Fatalf("verify auth exit = %d, want 0", code)
	}
}

func TestCLIExitVerifyMissing(t *testing.T) {
	_, _, code := runRoot(t, "verify-runbook", "order", "--fixture", fixtureDir(t))
	if code != 2 {
		t.Fatalf("verify order (missing runbook) exit = %d, want 2", code)
	}
}

func TestCLIExitVerifyUnknownService(t *testing.T) {
	_, _, code := runRoot(t, "verify-runbook", "nosuch", "--fixture", fixtureDir(t))
	if code != 1 {
		t.Fatalf("verify nosuch exit = %d, want 1", code)
	}
}

func TestCLIExitVerifyRunbookFile(t *testing.T) {
	fx := fixtureDir(t)
	path := filepath.Join(fx, "runbooks", "auth.md")
	_, _, code := runRoot(t, "verify-runbook", path, "--fixture", fx)
	if code != 0 {
		t.Fatalf("verify file %s exit = %d, want 0", path, code)
	}
}

func TestCLIExitDemo(t *testing.T) {
	_, _, code := runRoot(t, "demo", "--format", "json")
	if code != 0 {
		t.Fatalf("demo exit = %d, want 0", code)
	}
}
