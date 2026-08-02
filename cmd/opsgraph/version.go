package main

import (
	"fmt"

	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/sanjeev0120test/opsgraph/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the opsgraph version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "table" && format != "json" {
				return fail(2, "invalid --format %q (want table or json)", format)
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), map[string]string{
					"version": version.Version,
					"commit":  version.Commit,
					"date":    version.Date,
				})
			}
			// Use OutOrStdout explicitly: cobra's Println falls back to stderr
			// when Command.out is unset, which breaks release/install smoke checks
			// that capture stdout.
			fmt.Fprintln(cmd.OutOrStdout(), "opsgraph", version.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
