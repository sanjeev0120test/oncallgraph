// Package report renders AskResult as a markdown incident report.
package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/score"
)

// Markdown builds a complete incident report suitable for paste into Slack/PagerDuty notes.
func Markdown(res model.AskResult) string {
	sc := score.Compute(res)
	var b strings.Builder
	fmt.Fprintf(&b, "# Incident report: %s\n\n", res.Service.ID)
	fmt.Fprintf(&b, "- **Health:** %s\n", res.Service.Health)
	fmt.Fprintf(&b, "- **Severity score:** %d (%s)\n", sc.Score, sc.Level)
	fmt.Fprintf(&b, "- **Window:** last %s (as of %s)\n", res.Window, res.GeneratedAt.UTC().Format(time.RFC3339))
	if res.Owner != nil {
		fmt.Fprintf(&b, "- **Owner:** %s", res.Owner.Name)
		if res.Owner.Email != "" {
			fmt.Fprintf(&b, " <%s>", res.Owner.Email)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Changes\n")
	if len(res.Changes) == 0 {
		b.WriteString("_None in window._\n")
	}
	for _, c := range res.Changes {
		fmt.Fprintf(&b, "- `%s` %s — %q [%s]\n", c.Type, c.At.Format(time.RFC3339), c.Summary, c.EvidenceID)
	}
	b.WriteString("\n## Alerts\n")
	if len(res.Alerts) == 0 {
		b.WriteString("_None in window._\n")
	}
	for _, a := range res.Alerts {
		fmt.Fprintf(&b, "- **%s** (%s, %s) [%s]\n", a.Name, a.Severity, a.Status, a.EvidenceID)
	}
	b.WriteString("\n## Blast radius\n")
	b.WriteString("- Upstream: " + joinSvc(res.Upstream) + "\n")
	b.WriteString("- Downstream: " + joinSvc(res.Downstream) + "\n")
	if res.RunbookResult != nil {
		fmt.Fprintf(&b, "\n## Runbook\n- Path: `%s`\n- Status: **%s**\n", res.RunbookResult.Path, res.RunbookResult.Status)
	}
	b.WriteString("\n## Timeline\n")
	for _, t := range res.Timeline {
		fmt.Fprintf(&b, "- %s `%s` %s", t.At.Format(time.RFC3339), t.Kind, t.Summary)
		if t.EvidenceID != "" {
			fmt.Fprintf(&b, " [%s]", t.EvidenceID)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Recommendations\n")
	for i, r := range res.Recommendations {
		fmt.Fprintf(&b, "%d. %s\n", i+1, r)
	}
	if len(res.Evidence) > 0 {
		b.WriteString("\n## Evidence\n")
		for _, e := range res.Evidence {
			fmt.Fprintf(&b, "- `%s` (%s) %s\n", e.ID, e.Kind, e.Summary)
		}
	}
	return b.String()
}

func joinSvc(svcs []model.Service) string {
	if len(svcs) == 0 {
		return "_(none)_"
	}
	parts := make([]string, 0, len(svcs))
	for _, s := range svcs {
		parts = append(parts, fmt.Sprintf("%s (%s)", s.ID, s.Health))
	}
	return strings.Join(parts, ", ")
}
