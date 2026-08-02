package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvNamesFrozen(t *testing.T) {
	// Public env contract — renaming breaks automation / docs / installers.
	want := []string{
		"OPSGRAPH_CONFIG",
		"OPSGRAPH_DATA_DIR",
		"OPSGRAPH_FIXTURE",
	}
	for _, name := range want {
		if !strings.HasPrefix(name, "OPSGRAPH_") {
			t.Fatalf("bad env name %q", name)
		}
	}
	t.Setenv("OPSGRAPH_FIXTURE", fixtureDir(t))
	t.Setenv("OPSGRAPH_DATA_DIR", "")
	t.Setenv("OPSGRAPH_CONFIG", "")
	out, _, code := runRoot(t, "doctor")
	if code != 0 {
		t.Fatalf("doctor exit = %d", code)
	}
	if !strings.Contains(out, "env_OPSGRAPH_FIXTURE") {
		t.Fatalf("doctor missing env_OPSGRAPH_FIXTURE check:\n%s", out)
	}
}

func TestEnvFixtureAndDataDirExclusive(t *testing.T) {
	fx := fixtureDir(t)
	dir := t.TempDir()
	t.Setenv("OPSGRAPH_FIXTURE", fx)
	t.Setenv("OPSGRAPH_DATA_DIR", dir)
	t.Setenv("OPSGRAPH_CONFIG", "")
	_, _, code := runRoot(t, "ask", "checkout", "--format", "json")
	if code != 2 {
		t.Fatalf("OPSGRAPH_FIXTURE+OPSGRAPH_DATA_DIR exit = %d, want 2", code)
	}
	_, _, code = runRoot(t, "status")
	if code != 2 {
		t.Fatalf("status env fixture+data-dir exit = %d, want 2", code)
	}
}

func TestEnvFixtureFlagPrecedence(t *testing.T) {
	fx := fixtureDir(t)
	t.Setenv("OPSGRAPH_FIXTURE", filepath.Join(os.TempDir(), "opsgraph-missing-fixture"))
	t.Setenv("OPSGRAPH_DATA_DIR", "")
	t.Setenv("OPSGRAPH_CONFIG", "")
	_, _, code := runRoot(t, "ask", "checkout", "--fixture", fx, "--format", "json")
	if code != 0 {
		t.Fatalf("flag --fixture should win over bad OPSGRAPH_FIXTURE: exit=%d", code)
	}
}
