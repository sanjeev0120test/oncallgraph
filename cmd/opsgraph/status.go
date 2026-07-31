package main

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var (
		fixture    string
		configPath string
	)
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show connector configuration and ingested data counts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fail(2, "%v", err)
			}

			cmd.Println("CONNECTORS")
			cmd.Printf("  fixtures:     %v\n", cfg.Connectors.Fixtures.Enabled)
			cmd.Printf("  git:          %v (repo %q)\n", cfg.Connectors.Git.Enabled, cfg.Connectors.Git.RepoPath)
			cmd.Printf("  kubernetes:   %v (snapshot %q)\n", cfg.Connectors.Kubernetes.Enabled, cfg.Connectors.Kubernetes.Snapshot)
			cmd.Printf("  prometheus:   %v (phase-2)\n", cfg.Connectors.Prometheus.Enabled)
			cmd.Printf("  alertmanager: %v (phase-2)\n", cfg.Connectors.Alertmanager.Enabled)
			cmd.Printf("AI\n  enabled: %v  model: %s  embed: %s  url: %s\n",
				cfg.AI.Enabled, cfg.AI.Model, cfg.AI.EmbedModel, cfg.AI.OllamaURL)

			var ls *loadedStore
			if fixture != "" {
				ls, err = storeFromFixtureDir(fixture)
			} else if effPath := resolveConfigPath(configPath); effPath != "" {
				ls, err = storeFromConfig(cfg, dirOf(effPath), cfg.Since(), time.Now().UTC())
			} else {
				cmd.Println("\nNo data source (pass --fixture <pack> or add .opsgraph.yaml) - showing config only.")
				return nil
			}
			if err != nil {
				return fail(2, "%v", err)
			}
			defer ls.cleanup()

			counts, err := ls.store.Counts()
			if err != nil {
				return fail(2, "%v", err)
			}
			cmd.Println("\nINGESTED")
			for _, k := range sortedKeys(counts) {
				cmd.Printf("  %-13s %d\n", k+":", counts[k])
			}

			services, err := ls.store.ListServices()
			if err != nil {
				return fail(2, "%v", err)
			}
			if len(services) > 0 {
				cmd.Println("SERVICES")
				for _, s := range services {
					cmd.Printf("  %-16s %s\n", s.ID, s.Health)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&configPath, "config", "", "path to .opsgraph.yaml (default: ./.opsgraph.yaml if present)")
	return cmd
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func dirOf(p string) string {
	if p == ".opsgraph.yaml" {
		return "."
	}
	return filepath.Dir(p)
}
