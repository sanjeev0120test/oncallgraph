package ingest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/runbook"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

// openK8sSnapshotFS accepts either a directory (deployments.yaml inside) or a
// path to the deployments YAML file (siblings: events.yaml, releases.yaml).
// File names returned for fs.FS always use forward slashes (required by io/fs).
func openK8sSnapshotFS(snap string) (fsys fs.FS, depFile, evFile, relFile string, err error) {
	info, err := os.Stat(snap)
	if err != nil {
		return nil, "", "", "", err
	}
	if info.IsDir() {
		return os.DirFS(snap), "deployments.yaml", "events.yaml", "releases.yaml", nil
	}
	dir := filepath.Dir(snap)
	return os.DirFS(dir), filepath.ToSlash(filepath.Base(snap)), "events.yaml", "releases.yaml", nil
}

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
	if cfg.Connectors.Kubernetes.Enabled {
		if cfg.Connectors.Kubernetes.Snapshot == "" {
			fmt.Fprintln(os.Stderr, "warning: kubernetes connector enabled but snapshot path is empty; skipping")
		} else {
			snap := cfg.Connectors.Kubernetes.Snapshot
			if !filepath.IsAbs(snap) {
				snap = filepath.Join(configDir, snap)
			}
			fsys, depFile, evFile, relFile, err := openK8sSnapshotFS(snap)
			if err != nil {
				return fmt.Errorf("kubernetes snapshot: %w", err)
			}
			if err := ingestK8sFiles(s, fsys, depFile, evFile); err != nil {
				return fmt.Errorf("kubernetes snapshot: %w", err)
			}
			if err := ingestHelmReleases(s, fsys, relFile); err != nil {
				return fmt.Errorf("helm snapshot: %w", err)
			}
		}
	}
	if cfg.Connectors.Prometheus.Enabled {
		if cfg.Connectors.Prometheus.URL == "" {
			fmt.Fprintln(os.Stderr, "warning: prometheus connector enabled but url is empty; skipping")
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := IngestPrometheus(ctx, s, cfg.Connectors.Prometheus.URL, nil, now)
			cancel()
			if err != nil {
				return fmt.Errorf("prometheus connector: %w", err)
			}
		}
	}
	if cfg.Connectors.Alertmanager.Enabled {
		if cfg.Connectors.Alertmanager.URL == "" {
			fmt.Fprintln(os.Stderr, "warning: alertmanager connector enabled but url is empty; skipping")
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := IngestAlertmanager(ctx, s, cfg.Connectors.Alertmanager.URL, nil, now)
			cancel()
			if err != nil {
				return fmt.Errorf("alertmanager connector: %w", err)
			}
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
	ownerIDs := make([]string, 0, len(cfg.Owners))
	for id := range cfg.Owners {
		ownerIDs = append(ownerIDs, id)
	}
	sort.Strings(ownerIDs)
	for _, id := range ownerIDs {
		o := cfg.Owners[id]
		if err := s.UpsertOwner(model.Owner{ID: id, Name: o.Name, Team: o.Team, Email: o.Email}); err != nil {
			return err
		}
	}
	svcIDs := make([]string, 0, len(cfg.Services))
	for id := range cfg.Services {
		svcIDs = append(svcIDs, id)
	}
	sort.Strings(svcIDs)
	for _, id := range svcIDs {
		sc := cfg.Services[id]
		health := model.HealthUnknown
		name := id
		sources := []string{"config"}
		if existing, err := s.GetService(id); err == nil {
			// Preserve connector-derived health/name; config seed must not wipe them.
			health = existing.Health
			if existing.Name != "" {
				name = existing.Name
			}
			sources = addSource(existing.Sources, "config")
		} else if err != nil && !errors.Is(err, store.ErrNotFound) {
			return err
		}
		if err := s.UpsertService(model.Service{
			ID: id, Name: name, Aliases: sc.Aliases, OwnerID: sc.Owner,
			Health: health, Sources: sources,
		}); err != nil {
			return err
		}
		for _, dep := range sc.DependsOn {
			if err := ensureServiceStub(s, dep); err != nil {
				return err
			}
			if err := s.UpsertDependency(model.Dependency{
				FromServiceID: id, ToServiceID: dep, Type: "depends_on", Source: "config",
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
			fmt.Fprintf(os.Stderr, "warning: runbook %q not found; skipping\n", p)
			return nil
		}
		return fmt.Errorf("read runbook %q: %w", p, err)
	}
	rb, _, err := runbook.Parse(data, filepath.ToSlash(rbPath))
	if err != nil {
		return err
	}
	if rb.ServiceID == "" {
		fmt.Fprintf(os.Stderr, "warning: runbook %q has empty service in front matter; skipping\n", p)
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
