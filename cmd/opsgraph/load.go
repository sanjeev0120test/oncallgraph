package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

// sourceFlags are the common --fixture/--config/--data-dir trio.
type sourceFlags struct {
	fixture    string
	configPath string
	dataDir    string
}

func (f *sourceFlags) loadCtx(ctx context.Context, since time.Duration) (*loadedStore, *config.Config, error) {
	if err := validSince(since); err != nil {
		return nil, nil, err
	}
	fixture, err := resolveFixtureExclusive(f.fixture, f.dataDir)
	if err != nil {
		return nil, nil, err
	}
	cfgPath := configPathOrEnv(f.configPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, nil, err
	}
	if since == 0 {
		since = cfg.Since()
	}
	ls, err := loadAskStore(ctx, fixture, cfgPath, f.dataDir, cfg, since)
	return ls, cfg, err
}

// resolveFixtureExclusive resolves --fixture / OPSGRAPH_FIXTURE and rejects
// combinations with --data-dir / OPSGRAPH_DATA_DIR (flag or env).
func resolveFixtureExclusive(fixtureFlag, dataDirFlag string) (string, error) {
	fixture := strings.TrimSpace(fixtureFlag)
	if fixture == "" {
		fixture = strings.TrimSpace(os.Getenv("OPSGRAPH_FIXTURE"))
	}
	dataDir := strings.TrimSpace(dataDirFlag)
	if dataDir == "" {
		dataDir = strings.TrimSpace(os.Getenv("OPSGRAPH_DATA_DIR"))
	}
	if fixture != "" && dataDir != "" {
		return "", fail(2, "--fixture and --data-dir are mutually exclusive")
	}
	return fixture, nil
}

// failSource maps store/config load errors to CLI exit codes.
func failSource(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrEmptyStore) {
		return fail(1, "%v", err)
	}
	return fail(2, "%v", err)
}

func askService(ls *loadedStore, query string, since time.Duration) (model.AskResult, error) {
	if since == 0 {
		since = config.DefaultSince
	}
	return ask.Ask(ls.store, query, ask.Options{Since: since, Now: ls.now, WithRunbook: true})
}

// failAsk maps ask errors to CLI exit codes (1 = not found / ambiguous, 2 = other).
func failAsk(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ask.ErrServiceNotFound) || errors.Is(err, store.ErrAmbiguous) {
		return fail(1, "%v", err)
	}
	return fail(2, "%v", err)
}

// failLookup maps store service lookup errors (not found / ambiguous -> 1).
func failLookup(query string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return fail(1, "service %q not found", query)
	}
	if errors.Is(err, store.ErrAmbiguous) {
		return fail(1, "%v", err)
	}
	return fail(2, "%v", err)
}
