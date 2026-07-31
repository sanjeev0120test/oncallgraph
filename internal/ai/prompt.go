package ai

import (
	"fmt"
	"strings"

	"github.com/opsgraph/opsgraph/internal/model"
)

// maxPromptChars caps the prompt size. We use a character budget instead of a
// tokenizer to stay 100% offline (no BPE downloads) and dependency-light.
const maxPromptChars = 6000

// buildPrompt renders a deterministic instruction prompt from the assembled
// result plus optional retrieved context. The LLM's response is not
// deterministic, but the prompt is.
func buildPrompt(res model.AskResult, ragContext []string) string {
	var b strings.Builder
	b.WriteString("You are an on-call SRE assistant. Using ONLY the facts below, write a concise ")
	b.WriteString("3-5 sentence incident summary for the engineer who was just paged. ")
	b.WriteString("Name the prime-suspect change, the blast radius, and the single most important next step. ")
	b.WriteString("Do not invent facts or hostnames.\n\n")

	fmt.Fprintf(&b, "Service: %s (%s)\n", res.Service.ID, res.Service.Health)
	fmt.Fprintf(&b, "Window: last %s\n", res.Window)

	if len(res.Changes) > 0 {
		b.WriteString("Recent changes:\n")
		for _, c := range res.Changes {
			rev := c.Revision
			if rev == "" {
				rev = c.ID
			}
			fmt.Fprintf(&b, "- %s %q (%s) at %s\n", c.Type, c.Summary, rev, c.At.Format("15:04Z"))
		}
	}
	if len(res.Alerts) > 0 {
		b.WriteString("Alerts:\n")
		for _, a := range res.Alerts {
			fmt.Fprintf(&b, "- %s (%s, %s)\n", a.Name, a.Severity, a.Status)
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
		b.WriteString("\nRelated evidence:\n")
		for _, c := range ragContext {
			fmt.Fprintf(&b, "- %s\n", c)
		}
	}
	b.WriteString("\nSummary:")

	out := b.String()
	if len(out) > maxPromptChars {
		out = out[:maxPromptChars]
	}
	return out
}

func serviceHealthList(svcs []model.Service) string {
	parts := make([]string, 0, len(svcs))
	for _, s := range svcs {
		parts = append(parts, s.ID+" ("+s.Health+")")
	}
	return strings.Join(parts, ", ")
}
