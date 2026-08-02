package main

import (
	"errors"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/sanjeev0120test/opsgraph/internal/store"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var interval time.Duration
	var timeout time.Duration
	var once bool
	var format string
	cmd := &cobra.Command{
		Use:   "watch <service>",
		Short: "Poll a service until healthy (or timeout)",
		Long: "Poll a live or persistent data source until the service reports healthy.\n" +
			"Fixture packs are static snapshots — watch will timeout if the fixture service is not healthy.\n" +
			"Pass --once for a single health check (exit 0 healthy, 1 otherwise).",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServiceArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireArg("service", args[0]); err != nil {
				return err
			}
			if format != "table" && format != "json" {
				return fail(2, "invalid --format %q (want table or json)", format)
			}
			if once {
				healthy, svcID, health, line, cont, err := watchTick(cmd, &src, args[0], since)
				if err != nil {
					return err
				}
				if cont {
					cmd.PrintErrln(line)
					return fail(1, "watch --once: transient load failure")
				}
				if health == "" {
					health = "unknown"
				}
				if format == "json" {
					_ = output.JSON(cmd.OutOrStdout(), map[string]any{
						"service": svcID,
						"health":  health,
						"healthy": healthy,
						"ok":      healthy,
					})
				} else {
					cmd.Printf("%s  %s\n", time.Now().UTC().Format(time.RFC3339), line)
				}
				if healthy {
					return nil
				}
				return fail(1, "watch --once: %s is %s", svcID, health)
			}
			if format == "json" {
				return fail(2, "--format json is only supported with --once")
			}
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
			if src.fixture == "" && src.dataDir == "" && interval < 5*time.Second {
				cmd.PrintErrln("warning: short --interval may re-scrape live connectors often; prefer >=5s or --data-dir + external ingest")
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

				healthy, nextID, nextHealth, line, cont, err := watchTick(cmd, &src, args[0], since)
				if err != nil {
					return err
				}
				if cont {
					cmd.PrintErrln(line)
				} else {
					svcID = nextID
					lastHealth = nextHealth
					if line != last {
						cmd.Printf("%s  %s\n", time.Now().UTC().Format(time.RFC3339), line)
						last = line
					}
					if healthy {
						return nil
					}
				}

				remaining := time.Until(deadline)
				sleep := interval
				if remaining < sleep {
					sleep = remaining
				}
				if sleep <= 0 {
					// Do not busy-spin; surface timeout with last observed health.
					if lastHealth != "" {
						return fail(1, "watch timeout: %s still %s", svcID, lastHealth)
					}
					return fail(1, "watch timeout: %s", svcID)
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
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "poll interval (prefer >=5s with live connectors)")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "give up after this duration")
	cmd.Flags().BoolVar(&once, "once", false, "check health once and exit (0=healthy, 1=not)")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json (json requires --once)")
	bindFormatCompletion(cmd)
	return cmd
}

// watchTick loads once. cont=true means retry after sleep (transient error).
func watchTick(cmd *cobra.Command, src *sourceFlags, query string, since time.Duration) (healthy bool, svcID, health, line string, cont bool, err error) {
	ls, cfg, err := src.loadCtx(cmd.Context(), since)
	if err != nil {
		if watchFatalLoad(err) {
			return false, "", "", "", false, failSource(err)
		}
		return false, "", "", "warning: watch load: " + err.Error(), true, nil
	}
	win := since
	if win == 0 {
		win = cfg.Since()
	}
	res, err := askService(ls, query, win)
	ls.cleanup()
	if err != nil {
		if errors.Is(err, ask.ErrServiceNotFound) || errors.Is(err, store.ErrAmbiguous) {
			return false, "", "", "", false, failAsk(err)
		}
		return false, "", "", "warning: watch ask: " + err.Error(), true, nil
	}
	line = res.Service.ID + " " + res.Service.Health
	return res.Service.Health == model.HealthHealthy, res.Service.ID, res.Service.Health, line, false, nil
}

func watchFatalLoad(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrEmptyStore) || errors.Is(err, ErrNoDataSource) || errors.Is(err, ErrInvalidSince) {
		return true
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "parse") && strings.Contains(msg, "yaml"):
		return true
	case strings.Contains(msg, "read config"), strings.Contains(msg, "parse config"):
		return true
	default:
		return false
	}
}
