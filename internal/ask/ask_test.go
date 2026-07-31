package ask_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/opsgraph/opsgraph/fixtures"
	"github.com/opsgraph/opsgraph/internal/ask"
	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/opsgraph/opsgraph/internal/output"
	"github.com/opsgraph/opsgraph/internal/store"
)

func run(t *testing.T) (model.AskResult, []byte) {
	t.Helper()
	fsys, err := fixtures.CheckoutFS()
	if err != nil {
		t.Fatal(err)
	}
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now, err := ingest.IngestFixtureFS(s, fsys)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ask.Ask(s, "checkout", ask.Options{Since: time.Hour, Now: now, WithRunbook: true})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := output.JSON(&buf, res); err != nil {
		t.Fatal(err)
	}
	return res, buf.Bytes()
}

func TestAskIsDeterministic(t *testing.T) {
	_, a := run(t)
	_, b := run(t)
	if !bytes.Equal(a, b) {
		t.Fatalf("ask JSON not byte-identical across runs:\n--- a ---\n%s\n--- b ---\n%s", a, b)
	}
}

func TestAskContent(t *testing.T) {
	res, _ := run(t)

	if res.Service.Health != model.HealthDegraded {
		t.Fatalf("service health = %q", res.Service.Health)
	}
	if res.Window != "60m" {
		t.Fatalf("window = %q, want 60m", res.Window)
	}
	if len(res.Changes) != 2 {
		t.Fatalf("changes = %d, want 2", len(res.Changes))
	}
	if len(res.Alerts) != 1 || res.Alerts[0].Status != "firing" {
		t.Fatalf("alerts = %+v", res.Alerts)
	}

	up := map[string]string{}
	for _, u := range res.Upstream {
		up[u.ID] = u.Health
	}
	if up["auth"] != model.HealthUnhealthy {
		t.Fatalf("auth upstream health = %q", up["auth"])
	}
	if up["redis"] != model.HealthUnknown {
		t.Fatalf("redis should be synthesized unknown, got %q", up["redis"])
	}
	if len(res.Downstream) != 1 || res.Downstream[0].ID != "order" {
		t.Fatalf("downstream = %+v, want [order]", res.Downstream)
	}

	if res.RunbookResult == nil || res.RunbookResult.Status != model.StatusStale {
		t.Fatalf("runbook result = %+v", res.RunbookResult)
	}

	if len(res.Recommendations) == 0 || !strings.Contains(res.Recommendations[0], "most recent") {
		t.Fatalf("recommendations = %+v", res.Recommendations)
	}

	haveEv := map[string]bool{}
	for _, e := range res.Evidence {
		haveEv[e.ID] = true
	}
	for _, id := range []string{"ev-change-1", "ev-alert-1"} {
		if !haveEv[id] {
			t.Fatalf("evidence missing %q; have %v", id, haveEv)
		}
	}
}

func TestAskUnknownService(t *testing.T) {
	fsys, _ := fixtures.CheckoutFS()
	s, cleanup, _ := store.OpenTemp()
	t.Cleanup(cleanup)
	now, _ := ingest.IngestFixtureFS(s, fsys)
	if _, err := ask.Ask(s, "does-not-exist", ask.Options{Since: time.Hour, Now: now}); err == nil {
		t.Fatal("expected error for unknown service")
	}
}
