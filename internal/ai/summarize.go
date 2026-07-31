package ai

import (
	"fmt"
	"strings"

	"github.com/sanjeev0120test/oncallgraph/internal/model"
)

// LocalSummary builds a deterministic, offline, extractive summary from the
// assembled result. No network, no model, always available.
func LocalSummary(res model.AskResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s is %s.", res.Service.ID, res.Service.Health)

	if len(res.Changes) > 0 {
		c := res.Changes[0]
		fmt.Fprintf(&b, " The most recent %s was %q", c.Type, c.Summary)
		if c.Revision != "" {
			fmt.Fprintf(&b, " (%s)", c.Revision)
		}
		b.WriteString(", the prime suspect.")
	}

	if fa := firstFiring(res.Alerts); fa != "" {
		fmt.Fprintf(&b, " Alert %s is firing.", fa)
	}

	var unhealthy []string
	for _, u := range res.Upstream {
		if u.Health == model.HealthDegraded || u.Health == model.HealthUnhealthy {
			unhealthy = append(unhealthy, u.ID)
		}
	}
	if len(unhealthy) > 0 {
		fmt.Fprintf(&b, " Upstream %s %s unhealthy.", strings.Join(unhealthy, ", "), plural(len(unhealthy)))
	}

	if res.RunbookResult != nil && (res.RunbookResult.Status == model.StatusStale || res.RunbookResult.Status == model.StatusFail) {
		fmt.Fprintf(&b, " The runbook is %s and needs review.", res.RunbookResult.Status)
	}

	if res.Owner != nil {
		who := res.Owner.Name
		if who == "" {
			who = res.Owner.ID
		}
		fmt.Fprintf(&b, " Owner: %s.", who)
	}
	return b.String()
}

func firstFiring(alerts []model.Alert) string {
	for _, a := range alerts {
		if model.AlertActive(a.Status) {
			return a.Name
		}
	}
	return ""
}

func plural(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}
