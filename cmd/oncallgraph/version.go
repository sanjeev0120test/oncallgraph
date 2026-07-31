package main

import (
	"github.com/sanjeev0120test/oncallgraph/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the oncallgraph version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("oncallgraph", version.String())
			return nil
		},
	}
}
