package ingest

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestUpsertRemoteAlertSuppressedStaysFiring(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	_, ok, err := upsertRemoteAlert(s,
		map[string]string{"alertname": "Silenced", "severity": "critical", "service": "checkout"},
		map[string]string{"summary": "quiet"},
		"suppressed", time.Date(2026, 7, 31, 11, 0, 0, 0, time.UTC), "alertmanager")
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
