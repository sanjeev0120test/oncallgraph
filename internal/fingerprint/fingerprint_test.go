package fingerprint_test

import (
	"testing"

	"github.com/sanjeev0120test/oncallgraph/internal/fingerprint"
	"github.com/sanjeev0120test/oncallgraph/internal/model"
)

func TestFingerprintStable(t *testing.T) {
	res := model.AskResult{
		Service: model.Service{ID: "checkout", Health: model.HealthDegraded},
		Changes: []model.Change{{Type: "deploy", Revision: "abc123"}},
		Alerts:  []model.Alert{{Name: "CheckoutErrorRateHigh", Severity: "critical", Status: "firing"}},
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
