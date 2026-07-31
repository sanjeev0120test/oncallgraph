package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ai"
	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newDemoCmd() *cobra.Command {
	var (
		format string
		useAI  bool
	)
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Run a full simulated incident from the built-in fixture",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			ls, err := embeddedCheckout()
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()

			res, err := ask.Ask(ls.store, "checkout", ask.Options{Since: time.Hour, Now: ls.now, WithRunbook: true})
			if err != nil {
				return fail(2, "%v", err)
			}
			if useAI {
				cfg := config.Default()
				cfg.AI.Enabled = true
				res.AISummary = ai.Summarize(cmd.Context(), cfg, res)
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			return output.Table(cmd.OutOrStdout(), res)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().BoolVar(&useAI, "ai", false, "add a local AI summary (needs Ollama; degrades gracefully)")
	return cmd
}
