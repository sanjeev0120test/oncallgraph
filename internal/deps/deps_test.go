// Package deps holds cross-cutting build-invariant tests. It has no runtime code.
package deps

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultBuildHasNoK8sIO asserts the default build does not link any
// k8s.io/* package. The v1 Kubernetes connector is a pure-Go snapshot parser;
// client-go is intentionally not a dependency (see PLAN.md).
func TestDefaultBuildHasNoK8sIO(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	root := moduleRoot(t)

	cmd := exec.Command("go", "list", "-deps", "./cmd/oncallgraph")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps failed: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "k8s.io/") {
			t.Fatalf("default build unexpectedly links %q; client-go must stay out of the default build", line)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root (go.mod)")
		}
		dir = parent
	}
}
