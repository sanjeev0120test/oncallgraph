package fingerprint_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/fingerprint"
	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestFingerprintStable(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	res := model.AskResult{
		Service:     model.Service{ID: "checkout", Health: model.HealthDegraded},
		GeneratedAt: now,
		Changes: []model.Change{{
			Type: "deploy", Revision: "abc123", At: now.Add(-10 * time.Minute),
		}},
		Alerts: []model.Alert{{Name: "CheckoutErrorRateHigh", Severity: "critical", Status: "firing"}},
	}
	a := fingerprint.Of(res)
	b := fingerprint.Of(res)
	if a.Fingerprint == "" || a.Fingerprint != b.Fingerprint {
		t.Fatalf("unstable fingerprint: %q vs %q", a.Fingerprint, b.Fingerprint)
	}
	if a.Fingerprint[:4] != "inc_" {
		t.Fatalf("prefix: %s", a.Fingerprint)
	}
}

func TestFingerprintIgnoresOldChanges(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	base := model.AskResult{
		Service:     model.Service{ID: "checkout", Health: model.HealthDegraded},
		GeneratedAt: now,
		Alerts:      []model.Alert{{Name: "A", Severity: "warning", Status: "firing"}},
	}
	withOld := base
	withOld.Changes = []model.Change{{Type: "deploy", Revision: "old", At: now.Add(-2 * time.Hour)}}
	withNew := base
	withNew.Changes = []model.Change{{Type: "deploy", Revision: "new", At: now.Add(-5 * time.Minute)}}

	oldFP := fingerprint.Of(withOld)
	newFP := fingerprint.Of(withNew)
	if oldFP.Fingerprint == newFP.Fingerprint {
		t.Fatal("old change outside suspect window should not affect fingerprint like a fresh deploy")
	}
	for _, part := range oldFP.Inputs {
		if strings.HasPrefix(part, "change=") {
			t.Fatalf("old change leaked into fingerprint inputs: %v", oldFP.Inputs)
		}
	}
}
