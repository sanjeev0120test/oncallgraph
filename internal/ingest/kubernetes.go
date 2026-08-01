package ingest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

// v1 uses a checked-in YAML snapshot (pure Go, no client-go). A live-cluster
// connector via client-go is a documented future opt-in.

type k8sDeployments struct {
	Deployments []k8sDeployment `yaml:"deployments"`
}

type k8sDeployment struct {
	Name      string    `yaml:"name"`
	Namespace string    `yaml:"namespace"`
	ServiceID string    `yaml:"service_id"`
	Desired   int       `yaml:"desired"`
	Ready     int       `yaml:"ready"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

type k8sEvents struct {
	Events []k8sEvent `yaml:"events"`
}

type k8sEvent struct {
	ServiceID  string    `yaml:"service_id"`
	At         time.Time `yaml:"at"`
	Reason     string    `yaml:"reason"`
	Message    string    `yaml:"message"`
	Type       string    `yaml:"type"`
	EvidenceID string    `yaml:"evidence_id"`
}

// deploymentHealth maps replica readiness to a health value.
func deploymentHealth(desired, ready int) string {
	switch {
	case desired <= 0:
		return model.HealthUnknown
	case ready >= desired:
		return model.HealthHealthy
	case ready == 0:
		return model.HealthUnhealthy
	default:
		return model.HealthDegraded
	}
}

// ingestK8sSnapshot reads a fixture-pack snapshot under k8s/.
func ingestK8sSnapshot(s *store.Store, fsys fs.FS) error {
	if err := ingestK8sFiles(s, fsys, "k8s/deployments.yaml", "k8s/events.yaml"); err != nil {
		return err
	}
	return ingestHelmReleases(s, fsys, "k8s/releases.yaml")
}

// ingestK8sFiles reads the given deployment/event files (if present), updates
// service health, emits rollout changes, and records event evidence.
func ingestK8sFiles(s *store.Store, fsys fs.FS, depFile, evFile string) error {
	var deps k8sDeployments
	if _, err := readYAML(fsys, depFile, &deps); err != nil {
		return err
	}
	for _, d := range deps.Deployments {
		if d.ServiceID == "" {
			continue
		}
		if err := applyDeploymentHealth(s, d); err != nil {
			return err
		}
		if err := emitRollout(s, d); err != nil {
			return err
		}
	}

	var evs k8sEvents
	if _, err := readYAML(fsys, evFile, &evs); err != nil {
		return err
	}
	skipped := 0
	for _, e := range evs.Events {
		if e.ServiceID == "" {
			skipped++
			continue
		}
		evID := e.EvidenceID
		if evID == "" {
			// Synthesize a stable id so partial snapshots still contribute timeline evidence.
			evID = "ev-k8s-" + slug(e.ServiceID) + "-" + e.At.UTC().Format("20060102T150405") + "-" + slug(e.Reason)
		}
		if err := s.UpsertEvidence(model.Evidence{
			ID: evID, Source: "kubernetes", At: e.At, Kind: "k8s-event",
			Summary: eventSummary(e), RawRef: e.Reason, ServiceID: e.ServiceID,
		}); err != nil {
			return err
		}
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "warning: skipped %d k8s events (missing service_id)\n", skipped)
	}
	return nil
}

func applyDeploymentHealth(s *store.Store, d k8sDeployment) error {
	health := deploymentHealth(d.Desired, d.Ready)
	svc, err := s.GetService(d.ServiceID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			svc = &model.Service{ID: d.ServiceID, Name: d.ServiceID}
		} else {
			return err
		}
	}
	// Replica readiness must not paper over a worse app-level health already known.
	svc.Health = mergeHealth(svc.Health, health)
	svc.Sources = addSource(svc.Sources, "kubernetes")
	return s.UpsertService(*svc)
}

// mergeHealth keeps the worse of current and incoming. Replica-healthy never
// upgrades degraded/unhealthy (app can still be broken with pods Ready).
func mergeHealth(current, incoming string) string {
	if current == "" || current == model.HealthUnknown {
		return incoming
	}
	if healthRank(incoming) > healthRank(current) {
		return incoming
	}
	return current
}

func healthRank(h string) int {
	switch h {
	case model.HealthUnhealthy:
		return 3
	case model.HealthDegraded:
		return 2
	case model.HealthHealthy:
		return 1
	default:
		return 0
	}
}

func emitRollout(s *store.Store, d k8sDeployment) error {
	if d.UpdatedAt.IsZero() {
		return nil
	}
	evID := "ev-k8s-rollout-" + d.Name
	summary := fmt.Sprintf("rollout %s (%d/%d ready)", d.Name, d.Ready, d.Desired)
	if err := s.UpsertChange(model.Change{
		ID: "k8s-rollout-" + d.Name, ServiceID: d.ServiceID, At: d.UpdatedAt, Type: "rollout",
		Summary: summary, Source: "kubernetes", EvidenceID: evID,
	}); err != nil {
		return err
	}
	return s.UpsertEvidence(model.Evidence{
		ID: evID, Source: "kubernetes", At: d.UpdatedAt, Kind: "rollout", Summary: summary, ServiceID: d.ServiceID,
	})
}

func eventSummary(e k8sEvent) string {
	if e.Reason != "" {
		return e.Reason + ": " + e.Message
	}
	return e.Message
}

func addSource(sources []string, s string) []string {
	for _, existing := range sources {
		if existing == s {
			return sources
		}
	}
	return append(sources, s)
}
