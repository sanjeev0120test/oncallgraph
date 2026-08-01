package ingest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestOpenK8sSnapshotFSDirAndFile(t *testing.T) {
	root := t.TempDir()
	k8s := filepath.Join(root, "k8s")
	if err := os.MkdirAll(k8s, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "deployments:\n  - name: checkout\n    service_id: checkout\n    desired: 1\n    ready: 1\n"
	dep := filepath.Join(k8s, "deployments.yaml")
	if err := os.WriteFile(dep, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, snap := range []string{k8s, dep} {
		fsys, depFile, _, _, err := openK8sSnapshotFS(snap)
		if err != nil {
			t.Fatalf("snap %q: %v", snap, err)
		}
		s, cleanup, err := store.OpenTemp()
		if err != nil {
			t.Fatal(err)
		}
		if err := ingestK8sFiles(s, fsys, depFile, "events.yaml", time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)); err != nil {
			cleanup()
			t.Fatalf("ingest via %q: %v", snap, err)
		}
		svc, err := s.GetService("checkout")
		cleanup()
		if err != nil {
			t.Fatalf("service via %q: %v", snap, err)
		}
		if svc.Health != "healthy" {
			t.Fatalf("snap %q: health=%q", snap, svc.Health)
		}
	}
}
