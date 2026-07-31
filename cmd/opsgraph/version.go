package main

import (
	"github.com/opsgraph/opsgraph/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the opsgraph version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("opsgraph", version.String())
			return nil
		},
	}
}
