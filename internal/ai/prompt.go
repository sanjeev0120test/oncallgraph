package ai

import (
	"fmt"
	"strings"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// maxPromptChars caps the prompt size. We use a character budget instead of a
// tokenizer to stay 100% offline (no BPE downloads) and dependency-light.
const maxPromptChars = 6000

// buildPrompt renders a deterministic instruction prompt from the assembled
// result plus optional retrieved context. The LLM's response is not
// deterministic, but the prompt is.
func buildPrompt(res model.AskResult, ragContext []string) string {
	var b strings.Builder
	b.WriteString("You are an on-call SRE assistant. Using ONLY the facts below, write up to 8 ")
	b.WriteString("bullet points summarizing the incident. EACH bullet MUST cite at least one ")
	b.WriteString("evidence ID exactly as listed (e.g. ev-change-1). Do not invent facts or IDs.\n\n")

	fmt.Fprintf(&b, "Service: %s (%s)\n", res.Service.ID, res.Service.Health)
	fmt.Fprintf(&b, "Window: last %s\n", res.Window)

	if len(res.Evidence) > 0 {
		b.WriteString("Evidence IDs (cite these):\n")
		for _, e := range res.Evidence {
			fmt.Fprintf(&b, "- %s: %s\n", e.ID, e.Summary)
		}
	}
	if len(res.Changes) > 0 {
		b.WriteString("Recent changes:\n")
		for _, c := range res.Changes {
			rev := c.Revision
			if rev == "" {
				rev = c.ID
			}
			fmt.Fprintf(&b, "- %s %q (%s) evidence=%s\n", c.Type, c.Summary, rev, c.EvidenceID)
		}
	}
	if len(res.Alerts) > 0 {
		b.WriteString("Alerts:\n")
		for _, a := range res.Alerts {
			fmt.Fprintf(&b, "- %s (%s, %s) evidence=%s\n", a.Name, a.Severity, a.Status, a.EvidenceID)
		}
	}
	if len(res.Upstream) > 0 {
		b.WriteString("Upstream: " + serviceHealthList(res.Upstream) + "\n")
	}
	if len(res.Downstream) > 0 {
		b.WriteString("Downstream: " + serviceHealthList(res.Downstream) + "\n")
	}
	if res.RunbookResult != nil {
		fmt.Fprintf(&b, "Runbook %s: %s\n", res.RunbookResult.Path, res.RunbookResult.Status)
	}
	if len(ragContext) > 0 {
		b.WriteString("\nRelated context:\n")
		for _, c := range ragContext {
			fmt.Fprintf(&b, "- %s\n", c)
		}
	}
	footer := "\nBullets:"
	out := b.String() + footer
	if len(out) > maxPromptChars {
		out = truncatePrompt(out, footer, maxPromptChars)
	}
	return out
}

// truncatePrompt drops whole lines from the middle so we never cut mid-token
// and always keep the instruction footer.
func truncatePrompt(s, footer string, max int) string {
	if max < len(footer)+32 {
		return footer
	}
	budget := max - len(footer)
	body := strings.TrimSuffix(s, footer)
	if len(body) <= budget {
		return body + footer
	}
	lines := strings.Split(body, "\n")
	var kept []string
	n := 0
	for _, line := range lines {
		add := len(line) + 1
		if n+add > budget {
			break
		}
		kept = append(kept, line)
		n += add
	}
	return strings.Join(kept, "\n") + footer
}

func serviceHealthList(svcs []model.Service) string {
	parts := make([]string, 0, len(svcs))
	for _, s := range svcs {
		parts = append(parts, s.ID+" ("+s.Health+")")
	}
	return strings.Join(parts, ", ")
}
