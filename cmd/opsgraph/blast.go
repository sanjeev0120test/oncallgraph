package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newBlastCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:               "blast <service>",
		Short:             "Show 1-hop upstream/downstream blast radius for a service",
		Long:              "Shows immediate (1-hop) upstream and downstream neighbors. For recursive downstream impact, use `opsgraph impact`.",
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
			out := struct {
				Service    string              `json:"service"`
				Health     string              `json:"health"`
				Upstream   []map[string]string `json:"upstream"`
				Downstream []map[string]string `json:"downstream"`
			}{
				Service:    res.Service.ID,
				Health:     res.Service.Health,
				Upstream:   []map[string]string{},
				Downstream: []map[string]string{},
			}
			for _, u := range res.Upstream {
				out.Upstream = append(out.Upstream, map[string]string{"id": u.ID, "health": u.Health})
			}
			for _, d := range res.Downstream {
				out.Downstream = append(out.Downstream, map[string]string{"id": d.ID, "health": d.Health})
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), out)
			}
			cmd.Printf("SERVICE     %s (%s)\n", out.Service, out.Health)
			cmd.Println("UPSTREAM (1-hop)")
			if len(out.Upstream) == 0 {
				cmd.Println("  (none)")
			}
			for _, u := range out.Upstream {
				cmd.Printf("  %-16s %s\n", u["id"], u["health"])
			}
			cmd.Println("DOWNSTREAM (1-hop)")
			if len(out.Downstream) == 0 {
				cmd.Println("  (none)")
			}
			for _, d := range out.Downstream {
				cmd.Printf("  %-16s %s\n", d["id"], d["health"])
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
