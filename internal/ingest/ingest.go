// Package ingest loads incident data from fixtures (and, later, live sources)
// into the store. The fixture path is fully deterministic and offline.
package ingest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/store"
	"gopkg.in/yaml.v3"
)

type metaFile struct {
	Now time.Time `yaml:"now"`
}

// FixtureNow reads meta.yaml and returns the pinned clock. If absent, it
// returns the zero time and ok=false.
func FixtureNow(fsys fs.FS) (time.Time, bool, error) {
	var m metaFile
	found, err := readYAML(fsys, "meta.yaml", &m)
	if err != nil {
		return time.Time{}, false, err
	}
	if !found || m.Now.IsZero() {
		return time.Time{}, false, nil
	}
	return m.Now.UTC(), true, nil
}

// IngestFixtureFS ingests a fixture pack from a filesystem and returns the
// fixture's pinned "now" (or the real time if meta.yaml is absent).
func IngestFixtureFS(s *store.Store, fsys fs.FS) (time.Time, error) {
	if err := ingestEntities(s, fsys); err != nil {
		return time.Time{}, err
	}
	if err := ingestK8sSnapshot(s, fsys); err != nil {
		return time.Time{}, err
	}
	now, ok, err := FixtureNow(fsys)
	if err != nil {
		return time.Time{}, err
	}
	if !ok {
		return time.Now().UTC(), nil
	}
	return now, nil
}

// IngestFixtureDir ingests a fixture pack from an on-disk directory.
func IngestFixtureDir(s *store.Store, dir string) (time.Time, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return time.Time{}, fmt.Errorf("fixture dir %q: %w", dir, err)
	}
	if !info.IsDir() {
		return time.Time{}, fmt.Errorf("fixture path %q is not a directory", dir)
	}
	return IngestFixtureFS(s, os.DirFS(dir))
}

// readYAML decodes fsys/name into out. Returns found=false if the file is absent.
func readYAML(fsys fs.FS, name string, out any) (bool, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read %s: %w", name, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return true, nil
}
