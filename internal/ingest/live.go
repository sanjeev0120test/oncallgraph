package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/opsgraph/opsgraph/internal/runbook"
	"github.com/opsgraph/opsgraph/internal/store"
)

// LiveIngest seeds the store from config and runs the enabled live connectors
// (local git, kubernetes snapshot, optional prometheus/alertmanager). configDir
// is used to resolve relative runbook/snapshot paths. Missing git repo is skipped.
func LiveIngest(s *store.Store, cfg *config.Config, configDir string, since, now time.Time) error {
	if err := seedFromConfig(s, cfg, configDir); err != nil {
		return err
	}
	if cfg.Connectors.Git.Enabled {
		repoPath := cfg.Connectors.Git.RepoPath
		if repoPath == "" {
			repoPath = "."
		}
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(configDir, repoPath)
		}
		if err := IngestGit(s, repoPath, servicePaths(cfg), since, now); err != nil {
			// No repo / empty history is non-fatal — fixtures and other sources still work.
			if !isMissingGit(err) {
				return fmt.Errorf("git connector: %w", err)
			}
		}
	}
	if cfg.Connectors.Kubernetes.Enabled && cfg.Connectors.Kubernetes.Snapshot != "" {
		snap := cfg.Connectors.Kubernetes.Snapshot
		if !filepath.IsAbs(snap) {
			snap = filepath.Join(configDir, snap)
		}
		fsys := os.DirFS(snap)
		if err := ingestK8sFiles(s, fsys, "deployments.yaml", "events.yaml"); err != nil {
			return fmt.Errorf("kubernetes snapshot: %w", err)
		}
		if err := ingestHelmReleases(s, fsys, "releases.yaml"); err != nil {
			return fmt.Errorf("helm snapshot: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if cfg.Connectors.Prometheus.Enabled && cfg.Connectors.Prometheus.URL != "" {
		if err := IngestPrometheus(ctx, s, cfg.Connectors.Prometheus.URL, nil); err != nil {
			return fmt.Errorf("prometheus connector: %w", err)
		}
	}
	if cfg.Connectors.Alertmanager.Enabled && cfg.Connectors.Alertmanager.URL != "" {
		if err := IngestAlertmanager(ctx, s, cfg.Connectors.Alertmanager.URL, nil); err != nil {
			return fmt.Errorf("alertmanager connector: %w", err)
		}
	}
	return nil
}

func isMissingGit(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "repository does not exist") ||
		strings.Contains(msg, "reference not found")
}

func seedFromConfig(s *store.Store, cfg *config.Config, configDir string) error {
	for id, o := range cfg.Owners {
		if err := s.UpsertOwner(model.Owner{ID: id, Name: o.Name, Team: o.Team, Email: o.Email}); err != nil {
			return err
		}
	}
	for id, sc := range cfg.Services {
		if err := s.UpsertService(model.Service{
			ID: id, Name: id, Aliases: sc.Aliases, OwnerID: sc.Owner,
			Health: model.HealthUnknown, Sources: []string{"config"},
		}); err != nil {
			return err
		}
		for _, dep := range sc.DependsOn {
			if err := s.UpsertDependency(model.Dependency{
				FromServiceID: id, ToServiceID: dep, Type: "unknown", Source: "config",
			}); err != nil {
				return err
			}
		}
		if sc.Runbook != "" {
			if err := seedRunbook(s, sc.Runbook, configDir); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedRunbook(s *store.Store, rbPath, configDir string) error {
	p := rbPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(configDir, p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // a missing runbook is not fatal
		}
		return fmt.Errorf("read runbook %q: %w", p, err)
	}
	rb, _, err := runbook.Parse(data, filepath.ToSlash(rbPath))
	if err != nil {
		return err
	}
	if rb.ServiceID == "" {
		return nil
	}
	return s.UpsertRunbook(rb)
}

func servicePaths(cfg *config.Config) []ServicePaths {
	var out []ServicePaths
	for id, sc := range cfg.Services {
		if len(sc.GitPaths) == 0 {
			continue
		}
		out = append(out, ServicePaths{ServiceID: id, Paths: sc.GitPaths})
	}
	return out
}
