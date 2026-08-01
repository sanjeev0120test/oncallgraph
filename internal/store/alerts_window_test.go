package store_test

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestListAlertsKeepsActiveOutsideSince(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertService(model.Service{ID: "checkout", Name: "checkout", Health: model.HealthDegraded}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertAlert(model.Alert{
		ID: "old-fire", ServiceID: "checkout", At: now.Add(-3 * time.Hour),
		Name: "OldButFiring", Status: "firing", Severity: "critical", Source: "prometheus",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertAlert(model.Alert{
		ID: "old-resolved", ServiceID: "checkout", At: now.Add(-3 * time.Hour),
		Name: "OldResolved", Status: "resolved", Severity: "warning", Source: "prometheus",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertAlert(model.Alert{
		ID: "old-silenced", ServiceID: "checkout", At: now.Add(-3 * time.Hour),
		Name: "OldSilenced", Status: "suppressed", Severity: "critical", Source: "alertmanager",
	}); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("want firing+suppressed outside since, got %+v", alerts)
	}
	seen := map[string]bool{}
	for _, a := range alerts {
		seen[a.ID] = true
	}
	if !seen["old-fire"] || !seen["old-silenced"] || seen["old-resolved"] {
		t.Fatalf("unexpected alert set: %+v", alerts)
	}
}
