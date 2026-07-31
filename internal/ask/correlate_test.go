package ask_test

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestAskCorrelatesDeployBeforeAlert(t *testing.T) {
	res, _ := run(t)
	if len(res.Correlations) == 0 {
		t.Fatal("expected change→alert correlation for checkout fixture")
	}
	c := res.Correlations[0]
	if c.Kind != "change_then_alert" {
		t.Fatalf("kind = %q", c.Kind)
	}
	if c.Gap != "7m" {
		t.Fatalf("gap = %q, want 7m", c.Gap)
	}
	if c.ChangeEvidence != "ev-change-1" || c.AlertEvidence != "ev-alert-1" {
		t.Fatalf("evidence = %+v", c)
	}
}

func TestRecentSuspectChangeRespectsWindow(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	res := model.AskResult{
		GeneratedAt: now,
		Changes: []model.Change{{
			ID: "old", Type: "deploy", Summary: "old deploy", At: now.Add(-45 * time.Minute),
		}},
	}
	if _, ok := ask.RecentSuspectChange(res); ok {
		t.Fatal("45m-old change should not be a suspect")
	}
	res.Changes[0].At = now.Add(-10 * time.Minute)
	if c, ok := ask.RecentSuspectChange(res); !ok || c.ID != "old" {
		t.Fatalf("10m-old change should be suspect, got ok=%v %+v", ok, c)
	}
}
