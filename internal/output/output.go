// Package output renders ask/verify results as deterministic JSON or a
// human-readable table. Only JSON is used for golden comparisons.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/opsgraph/opsgraph/internal/model"
)

// JSON writes res as indented, deterministic JSON with a trailing newline.
func JSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}

// Table writes a compact human-readable summary of an AskResult.
func Table(w io.Writer, res model.AskResult) error {
	p := func(format string, args ...any) { fmt.Fprintf(w, format, args...) }

	p("SERVICE   %s (%s)", res.Service.ID, res.Service.Health)
	if res.Owner != nil {
		p("   owner: %s", ownerLine(res.Owner))
	}
	p("\n")
	p("WINDOW    last %s (as of %s)\n", res.Window, res.GeneratedAt.Format(time.RFC3339))

	if len(res.Changes) == 0 {
		p("CHANGES   none in window\n")
	}
	for i, c := range res.Changes {
		label := "CHANGES"
		if i > 0 {
			label = "       "
		}
		rev := c.Revision
		if rev == "" {
			rev = c.ID
		}
		p("%s   %s  %s ago  %s  %q  [%s]\n", label, c.Type, ago(res.GeneratedAt, c.At), rev, c.Summary, c.EvidenceID)
	}

	if len(res.Alerts) == 0 {
		p("ALERTS    none in window\n")
	}
	for i, a := range res.Alerts {
		label := "ALERTS"
		if i > 0 {
			label = "      "
		}
		p("%s    %s (%s, %s)  [%s]\n", label, a.Name, a.Severity, a.Status, a.EvidenceID)
	}

	p("BLAST     upstream: %s   downstream: %s\n", svcList(res.Upstream), svcList(res.Downstream))

	if res.RunbookResult != nil {
		rb := res.RunbookResult
		p("RUNBOOK   %s -> %s (%s)\n", rb.Path, strings.ToUpper(rb.Status), stepTally(rb.Steps))
		for _, s := range rb.Steps {
			p("          %d. [%-6s] %s\n", s.Number, s.Status, s.Text)
		}
	}

	p("NEXT\n")
	for i, r := range res.Recommendations {
		p("          %d. %s\n", i+1, r)
	}

	if res.AISummary != "" {
		p("AI\n          %s\n", indentLines(res.AISummary, "          "))
	}
	return nil
}

// VerifyTable writes a human-readable runbook verification result.
func VerifyTable(w io.Writer, vr model.VerifyResult) error {
	fmt.Fprintf(w, "RUNBOOK %s (service %s) -> %s\n", vr.Path, vr.ServiceID, strings.ToUpper(vr.Status))
	for _, s := range vr.Steps {
		fmt.Fprintf(w, "  %d. [%-6s] %s\n", s.Number, s.Status, s.Text)
		if s.Message != "" {
			fmt.Fprintf(w, "        %s\n", s.Message)
		}
	}
	return nil
}

func ownerLine(o *model.Owner) string {
	s := o.Name
	if s == "" {
		s = o.ID
	}
	if o.Email != "" {
		s += " <" + o.Email + ">"
	}
	return s
}

func svcList(svcs []model.Service) string {
	if len(svcs) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(svcs))
	for _, s := range svcs {
		parts = append(parts, s.ID+" ("+s.Health+")")
	}
	return strings.Join(parts, ", ")
}

func stepTally(steps []model.StepVerifyResult) string {
	counts := map[string]int{}
	for _, s := range steps {
		counts[s.Status]++
	}
	order := []string{model.StatusPass, model.StatusFail, model.StatusStale, model.StatusManual, model.StatusError}
	var parts []string
	for _, k := range order {
		if counts[k] > 0 {
			parts = append(parts, strconv.Itoa(counts[k])+" "+k)
		}
	}
	return strings.Join(parts, ", ")
}

func ago(now, then time.Time) string {
	d := now.Sub(then)
	if d < time.Minute {
		return strconv.Itoa(int(d.Seconds())) + "s"
	}
	if d < time.Hour {
		return strconv.Itoa(int(d.Minutes())) + "m"
	}
	return strconv.Itoa(int(d.Hours())) + "h"
}

func indentLines(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return strings.Join(lines, "\n"+prefix)
}
