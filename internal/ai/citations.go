package ai

import (
	"regexp"
	"strings"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

var (
	bulletRe = regexp.MustCompile(`(?m)^\s*(?:[-*]|\d+\.)\s+(.*\S)\s*$`)
	// Loose token match; acceptance is decided against the known evidence set
	// so non-ev-* IDs (if ever used) still validate.
	citeTokenRe = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9][-a-zA-Z0-9]*\b`)
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
		tokens := citeTokenRe.FindAllString(line, -1)
		cited := 0
		ok := true
		for _, tok := range tokens {
			if !known[tok] {
				continue
			}
			cited++
		}
		// Require at least one known evidence ID; ignore other words.
		if cited == 0 {
			ok = false
		}
		// Reject if the line contains an ev-* looking token that is unknown.
		for _, tok := range tokens {
			if strings.HasPrefix(tok, "ev-") && !known[tok] {
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
