package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/model"
)

func sampleResult() model.AskResult {
	return model.AskResult{
		Service: model.Service{ID: "checkout", Health: model.HealthDegraded},
		Owner:   &model.Owner{ID: "payments", Name: "Payments Team"},
		Window:  "60m",
		Changes: []model.Change{{ID: "c1", Type: "deploy", Summary: "deploy v1.4.2", Revision: "abc123"}},
		Alerts:  []model.Alert{{Name: "CheckoutErrorRateHigh", Severity: "critical", Status: "firing"}},
		Upstream: []model.Service{
			{ID: "auth", Health: model.HealthUnhealthy},
			{ID: "redis", Health: model.HealthUnknown},
		},
		RunbookResult: &model.VerifyResult{Path: "runbooks/checkout.md", Status: model.StatusStale},
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
	cfg.AI.OllamaURL = "http://127.0.0.1:1" // nothing listening -> connection refused
	cfg.AI.Timeout = "3s"
	res := sampleResult()
	if got := Summarize(context.Background(), cfg, res); got != LocalSummary(res) {
		t.Fatalf("unreachable Ollama should degrade to LocalSummary, got:\n%s", got)
	}
}
