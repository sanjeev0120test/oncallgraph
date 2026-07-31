package main

import (
	"os"
	"path/filepath"

	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/store"
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
			counts, err := s.Counts()
			if err != nil {
				return fail(2, "%v", err)
			}
			if counts["services"] == 0 {
				return fail(1, "fixture has zero services after ingest")
			}
			cmd.Printf("ok - fixture %q valid (now=%s)\n", dir, now.UTC().Format("2006-01-02T15:04:05Z"))
			for _, k := range []string{"services", "owners", "changes", "dependencies", "alerts", "runbooks", "evidence"} {
				cmd.Printf("  %-13s %d\n", k+":", counts[k])
			}
			return nil
		},
	}
}
