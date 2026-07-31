package main

import (
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/opsgraph/opsgraph/fixtures"
	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/store"
)

// loadedStore holds an opened store plus the effective "now" and a cleanup func.
type loadedStore struct {
	store   *store.Store
	now     time.Time
	cleanup func()
}

// storeFromFixtureDir ingests an on-disk fixture pack into an ephemeral store.
func storeFromFixtureDir(dir string) (*loadedStore, error) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		return nil, err
	}
	now, err := ingest.IngestFixtureDir(s, dir)
	if err != nil {
		cleanup()
		return nil, err
	}
	return &loadedStore{store: s, now: now, cleanup: cleanup}, nil
}

// storeFromFixtureFS ingests a fixture pack filesystem (e.g. embedded) into an
// ephemeral store.
func storeFromFixtureFS(fsys fs.FS) (*loadedStore, error) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		return nil, err
	}
	now, err := ingest.IngestFixtureFS(s, fsys)
	if err != nil {
		cleanup()
		return nil, err
	}
	return &loadedStore{store: s, now: now, cleanup: cleanup}, nil
}

// embeddedCheckout returns an ephemeral store loaded from the built-in demo pack.
func embeddedCheckout() (*loadedStore, error) {
	fsys, err := fixtures.CheckoutFS()
	if err != nil {
		return nil, err
	}
	return storeFromFixtureFS(fsys)
}

// resolveConfigPath returns the config path to use: the explicit flag, or
// ./.opsgraph.yaml if it exists, or "" if there is no config.
func resolveConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}
	if _, err := os.Stat(".opsgraph.yaml"); err == nil {
		return ".opsgraph.yaml"
	}
	return ""
}

// storeFromConfig builds an ephemeral store from the live connectors described
// in cfg (seeded services/owners/runbooks + local git + k8s snapshot).
func storeFromConfig(cfg *config.Config, configDir string, since time.Duration, now time.Time) (*loadedStore, error) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		return nil, err
	}
	if err := ingest.LiveIngest(s, cfg, configDir, now.Add(-since), now); err != nil {
		cleanup()
		return nil, err
	}
	return &loadedStore{store: s, now: now, cleanup: cleanup}, nil
}

func validFormat(f string) error {
	if f != "table" && f != "json" {
		return fmt.Errorf("invalid --format %q (want table or json)", f)
	}
	return nil
}
