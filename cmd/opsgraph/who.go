package main

import (
	"errors"
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
		Use:   "who <service>",
		Short: "Show who owns a service and who last changed it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(format); err != nil {
				return fail(2, "%v", err)
			}
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
			type who struct {
				Service    string `json:"service"`
				OwnerID    string `json:"owner_id,omitempty"`
				OwnerName  string `json:"owner_name,omitempty"`
				OwnerEmail string `json:"owner_email,omitempty"`
				LastChange string `json:"last_change,omitempty"`
				LastAuthor string `json:"last_author,omitempty"`
				LastType   string `json:"last_type,omitempty"`
				LastRev    string `json:"last_revision,omitempty"`
				EvidenceID string `json:"evidence_id,omitempty"`
			}
			w := who{Service: res.Service.ID}
			if res.Owner != nil {
				w.OwnerID = res.Owner.ID
				w.OwnerName = res.Owner.Name
				w.OwnerEmail = res.Owner.Email
			}
			if len(res.Changes) > 0 {
				c := res.Changes[0]
				w.LastChange = c.Summary
				w.LastAuthor = c.Author
				w.LastType = c.Type
				w.LastRev = c.Revision
				w.EvidenceID = c.EvidenceID
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
			} else {
				cmd.Println("CHANGED  (none in window)")
			}
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	return cmd
}
