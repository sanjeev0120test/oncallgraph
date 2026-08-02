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

	cmd := exec.Command("go", "list", "-deps", "./cmd/opsgraph")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
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

// TestGoModHygiene forbids replace/exclude and direct k8s.io requires.
// Enterprise offline CLIs must stay on a clean, portable module graph.
func TestGoModHygiene(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(raw), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "replace ") || trim == "replace (" {
			t.Fatalf("go.mod:%d: replace directives are forbidden", i+1)
		}
		if strings.HasPrefix(trim, "exclude ") || trim == "exclude (" {
			t.Fatalf("go.mod:%d: exclude directives are forbidden", i+1)
		}
		if strings.Contains(trim, "k8s.io/") {
			t.Fatalf("go.mod:%d: k8s.io modules are forbidden in the default module graph", i+1)
		}
	}
}

// TestCmdOpsgraphIsPureGo asserts no package in the default link graph has
// CgoFiles, so CGO_ENABLED=0 cross-builds stay valid on every OS.
func TestCmdOpsgraphIsPureGo(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	root := moduleRoot(t)
	// Force CGO off even when the race job sets CGO_ENABLED=1.
	cmd := exec.Command("go", "list", "-deps", "-f", "{{if .CgoFiles}}{{.ImportPath}}{{end}}", "./cmd/opsgraph")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	if hit := strings.TrimSpace(string(out)); hit != "" {
		t.Fatalf("default build links cgo packages (want pure-Go):\n%s", hit)
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
