package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/report"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	cmd := &cobra.Command{
		Use:   "report <service>",
		Short: "Export a markdown incident report for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ls, cfg, err := src.load(since)
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
			cmd.Print(report.Markdown(res))
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	return cmd
}
