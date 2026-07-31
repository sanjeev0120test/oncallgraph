package ask

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// r1ChangeWindow is the fixed lookback for recommendation R1. Independent of
// --since so a wide query window does not falsely elevate an old change.
const r1ChangeWindow = 30 * time.Minute

const r6Handoff = "Write a short handoff note with evidence IDs before closing the incident."

// recommend produces deterministic, ordered next-step recommendations (R1–R6+).
func recommend(res model.AskResult) []string {
	var recs []string

	// R1: most recent change within a fixed 30m threshold is the prime suspect.
	r1Fired := false
	if c, ok := recentChange(res); ok {
		r1Fired = true
		ref := c.Revision
		if ref == "" {
			ref = c.ID
		}
		recs = append(recs, fmt.Sprintf("Inspect the most recent %s first: %q (%s).", changeNoun(c.Type), c.Summary, ref))
	}

	// R1b: queried service itself is unhealthy/degraded.
	if res.Service.Health == model.HealthDegraded || res.Service.Health == model.HealthUnhealthy {
		recs = append(recs, fmt.Sprintf("Investigate %s health (%s) and stabilize before further changes.", res.Service.ID, res.Service.Health))
	}

	// R2: unhealthy upstreams block safe changes (id order from Upstream).
	for _, u := range res.Upstream {
		if u.Health == model.HealthDegraded || u.Health == model.HealthUnhealthy {
			recs = append(recs, fmt.Sprintf("Check upstream %s (%s) before changing %s.", u.ID, u.Health, res.Service.ID))
		}
	}

	// R3: firing/pending alerts need acknowledgement.
	if fa := firstActiveAlert(res.Alerts); fa != nil {
		label := fa.Status
		if label == "" {
			label = "firing"
		}
		if r1Fired {
			recs = append(recs, fmt.Sprintf("Acknowledge %s alert %s and correlate it with the recent change.", label, fa.Name))
		} else {
			recs = append(recs, fmt.Sprintf("Acknowledge %s alert %s and investigate related signals.", label, fa.Name))
		}
	}

	// R4: a stale/failed runbook must be fixed.
	if rb := res.RunbookResult; rb != nil && (rb.Status == model.StatusStale || rb.Status == model.StatusFail) {
		steps := offendingSteps(rb.Steps)
		recs = append(recs, fmt.Sprintf("Runbook %s is %s - review step(s) %s.", rb.Path, rb.Status, steps))
	}

	// R5: loop in the owner when known; otherwise ask to assign one.
	if res.Owner != nil {
		who := res.Owner.Name
		if who == "" {
			who = res.Owner.ID
		}
		if res.Owner.Email != "" {
			who += " <" + res.Owner.Email + ">"
		}
		recs = append(recs, fmt.Sprintf("Notify owner %s.", who))
	} else if res.Service.ID != "" {
		recs = append(recs, fmt.Sprintf("Assign an owner for %s.", res.Service.ID))
	}

	// R6: always appended so every answer ends with a stable handoff step.
	recs = append(recs, r6Handoff)
	return recs
}

func recentChange(res model.AskResult) (model.Change, bool) {
	if len(res.Changes) == 0 || res.GeneratedAt.IsZero() {
		return model.Change{}, false
	}
	c := res.Changes[0]
	if res.GeneratedAt.Sub(c.At) > r1ChangeWindow {
		return model.Change{}, false
	}
	return c, true
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

func firstActiveAlert(alerts []model.Alert) *model.Alert {
	for i := range alerts {
		switch alerts[i].Status {
		case "firing", "pending":
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
