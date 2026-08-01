package score_test

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/score"
)

func TestComputeCheckoutLike(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	res := model.AskResult{
		Service:     model.Service{ID: "checkout", Health: model.HealthDegraded},
		GeneratedAt: now,
		Changes:     []model.Change{{ID: "c1", At: now.Add(-10 * time.Minute)}},
		Alerts:      []model.Alert{{Name: "CheckoutErrorRateHigh", Severity: "critical", Status: "firing"}},
		Upstream: []model.Service{
			{ID: "auth", Health: model.HealthUnhealthy},
		},
		RunbookResult: &model.VerifyResult{Status: model.StatusStale},
	}
	sc := score.Compute(res)
	if sc.Score < 50 {
		t.Fatalf("expected high score, got %d (%s)", sc.Score, sc.Level)
	}
	if sc.Level != "high" && sc.Level != "critical" {
		t.Fatalf("level = %q", sc.Level)
	}
}

func TestComputeHealthyLow(t *testing.T) {
	sc := score.Compute(model.AskResult{Service: model.Service{ID: "x", Health: model.HealthHealthy}})
	if sc.Score != 0 || sc.Level != "low" {
		t.Fatalf("got %+v", sc)
	}
}

func TestComputeDownstreamImpact(t *testing.T) {
	// Healthy neighbors alone must not inflate severity.
	healthy := score.Compute(model.AskResult{
		Service: model.Service{ID: "a", Health: model.HealthDegraded},
		Downstream: []model.Service{
			{ID: "b", Health: model.HealthHealthy},
			{ID: "c", Health: model.HealthHealthy},
		},
	})
	if healthy.Breakdown["downstream_impact"] != 0 {
		t.Fatalf("2 healthy downstream should not score: %+v", healthy.Breakdown)
	}
	wide := score.Compute(model.AskResult{
		Service: model.Service{ID: "a", Health: model.HealthHealthy},
		Downstream: []model.Service{
			{ID: "b", Health: model.HealthHealthy},
			{ID: "c", Health: model.HealthHealthy},
			{ID: "d", Health: model.HealthHealthy},
		},
	})
	if wide.Breakdown["downstream_impact"] != 3 {
		t.Fatalf("wide blast want 3, got %+v", wide.Breakdown)
	}
	bad := score.Compute(model.AskResult{
		Service: model.Service{ID: "a", Health: model.HealthHealthy},
		Downstream: []model.Service{
			{ID: "b", Health: model.HealthUnhealthy},
			{ID: "c", Health: model.HealthHealthy},
		},
	})
	if bad.Breakdown["downstream_impact"] != 5 {
		t.Fatalf("one unhealthy downstream want 5, got %+v", bad.Breakdown)
	}
}
