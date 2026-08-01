// Package graphviz renders the service dependency graph (ASCII or Mermaid).
package graphviz

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// ASCII returns a simple text dependency map. Edge direction: From → To means From depends on To.
func ASCII(services []model.Service, deps []model.Dependency) string {
	health := map[string]string{}
	for _, s := range services {
		health[s.ID] = s.Health
	}
	var b strings.Builder
	b.WriteString("Dependency graph (From → To = From depends on To)\n")
	if len(deps) == 0 {
		b.WriteString("(no edges)\n")
		return b.String()
	}
	sorted := append([]model.Dependency(nil), deps...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].FromServiceID != sorted[j].FromServiceID {
			return sorted[i].FromServiceID < sorted[j].FromServiceID
		}
		return sorted[i].ToServiceID < sorted[j].ToServiceID
	})
	for _, d := range sorted {
		fh, th := health[d.FromServiceID], health[d.ToServiceID]
		if fh == "" {
			fh = "unknown"
		}
		if th == "" {
			th = "unknown"
		}
		fmt.Fprintf(&b, "  %s [%s] → %s [%s]  (%s)\n", d.FromServiceID, fh, d.ToServiceID, th, d.Type)
	}
	return b.String()
}

// Mermaid returns a Mermaid flowchart definition.
func Mermaid(services []model.Service, deps []model.Dependency) string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	svcs := append([]model.Service(nil), services...)
	sort.SliceStable(svcs, func(i, j int) bool { return svcs[i].ID < svcs[j].ID })
	seen := map[string]bool{}
	for _, s := range svcs {
		seen[s.ID] = true
		label := mermaidEscape(s.ID) + `\n` + mermaidEscape(s.Health)
		fmt.Fprintf(&b, "  %s[\"%s\"]\n", safeID(s.ID), label)
	}
	sorted := append([]model.Dependency(nil), deps...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].FromServiceID != sorted[j].FromServiceID {
			return sorted[i].FromServiceID < sorted[j].FromServiceID
		}
		return sorted[i].ToServiceID < sorted[j].ToServiceID
	})
	for _, d := range sorted {
		if !seen[d.FromServiceID] {
			fmt.Fprintf(&b, "  %s[\"%s\"]\n", safeID(d.FromServiceID), mermaidEscape(d.FromServiceID))
			seen[d.FromServiceID] = true
		}
		if !seen[d.ToServiceID] {
			fmt.Fprintf(&b, "  %s[\"%s\"]\n", safeID(d.ToServiceID), mermaidEscape(d.ToServiceID))
			seen[d.ToServiceID] = true
		}
		fmt.Fprintf(&b, "  %s -->|%s| %s\n", safeID(d.FromServiceID), mermaidEscape(d.Type), safeID(d.ToServiceID))
	}
	return b.String()
}

func safeID(id string) string {
	r := strings.NewReplacer("-", "_", ".", "_", "/", "_", " ", "_", "\"", "")
	out := r.Replace(id)
	if out == "" {
		out = "unknown"
	}
	return "n_" + out
}

func mermaidEscape(s string) string {
	s = strings.ReplaceAll(s, `"`, "'")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
