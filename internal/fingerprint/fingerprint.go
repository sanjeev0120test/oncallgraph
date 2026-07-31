// Package fingerprint builds a deterministic incident fingerprint for dedup.
package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// Result is a stable hash of the incident's distinguishing facts.
type Result struct {
	Fingerprint string   `json:"fingerprint"`
	Service     string   `json:"service"`
	Inputs      []string `json:"inputs"`
}

// Of returns a short hex fingerprint from health, firing alerts, and recent
// suspect-window changes (same 30m window as recommendation R1).
func Of(res model.AskResult) Result {
	var parts []string
	parts = append(parts, "service="+res.Service.ID)
	parts = append(parts, "health="+res.Service.Health)
	for _, a := range res.Alerts {
		if model.AlertActive(a.Status) {
			parts = append(parts, "alert="+a.Name+":"+a.Severity)
		}
	}
	for _, c := range res.Changes {
		// When GeneratedAt is set, only suspect-window changes count (align with R1).
		if !res.GeneratedAt.IsZero() && res.GeneratedAt.Sub(c.At) > ask.SuspectChangeWindow {
			continue
		}
		ref := c.Revision
		if ref == "" {
			ref = c.ID
		}
		parts = append(parts, "change="+c.Type+":"+ref)
	}
	for _, u := range res.Upstream {
		if u.Health == model.HealthUnhealthy || u.Health == model.HealthDegraded {
			parts = append(parts, "up="+u.ID+":"+u.Health)
		}
	}
	if res.RunbookResult != nil {
		parts = append(parts, "runbook="+res.RunbookResult.Status)
	}
	sort.Strings(parts[1:]) // keep service first; sort the rest for stability
	canon := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(canon))
	short := hex.EncodeToString(sum[:8])
	return Result{
		Fingerprint: fmt.Sprintf("inc_%s", short),
		Service:     res.Service.ID,
		Inputs:      parts,
	}
}
