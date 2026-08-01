package main

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/ingest"
	"github.com/sanjeev0120test/opsgraph/internal/store"
	"github.com/spf13/cobra"
)

func newIngestCmd() *cobra.Command {
	var (
		fixture    string
		configPath string
		dataDir    string
		replace    bool
		since      time.Duration
	)
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Load fixture or live connectors into the local store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validSince(since); err != nil {
				return fail(2, "%v", err)
			}
			cfg, err := config.Load(configPath)
			if err != nil {
				return fail(2, "%v", err)
			}
			effPath := resolveConfigPath(configPath)
			configDir := "."
			if effPath != "" {
				configDir = filepath.Dir(effPath)
			}
			dir := resolveDataDir(dataDir, cfg, configDir)
			s, err := store.Open(dir)
			if err != nil {
				return fail(2, "%v", err)
			}
			defer s.Close()

			// Fixture packs are complete snapshots: always replace so removed
			// entities do not linger. Live ingest replaces only with --replace.
			if replace || fixture != "" {
				if err := s.Reset(); err != nil {
					return fail(2, "reset store: %v", err)
				}
			}

			lookback := since
			if lookback <= 0 {
				lookback = cfg.Since()
			}

			now := time.Now().UTC()
			switch {
			case fixture != "":
				now, err = ingest.IngestFixtureDir(s, fixture)
			default:
				if effPath == "" {
					return fail(2, "no data source: pass --fixture <pack> or add a .opsgraph.yaml")
				}
				err = ingest.LiveIngest(s, cfg, configDir, now.Add(-ask.ChangeLookback(lookback)), now)
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
	cmd.Flags().StringVar(&configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "persistent store directory (default: config data_dir or .opsgraph/data)")
	cmd.Flags().BoolVar(&replace, "replace", false, "clear the store before ingest (fixture ingest always replaces)")
	cmd.Flags().DurationVar(&since, "since", 0, "change lookback for live git/helm ingest (default: config default_since)")
	return cmd
}
