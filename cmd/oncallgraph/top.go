package main

import (
	"sort"
	"time"

	"github.com/sanjeev0120test/oncallgraph/internal/output"
	"github.com/sanjeev0120test/oncallgraph/internal/score"
	"github.com/spf13/cobra"
)

func newTopCmd() *cobra.Command {
	var src sourceFlags
	var format string
	var limit int
	var since time.Duration
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Rank services by incident severity score",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			if limit <= 0 {
				limit = 10
			}
			ls, cfg, err := src.load(since)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			svcs, err := ls.store.ListServices()
			if err != nil {
				return fail(2, "%v", err)
			}
			type row struct {
				Service string `json:"service"`
				Health  string `json:"health"`
				Score   int    `json:"score"`
				Level   string `json:"level"`
			}
			rows := make([]row, 0, len(svcs))
			for _, s := range svcs {
				res, err := askService(ls, s.ID, since)
				if err != nil {
					continue
				}
				sc := score.Compute(res)
				rows = append(rows, row{Service: s.ID, Health: s.Health, Score: sc.Score, Level: sc.Level})
			}
			sort.SliceStable(rows, func(i, j int) bool {
				if rows[i].Score != rows[j].Score {
					return rows[i].Score > rows[j].Score
				}
				return rows[i].Service < rows[j].Service
			})
			if len(rows) > limit {
				rows = rows[:limit]
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), rows)
			}
			cmd.Printf("%-4s %-16s %-12s %-6s %s\n", "#", "SERVICE", "HEALTH", "SCORE", "LEVEL")
			for i, r := range rows {
				cmd.Printf("%-4d %-16s %-12s %-6d %s\n", i+1, r.Service, r.Health, r.Score, r.Level)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().IntVar(&limit, "limit", 10, "max services to show")
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	return cmd
}
