package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var (
		fixture    string
		configPath string
		dataDir    string
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
			dir := resolveDataDir(dataDir, cfg)
			dbPath := filepath.Join(dir, "state.db")

			cmd.Println("STORE")
			cmd.Printf("  data_dir: %s\n", dir)
			cmd.Printf("  db:       %s\n", dbPath)

			cmd.Println("CONNECTORS")
			cmd.Printf("  fixtures:     %v\n", cfg.Connectors.Fixtures.Enabled)
			gitPath := cfg.Connectors.Git.RepoPath
			if gitPath == "" {
				gitPath = "."
			}
			gitOK := pathExists(filepath.Join(gitPath, ".git"))
			cmd.Printf("  git:          %v (repo %q has_git=%v)\n", cfg.Connectors.Git.Enabled, gitPath, gitOK)
			cmd.Printf("  kubernetes:   %v (snapshot %q)\n", cfg.Connectors.Kubernetes.Enabled, cfg.Connectors.Kubernetes.Snapshot)
			cmd.Printf("  prometheus:   %v (url %q)\n", cfg.Connectors.Prometheus.Enabled, cfg.Connectors.Prometheus.URL)
			cmd.Printf("  alertmanager: %v (url %q)\n", cfg.Connectors.Alertmanager.Enabled, cfg.Connectors.Alertmanager.URL)

			ollamaOK := probeOllama(cmd.Context(), cfg.AI.OllamaURL)
			cmd.Printf("AI\n  enabled: %v  model: %s  embed: %s  url: %s  reachable: %v\n",
				cfg.AI.Enabled, cfg.AI.Model, cfg.AI.EmbedModel, cfg.AI.OllamaURL, ollamaOK)

			var ls *loadedStore
			if fixture != "" {
				ls, err = storeFromFixtureDir(fixture)
			} else if pathExists(dbPath) {
				ls, err = storeFromDataDir(dir, time.Now().UTC())
			} else if effPath := resolveConfigPath(configPath); effPath != "" {
				ls, err = storeFromConfig(cfg, dirOf(effPath), cfg.Since(), time.Now().UTC())
			} else {
				cmd.Println("\nNo data source (pass --fixture <pack>, run `opsgraph ingest`, or add .opsgraph.yaml) - showing config only.")
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
			ver, _ := ls.store.UserVersion()
			cmd.Printf("\nINGESTED (schema v%d)\n", ver)
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
	cmd.Flags().StringVar(&configPath, "config", "", "path to .opsgraph.yaml (legacy .oncallgraph.yaml also accepted)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "persistent store directory")
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
	d := filepath.Dir(p)
	if d == "" {
		return "."
	}
	return d
}

func pathExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func probeOllama(ctx context.Context, url string) bool {
	if url == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimSlash(url)+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
