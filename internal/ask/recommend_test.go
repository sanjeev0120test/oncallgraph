package ask

import (
	"strings"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestRecommendR1Within30m(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	res := model.AskResult{
		Service:     model.Service{ID: "checkout"},
		GeneratedAt: now,
		Changes: []model.Change{{
			ID: "c1", Type: "deploy", Summary: "deploy v1", Revision: "abc",
			At: now.Add(-22 * time.Minute),
		}},
	}
	recs := recommend(res)
	if len(recs) == 0 || !strings.Contains(recs[0], "most recent deploy") {
		t.Fatalf("R1 missing: %v", recs)
	}
	if recs[len(recs)-1] != r6Handoff {
		t.Fatalf("R6 missing: %v", recs)
	}
}

func TestRecommendR1SkipsFutureChange(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	res := model.AskResult{
		Service:     model.Service{ID: "checkout"},
		GeneratedAt: now,
		Changes: []model.Change{{
			ID: "c1", Type: "deploy", Summary: "from the future", Revision: "fut",
			At: now.Add(5 * time.Minute),
		}},
	}
	recs := recommend(res)
	for _, r := range recs {
		if strings.Contains(r, "most recent") {
			t.Fatalf("R1 should not fire for future change: %v", recs)
		}
	}
}

func TestRecommendR1SkipsOlderThan30m(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	res := model.AskResult{
		Service:     model.Service{ID: "checkout"},
		GeneratedAt: now,
		Changes: []model.Change{{
			ID: "c1", Type: "deploy", Summary: "old deploy", Revision: "old",
			At: now.Add(-45 * time.Minute),
		}},
	}
	recs := recommend(res)
	for _, r := range recs {
		if strings.Contains(r, "most recent") {
			t.Fatalf("R1 should not fire for >30m change: %v", recs)
		}
	}
	if recs[len(recs)-1] != r6Handoff {
		t.Fatalf("R6 missing: %v", recs)
	}
}

func TestRecommendFullOrder(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	res := model.AskResult{
		Service:     model.Service{ID: "checkout", Health: model.HealthDegraded},
		GeneratedAt: now,
		Owner:       &model.Owner{ID: "payments", Name: "Payments Team", Email: "payments@example.com"},
		Changes: []model.Change{{
			ID: "c1", Type: "deploy", Summary: "deploy checkout v1.4.2", Revision: "abc123",
			At: now.Add(-22 * time.Minute),
		}},
		Alerts:   []model.Alert{{Name: "CheckoutErrorRateHigh", Status: "firing"}},
		Upstream: []model.Service{{ID: "auth", Health: model.HealthUnhealthy}},
		RunbookResult: &model.VerifyResult{
			Path: "runbooks/checkout.md", Status: model.StatusStale,
			Steps: []model.StepVerifyResult{{Number: 2, Status: model.StatusStale}},
		},
	}
	recs := recommend(res)
	wantPrefix := []string{
		`Inspect the most recent deploy first: "deploy checkout v1.4.2" (abc123).`,
		"Investigate checkout health (degraded) and stabilize before further changes.",
		"Check upstream auth (unhealthy) before changing checkout.",
		"Acknowledge firing alert CheckoutErrorRateHigh and correlate it with the recent change.",
		"Runbook runbooks/checkout.md is stale - review step(s) 2.",
		"Notify owner Payments Team <payments@example.com>.",
		r6Handoff,
	}
	if len(recs) != len(wantPrefix) {
		t.Fatalf("got %d recs, want %d:\n%v", len(recs), len(wantPrefix), recs)
	}
	for i, w := range wantPrefix {
		if recs[i] != w {
			t.Fatalf("rec[%d]=\n%q\nwant\n%q", i, recs[i], w)
		}
	}
}

func TestRecommendEmptyStillHasR6(t *testing.T) {
	recs := recommend(model.AskResult{Service: model.Service{ID: "x"}, GeneratedAt: time.Now().UTC()})
	if len(recs) != 2 {
		t.Fatalf("empty incident should have owner+R6, got %v", recs)
	}
	if recs[0] != "Assign an owner for x." || recs[1] != r6Handoff {
		t.Fatalf("unexpected recs: %v", recs)
	}
}
