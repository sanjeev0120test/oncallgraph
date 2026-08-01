package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/fixtures"
	"github.com/sanjeev0120test/opsgraph/internal/ask"
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
// ./.opsgraph.yaml if present, or "".
func resolveConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}
	if _, err := os.Stat(".opsgraph.yaml"); err == nil {
		return ".opsgraph.yaml"
	}
	return ""
}

// resolveDataDir picks --data-dir (CWD-relative), else config.data_dir resolved
// against the config file's directory, else the default under configDir/CWD.
func resolveDataDir(flag string, cfg *config.Config, configDir string) string {
	if flag != "" {
		return flag
	}
	dir := config.DefaultDataDir
	if cfg != nil && strings.TrimSpace(cfg.DataDir) != "" {
		dir = cfg.DataDir
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	if configDir == "" {
		configDir = "."
	}
	return filepath.Clean(filepath.Join(configDir, dir))
}

// storeFromDataDir opens the persistent store under dataDir (no re-ingest).
// When ingest persisted as_of (fixture clock), that wins over wall clock so
// ask/verify match --fixture determinism.
func storeFromDataDir(dataDir string, now time.Time) (*loadedStore, error) {
	s, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}
	if asOf, ok, err := s.AsOf(); err != nil {
		_ = s.Close()
		return nil, err
	} else if ok {
		now = asOf
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
	lookback := ask.ChangeLookback(since)
	if err := ingest.LiveIngest(s, cfg, configDir, now.Add(-lookback), now); err != nil {
		cleanup()
		return nil, err
	}
	return &loadedStore{store: s, now: now, cleanup: cleanup}, nil
}

// liveConnectorsEnabled reports whether config can scrape *usable* fresh
// incident signals. Git-alone is excluded. Enabled-but-empty k8s/prom/AM
// must not displace a populated state.db with an empty ephemeral scrape.
func liveConnectorsEnabled(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	c := cfg.Connectors
	if c.Kubernetes.Enabled && strings.TrimSpace(c.Kubernetes.Snapshot) != "" {
		return true
	}
	if c.Prometheus.Enabled && strings.TrimSpace(c.Prometheus.URL) != "" {
		return true
	}
	if c.Alertmanager.Enabled && strings.TrimSpace(c.Alertmanager.URL) != "" {
		return true
	}
	return false
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
	effPath := resolveConfigPath(configPath)
	configDir := "."
	if effPath != "" {
		configDir = dirOf(effPath)
	}
	// Live connectors beat a stale state.db so ask/why/watch see fresh signals
	// when a config is present. Explicit --data-dir still forces the store.
	if effPath != "" && liveConnectorsEnabled(cfg) {
		return storeFromConfig(cfg, configDir, since, time.Now().UTC())
	}
	dir := resolveDataDir("", cfg, configDir)
	if counts, err := peekCounts(dir); err == nil {
		if counts["services"] > 0 {
			return storeFromDataDir(dir, time.Now().UTC())
		}
		return nil, fmt.Errorf("%w at %s: run `opsgraph ingest` first or pass `--fixture`", ErrEmptyStore, dir)
	}
	if effPath == "" {
		return nil, fmt.Errorf("no data source: pass --fixture <pack>, run `opsgraph ingest`, or add a .opsgraph.yaml")
	}
	return storeFromConfig(cfg, configDir, since, time.Now().UTC())
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

// validSince rejects negative lookback windows (a negative duration would
// invert the cutoff and silently drop all history).
func validSince(d time.Duration) error {
	if d < 0 {
		return fmt.Errorf("invalid --since %s (must be >= 0)", d)
	}
	return nil
}
