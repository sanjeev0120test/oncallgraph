package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
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
		Long: "Poll a live or persistent data source until the service reports healthy.\n" +
			"Fixture packs are static snapshots — watch will timeout if the fixture service is not healthy.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if interval <= 0 {
				return fail(2, "invalid --interval %s (must be > 0)", interval)
			}
			if timeout <= 0 {
				return fail(2, "invalid --timeout %s (must be > 0)", timeout)
			}
			if src.fixture != "" {
				cmd.PrintErrln("note: --fixture is a static snapshot; watch only succeeds if the fixture service is already healthy")
			}
			if src.dataDir != "" {
				cmd.PrintErrln("note: --data-dir re-reads the store each tick; re-run `opsgraph ingest` (or use live config) for fresh health")
			}
			deadline := time.Now().Add(timeout)
			var last string
			lastHealth := ""
			svcID := args[0]
			for {
				if err := cmd.Context().Err(); err != nil {
					return fail(1, "cancelled")
				}
				if time.Now().After(deadline) {
					if lastHealth != "" {
						return fail(1, "watch timeout: %s still %s", svcID, lastHealth)
					}
					return fail(1, "watch timeout: %s", svcID)
				}
				ls, cfg, err := src.load(since)
				if err != nil {
					return failSource(err)
				}
				win := since
				if win == 0 {
					win = cfg.Since()
				}
				res, err := askService(ls, args[0], win)
				ls.cleanup()
				if err != nil {
					return failAsk(err)
				}
				svcID = res.Service.ID
				lastHealth = res.Service.Health
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
				remaining := time.Until(deadline)
				sleep := interval
				if remaining < sleep {
					sleep = remaining
				}
				select {
				case <-cmd.Context().Done():
					return fail(1, "cancelled")
				case <-time.After(sleep):
				}
			}
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "poll interval")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "give up after this duration")
	return cmd
}
