package store_test

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestListAlertsOrdersFiringBeforePending(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertAlert(model.Alert{
		ID: "p", ServiceID: "checkout", At: now, Name: "PendingOne", Status: "pending", Source: "prometheus",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertAlert(model.Alert{
		ID: "f", ServiceID: "checkout", At: now.Add(-time.Minute), Name: "FiringOne", Status: "firing", Source: "prometheus",
	}); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) < 2 || alerts[0].Status != "firing" {
		t.Fatalf("firing must sort before pending: %+v", alerts)
	}
}
