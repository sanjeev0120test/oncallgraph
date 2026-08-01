package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/sanjeev0120test/opsgraph/internal/score"
	"github.com/spf13/cobra"
)

func newCompareCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:   "compare <service-a> <service-b>",
		Short: "Compare health, blast radius, and severity of two services",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, cfg, err := src.load(since)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			a, err := askService(ls, args[0], since)
			if err != nil {
				return failAsk(err)
			}
			b, err := askService(ls, args[1], since)
			if err != nil {
				return failAsk(err)
			}
			sa, sb := score.Compute(a), score.Compute(b)
			type side struct {
				Service    string   `json:"service"`
				Health     string   `json:"health"`
				Score      int      `json:"score"`
				Level      string   `json:"level"`
				Changes    int      `json:"changes"`
				Alerts     int      `json:"alerts"` // active (firing/pending) only
				Upstream   []string `json:"upstream"`
				Downstream []string `json:"downstream"`
			}
			out := struct {
				A side `json:"a"`
				B side `json:"b"`
			}{
				A: side{
					Service: a.Service.ID, Health: a.Service.Health, Score: sa.Score, Level: sa.Level,
					Changes: len(a.Changes), Alerts: countActiveAlerts(a.Alerts),
					Upstream: svcIDs(a.Upstream), Downstream: svcIDs(a.Downstream),
				},
				B: side{
					Service: b.Service.ID, Health: b.Service.Health, Score: sb.Score, Level: sb.Level,
					Changes: len(b.Changes), Alerts: countActiveAlerts(b.Alerts),
					Upstream: svcIDs(b.Upstream), Downstream: svcIDs(b.Downstream),
				},
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), out)
			}
			cmd.Printf("%-12s %-12s %-12s\n", "FIELD", out.A.Service, out.B.Service)
			cmd.Printf("%-12s %-12s %-12s\n", "health", out.A.Health, out.B.Health)
			cmd.Printf("%-12s %-12d %-12d\n", "score", out.A.Score, out.B.Score)
			cmd.Printf("%-12s %-12s %-12s\n", "level", out.A.Level, out.B.Level)
			cmd.Printf("%-12s %-12d %-12d\n", "changes", out.A.Changes, out.B.Changes)
			cmd.Printf("%-12s %-12d %-12d\n", "alerts", out.A.Alerts, out.B.Alerts)
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}

func svcIDs(svcs []model.Service) []string {
	out := make([]string, 0, len(svcs))
	for _, s := range svcs {
		out = append(out, s.ID)
	}
	return out
}

func countActiveAlerts(alerts []model.Alert) int {
	n := 0
	for _, a := range alerts {
		if model.AlertActive(a.Status) {
			n++
		}
	}
	return n
}
