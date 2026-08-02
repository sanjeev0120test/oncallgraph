package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newTimelineCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	var limit int
	cmd := &cobra.Command{
		Use:               "timeline <service>",
		Short:             "Show only the incident timeline for a service",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServiceArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireArg("service", args[0]); err != nil {
				return err
			}
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			if limit < 0 {
				return fail(2, "invalid --limit %d (must be >= 0)", limit)
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
			events := res.Timeline
			if limit > 0 && len(events) > limit {
				events = events[:limit]
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), events)
			}
			if len(res.Timeline) == 0 {
				cmd.Println("(no events)")
				return nil
			}
			for _, t := range events {
				ev := t.EvidenceID
				if ev == "" {
					ev = "-"
				}
				cmd.Printf("%s  %-10s  %-40s  [%s]\n", t.At.Format(time.RFC3339), t.Kind, trunc(t.Summary, 40), ev)
			}
			if limit > 0 && len(res.Timeline) > limit {
				cmd.Printf("… +%d more (raise --limit)\n", len(res.Timeline)-limit)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().IntVar(&limit, "limit", 0, "max events to show (0 = all)")
	return cmd
}

func trunc(s string, n int) string {
	if n <= 1 {
		return "…"
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
