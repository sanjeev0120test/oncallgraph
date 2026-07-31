package main

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/opsgraph/opsgraph/fixtures"
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

func validFormat(f string) error {
	if f != "table" && f != "json" {
		return fmt.Errorf("invalid --format %q (want table or json)", f)
	}
	return nil
}
