package main

import (
	"fmt"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/sanjeev0120test/opsgraph/internal/report"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:               "report <service>",
		Short:             "Export a markdown incident report for a service",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServiceArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireArg("service", args[0]); err != nil {
				return err
			}
			if format != "markdown" && format != "md" && format != "json" {
				return fail(2, "invalid --format %q (want markdown|json)", format)
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
			md := report.Markdown(res)
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), map[string]any{
					"service":  res.Service.ID,
					"health":   res.Service.Health,
					"markdown": md,
				})
			}
			fmt.Fprint(cmd.OutOrStdout(), md)
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "markdown", "output format: markdown|json")
	_ = cmd.RegisterFlagCompletionFunc("format", completeFormatMarkdownJSON)
	return cmd
}
