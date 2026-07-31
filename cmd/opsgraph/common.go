package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/sanjeev0120test/opsgraph/fixtures"
	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/ingest"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

// ErrEmptyStore means a persistent --data-dir has no services yet.
var ErrEmptyStore = errors.New("empty store")

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
// ./.opsgraph.yaml / legacy ./.oncallgraph.yaml if present, or "".
func resolveConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}
	for _, candidate := range []string{".opsgraph.yaml", ".oncallgraph.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// resolveDataDir picks --data-dir, else config.data_dir, else the default.
// When using the built-in default and only a brief-rename legacy store exists
// (.oncallgraph/data), that path is used so users are not silently reset.
func resolveDataDir(flag string, cfg *config.Config) string {
	if flag != "" {
		return flag
	}
	if cfg != nil && cfg.DataDir != "" && cfg.DataDir != config.DefaultDataDir {
		return cfg.DataDir
	}
	preferred := config.DefaultDataDir
	legacy := ".oncallgraph/data"
	if _, err := os.Stat(filepath.Join(preferred, "state.db")); err == nil {
		return preferred
	}
	if _, err := os.Stat(filepath.Join(legacy, "state.db")); err == nil {
		return legacy
	}
	return preferred
}

// storeFromDataDir opens the persistent store under dataDir (no re-ingest).
func storeFromDataDir(dataDir string, now time.Time) (*loadedStore, error) {
	s, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}
	return &loadedStore{store: s, now: now, cleanup: func() { _ = s.Close() }}, nil
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

// loadAskStore resolves fixture / data-dir / live-config into a store.
func loadAskStore(fixture, configPath, dataDirFlag string, cfg *config.Config, since time.Duration) (*loadedStore, error) {
	if fixture != "" {
		return storeFromFixtureDir(fixture)
	}
	if dataDirFlag != "" {
		ls, err := storeFromDataDir(dataDirFlag, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		counts, err := ls.store.Counts()
		if err != nil {
			ls.cleanup()
			return nil, err
		}
		if counts["services"] == 0 {
			ls.cleanup()
			return nil, fmt.Errorf("%w at %s: run `opsgraph ingest` first or pass `--fixture`", ErrEmptyStore, dataDirFlag)
		}
		return ls, nil
	}
	// Prefer an already-ingested persistent store when present.
	dir := resolveDataDir("", cfg)
	if counts, err := peekCounts(dir); err == nil {
		if counts["services"] > 0 {
			return storeFromDataDir(dir, time.Now().UTC())
		}
		// state.db exists but is empty — do not silently fall back to live config.
		return nil, fmt.Errorf("%w at %s: run `opsgraph ingest` first or pass `--fixture`", ErrEmptyStore, dir)
	}
	effPath := resolveConfigPath(configPath)
	if effPath == "" {
		return nil, fmt.Errorf("no data source: pass --fixture <pack>, run `opsgraph ingest`, or add a .opsgraph.yaml")
	}
	return storeFromConfig(cfg, dirOf(effPath), since, time.Now().UTC())
}

func peekCounts(dataDir string) (map[string]int, error) {
	dbPath := filepath.Join(dataDir, "state.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err
	}
	s, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	return s.Counts()
}

func validFormat(f string) error {
	if f != "table" && f != "json" {
		return fmt.Errorf("invalid --format %q (want table or json)", f)
	}
	return nil
}
