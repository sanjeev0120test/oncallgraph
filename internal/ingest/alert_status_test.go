package ingest

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/store"
)

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

func TestUpsertRemoteAlertSuppressedStaysFiring(t *testing.T) {
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
	if len(alerts) != 1 || alerts[0].Status != "firing" {
		t.Fatalf("suppressed should stay firing, got %+v", alerts)
	}
}
