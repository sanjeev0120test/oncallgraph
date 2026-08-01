package ingest

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestSlugEmptyFallsBack(t *testing.T) {
	if got := slug("@@@"); got != "unknown" {
		t.Fatalf("slug exotic = %q", got)
	}
	if got := slug("Checkout_API"); got != "checkout-api" {
		t.Fatalf("slug normal = %q", got)
	}
}

func TestUpsertRemoteAlertInactiveResolved(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	_, ok, err := upsertRemoteAlert(s,
		map[string]string{"alertname": "WasFiring", "severity": "warning", "service": "checkout"},
		map[string]string{"summary": "done"},
		"inactive", now, "prometheus", now)
	if err != nil || !ok {
		t.Fatalf("upsert: ok=%v err=%v", ok, err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Status != "resolved" {
		t.Fatalf("inactive should resolve, got %+v", alerts)
	}
}

func TestUpsertRemoteAlertUnknownStateSkipped(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	_, ok, err := upsertRemoteAlert(s,
		map[string]string{"alertname": "Weird", "severity": "warning", "service": "checkout"},
		map[string]string{"summary": "x"},
		"weird-state", now, "prometheus", now)
	if err != nil || ok {
		t.Fatalf("unknown state must skip: ok=%v err=%v", ok, err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("unknown state must not invent alerts: %+v", alerts)
	}
}

func TestUpsertRemoteAlertAmbiguousAliasSkipped(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertService(model.Service{ID: "a", Name: "A", Aliases: []string{"shared"}, Health: "healthy"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertService(model.Service{ID: "b", Name: "B", Aliases: []string{"shared"}, Health: "healthy"}); err != nil {
		t.Fatal(err)
	}
	_, ok, err := upsertRemoteAlert(s,
		map[string]string{"alertname": "X", "severity": "warning", "service": "shared"},
		map[string]string{"summary": "x"},
		"firing", now, "prometheus", now)
	if err != nil || ok {
		t.Fatalf("ambiguous must skip: ok=%v err=%v", ok, err)
	}
}

func TestResolveActiveAlertsNotInClearsSuppressed(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	id, ok, err := upsertRemoteAlert(s,
		map[string]string{"alertname": "Silenced", "severity": "critical", "service": "checkout"},
		map[string]string{"summary": "quiet"},
		"suppressed", now.Add(-time.Hour), "alertmanager", now)
	if err != nil || !ok {
		t.Fatalf("upsert: ok=%v err=%v", ok, err)
	}
	n, err := s.ResolveActiveAlertsNotIn("alertmanager", map[string]bool{})
	if err != nil || n != 1 {
		t.Fatalf("resolve suppressed: n=%d err=%v", n, err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Status != "resolved" || alerts[0].ID != id {
		t.Fatalf("suppressed zombie must resolve: %+v", alerts)
	}
}

func TestUpsertRemoteAlertSuppressedNotActive(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	_, ok, err := upsertRemoteAlert(s,
		map[string]string{"alertname": "Silenced", "severity": "critical", "service": "checkout"},
		map[string]string{"summary": "quiet"},
		"suppressed", time.Date(2026, 7, 31, 11, 0, 0, 0, time.UTC), "alertmanager", now)
	if err != nil || !ok {
		t.Fatalf("upsert: ok=%v err=%v", ok, err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Status != "suppressed" {
		t.Fatalf("suppressed should keep status, got %+v", alerts)
	}
	if model.AlertActive(alerts[0].Status) {
		t.Fatal("suppressed must not count as active for R3/score")
	}
}
