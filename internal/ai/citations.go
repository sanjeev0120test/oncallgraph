package ai

import (
	"regexp"
	"strings"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

var (
	bulletRe = regexp.MustCompile(`(?m)^\s*(?:[-*]|\d+\.)\s+(.*\S)\s*$`)
	evIDRe   = regexp.MustCompile(`\bev-[a-zA-Z0-9][-a-zA-Z0-9]*\b`)
)

const maxAIBullets = 8

// filterCitedBullets keeps at most maxAIBullets lines that cite known evidence
// IDs. Unknown citations are dropped. Returns ok=false when nothing survives.
func filterCitedBullets(raw string, res model.AskResult) (string, bool) {
	known := map[string]bool{}
	for _, e := range res.Evidence {
		known[e.ID] = true
	}
	for _, c := range res.Changes {
		if c.EvidenceID != "" {
			known[c.EvidenceID] = true
		}
	}
	for _, a := range res.Alerts {
		if a.EvidenceID != "" {
			known[a.EvidenceID] = true
		}
	}

	var kept []string
	for _, m := range bulletRe.FindAllStringSubmatch(raw, -1) {
		line := strings.TrimSpace(m[1])
		ids := evIDRe.FindAllString(line, -1)
		if len(ids) == 0 {
			continue
		}
		ok := false
		for _, id := range ids {
			if known[id] {
				ok = true
			} else {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		kept = append(kept, "- "+line)
		if len(kept) >= maxAIBullets {
			break
		}
	}
	if len(kept) == 0 {
		return "", false
	}
	return strings.Join(kept, "\n"), true
}
