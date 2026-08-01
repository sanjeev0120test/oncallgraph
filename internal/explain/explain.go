// Package explain builds a deterministic root-cause narrative from AskResult.
package explain

import (
	"fmt"
	"strings"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// Narrative returns a multi-paragraph, evidence-aware explanation.
func Narrative(res model.AskResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s is currently %s", res.Service.ID, res.Service.Health)
	if res.Owner != nil {
		who := res.Owner.Name
		if who == "" {
			who = res.Owner.ID
		}
		fmt.Fprintf(&b, " (owner: %s)", who)
	}
	b.WriteString(".\n\n")

	if c, ok := ask.RecentSuspectChange(res); ok {
		fmt.Fprintf(&b, "Prime suspect: the most recent %s %q", c.Type, c.Summary)
		if c.Revision != "" {
			fmt.Fprintf(&b, " (%s)", c.Revision)
		}
		if c.Author != "" {
			fmt.Fprintf(&b, " by %s", c.Author)
		}
		if c.EvidenceID != "" {
			fmt.Fprintf(&b, " [%s]", c.EvidenceID)
		}
		b.WriteString(".\n")
	} else if len(res.Correlations) > 0 {
		b.WriteString("No change in the 30m suspect window; older linked change exists.\n")
	} else if len(res.Changes) > 0 {
		b.WriteString("No change in the 30m suspect window; older changes exist in the lookback.\n")
	} else {
		b.WriteString("No recent changes were found in the lookback window.\n")
	}

	if len(res.Correlations) > 0 {
		fmt.Fprintf(&b, "Linked evidence: %s.\n", res.Correlations[0].Summary)
	}

	firing := []string{}
	for _, a := range res.Alerts {
		if model.AlertActive(a.Status) {
			firing = append(firing, a.Name)
		}
	}
	if len(firing) > 0 {
		fmt.Fprintf(&b, "Active alerts: %s.\n", strings.Join(firing, ", "))
	}

	var badUp []string
	for _, u := range res.Upstream {
		if u.Health == model.HealthUnhealthy || u.Health == model.HealthDegraded {
			badUp = append(badUp, fmt.Sprintf("%s (%s)", u.ID, u.Health))
		}
	}
	if len(badUp) > 0 {
		fmt.Fprintf(&b, "Upstream pressure: %s — fix upstream before changing %s.\n",
			strings.Join(badUp, ", "), res.Service.ID)
	}

	if len(res.Downstream) > 0 {
		var bad, ids []string
		for _, d := range res.Downstream {
			ids = append(ids, d.ID)
			if d.Health == model.HealthUnhealthy || d.Health == model.HealthDegraded {
				bad = append(bad, fmt.Sprintf("%s (%s)", d.ID, d.Health))
			}
		}
		if len(bad) > 0 {
			fmt.Fprintf(&b, "Unhealthy downstream: %s.\n", strings.Join(bad, ", "))
		}
		fmt.Fprintf(&b, "Blast radius downstream: %s.\n", strings.Join(ids, ", "))
	}

	if res.RunbookResult != nil {
		fmt.Fprintf(&b, "\nRunbook %s is %s.", res.RunbookResult.Path, res.RunbookResult.Status)
		for _, st := range res.RunbookResult.Steps {
			if st.Status == model.StatusStale || st.Status == model.StatusFail || st.Status == model.StatusError {
				fmt.Fprintf(&b, " Step %d needs review: %s", st.Number, st.Text)
				break
			}
		}
		b.WriteString("\n")
	}

	if len(res.Recommendations) > 0 {
		b.WriteString("\nNext steps:\n")
		recs := res.Recommendations
		if len(recs) > 5 {
			// Keep head + trailing handoff (R6).
			recs = append(append([]string{}, recs[:4]...), recs[len(recs)-1])
		}
		for i, r := range recs {
			fmt.Fprintf(&b, "  %d. %s\n", i+1, r)
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}
