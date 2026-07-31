package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func sampleResult() model.AskResult {
	return model.AskResult{
		Service: model.Service{ID: "checkout", Health: model.HealthDegraded},
		Owner:   &model.Owner{ID: "payments", Name: "Payments Team"},
		Window:  "60m",
		Changes: []model.Change{{ID: "c1", Type: "deploy", Summary: "deploy v1.4.2", Revision: "abc123", EvidenceID: "ev-change-1"}},
		Alerts:  []model.Alert{{Name: "CheckoutErrorRateHigh", Severity: "critical", Status: "firing", EvidenceID: "ev-alert-1"}},
		Upstream: []model.Service{
			{ID: "auth", Health: model.HealthUnhealthy},
			{ID: "redis", Health: model.HealthUnknown},
		},
		RunbookResult: &model.VerifyResult{Path: "runbooks/checkout.md", Status: model.StatusStale},
		Evidence: []model.Evidence{
			{ID: "ev-change-1", Summary: "deploy checkout v1.4.2"},
			{ID: "ev-alert-1", Summary: "error rate high"},
		},
	}
}

func TestLocalSummaryIsUseful(t *testing.T) {
	s := LocalSummary(sampleResult())
	for _, want := range []string{"checkout", "degraded", "deploy v1.4.2", "auth", "Payments Team"} {
		if !strings.Contains(s, want) {
			t.Fatalf("LocalSummary missing %q:\n%s", want, s)
		}
	}
}

func TestSummarizeDisabledFallsBackToLocal(t *testing.T) {
	cfg := config.Default() // AI.Enabled == false
	res := sampleResult()
	if got := Summarize(context.Background(), cfg, res); got != LocalSummary(res) {
		t.Fatalf("disabled AI should return LocalSummary, got:\n%s", got)
	}
}

func TestSummarizeUnreachableFallsBackToLocal(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Enabled = true
	cfg.AI.OllamaURL = "http://127.0.0.1:9"
	cfg.AI.Timeout = "50ms"
	res := sampleResult()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := Summarize(ctx, cfg, res)
	if !strings.Contains(got, UnavailableMessage) {
		t.Fatalf("unreachable should mention unavailable, got:\n%s", got)
	}
	if !strings.Contains(got, LocalSummary(res)) {
		t.Fatalf("unreachable should include LocalSummary, got:\n%s", got)
	}
}

func TestStubSummarizer(t *testing.T) {
	res := sampleResult()
	got := SummarizeWith(context.Background(), StubSummarizer{Text: "stub-ok"}, res)
	if got != "stub-ok" {
		t.Fatalf("got %q", got)
	}
}

func TestFilterCitedBullets(t *testing.T) {
	res := sampleResult()
	raw := `
- Deploy looks bad [ev-change-1]
- Invented claim [ev-fake-99]
- Alert firing [ev-alert-1]
plain prose without citation
`
	got, ok := filterCitedBullets(raw, res)
	if !ok {
		t.Fatal("expected kept bullets")
	}
	if !strings.Contains(got, "ev-change-1") || !strings.Contains(got, "ev-alert-1") {
		t.Fatalf("missing good citations: %s", got)
	}
	if strings.Contains(got, "ev-fake-99") {
		t.Fatalf("should drop unknown citations: %s", got)
	}
}
