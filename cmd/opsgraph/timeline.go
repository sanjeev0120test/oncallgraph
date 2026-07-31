package main

import (
	"errors"
	"time"

	"github.com/opsgraph/opsgraph/internal/ask"
	"github.com/opsgraph/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newTimelineCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:   "timeline <service>",
		Short: "Show only the incident timeline for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, cfg, err := src.load(since)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			res, err := askService(ls, args[0], since)
			if err != nil {
				if errors.Is(err, ask.ErrServiceNotFound) {
					return fail(1, "%v", err)
				}
				return fail(2, "%v", err)
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), res.Timeline)
			}
			for _, t := range res.Timeline {
				ev := t.EvidenceID
				if ev == "" {
					ev = "-"
				}
				cmd.Printf("%s  %-10s  %-40s  [%s]\n", t.At.Format(time.RFC3339), t.Kind, trunc(t.Summary, 40), ev)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&src.fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&src.configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&src.dataDir, "data-dir", "", "persistent store directory")
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
