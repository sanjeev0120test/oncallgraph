package main

import (
	"fmt"

	"github.com/sanjeev0120test/opsgraph/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the opsgraph version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Use OutOrStdout explicitly: cobra's Println falls back to stderr
			// when Command.out is unset, which breaks release/install smoke checks
			// that capture stdout.
			fmt.Fprintln(cmd.OutOrStdout(), "opsgraph", version.String())
			return nil
		},
	}
}
