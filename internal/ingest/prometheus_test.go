package ingest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ingest"
	"github.com/sanjeev0120test/opsgraph/internal/store"
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

func TestIngestPrometheusServiceNameLabel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "status":"success",
		  "data":{"alerts":[{
		    "labels":{"alertname":"HighLatency","severity":"warning","service_name":"checkout"},
		    "annotations":{"summary":"p99 high"},
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
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Name != "HighLatency" {
		t.Fatalf("service_name label should map to checkout: %+v", alerts)
	}
}

func TestIngestPrometheusResolvesZombieAlerts(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) == 1 {
			_, _ = w.Write([]byte(`{
			  "status":"success",
			  "data":{"alerts":[{
			    "labels":{"alertname":"OldFire","severity":"critical","service":"checkout"},
			    "annotations":{"summary":"old"},
			    "state":"firing",
			    "activeAt":"2026-07-31T11:00:00Z"
			  }]}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"alerts":[]}}`))
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
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client()); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Status != "resolved" {
		t.Fatalf("zombie should resolve, got %+v", alerts)
	}
}

func TestIngestPrometheusDroppedLabelsSkipResolve(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) == 1 {
			_, _ = w.Write([]byte(`{
			  "status":"success",
			  "data":{"alerts":[{
			    "labels":{"alertname":"KeepFiring","severity":"critical","service":"checkout"},
			    "annotations":{"summary":"keep"},
			    "state":"firing",
			    "activeAt":"2026-07-31T11:00:00Z"
			  }]}
			}`))
			return
		}
		// Bad scrape: alert without service label — must not resolve KeepFiring.
		_, _ = w.Write([]byte(`{
		  "status":"success",
		  "data":{"alerts":[{
		    "labels":{"alertname":"Orphan","severity":"warning"},
		    "annotations":{"summary":"no service"},
		    "state":"firing",
		    "activeAt":"2026-07-31T11:30:00Z"
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
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client()); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Status != "firing" {
		t.Fatalf("partial scrape must not resolve real firing: %+v", alerts)
	}
}
