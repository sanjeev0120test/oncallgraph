package main

import (
	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/opsgraph/opsgraph/internal/output"
	"github.com/opsgraph/opsgraph/internal/runbook"
	"github.com/spf13/cobra"
)

func newVerifyRunbookCmd() *cobra.Command {
	var (
		fixture string
		format  string
	)
	cmd := &cobra.Command{
		Use:   "verify-runbook <service>",
		Short: "Check whether a service's runbook is still valid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			if fixture == "" {
				return fail(2, "no data source: pass --fixture <pack>")
			}
			ls, err := storeFromFixtureDir(fixture)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()

			// Resolve alias/name to a canonical id first.
			svc, err := ls.store.GetServiceByNameOrAlias(args[0])
			if err != nil {
				return fail(1, "service %q not found", args[0])
			}
			res, err := runbook.NewVerifier(ls.store, ls.now).VerifyService(svc.ID)
			if err != nil {
				return fail(2, "%v", err)
			}

			if format == "json" {
				if err := output.JSON(cmd.OutOrStdout(), res); err != nil {
					return err
				}
			} else if err := output.VerifyTable(cmd.OutOrStdout(), res); err != nil {
				return err
			}

			switch res.Status {
			case model.StatusPass:
				return nil
			case model.StatusStale, model.StatusFail:
				return &exitError{code: 1, err: errStatus(res.Status)}
			default: // missing / error
				return &exitError{code: 2, err: errStatus(res.Status)}
			}
		},
	}
	cmd.Flags().StringVar(&fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}

type statusErr string

func (e statusErr) Error() string { return "runbook status: " + string(e) }

func errStatus(s string) error { return statusErr(s) }
