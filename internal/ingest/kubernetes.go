package ingest

import (
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/opsgraph/opsgraph/internal/store"
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

// ingestK8sSnapshot reads k8s/deployments.yaml and k8s/events.yaml (if present),
// updates service health, emits rollout changes, and records event evidence.
func ingestK8sSnapshot(s *store.Store, fsys fs.FS) error {
	var deps k8sDeployments
	if _, err := readYAML(fsys, "k8s/deployments.yaml", &deps); err != nil {
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
	if _, err := readYAML(fsys, "k8s/events.yaml", &evs); err != nil {
		return err
	}
	for _, e := range evs.Events {
		if e.EvidenceID == "" {
			continue
		}
		if err := s.UpsertEvidence(model.Evidence{
			ID: e.EvidenceID, Source: "kubernetes", At: e.At, Kind: "k8s-event",
			Summary: eventSummary(e), RawRef: e.Reason,
		}); err != nil {
			return err
		}
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
	svc.Health = health
	svc.Sources = addSource(svc.Sources, "kubernetes")
	return s.UpsertService(*svc)
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
		ID: evID, Source: "kubernetes", At: d.UpdatedAt, Kind: "rollout", Summary: summary,
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
