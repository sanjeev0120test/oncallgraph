package main

import (
	"github.com/sanjeev0120test/oncallgraph/internal/graphviz"
	"github.com/spf13/cobra"
)

func newGraphCmd() *cobra.Command {
	var src sourceFlags
	var format string
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Render the service dependency graph (ascii or mermaid)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "ascii" && format != "mermaid" {
				return fail(2, "invalid --format %q (want ascii|mermaid)", format)
			}
			ls, _, err := src.load(0)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()
			svcs, err := ls.store.ListServices()
			if err != nil {
				return fail(2, "%v", err)
			}
			deps, err := ls.store.ListAllDependencies()
			if err != nil {
				return fail(2, "%v", err)
			}
			if format == "mermaid" {
				cmd.Print(graphviz.Mermaid(svcs, deps))
				return nil
			}
			cmd.Print(graphviz.ASCII(svcs, deps))
			return nil
		},
	}
	cmd.Flags().StringVar(&src.fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&src.configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&src.dataDir, "data-dir", "", "persistent store directory")
	cmd.Flags().StringVar(&format, "format", "ascii", "output format: ascii|mermaid")
	return cmd
}
