package main

import (
	"errors"
	"time"

	"github.com/sanjeev0120test/oncallgraph/internal/ask"
	"github.com/sanjeev0120test/oncallgraph/internal/explain"
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
			cmd.Print(explain.Narrative(res))
			return nil
		},
	}
	cmd.Flags().StringVar(&src.fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&src.configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&src.dataDir, "data-dir", "", "persistent store directory")
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	return cmd
}
