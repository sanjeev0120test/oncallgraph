package main

import (
	"time"

	"github.com/sanjeev0120test/oncallgraph/internal/output"
	"github.com/spf13/cobra"
)

func newAlertsCmd() *cobra.Command {
	var src sourceFlags
	var format string
	var firingOnly bool
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "List alerts across the fleet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, _, err := src.load(0)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()
			list, err := ls.store.ListAllAlerts()
			if err != nil {
				return fail(2, "%v", err)
			}
			if firingOnly {
				filtered := list[:0]
				for _, a := range list {
					if a.Status == "firing" {
						filtered = append(filtered, a)
					}
				}
				list = filtered
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), list)
			}
			if len(list) == 0 {
				cmd.Println("(no alerts)")
				return nil
			}
			for _, a := range list {
				cmd.Printf("%s  %-10s %-10s %-14s %s [%s]\n",
					a.At.Format(time.RFC3339), a.Status, a.Severity, a.ServiceID, a.Name, a.EvidenceID)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().BoolVar(&firingOnly, "firing", false, "only show firing alerts")
	return cmd
}
