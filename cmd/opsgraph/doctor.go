package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sanjeev0120test/opsgraph/fixtures"
	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/store"
	"github.com/sanjeev0120test/opsgraph/internal/version"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var configPath string
	var dataDir string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run offline health checks for the local opsgraph environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			okN, warnN, failN := 0, 0, 0
			check := func(name string, good bool, detail string) {
				status := "ok"
				if !good {
					status = "FAIL"
					failN++
				} else {
					okN++
				}
				cmd.Printf("[%s] %-18s %s\n", status, name, detail)
			}
			warnf := func(name, detail string) {
				warnN++
				cmd.Printf("[WARN] %-18s %s\n", name, detail)
			}

			check("binary", true, fmt.Sprintf("opsgraph %s (%s/%s)", version.String(), runtime.GOOS, runtime.GOARCH))
			check("cwd", true, mustGetwd())

			cfg, err := config.Load(configPath)
			check("config_load", err == nil, errOrOK(err))
			if err == nil {
				if configPath == "" && resolveConfigPath("") == "" {
					warnf("config_file", "no .opsgraph.yaml (using built-in defaults)")
				} else if eff := resolveConfigPath(configPath); eff != "" {
					check("config_file", true, eff)
				}
				cfgDir := "."
				if eff := resolveConfigPath(configPath); eff != "" {
					cfgDir = dirOf(eff)
				}
				dir := resolveDataDir(dataDir, cfg, cfgDir)
				db := filepath.Join(dir, "state.db")
				if pathExists(db) {
					s, openErr := store.Open(dir)
					if openErr != nil {
						check("persistent_db", false, openErr.Error())
					} else {
						ver, verErr := s.UserVersion()
						_ = s.Close()
						if verErr != nil {
							check("persistent_db", false, verErr.Error())
						} else if ver != store.SchemaVersion {
							check("persistent_db", false, fmt.Sprintf("%s schema v%d (want v%d)", db, ver, store.SchemaVersion))
						} else {
							check("persistent_db", true, fmt.Sprintf("%s (schema v%d)", db, ver))
						}
					}
				} else {
					warnf("persistent_db", "no state.db yet (run ingest)")
				}
				if !cfg.Connectors.Git.Enabled {
					warnf("git_repo", "connector disabled")
				} else {
					gitPath := resolveGitRepoPath(cfg.Connectors.Git.RepoPath, cfgDir)
					gitDir := filepath.Join(gitPath, ".git")
					gitOK := pathExists(gitDir)
					if gitOK {
						check("git_repo", true, gitPath)
						// ask/who prefer live k8s/prom/AM only — git needs ingest refresh.
						if !liveConnectorsEnabled(cfg) && pathExists(db) {
							warnf("git_refresh", "ask uses persisted store; run `opsgraph ingest` to pick up new commits")
						}
					} else if pathExists(gitPath) {
						warnf("git_repo", gitPath+" exists but has no .git (not a repository)")
					} else {
						// Missing git is non-fatal for live ingest; keep doctor usable offline.
						warnf("git_repo", gitPath+" missing (optional)")
					}
				}
				if !cfg.Connectors.Kubernetes.Enabled {
					warnf("k8s_snapshot", "connector disabled")
				} else if cfg.Connectors.Kubernetes.Snapshot == "" {
					check("k8s_snapshot", false, "enabled but snapshot path is empty")
				} else {
					snap := cfg.Connectors.Kubernetes.Snapshot
					if !filepath.IsAbs(snap) {
						snap = filepath.Join(cfgDir, snap)
					}
					if pathExists(snap) {
						check("k8s_snapshot", true, snap)
					} else {
						check("k8s_snapshot", false, snap+" missing")
					}
				}
				if !cfg.Connectors.Prometheus.Enabled {
					warnf("prometheus", "connector disabled")
				} else if strings.TrimSpace(cfg.Connectors.Prometheus.URL) == "" {
					check("prometheus", false, "enabled but url is empty")
				} else if probeHTTP(cmd.Context(), cfg.Connectors.Prometheus.URL, "/api/v1/alerts") {
					check("prometheus", true, cfg.Connectors.Prometheus.URL)
				} else {
					check("prometheus", false, cfg.Connectors.Prometheus.URL+" unreachable")
				}
				if !cfg.Connectors.Alertmanager.Enabled {
					warnf("alertmanager", "connector disabled")
				} else if strings.TrimSpace(cfg.Connectors.Alertmanager.URL) == "" {
					check("alertmanager", false, "enabled but url is empty")
				} else if probeHTTP(cmd.Context(), cfg.Connectors.Alertmanager.URL, "/api/v2/alerts") {
					check("alertmanager", true, cfg.Connectors.Alertmanager.URL)
				} else {
					check("alertmanager", false, cfg.Connectors.Alertmanager.URL+" unreachable")
				}
				if probeOllama(cmd.Context(), cfg.AI.OllamaURL) {
					check("ollama", true, cfg.AI.OllamaURL)
				} else {
					warnf("ollama", "unreachable at "+cfg.AI.OllamaURL+" (optional)")
				}
			}

			_, embErr := fixtures.CheckoutFS()
			check("embedded_fixture", embErr == nil, "fixtures.CheckoutFS")

			cmd.Printf("\nsummary: %d ok, %d warn, %d fail\n", okN, warnN, failN)
			if failN > 0 {
				return fail(1, "doctor found %d failure(s)", failN)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "persistent store directory to validate")
	return cmd
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "(unknown)"
	}
	return wd
}

func errOrOK(err error) string {
	if err != nil {
		return err.Error()
	}
	return "ok"
}
