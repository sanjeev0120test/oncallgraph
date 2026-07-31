package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ingest"
	"github.com/sanjeev0120test/opsgraph/internal/store"
	"github.com/spf13/cobra"
)

func newValidateFixtureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-fixture <dir>",
		Short: "Validate a fixture pack can be ingested and has required files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			required := []string{
				"meta.yaml", "services.yaml", "owners.yaml", "changes.yaml",
				"dependencies.yaml", "alerts.yaml",
			}
			var missing []string
			for _, f := range required {
				if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
					missing = append(missing, f)
				}
			}
			if len(missing) > 0 {
				return fail(1, "missing required files: %v", missing)
			}
			s, cleanup, err := store.OpenTemp()
			if err != nil {
				return fail(2, "%v", err)
			}
			defer cleanup()
			now, err := ingest.IngestFixtureDir(s, dir)
			if err != nil {
				return fail(1, "ingest failed: %v", err)
			}
			if now.IsZero() {
				return fail(1, "fixture meta.yaml must set a non-zero now: timestamp for determinism")
			}
			counts, err := s.Counts()
			if err != nil {
				return fail(2, "%v", err)
			}
			if counts["services"] == 0 {
				return fail(1, "fixture has zero services after ingest")
			}
			svcs, err := s.ListServices()
			if err != nil {
				return fail(2, "%v", err)
			}
			known := map[string]bool{}
			for _, svc := range svcs {
				known[svc.ID] = true
			}
			deps, err := s.ListAllDependencies()
			if err != nil {
				return fail(2, "%v", err)
			}
			for _, d := range deps {
				if !known[d.FromServiceID] {
					return fail(1, "dependency from unknown service %q", d.FromServiceID)
				}
				if !known[d.ToServiceID] {
					return fail(1, "dependency to unknown service %q (synthesize or declare it)", d.ToServiceID)
				}
			}
			changes, err := s.ListAllChanges()
			if err != nil {
				return fail(2, "%v", err)
			}
			for _, c := range changes {
				if c.ServiceID != "" && !known[c.ServiceID] {
					return fail(1, "change %q references unknown service %q", c.ID, c.ServiceID)
				}
			}
			alerts, err := s.ListAllAlerts()
			if err != nil {
				return fail(2, "%v", err)
			}
			for _, a := range alerts {
				if a.ServiceID != "" && !known[a.ServiceID] {
					return fail(1, "alert %q references unknown service %q", a.ID, a.ServiceID)
				}
			}
			cmd.Printf("ok - fixture %q valid (now=%s)\n", dir, now.UTC().Format(time.RFC3339))
			for _, k := range []string{"services", "owners", "changes", "dependencies", "alerts", "runbooks", "evidence"} {
				cmd.Printf("  %-13s %d\n", k+":", counts[k])
			}
			return nil
		},
	}
}
