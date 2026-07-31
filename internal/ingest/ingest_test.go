package ingest_test

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/opsgraph/opsgraph/fixtures"
	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/store"
)

func TestIngestCheckoutFixture(t *testing.T) {
	fsys, err := fixtures.CheckoutFS()
	if err != nil {
		t.Fatalf("CheckoutFS: %v", err)
	}
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	t.Cleanup(cleanup)

	now, err := ingest.IngestFixtureFS(s, fsys)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	want := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if !now.Equal(want) {
		t.Fatalf("fixture now = %v, want %v", now, want)
	}

	svc, err := s.GetServiceByNameOrAlias("checkout-api")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if svc.ID != "checkout" || svc.Health != "degraded" {
		t.Fatalf("checkout resolved wrong: %+v", svc)
	}

	changes, err := s.ListChanges("checkout", now.Add(-60*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 { // fixture deploy + k8s rollout, both within window
		t.Fatalf("want 2 in-window changes, got %d: %+v", len(changes), changes)
	}

	alerts, err := s.ListAlerts("checkout", now.Add(-60*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Status != "firing" {
		t.Fatalf("alerts wrong: %+v", alerts)
	}

	deps, err := s.ListDependencies("checkout")
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 3 {
		t.Fatalf("want 3 dependency edges, got %d: %+v", len(deps), deps)
	}

	rb, err := s.GetRunbook("checkout")
	if err != nil {
		t.Fatalf("runbook: %v", err)
	}
	if len(rb.Steps) != 3 {
		t.Fatalf("want 3 runbook steps, got %d: %+v", len(rb.Steps), rb.Steps)
	}
	if rb.Steps[0].Check != "deploy_age_lt:60m" || rb.Steps[1].Check != "service_healthy:checkout" || rb.Steps[2].Check != "manual" {
		t.Fatalf("runbook checks parsed wrong: %+v", rb.Steps)
	}

	// Auth health should have been set unhealthy by the snapshot/services.
	auth, err := s.GetService("auth")
	if err != nil || auth.Health != "unhealthy" {
		t.Fatalf("auth health wrong: %+v (%v)", auth, err)
	}

	ev, err := s.ListEvidence([]string{"ev-change-1", "ev-alert-1", "ev-k8s-rollout-checkout"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 3 {
		t.Fatalf("want 3 evidence rows, got %d: %+v", len(ev), ev)
	}
}

func TestIngestFixtureDirAbsolutePath(t *testing.T) {
	// Exercises OS-native absolute paths (incl. Windows backslashes via Abs).
	rel := filepath.Join("..", "..", "fixtures", "incident_checkout")
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatal(err)
	}
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now, err := ingest.IngestFixtureDir(s, abs)
	if err != nil {
		t.Fatalf("IngestFixtureDir(%q) on %s: %v", abs, runtime.GOOS, err)
	}
	want := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if !now.Equal(want) {
		t.Fatalf("fixture now = %v, want %v", now, want)
	}
	svc, err := s.GetServiceByNameOrAlias("checkout-api")
	if err != nil {
		t.Fatalf("resolve alias after abs-path ingest: %v", err)
	}
	if svc.ID != "checkout" {
		t.Fatalf("got %q", svc.ID)
	}
}
