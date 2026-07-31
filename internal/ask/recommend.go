package ask

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/opsgraph/opsgraph/internal/model"
)

// recommend produces deterministic, ordered next-step recommendations.
func recommend(res model.AskResult) []string {
	var recs []string

	// R1: most recent change is the prime suspect.
	if len(res.Changes) > 0 {
		c := res.Changes[0]
		ref := c.Revision
		if ref == "" {
			ref = c.ID
		}
		recs = append(recs, fmt.Sprintf("Inspect the most recent %s first: %q (%s).", changeNoun(c.Type), c.Summary, ref))
	}

	// R2: unhealthy upstreams block safe changes (id order from Upstream).
	for _, u := range res.Upstream {
		if u.Health == model.HealthDegraded || u.Health == model.HealthUnhealthy {
			recs = append(recs, fmt.Sprintf("Check upstream %s (%s) before changing %s.", u.ID, u.Health, res.Service.ID))
		}
	}

	// R3: firing alerts need acknowledgement.
	if fa := firstFiring(res.Alerts); fa != nil {
		recs = append(recs, fmt.Sprintf("Acknowledge firing alert %s and correlate it with the recent change.", fa.Name))
	}

	// R4: a stale/failed runbook must be fixed.
	if rb := res.RunbookResult; rb != nil && (rb.Status == model.StatusStale || rb.Status == model.StatusFail) {
		steps := offendingSteps(rb.Steps)
		recs = append(recs, fmt.Sprintf("Runbook %s is %s - review step(s) %s.", rb.Path, rb.Status, steps))
	}

	// R5: always loop in the owner.
	if res.Owner != nil {
		who := res.Owner.Name
		if who == "" {
			who = res.Owner.ID
		}
		if res.Owner.Email != "" {
			who += " <" + res.Owner.Email + ">"
		}
		recs = append(recs, fmt.Sprintf("Notify owner %s.", who))
	}

	if len(recs) == 0 {
		recs = append(recs, "No changes, alerts, or unhealthy dependencies in window; monitor and confirm scope.")
	}
	return recs
}

func changeNoun(t string) string {
	switch t {
	case "deploy":
		return "deploy"
	case "rollout":
		return "rollout"
	case "commit":
		return "commit"
	default:
		return "change"
	}
}

func firstFiring(alerts []model.Alert) *model.Alert {
	for i := range alerts {
		if alerts[i].Status == "firing" {
			return &alerts[i]
		}
	}
	return nil
}

func offendingSteps(steps []model.StepVerifyResult) string {
	var nums []string
	for _, s := range steps {
		if s.Status == model.StatusStale || s.Status == model.StatusFail || s.Status == model.StatusError {
			nums = append(nums, strconv.Itoa(s.Number))
		}
	}
	if len(nums) == 0 {
		return "-"
	}
	return strings.Join(nums, ", ")
}
