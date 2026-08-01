package main

import (
	"sort"

	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newOwnersCmd() *cobra.Command {
	var src sourceFlags
	var format string
	cmd := &cobra.Command{
		Use:   "owners",
		Short: "List owners and the services they own",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, _, err := src.loadCtx(cmd.Context(), 0)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			owners, err := ls.store.ListOwners()
			if err != nil {
				return fail(2, "%v", err)
			}
			type row struct {
				ID       string   `json:"id"`
				Name     string   `json:"name"`
				Email    string   `json:"email,omitempty"`
				Services []string `json:"services"`
			}
			out := make([]row, 0, len(owners))
			for _, o := range owners {
				svcs, err := ls.store.ListServicesByOwner(o.ID)
				if err != nil {
					return fail(2, "%v", err)
				}
				ids := make([]string, 0, len(svcs))
				for _, s := range svcs {
					ids = append(ids, s.ID)
				}
				sort.Strings(ids)
				out = append(out, row{ID: o.ID, Name: o.Name, Email: o.Email, Services: ids})
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), out)
			}
			if len(out) == 0 {
				cmd.Println("(no owners)")
				return nil
			}
			cmd.Printf("%-14s %-22s %-28s %s\n", "ID", "NAME", "EMAIL", "SERVICES")
			for _, r := range out {
				svc := ""
				for i, id := range r.Services {
					if i > 0 {
						svc += ", "
					}
					svc += id
				}
				cmd.Printf("%-14s %-22s %-28s %s\n", r.ID, r.Name, r.Email, svc)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
