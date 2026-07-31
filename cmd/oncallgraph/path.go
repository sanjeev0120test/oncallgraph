package main

import (
	"github.com/sanjeev0120test/oncallgraph/internal/output"
	"github.com/sanjeev0120test/oncallgraph/internal/pathfind"
	"github.com/spf13/cobra"
)

func newPathCmd() *cobra.Command {
	var src sourceFlags
	var format string
	cmd := &cobra.Command{
		Use:   "path <from> <to>",
		Short: "Find the shortest depends-on path between two services",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, _, err := src.load(0)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()
			deps, err := ls.store.ListAllDependencies()
			if err != nil {
				return fail(2, "%v", err)
			}
			p, err := pathfind.Shortest(deps, args[0], args[1])
			if err != nil {
				return fail(1, "%v", err)
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), p)
			}
			cmd.Printf("PATH  %v\n", p.Nodes)
			cmd.Printf("HOPS  %d\n", p.Hops)
			return nil
		},
	}
	cmd.Flags().StringVar(&src.fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&src.configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&src.dataDir, "data-dir", "", "persistent store directory")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
