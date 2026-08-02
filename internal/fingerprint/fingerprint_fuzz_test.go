package fingerprint

import (
	"strings"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func FuzzOfStablePrefix(f *testing.F) {
	f.Add("checkout", "degraded", "AlertA", "critical", "deploy", "abc123")
	f.Add("auth", "unhealthy", "", "", "commit", "")
	f.Fuzz(func(t *testing.T, svc, health, alert, sev, changeType, rev string) {
		now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
		res := model.AskResult{
			Service:     model.Service{ID: svc, Health: health},
			GeneratedAt: now,
		}
		if alert != "" {
			res.Alerts = []model.Alert{{Name: alert, Severity: sev, Status: "firing"}}
		}
		if changeType != "" {
			res.Changes = []model.Change{{
				ID: "c1", Type: changeType, Revision: rev, Summary: "x",
				At: now.Add(-5 * time.Minute),
			}}
		}
		a := Of(res)
		b := Of(res)
		if a.Fingerprint != b.Fingerprint {
			t.Fatalf("unstable fingerprint: %q vs %q", a.Fingerprint, b.Fingerprint)
		}
		if !strings.HasPrefix(a.Fingerprint, "inc_") {
			t.Fatalf("prefix: %q", a.Fingerprint)
		}
		if a.Service != svc {
			t.Fatalf("service=%q want %q", a.Service, svc)
		}
	})
}
