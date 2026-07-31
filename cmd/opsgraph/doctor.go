package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/opsgraph/opsgraph/fixtures"
	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/version"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var configPath string
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
				dir := resolveDataDir("", cfg)
				db := filepath.Join(dir, "state.db")
				if pathExists(db) {
					check("persistent_db", true, db)
				} else {
					warnf("persistent_db", "no state.db yet (run ingest)")
				}
				gitPath := cfg.Connectors.Git.RepoPath
				if gitPath == "" {
					gitPath = "."
				}
				gitOK := pathExists(filepath.Join(gitPath, ".git")) || pathExists(gitPath)
				check("git_repo", gitOK, gitPath)
				if probeOllama(context.Background(), cfg.AI.OllamaURL) {
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
