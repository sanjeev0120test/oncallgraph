package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/fingerprint"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newFingerprintCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:   "fingerprint <service>",
		Short: "Compute a deterministic incident fingerprint for dedup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireArg("service", args[0]); err != nil {
				return err
			}
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, cfg, err := src.loadCtx(cmd.Context(), since)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			res, err := askService(ls, args[0], since)
			if err != nil {
				return failAsk(err)
			}
			fp := fingerprint.Of(res)
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), fp)
			}
			cmd.Printf("FINGERPRINT  %s\n", fp.Fingerprint)
			cmd.Printf("SERVICE      %s\n", fp.Service)
			if len(fp.Inputs) == 0 {
				cmd.Println("INPUT        (none)")
			}
			for _, in := range fp.Inputs {
				cmd.Printf("INPUT        %s\n", in)
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
