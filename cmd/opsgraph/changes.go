package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newChangesCmd() *cobra.Command {
	var src sourceFlags
	var format string
	var service string
	var limit int
	var since time.Duration
	cmd := &cobra.Command{
		Use:   "changes",
		Short: "List recent changes (optionally filtered by service)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			if limit <= 0 {
				limit = 20
			}
			ls, cfg, err := src.load(since)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			cutoff := ls.now.Add(-since)

			var list []model.Change
			if service != "" {
				svc, err := ls.store.GetServiceByNameOrAlias(service)
				if err != nil {
					return fail(1, "service %q not found", service)
				}
				list, err = ls.store.ListChanges(svc.ID, cutoff)
				if err != nil {
					return fail(2, "%v", err)
				}
			} else {
				all, err := ls.store.ListAllChanges()
				if err != nil {
					return fail(2, "%v", err)
				}
				for _, c := range all {
					if c.At.Before(cutoff) {
						continue
					}
					list = append(list, c)
				}
			}
			if len(list) > limit {
				list = list[:limit]
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), list)
			}
			for _, c := range list {
				cmd.Printf("%s  %-10s %-14s %s [%s]\n", c.At.Format(time.RFC3339), c.Type, c.ServiceID, c.Summary, c.EvidenceID)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().StringVar(&service, "service", "", "filter to one service name/alias")
	cmd.Flags().IntVar(&limit, "limit", 20, "max changes to show")
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window (default: config default_since or 60m)")
	return cmd
}
