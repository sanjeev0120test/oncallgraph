package main

import (
	"errors"
	"sort"
	"time"

	"github.com/sanjeev0120test/oncallgraph/internal/ask"
	"github.com/sanjeev0120test/oncallgraph/internal/output"
	"github.com/sanjeev0120test/oncallgraph/internal/score"
	"github.com/spf13/cobra"
)

func newScoreCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:   "score <service>",
		Short: "Compute a deterministic incident severity score (0-100)",
		Args:  cobra.ExactArgs(1),
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
			res, err := askService(ls, args[0], since)
			if err != nil {
				if errors.Is(err, ask.ErrServiceNotFound) {
					return fail(1, "%v", err)
				}
				return fail(2, "%v", err)
			}
			sc := score.Compute(res)
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), sc)
			}
			cmd.Printf("SCORE   %d (%s)\n", sc.Score, sc.Level)
			cmd.Println("BREAKDOWN")
			keys := make([]string, 0, len(sc.Breakdown))
			for k := range sc.Breakdown {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				cmd.Printf("  %-22s %d\n", k, sc.Breakdown[k])
			}
			if len(sc.Highlights) > 0 {
				cmd.Println("HIGHLIGHTS")
				for _, h := range sc.Highlights {
					cmd.Printf("  - %s\n", h)
				}
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
