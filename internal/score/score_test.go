package score_test

import (
	"testing"
	"time"

	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/opsgraph/opsgraph/internal/score"
)

func TestComputeCheckoutLike(t *testing.T) {
	res := model.AskResult{
		Service: model.Service{ID: "checkout", Health: model.HealthDegraded},
		Changes: []model.Change{{ID: "c1", At: time.Now()}},
		Alerts:  []model.Alert{{Name: "CheckoutErrorRateHigh", Severity: "critical", Status: "firing"}},
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
