package explain

import (
	"strings"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestNarrativeHotIncident(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	n := Narrative(model.AskResult{
		Service:     model.Service{ID: "checkout", Health: model.HealthDegraded},
		Owner:       &model.Owner{ID: "payments", Name: "Payments Team"},
		GeneratedAt: now,
		Changes: []model.Change{{
			ID: "c1", Type: "deploy", Summary: "deploy v1", Revision: "abc",
			Author: "cd-bot", EvidenceID: "ev1", At: now.Add(-10 * time.Minute),
		}},
		Alerts:          []model.Alert{{Name: "CheckoutErrorRateHigh", Status: "firing"}},
		Upstream:        []model.Service{{ID: "auth", Health: model.HealthUnhealthy}},
		Downstream:      []model.Service{{ID: "order", Health: model.HealthHealthy}},
		RunbookResult:   &model.VerifyResult{Path: "runbooks/checkout.md", Status: model.StatusStale, Steps: []model.StepVerifyResult{{Number: 2, Text: "check deploy age", Status: model.StatusStale}}},
		Recommendations: []string{"R1", "R2", "R3", "R4", "R5", "R6 handoff"},
	})
	for _, want := range []string{
		"checkout is currently degraded",
		"Payments Team",
		"Prime suspect",
		"Active alerts: CheckoutErrorRateHigh",
		"Upstream pressure: auth",
		"Blast radius downstream: order",
		"runbooks/checkout.md is stale",
		"Next steps:",
		"R6 handoff",
	} {
		if !strings.Contains(n, want) {
			t.Fatalf("narrative missing %q:\n%s", want, n)
		}
	}
}

func TestNarrativeQuietService(t *testing.T) {
	n := Narrative(model.AskResult{
		Service: model.Service{ID: "order", Health: model.HealthHealthy},
	})
	if !strings.Contains(n, "No recent changes") {
		t.Fatalf("want quiet lookback text:\n%s", n)
	}
}
