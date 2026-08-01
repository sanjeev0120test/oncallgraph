package main

import (
	"errors"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ai"
	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/sanjeev0120test/opsgraph/internal/store"
	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	var (
		fixture    string
		configPath string
		dataDir    string
		format     string
		since      time.Duration
		withRB     bool
		useAI      bool
	)
	cmd := &cobra.Command{
		Use:   "ask <service>",
		Short: "Show evidence-backed incident context for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireArg("service", args[0]); err != nil {
				return err
			}
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
			if err := validSince(since); err != nil {
				return fail(2, "%v", err)
			}
			cfg, err := config.Load(configPath)
			if err != nil {
				return fail(2, "%v", err)
			}
			if since == 0 {
				since = cfg.Since()
			}

			ls, err := loadAskStore(cmd.Context(), fixture, configPath, dataDir, cfg, since)
			if err != nil {
				if errors.Is(err, ErrEmptyStore) {
					return fail(1, "%v", err)
				}
				return fail(2, "%v", err)
			}
			defer ls.cleanup()
			res, err := ask.Ask(ls.store, args[0], ask.Options{Since: since, Now: ls.now, WithRunbook: withRB})
			if err != nil {
				if errors.Is(err, ask.ErrServiceNotFound) || errors.Is(err, store.ErrAmbiguous) {
					return fail(1, "%v", err)
				}
				return fail(2, "%v", err)
			}

			if useAI {
				cfg.AI.Enabled = true
				res.AISummary = ai.Summarize(cmd.Context(), cfg, res)
			}

			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			return output.Table(cmd.OutOrStdout(), res)
		},
	}
	cmd.Flags().StringVar(&fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "read from a persistent store (from `opsgraph ingest`)")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window (default: config default_since or 60m)")
	cmd.Flags().BoolVar(&withRB, "runbook", true, "verify the service runbook")
	cmd.Flags().BoolVar(&useAI, "ai", false, "add a local AI summary (needs Ollama; degrades gracefully)")
	return cmd
}
