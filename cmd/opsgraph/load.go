package main

import (
	"time"

	"github.com/opsgraph/opsgraph/internal/ask"
	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/model"
)

// sourceFlags are the common --fixture/--config/--data-dir trio.
type sourceFlags struct {
	fixture    string
	configPath string
	dataDir    string
}

func (f *sourceFlags) load(since time.Duration) (*loadedStore, *config.Config, error) {
	cfg, err := config.Load(f.configPath)
	if err != nil {
		return nil, nil, err
	}
	if since == 0 {
		since = cfg.Since()
	}
	ls, err := loadAskStore(f.fixture, f.configPath, f.dataDir, cfg, since)
	return ls, cfg, err
}

func askService(ls *loadedStore, query string, since time.Duration) (model.AskResult, error) {
	if since == 0 {
		since = time.Hour
	}
	return ask.Ask(ls.store, query, ask.Options{Since: since, Now: ls.now, WithRunbook: true})
}
