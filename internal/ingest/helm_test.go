package ingest_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/store"
)

func TestIngestHelmReleases(t *testing.T) {
	dir := t.TempDir()
	content := `releases:
  - name: checkout
    service_id: checkout
    chart: checkout
    version: 1.4.2
    revision: 7
    updated_at: 2026-07-31T11:38:00Z
`
	if err := os.WriteFile(filepath.Join(dir, "releases.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)

	// Use LiveIngest path via public IngestFixtureDir-style: call through kubernetes live helper.
	// Direct path: ingest via LiveIngest with a minimal config pointing at this snapshot.
	cfgPath := filepath.Join(dir, "cfg.yaml")
	cfg := `version: 1
services:
  checkout:
    owner: payments
connectors:
  git:
    enabled: false
  kubernetes:
    enabled: true
    snapshot: .
  prometheus:
    enabled: false
  alertmanager:
    enabled: false
owners:
  payments:
    name: Payments
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	cfgLoaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := ingest.LiveIngest(s, cfgLoaded, dir, now.Add(-time.Hour), now); err != nil {
		t.Fatal(err)
	}
	changes, err := s.ListChanges("checkout", now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range changes {
		if c.Source == "helm" && c.Type == "deploy" && c.Revision == "1.4.2" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("helm deploy not found: %+v", changes)
	}
}
