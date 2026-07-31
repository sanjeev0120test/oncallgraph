package main

import (
	"errors"

	"github.com/sanjeev0120test/oncallgraph/internal/impact"
	"github.com/sanjeev0120test/oncallgraph/internal/output"
	"github.com/sanjeev0120test/oncallgraph/internal/store"
	"github.com/spf13/cobra"
)

func newImpactCmd() *cobra.Command {
	var src sourceFlags
	var format string
	cmd := &cobra.Command{
		Use:   "impact <service>",
		Short: "Show recursive downstream impact if a service fails",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, _, err := src.load(0)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()
			svc, err := ls.store.GetServiceByNameOrAlias(args[0])
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return fail(1, "service %q not found", args[0])
				}
				return fail(2, "%v", err)
			}
			svcs, err := ls.store.ListServices()
			if err != nil {
				return fail(2, "%v", err)
			}
			deps, err := ls.store.ListAllDependencies()
			if err != nil {
				return fail(2, "%v", err)
			}
			res := impact.Downstream(svc.ID, svcs, deps)
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			cmd.Printf("ROOT       %s\n", res.Root)
			cmd.Printf("AFFECTED   %d (max depth %d)\n", len(res.Affected), res.MaxDepth)
			if len(res.Affected) == 0 {
				cmd.Println("TREE       (no downstream dependents)")
				return nil
			}
			cmd.Println("TREE")
			printImpactNode(cmd, res.Tree, "")
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}

func printImpactNode(cmd *cobra.Command, n impact.Node, prefix string) {
	health := n.Health
	if health == "" {
		health = "unknown"
	}
	if n.Depth == 0 {
		cmd.Printf("%s%s [%s]\n", prefix, n.ID, health)
	} else {
		cmd.Printf("%s└─ %s [%s]\n", prefix, n.ID, health)
	}
	childPrefix := prefix + "   "
	for _, c := range n.Children {
		printImpactNode(cmd, c, childPrefix)
	}
}
