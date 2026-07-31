package main

import (
	"strings"

	"github.com/sanjeev0120test/oncallgraph/internal/model"
	"github.com/sanjeev0120test/oncallgraph/internal/output"
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	var src sourceFlags
	var format string
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Fleet health dashboard: counts by health state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, _, err := src.load(0)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			svcs, err := ls.store.ListServices()
			if err != nil {
				return fail(2, "%v", err)
			}
			counts := map[string]int{
				model.HealthHealthy: 0, model.HealthDegraded: 0,
				model.HealthUnhealthy: 0, model.HealthUnknown: 0,
			}
			by := map[string][]string{}
			for _, s := range svcs {
				counts[s.Health]++
				by[s.Health] = append(by[s.Health], s.ID)
			}
			out := struct {
				Total   int                 `json:"total"`
				Counts  map[string]int      `json:"counts"`
				ByState map[string][]string `json:"by_state"`
			}{Total: len(svcs), Counts: counts, ByState: by}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), out)
			}
			cmd.Printf("FLEET HEALTH  (%d services)\n", out.Total)
			for _, k := range []string{model.HealthUnhealthy, model.HealthDegraded, model.HealthUnknown, model.HealthHealthy} {
				ids := by[k]
				if len(ids) == 0 {
					cmd.Printf("  %-12s %d\n", k, counts[k])
					continue
				}
				cmd.Printf("  %-12s %d  (%s)\n", k, counts[k], strings.Join(ids, ", "))
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
