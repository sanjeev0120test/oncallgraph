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
