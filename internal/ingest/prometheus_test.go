package ingest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ingest"
	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestIngestPrometheusRejectsErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"boom","data":{"alerts":[]}}`))
	}))
	t.Cleanup(srv.Close)
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	err = ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{})
	if err == nil || !strings.Contains(err.Error(), "want success") {
		t.Fatalf("want status error, got %v", err)
	}
}

func TestIngestPrometheusRejectsEmptyStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"alerts":[]}}`))
	}))
	t.Cleanup(srv.Close)
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err == nil {
		t.Fatal("empty status must fail")
	}
}

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

	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Name != "CheckoutErrorRateHigh" || alerts[0].Source != "prometheus" {
		t.Fatalf("alerts = %+v", alerts)
	}
	// Fingerprinted id: prometheus-<slug-name>-<slug-svc>-<8hex|none>
	if got := alerts[0].ID; len(got) < len("prometheus-checkouterrorratehigh-checkout-")+4 {
		t.Fatalf("alert id missing fingerprint suffix: %q", got)
	}
}

func TestIngestPrometheusDistinctInstances(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "status":"success",
		  "data":{"alerts":[
		    {"labels":{"alertname":"HighErr","severity":"critical","service":"checkout","instance":"a:9090"},
		     "annotations":{"summary":"a"},"state":"firing","activeAt":"2026-07-31T11:45:00Z"},
		    {"labels":{"alertname":"HighErr","severity":"critical","service":"checkout","instance":"b:9090"},
		     "annotations":{"summary":"b"},"state":"firing","activeAt":"2026-07-31T11:45:00Z"}
		  ]}
		}`))
	}))
	t.Cleanup(srv.Close)
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("want 2 distinct series alerts, got %+v", alerts)
	}
	if alerts[0].ID == alerts[1].ID {
		t.Fatalf("instance labels must produce distinct ids: %q", alerts[0].ID)
	}
}

func TestIngestPrometheusJobLabelKnownService(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "status":"success",
		  "data":{"alerts":[{
		    "labels":{"alertname":"JobMapped","severity":"warning","job":"checkout"},
		    "annotations":{"summary":"via job"},
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
	if err := s.UpsertService(model.Service{ID: "checkout", Name: "checkout", Health: "unknown"}); err != nil {
		t.Fatal(err)
	}
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Name != "JobMapped" {
		t.Fatalf("job label should map to known service: %+v", alerts)
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
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err != nil {
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
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err != nil {
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

func TestIngestPrometheusUnlabeledDropStillResolves(t *testing.T) {
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
		// Unlabeled orphan is dropped from upsert, but KeepFiring is absent from
		// seen → resolve (zombie cleanup must not freeze on bad labels).
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
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := ingest.IngestPrometheus(context.Background(), s, srv.URL, srv.Client(), time.Time{}); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.ListAlerts("checkout", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Status != "resolved" {
		t.Fatalf("absent labeled alert should resolve after unlabeled-only scrape: %+v", alerts)
	}
}
