package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/sanjeev0120test/oncallgraph/internal/ask"
	"github.com/sanjeev0120test/oncallgraph/internal/model"
	"github.com/spf13/cobra"
)

func newWhyCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	cmd := &cobra.Command{
		Use:   "why <service>",
		Short: "One-line root-cause hypothesis for a paged service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ls, cfg, err := src.load(since)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			res, err := askService(ls, args[0], since)
			if err != nil {
				if errors.Is(err, ask.ErrServiceNotFound) {
					return fail(1, "%v", err)
				}
				return fail(2, "%v", err)
			}
			cmd.Println(whyLine(res))
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	return cmd
}

func whyLine(res model.AskResult) string {
	line := fmt.Sprintf("%s is %s", res.Service.ID, res.Service.Health)
	if len(res.Changes) > 0 {
		c := res.Changes[0]
		line += fmt.Sprintf("; prime suspect %s %q", c.Type, c.Summary)
		if c.EvidenceID != "" {
			line += " [" + c.EvidenceID + "]"
		}
	}
	for _, u := range res.Upstream {
		if u.Health == model.HealthUnhealthy || u.Health == model.HealthDegraded {
			line += fmt.Sprintf("; upstream %s is %s", u.ID, u.Health)
			break
		}
	}
	for _, a := range res.Alerts {
		if a.Status == "firing" {
			line += "; alert " + a.Name + " firing"
			break
		}
	}
	return line + "."
}
