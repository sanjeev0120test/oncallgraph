package main

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/store"
	"github.com/spf13/cobra"
)

func newIngestCmd() *cobra.Command {
	var (
		fixture    string
		configPath string
		dataDir    string
	)
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Load fixture or live connectors into the local store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fail(2, "%v", err)
			}
			dir := resolveDataDir(dataDir, cfg)
			s, err := store.Open(dir)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer s.Close()

			now := time.Now().UTC()
			switch {
			case fixture != "":
				now, err = ingest.IngestFixtureDir(s, fixture)
			default:
				effPath := resolveConfigPath(configPath)
				if effPath == "" {
					return fail(2, "no data source: pass --fixture <pack> or add a .opsgraph.yaml")
				}
				err = ingest.LiveIngest(s, cfg, filepath.Dir(effPath), now.Add(-cfg.Since()), now)
			}
			if err != nil {
				return fail(2, "%v", err)
			}

			counts, err := s.Counts()
			if err != nil {
				return fail(2, "%v", err)
			}
			cmd.Printf("ingested into %s (schema v%d)\n", s.Path(), store.SchemaVersion)
			keys := make([]string, 0, len(counts))
			for k := range counts {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				cmd.Printf("  %-13s %d\n", k+":", counts[k])
			}
			cmd.Printf("  as_of:        %s\n", now.UTC().Format(time.RFC3339))
			return nil
		},
	}
	cmd.Flags().StringVar(&fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&configPath, "config", "", "path to .opsgraph.yaml (default: ./.opsgraph.yaml if present)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "persistent store directory (default: config data_dir or .opsgraph/data)")
	return cmd
}
