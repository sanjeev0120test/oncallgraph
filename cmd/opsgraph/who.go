package main

import (
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/spf13/cobra"
)

func newWhoCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var format string
	cmd := &cobra.Command{
		Use:               "who <service>",
		Short:             "Show who owns a service and who last changed it",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServiceArg,
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
			type who struct {
				Service           string `json:"service"`
				OwnerID           string `json:"owner_id,omitempty"`
				OwnerName         string `json:"owner_name,omitempty"`
				OwnerEmail        string `json:"owner_email,omitempty"`
				LastChange        string `json:"last_change,omitempty"`
				LastAuthor        string `json:"last_author,omitempty"`
				LastType          string `json:"last_type,omitempty"`
				LastRev           string `json:"last_revision,omitempty"`
				EvidenceID        string `json:"evidence_id,omitempty"`
				OlderLookbackOnly bool   `json:"older_lookback_only,omitempty"`
			}
			w := who{Service: res.Service.ID}
			if res.Owner != nil {
				w.OwnerID = res.Owner.ID
				w.OwnerName = res.Owner.Name
				w.OwnerEmail = res.Owner.Email
			}
			// Align with R1 / explain / fingerprint: 30m suspect window, not full lookback.
			if c, ok := ask.RecentSuspectChange(res); ok {
				w.LastChange = c.Summary
				w.LastAuthor = c.Author
				w.LastType = c.Type
				w.LastRev = c.Revision
				w.EvidenceID = c.EvidenceID
			} else if len(res.Changes) > 0 {
				w.OlderLookbackOnly = true
			}
			if format == "json" {
				return output.JSON(cmd.OutOrStdout(), w)
			}
			cmd.Printf("SERVICE  %s\n", w.Service)
			if w.OwnerName != "" || w.OwnerID != "" {
				cmd.Printf("OWNER    %s", w.OwnerName)
				if w.OwnerEmail != "" {
					cmd.Printf(" <%s>", w.OwnerEmail)
				}
				cmd.Printf(" (%s)\n", w.OwnerID)
			} else {
				cmd.Println("OWNER    (unknown)")
			}
			if w.LastChange != "" {
				cmd.Printf("CHANGED  %s by %s (%s %s) [%s]\n", w.LastChange, w.LastAuthor, w.LastType, w.LastRev, w.EvidenceID)
			} else if len(res.Changes) > 0 {
				cmd.Println("CHANGED  (none in 30m suspect window; older lookback changes exist)")
			} else {
				cmd.Println("CHANGED  (none in 30m suspect window)")
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	bindFormatCompletion(cmd)
	return cmd
}
