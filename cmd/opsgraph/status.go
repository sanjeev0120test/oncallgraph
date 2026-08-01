package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
			if fixture != "" && dataDir != "" {
				return fail(2, "--fixture and --data-dir are mutually exclusive")
			}
			cfg, err := config.Load(configPath)
			if err != nil {
				return fail(2, "%v", err)
			}
			effPath := resolveConfigPath(configPath)
			configDir := "."
			if effPath != "" {
				configDir = dirOf(effPath)
			}
			dir := resolveDataDir(dataDir, cfg, configDir)
			dbPath := filepath.Join(dir, "state.db")

			cmd.Println("STORE")
			cmd.Printf("  data_dir: %s\n", dir)
			cmd.Printf("  db:       %s\n", dbPath)

			cmd.Println("CONNECTORS")
			cmd.Printf("  fixtures:     %v\n", cfg.Connectors.Fixtures.Enabled)
			gitPath := resolveGitRepoPath(cfg.Connectors.Git.RepoPath, configDir)
			gitOK := pathExists(filepath.Join(gitPath, ".git"))
			cmd.Printf("  git:          %v (repo %q has_git=%v)\n", cfg.Connectors.Git.Enabled, gitPath, gitOK)
			if cfg.Connectors.Git.Enabled && !liveConnectorsEnabled(cfg) {
				cmd.Println("  git_note:     ask/who/watch use persisted store; run `opsgraph ingest` to refresh git changes")
			}
			cmd.Printf("  kubernetes:   %v (snapshot %q)\n", cfg.Connectors.Kubernetes.Enabled, cfg.Connectors.Kubernetes.Snapshot)
			cmd.Printf("  prometheus:   %v (url %q", cfg.Connectors.Prometheus.Enabled, cfg.Connectors.Prometheus.URL)
			if cfg.Connectors.Prometheus.Enabled {
				cmd.Printf(" reachable=%v", probeHTTP(cmd.Context(), cfg.Connectors.Prometheus.URL, "/api/v1/alerts"))
			}
			cmd.Printf(")\n")
			cmd.Printf("  alertmanager: %v (url %q", cfg.Connectors.Alertmanager.Enabled, cfg.Connectors.Alertmanager.URL)
			if cfg.Connectors.Alertmanager.Enabled {
				cmd.Printf(" reachable=%v", probeHTTP(cmd.Context(), cfg.Connectors.Alertmanager.URL, "/api/v2/alerts"))
			}
			cmd.Printf(")\n")

			ollamaOK := probeOllama(cmd.Context(), cfg.AI.OllamaURL)
			cmd.Printf("AI\n  enabled: %v  model: %s  embed: %s  url: %s  reachable: %v\n",
				cfg.AI.Enabled, cfg.AI.Model, cfg.AI.EmbedModel, cfg.AI.OllamaURL, ollamaOK)

			// Same source selection as ask (live k8s/prom/AM preferred over stale db).
			ls, err := loadAskStore(cmd.Context(), fixture, configPath, dataDir, cfg, cfg.Since())
			if err != nil {
				if fixture == "" && dataDir == "" && (errors.Is(err, ErrEmptyStore) || isNoDataSource(err)) {
					cmd.Println("\nNo data source (pass --fixture <pack>, run `opsgraph ingest`, or add .opsgraph.yaml) - showing config only.")
					return fail(1, "no ingested data")
				}
				// Empty explicit --data-dir is exit 1 (same contract as ask).
				return failSource(err)
			}
			defer ls.cleanup()

			counts, err := ls.store.Counts()
			if err != nil {
				return fail(2, "%v", err)
			}
			ver, err := ls.store.UserVersion()
			if err != nil {
				return fail(2, "schema version: %v", err)
			}
			active := ls.source
			if active == "" {
				active = "unknown"
			}
			cmd.Printf("\nACTIVE SOURCE  %s\n", active)
			switch active {
			case "live":
				cmd.Println("  note: ephemeral live scrape (may differ from on-disk db above)")
			case "fixture":
				cmd.Println("  note: fixture pack (ephemeral; not written to data_dir)")
			case "persisted":
				cmd.Println("  note: reading on-disk state.db")
			}
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
	cmd.Flags().StringVar(&configPath, "config", "", "path to .opsgraph.yaml")
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
	return probeHTTP(ctx, url, "/api/tags")
}

// probeHTTP GETs baseURL+path with a short timeout. Empty URL is unreachable.
func probeHTTP(ctx context.Context, baseURL, path string) bool {
	if strings.TrimSpace(baseURL) == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimSlash(baseURL)+path, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "opsgraph/1.0")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	// Ollama historically omits Content-Type; Prom/AM should speak JSON.
	if ct == "" {
		return strings.Contains(path, "/api/tags")
	}
	return strings.Contains(ct, "json") || strings.Contains(ct, "javascript")
}

// resolveGitRepoPath mirrors LiveIngest: relative repo_path is config-dir based.
func resolveGitRepoPath(repoPath, configDir string) string {
	if repoPath == "" {
		repoPath = "."
	}
	if filepath.IsAbs(repoPath) {
		return repoPath
	}
	if configDir == "" {
		configDir = "."
	}
	return filepath.Join(configDir, repoPath)
}

func isNoDataSource(err error) bool {
	return errors.Is(err, ErrNoDataSource) ||
		(err != nil && strings.Contains(err.Error(), "no data source"))
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
