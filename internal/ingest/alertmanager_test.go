package ingest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/store"
)

func TestIngestAlertmanager(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/alerts" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{
		  "labels":{"alertname":"AuthDown","severity":"critical","service":"auth"},
		  "annotations":{"summary":"auth is down"},
		  "startsAt":"2026-07-31T11:40:00Z",
		  "status":{"state":"active"}
		}]`))
	}))
	t.Cleanup(srv.Close)

	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)

	if err := ingest.IngestAlertmanager(context.Background(), s, srv.URL, srv.Client()); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("auth", time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Name != "AuthDown" || alerts[0].Source != "alertmanager" {
		t.Fatalf("alerts = %+v", alerts)
	}
}
