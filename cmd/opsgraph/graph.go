package main

import (
	"fmt"

	"github.com/sanjeev0120test/opsgraph/internal/graphviz"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newGraphCmd() *cobra.Command {
	var src sourceFlags
	var format string
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Render the service dependency graph (ascii, mermaid, or json)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch format {
			case "ascii", "table", "mermaid", "json":
				// table is an alias for ascii (shared --format vocabulary).
			default:
				return fail(2, "invalid --format %q (want ascii|table|mermaid|json)", format)
			}
			ls, _, err := src.loadCtx(cmd.Context(), 0)
			if err != nil {
				return failSource(err)
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
			out := cmd.OutOrStdout()
			if format == "json" {
				return output.JSON(out, graphviz.JSONGraph(svcs, deps))
			}
			var body string
			if format == "mermaid" {
				body = graphviz.Mermaid(svcs, deps)
			} else {
				body = graphviz.ASCII(svcs, deps)
			}
			if _, err := fmt.Fprint(out, body); err != nil {
				return fail(2, "%v", err)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&format, "format", "ascii", "output format: ascii|table|mermaid|json")
	return cmd
}
