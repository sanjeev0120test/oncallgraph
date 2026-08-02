package report

import (
	"strings"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestMarkdownReport(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	md := Markdown(model.AskResult{
		Service:     model.Service{ID: "checkout", Health: model.HealthDegraded},
		Owner:       &model.Owner{Name: "Payments Team", Email: "payments@example.com"},
		Window:      "60m",
		GeneratedAt: now,
		Changes: []model.Change{{
			Type: "deploy", Summary: "deploy v1", Author: "cd-bot", EvidenceID: "ev-c",
			At: now.Add(-10 * time.Minute),
		}},
		Alerts: []model.Alert{{Name: "CheckoutErrorRateHigh", Severity: "critical", Status: "firing", EvidenceID: "ev-a"}},
		Upstream: []model.Service{{ID: "auth", Health: model.HealthUnhealthy}},
		Recommendations: []string{"Investigate recent deploy"},
	})
	for _, want := range []string{
		"# Incident report: checkout",
		"**Health:** degraded",
		"**Severity score:**",
		"Payments Team <payments@example.com>",
		"## Changes",
		"deploy v1",
		"## Alerts",
		"CheckoutErrorRateHigh",
		"## Blast radius",
		"auth",
		"## Recommendations",
		"Investigate recent deploy",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report missing %q:\n%s", want, md)
		}
	}
}

func TestMarkdownEmptyWindow(t *testing.T) {
	md := Markdown(model.AskResult{
		Service:     model.Service{ID: "api", Health: model.HealthHealthy},
		Window:      "60m",
		GeneratedAt: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
	})
	if !strings.Contains(md, "_None in window._") {
		t.Fatalf("want empty-window placeholder:\n%s", md)
	}
}
