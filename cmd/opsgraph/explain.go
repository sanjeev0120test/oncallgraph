package main

import (
	"fmt"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/explain"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newExplainCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:               "explain <service>",
		Short:             "Deterministic root-cause narrative for a service",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServiceArg,
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
			narrative := explain.Narrative(res)
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), map[string]string{
					"service":   res.Service.ID,
					"narrative": narrative,
				})
			}
			fmt.Fprint(cmd.OutOrStdout(), narrative)
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	bindFormatCompletion(cmd)
	return cmd
}
