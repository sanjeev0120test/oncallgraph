package main

import (
	"fmt"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/explain"
	"github.com/spf13/cobra"
)

func newExplainCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	cmd := &cobra.Command{
		Use:   "explain <service>",
		Short: "Deterministic root-cause narrative for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireArg("service", args[0]); err != nil {
				return err
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
			fmt.Fprint(cmd.OutOrStdout(), explain.Narrative(res))
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	return cmd
}
