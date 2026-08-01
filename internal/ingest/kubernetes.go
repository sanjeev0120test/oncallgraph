package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
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
	Namespace  string    `yaml:"namespace"`
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
func ingestK8sSnapshot(s *store.Store, fsys fs.FS, now time.Time) error {
	if err := ingestK8sFiles(s, fsys, "k8s/deployments.yaml", "k8s/events.yaml", now); err != nil {
		return err
	}
	return ingestHelmReleases(s, fsys, "k8s/releases.yaml")
}

// ingestK8sFiles reads the given deployment/event files (if present), updates
// service health, emits rollout changes, and records event evidence.
func ingestK8sFiles(s *store.Store, fsys fs.FS, depFile, evFile string, now time.Time) error {
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
		if err := emitRollout(s, d, now); err != nil {
			return err
		}
	}

	var evs k8sEvents
	if _, err := readYAML(fsys, evFile, &evs); err != nil {
		return err
	}
	skippedNoSvc, skippedNoAt := 0, 0
	for _, e := range evs.Events {
		if e.ServiceID == "" {
			skippedNoSvc++
			continue
		}
		if e.At.IsZero() {
			skippedNoAt++
			continue
		}
		evID := e.EvidenceID
		if evID == "" {
			// Synthesize a stable id so partial snapshots still contribute timeline evidence.
			// Namespace + message digest avoid same-second collisions across namespaces.
			ns := e.Namespace
			if ns == "" {
				ns = "default"
			}
			sum := sha256.Sum256([]byte(ns + "|" + e.Message + "|" + e.Reason))
			reason := slug(e.Reason)
			if reason == "" {
				reason = "event"
			}
			evID = "ev-k8s-" + slug(e.ServiceID) + "-" + slug(ns) + "-" + e.At.UTC().Format("20060102T150405") + "-" + reason + "-" + hex.EncodeToString(sum[:4])
		}
		if err := s.UpsertEvidence(model.Evidence{
			ID: evID, Source: "kubernetes", At: e.At, Kind: "k8s-event",
			Summary: eventSummary(e), RawRef: e.Reason, ServiceID: e.ServiceID,
		}); err != nil {
			return err
		}
	}
	if skippedNoSvc > 0 {
		fmt.Fprintf(os.Stderr, "warning: skipped %d k8s events (missing service_id)\n", skippedNoSvc)
	}
	if skippedNoAt > 0 {
		fmt.Fprintf(os.Stderr, "warning: skipped %d k8s events (missing/zero at)\n", skippedNoAt)
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
	svc.Health = mergeHealth(svc.Health, health, svc.Sources)
	svc.Sources = addSource(svc.Sources, "kubernetes")
	return s.UpsertService(*svc)
}

// mergeHealth merges replica-derived health with prior state.
// When prior health came from kubernetes (sources contains "kubernetes"),
// the latest replica scrape wins so recovery is visible. Otherwise replica
// health cannot upgrade a worse app-level state (pods Ready ≠ app healthy).
func mergeHealth(current, incoming string, sources []string) string {
	if current == "" || current == model.HealthUnknown {
		return incoming
	}
	if hasSource(sources, "kubernetes") {
		return incoming
	}
	if healthRank(incoming) > healthRank(current) {
		return incoming
	}
	return current
}

func hasSource(sources []string, s string) bool {
	for _, x := range sources {
		if x == s {
			return true
		}
	}
	return false
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

func emitRollout(s *store.Store, d k8sDeployment, now time.Time) error {
	if strings.TrimSpace(d.Name) == "" {
		return nil
	}
	at := d.UpdatedAt
	if at.IsZero() {
		// Still emit evidence so health-only snapshots contribute R1/timeline.
		at = now
		if at.IsZero() {
			at = time.Now().UTC()
		}
		fmt.Fprintf(os.Stderr, "warning: deployment %q missing updated_at; using scrape time\n", d.Name)
	}
	// Keep short legacy ids for default-namespace fixtures; namespace elsewhere
	// so same deployment name in two namespaces cannot overwrite.
	suffix := d.Name
	if ns := d.Namespace; ns != "" && ns != "default" {
		suffix = slug(ns) + "-" + d.Name
	}
	evID := "ev-k8s-rollout-" + suffix
	changeID := "k8s-rollout-" + suffix
	summary := fmt.Sprintf("rollout %s (%d/%d ready)", d.Name, d.Ready, d.Desired)
	if err := s.UpsertChange(model.Change{
		ID: changeID, ServiceID: d.ServiceID, At: at, Type: "rollout",
		Summary: summary, Source: "kubernetes", EvidenceID: evID,
	}); err != nil {
		return err
	}
	return s.UpsertEvidence(model.Evidence{
		ID: evID, Source: "kubernetes", At: at, Kind: "rollout",
		Summary: summary, RawRef: d.Name, ServiceID: d.ServiceID,
	})
}

func eventSummary(e k8sEvent) string {
	if e.Reason != "" {
		return e.Reason + ": " + e.Message
	}
	return e.Message
}

func addSource(sources []string, s string) []string {
	if hasSource(sources, s) {
		return sources
	}
	return append(sources, s)
}
