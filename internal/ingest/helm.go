package ingest

import (
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

type helmReleases struct {
	Releases []helmRelease `yaml:"releases"`
}

type helmRelease struct {
	Name      string    `yaml:"name"`
	ServiceID string    `yaml:"service_id"`
	Chart     string    `yaml:"chart"`
	Version   string    `yaml:"version"`
	Revision  int       `yaml:"revision"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

// ingestHelmReleases reads an optional helm releases snapshot and emits deploy
// changes. Missing file is a no-op. now fills missing updated_at.
func ingestHelmReleases(s *store.Store, fsys fs.FS, path string, now time.Time) error {
	var f helmReleases
	found, err := readYAML(fsys, path, &f)
	if err != nil || !found {
		return err
	}
	for _, r := range f.Releases {
		if r.ServiceID == "" {
			continue
		}
		if err := emitHelmDeploy(s, r, now); err != nil {
			return err
		}
	}
	return nil
}

func emitHelmDeploy(s *store.Store, r helmRelease, now time.Time) error {
	at := r.UpdatedAt
	if at.IsZero() {
		at = now
		if at.IsZero() {
			at = time.Now().UTC()
		}
		name := r.Name
		if name == "" {
			name = r.ServiceID
		}
		fmt.Fprintf(os.Stderr, "warning: helm release %q missing updated_at; using scrape time\n", name)
	}
	rev := fmt.Sprintf("%d", r.Revision)
	if r.Version != "" {
		rev = r.Version
	}
	name := r.Name
	if name == "" {
		name = r.ServiceID
	}
	// Include service_id so two services cannot collide on the same release name.
	evID := "ev-helm-" + r.ServiceID + "-" + name
	summary := fmt.Sprintf("helm release %s chart=%s rev=%s", name, r.Chart, rev)
	if err := s.UpsertChange(model.Change{
		ID: "helm-" + r.ServiceID + "-" + name, ServiceID: r.ServiceID, At: at, Type: "deploy",
		Summary: summary, Revision: rev, Source: "helm", EvidenceID: evID,
	}); err != nil {
		return err
	}
	return s.UpsertEvidence(model.Evidence{
		ID: evID, ServiceID: r.ServiceID, Source: "helm", At: at, Kind: "deploy", Summary: summary, RawRef: rev,
	})
}
