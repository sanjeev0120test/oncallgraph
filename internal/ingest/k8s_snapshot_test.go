package ingest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestK8sAllowIsPerService(t *testing.T) {
	root := t.TempDir()
	body := "" +
		"deployments:\n" +
		"  - name: checkout-api\n    service_id: checkout\n    namespace: prod\n    desired: 2\n    ready: 0\n" +
		"  - name: auth-api\n    service_id: auth\n    namespace: prod\n    desired: 1\n    ready: 0\n"
	dep := filepath.Join(root, "deployments.yaml")
	if err := os.WriteFile(dep, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ev := filepath.Join(root, "events.yaml")
	if err := os.WriteFile(ev, []byte("events: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	allow := k8sAllow{ByService: map[string]svcK8sAllow{
		"checkout": {
			Deployments: map[string]bool{"checkout-api": true},
			hasDeps:     true,
		},
		// auth listed with empty allow → block all auth deployments
		"auth": {Deployments: map[string]bool{}, hasDeps: true},
	}}
	if err := ingestK8sFiles(s, os.DirFS(root), "deployments.yaml", "events.yaml",
		time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC), allow); err != nil {
		t.Fatal(err)
	}
	checkout, err := s.GetService("checkout")
	if err != nil {
		t.Fatal(err)
	}
	if checkout.Health != "unhealthy" {
		t.Fatalf("checkout should ingest allowed deploy, health=%q", checkout.Health)
	}
	if _, err := s.GetService("auth"); err == nil {
		t.Fatal("auth deploy must be filtered by per-service allow")
	}
}

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
		if err := ingestK8sFiles(s, fsys, depFile, "events.yaml", time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC), k8sAllow{}); err != nil {
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
