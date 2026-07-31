package main

import (
	"errors"

	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/sanjeev0120test/opsgraph/internal/store"
	"github.com/spf13/cobra"
)

func newEvidenceCmd() *cobra.Command {
	var src sourceFlags
	var format string
	cmd := &cobra.Command{
		Use:   "evidence <id>",
		Short: "Look up a single evidence record by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, _, err := src.load(0)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			ev, err := ls.store.GetEvidence(args[0])
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return fail(1, "evidence %q not found", args[0])
				}
				return fail(2, "%v", err)
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), ev)
			}
			cmd.Printf("ID       %s\n", ev.ID)
			cmd.Printf("KIND     %s\n", ev.Kind)
			cmd.Printf("SOURCE   %s\n", ev.Source)
			cmd.Printf("AT       %s\n", ev.At.UTC().Format("2006-01-02T15:04:05Z"))
			cmd.Printf("SUMMARY  %s\n", ev.Summary)
			if ev.ServiceID != "" {
				cmd.Printf("SERVICE  %s\n", ev.ServiceID)
			}
			if ev.RawRef != "" {
				cmd.Printf("RAW_REF  %s\n", ev.RawRef)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
