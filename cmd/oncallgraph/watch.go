package main

import (
	"errors"
	"time"

	"github.com/sanjeev0120test/oncallgraph/internal/ask"
	"github.com/sanjeev0120test/oncallgraph/internal/model"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var interval time.Duration
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "watch <service>",
		Short: "Poll a service until healthy (or timeout)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if interval <= 0 {
				interval = 2 * time.Second
			}
			if timeout <= 0 {
				timeout = 10 * time.Second
			}
			deadline := time.Now().Add(timeout)
			var last string
			for {
				ls, cfg, err := src.load(since)
				if err != nil {
					return fail(2, "%v", err)
				}
				win := since
				if win == 0 {
					win = cfg.Since()
				}
				res, err := askService(ls, args[0], win)
				ls.cleanup()
				if err != nil {
					if errors.Is(err, ask.ErrServiceNotFound) {
						return fail(1, "%v", err)
					}
					return fail(2, "%v", err)
				}
				line := res.Service.ID + " " + res.Service.Health
				if line != last {
					cmd.Printf("%s  %s\n", time.Now().UTC().Format(time.RFC3339), line)
					last = line
				}
				if res.Service.Health == model.HealthHealthy {
					return nil
				}
				if time.Now().After(deadline) {
					return fail(1, "watch timeout: %s still %s", res.Service.ID, res.Service.Health)
				}
				// In CI/tests, one sample is enough when --timeout is tiny and fixture is degraded.
				select {
				case <-cmd.Context().Done():
					return fail(1, "cancelled")
				case <-time.After(interval):
				}
			}
		},
	}
	cmd.Flags().StringVar(&src.fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&src.configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&src.dataDir, "data-dir", "", "persistent store directory")
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "poll interval")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "give up after this duration")
	return cmd
}
