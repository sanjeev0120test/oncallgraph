package main

import (
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newServicesCmd() *cobra.Command {
	var src sourceFlags
	var health string
	var format string
	cmd := &cobra.Command{
		Use:   "services",
		Short: "List services and their health",
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
			type row struct {
				ID      string   `json:"id"`
				Name    string   `json:"name"`
				Health  string   `json:"health"`
				OwnerID string   `json:"owner_id,omitempty"`
				Aliases []string `json:"aliases,omitempty"`
			}
			out := make([]row, 0, len(svcs))
			for _, s := range svcs {
				if health != "" && s.Health != health {
					continue
				}
				out = append(out, row{ID: s.ID, Name: s.Name, Health: s.Health, OwnerID: s.OwnerID, Aliases: s.Aliases})
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), out)
			}
			if len(out) == 0 {
				cmd.Println("(no services)")
				return nil
			}
			cmd.Printf("%-16s %-12s %-16s %s\n", "ID", "HEALTH", "OWNER", "ALIASES")
			for _, r := range out {
				alias := ""
				if len(r.Aliases) > 0 {
					alias = r.Aliases[0]
					if len(r.Aliases) > 1 {
						alias += ",…"
					}
				}
				cmd.Printf("%-16s %-12s %-16s %s\n", r.ID, r.Health, r.OwnerID, alias)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&health, "health", "", "filter by health: healthy|degraded|unhealthy|unknown")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
