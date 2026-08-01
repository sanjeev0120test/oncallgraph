package main

import (
	"path/filepath"
	"sort"
	"strings"
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
		merge      bool
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
			if replace && merge {
				return fail(2, "use either --replace or --merge, not both")
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

			// Fixtures always replace. Live ingest replaces by default so
			// removed entities cannot linger; pass --merge to upsert-only.
			doReset := fixture != "" || replace || !merge

			lookback := since
			if lookback <= 0 {
				lookback = cfg.Since()
			}

			now := time.Now().UTC()
			liveClock := false
			switch {
			case fixture != "":
				if doReset {
					if err := s.Reset(); err != nil {
						return fail(2, "reset store: %v", err)
					}
				}
				now, err = ingest.IngestFixtureDir(s, fixture)
			default:
				if effPath == "" {
					return fail(2, "no data source: pass --fixture <pack> or add a .opsgraph.yaml")
				}
				sinceAt := now.Add(-ask.ChangeLookback(lookback))
				// Always scrape into a temp store first. On replace, atomically
				// swap the validated DB (no second scrape that can wipe state).
				tmp, cleanup, terr := store.OpenTemp()
				if terr != nil {
					return fail(2, "%v", terr)
				}
				terr = ingest.LiveIngest(cmd.Context(), tmp, cfg, configDir, sinceAt, now)
				if terr != nil {
					cleanup()
					return fail(2, "%v", terr)
				}
				if doReset {
					terr = s.ReplaceFromFile(tmp.Path())
					cleanup()
					if terr != nil {
						return fail(2, "%v", terr)
					}
				} else {
					cleanup()
					// Merge: re-scrape into the persistent store (upsert-only).
					terr = ingest.LiveIngest(cmd.Context(), s, cfg, configDir, sinceAt, now)
					if terr != nil {
						return fail(2, "%v", terr)
					}
				}
				if err := s.ClearAsOf(); err != nil {
					return fail(2, "%v", err)
				}
				liveClock = true
				err = nil
			}
			if err != nil {
				return fail(2, "%v", err)
			}
			if collisions, cerr := s.FindAliasCollisions(); cerr != nil {
				return fail(2, "%v", cerr)
			} else if len(collisions) > 0 {
				return fail(1, "ambiguous service aliases after ingest: %s", strings.Join(collisions, "; "))
			}
			countsCheck, cerr := s.Counts()
			if cerr != nil {
				return fail(2, "%v", cerr)
			}
			if countsCheck["services"] == 0 {
				return fail(1, "ingest produced zero services")
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
			if liveClock {
				cmd.Printf("  ingested_at:  %s\n", now.UTC().Format(time.RFC3339))
				cmd.Printf("  as_of:        (wall clock)\n")
			} else {
				cmd.Printf("  as_of:        %s\n", now.UTC().Format(time.RFC3339))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "persistent store directory (default: config data_dir or .opsgraph/data)")
	cmd.Flags().BoolVar(&replace, "replace", false, "clear the store before ingest (implied unless --merge)")
	cmd.Flags().BoolVar(&merge, "merge", false, "upsert without clearing prior rows (live ingest only; fixtures always replace)")
	cmd.Flags().DurationVar(&since, "since", 0, "change lookback for live git/helm ingest (default: config default_since)")
	return cmd
}
