package main

import (
	"fmt"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newWhyCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:   "why <service>",
		Short: "One-line root-cause hypothesis for a paged service",
		Example: `  opsgraph why checkout --fixture fixtures/incident_checkout
  opsgraph why checkout --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireArg("service", args[0]); err != nil {
				return err
			}
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, cfg, err := src.loadCtx(cmd.Context(), since)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			res, err := askService(ls, args[0], since)
			if err != nil {
				return failAsk(err)
			}
			line := whyLine(res)
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), map[string]string{
					"service": res.Service.ID,
					"why":     line,
				})
			}
			fmt.Fprintln(cmd.OutOrStdout(), line)
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}

func whyLine(res model.AskResult) string {
	line := fmt.Sprintf("%s is %s", res.Service.ID, res.Service.Health)
	if c, ok := ask.RecentSuspectChange(res); ok {
		line += fmt.Sprintf("; prime suspect %s %q", c.Type, c.Summary)
		if c.EvidenceID != "" {
			line += " [" + c.EvidenceID + "]"
		}
	} else if len(res.Correlations) > 0 {
		line += "; no 30m suspect; older linked change"
	} else if len(res.Changes) > 0 {
		line += "; no 30m suspect; older lookback changes"
	}
	if len(res.Correlations) > 0 {
		line += "; " + res.Correlations[0].Summary
	}
	for _, u := range res.Upstream {
		if u.Health == model.HealthUnhealthy || u.Health == model.HealthDegraded {
			line += fmt.Sprintf("; upstream %s is %s", u.ID, u.Health)
			break
		}
	}
	alertNoted := false
	for _, a := range res.Alerts {
		if model.AlertActive(a.Status) {
			line += "; alert " + a.Name + " " + a.Status
			if a.Severity != "" {
				line += " (" + a.Severity + ")"
			}
			alertNoted = true
			break
		}
	}
	if !alertNoted {
		for _, a := range res.Alerts {
			if a.Status == "suppressed" {
				line += "; alert " + a.Name + " suppressed"
				break
			}
		}
	}
	for _, d := range res.Downstream {
		if d.Health == model.HealthUnhealthy || d.Health == model.HealthDegraded {
			line += fmt.Sprintf("; downstream %s is %s", d.ID, d.Health)
			break
		}
	}
	return line + "."
}
