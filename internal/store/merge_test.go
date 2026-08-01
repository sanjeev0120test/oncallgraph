package store_test

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestMergeFromUpsertsAndResolvesZombies(t *testing.T) {
	dst, cleanupDst, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupDst)
	src, cleanupSrc, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupSrc)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	svc := model.Service{ID: "checkout", Name: "checkout", Health: model.HealthDegraded, Sources: []string{"config"}}
	if err := dst.UpsertService(svc); err != nil {
		t.Fatal(err)
	}
	if err := dst.UpsertAlert(model.Alert{
		ID: "am-old", ServiceID: "checkout", At: now.Add(-time.Hour),
		Name: "Gone", Status: "firing", Severity: "critical", Source: "alertmanager",
	}); err != nil {
		t.Fatal(err)
	}
	if err := src.UpsertService(svc); err != nil {
		t.Fatal(err)
	}
	if err := src.UpsertAlert(model.Alert{
		ID: "am-new", ServiceID: "checkout", At: now,
		Name: "Still", Status: "firing", Severity: "warning", Source: "alertmanager",
	}); err != nil {
		t.Fatal(err)
	}
	if err := src.SetMeta("connector:alertmanager", "ok"); err != nil {
		t.Fatal(err)
	}
	if err := dst.MergeFrom(src); err != nil {
		t.Fatal(err)
	}
	alerts, err := dst.ListAllAlerts()
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]string{}
	for _, a := range alerts {
		byID[a.ID] = a.Status
	}
	if byID["am-new"] != "firing" {
		t.Fatalf("merged alert missing: %+v", byID)
	}
	if byID["am-old"] != "resolved" {
		t.Fatalf("zombie should resolve: %+v", byID)
	}
	if v, ok, err := dst.GetMeta("connector:alertmanager"); err != nil || !ok || v != "ok" {
		t.Fatalf("meta copy: v=%q ok=%v err=%v", v, ok, err)
	}
}

func TestMergeFromQuietConnectorDoesNotWipeFirings(t *testing.T) {
	dst, cleanupDst, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupDst)
	src, cleanupSrc, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupSrc)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	svc := model.Service{ID: "checkout", Name: "checkout", Health: model.HealthDegraded}
	if err := dst.UpsertService(svc); err != nil {
		t.Fatal(err)
	}
	if err := dst.UpsertAlert(model.Alert{
		ID: "still-firing", ServiceID: "checkout", At: now.Add(-time.Hour),
		Name: "Keep", Status: "firing", Source: "prometheus",
	}); err != nil {
		t.Fatal(err)
	}
	if err := src.UpsertService(svc); err != nil {
		t.Fatal(err)
	}
	if err := src.SetMeta("connector:prometheus", "ok"); err != nil {
		t.Fatal(err)
	}
	if err := dst.MergeFrom(src); err != nil {
		t.Fatal(err)
	}
	alerts, err := dst.ListAllAlerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Status != "firing" {
		t.Fatalf("quiet merge must keep dst firing: %+v", alerts)
	}
}

func TestMergeFromPrunesRemovedTopology(t *testing.T) {
	dst, cleanupDst, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupDst)
	src, cleanupSrc, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupSrc)
	for _, id := range []string{"checkout", "auth", "payments"} {
		if err := dst.UpsertService(model.Service{ID: id, Name: id, Health: model.HealthUnknown}); err != nil {
			t.Fatal(err)
		}
		if err := src.UpsertService(model.Service{ID: id, Name: id, Health: model.HealthUnknown}); err != nil {
			t.Fatal(err)
		}
	}
	if err := dst.UpsertDependency(model.Dependency{
		FromServiceID: "checkout", ToServiceID: "auth", Type: "depends_on", Source: "config",
	}); err != nil {
		t.Fatal(err)
	}
	if err := dst.UpsertDependency(model.Dependency{
		FromServiceID: "checkout", ToServiceID: "payments", Type: "depends_on", Source: "config",
	}); err != nil {
		t.Fatal(err)
	}
	// Current config only keeps checkout→auth.
	if err := src.UpsertDependency(model.Dependency{
		FromServiceID: "checkout", ToServiceID: "auth", Type: "depends_on", Source: "config",
	}); err != nil {
		t.Fatal(err)
	}
	if err := src.SetMeta("topology:seeded", "ok"); err != nil {
		t.Fatal(err)
	}
	if err := dst.MergeFrom(src); err != nil {
		t.Fatal(err)
	}
	deps, err := dst.ListAllDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 1 || deps[0].ToServiceID != "auth" {
		t.Fatalf("expected only checkout→auth after prune, got %+v", deps)
	}
	if v, ok, err := dst.GetMeta("topology:seeded"); err != nil || !ok || v != "ok" {
		t.Fatalf("topology:seeded meta should copy: v=%q ok=%v err=%v", v, ok, err)
	}
}

func TestMergeFromClearsStaleK8sHealth(t *testing.T) {
	dst, cleanupDst, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupDst)
	src, cleanupSrc, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupSrc)
	if err := dst.UpsertService(model.Service{
		ID: "checkout", Name: "checkout", Health: model.HealthUnhealthy, Sources: []string{"kubernetes", "config"},
	}); err != nil {
		t.Fatal(err)
	}
	// Fresh scrape: service still seeded, but left the k8s snapshot.
	if err := src.UpsertService(model.Service{
		ID: "checkout", Name: "checkout", Health: model.HealthUnknown, Sources: []string{"config"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := src.SetMeta("connector:kubernetes", "ok"); err != nil {
		t.Fatal(err)
	}
	if err := src.SetMeta("topology:seeded", "ok"); err != nil {
		t.Fatal(err)
	}
	if err := dst.MergeFrom(src); err != nil {
		t.Fatal(err)
	}
	svc, err := dst.GetService("checkout")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Health != model.HealthUnknown {
		t.Fatalf("stale k8s health should clear when absent from scrape: %+v", svc)
	}
	for _, s := range svc.Sources {
		if s == "kubernetes" {
			t.Fatalf("kubernetes source should drop after leaving snapshot: %v", svc.Sources)
		}
	}
}

func TestMergeFromPreservesDestinationHealth(t *testing.T) {
	dst, cleanupDst, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupDst)
	src, cleanupSrc, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupSrc)
	if err := dst.UpsertService(model.Service{
		ID: "checkout", Name: "Checkout", Health: model.HealthDegraded, Sources: []string{"kubernetes"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := src.UpsertService(model.Service{
		ID: "checkout", Name: "checkout", Health: model.HealthUnknown, Sources: []string{"config"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := dst.MergeFrom(src); err != nil {
		t.Fatal(err)
	}
	svc, err := dst.GetService("checkout")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Health != model.HealthDegraded {
		t.Fatalf("merge wiped health: %+v", svc)
	}
	seen := map[string]bool{}
	for _, s := range svc.Sources {
		seen[s] = true
	}
	if !seen["kubernetes"] || !seen["config"] {
		t.Fatalf("sources not merged: %v", svc.Sources)
	}
}
