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

func TestIngestPrometheus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/alerts" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "status":"success",
		  "data":{"alerts":[{
		    "labels":{"alertname":"CheckoutErrorRateHigh","severity":"critical","service":"checkout"},
		    "annotations":{"summary":"error rate high"},
		    "state":"firing",
		    "activeAt":"2026-07-31T11:45:00Z"
		  }]}
		}`))
	}))
	t.Cleanup(srv.Close)

	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)

	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client()); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Name != "CheckoutErrorRateHigh" || alerts[0].Source != "prometheus" {
		t.Fatalf("alerts = %+v", alerts)
	}
}
