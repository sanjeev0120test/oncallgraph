package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	_ = os.Chdir(dir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if cfg.DataDir != DefaultDataDir {
		t.Fatalf("default data dir = %q", cfg.DataDir)
	}
	if cfg.Since() != time.Hour {
		t.Fatalf("default since = %v", cfg.Since())
	}
}

func TestLoadExplicitMissingIsError(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error for explicit missing config")
	}
}

func TestLoadMalformedIsError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(p, []byte("version: [this is not: valid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadImplicitOpsgraphYAML(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	_ = os.Chdir(dir)

	content := `version: 1
default_since: 15m
services:
  checkout:
    aliases: ["checkout-api"]
`
	if err := os.WriteFile(".opsgraph.yaml", []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Since() != 15*time.Minute {
		t.Fatalf("since = %v, want 15m", cfg.Since())
	}
	if _, ok := cfg.Services["checkout"]; !ok {
		t.Fatalf("expected checkout service from .opsgraph.yaml: %+v", cfg.Services)
	}
}

func TestLoadPrefersOpsgraphOverOncallgraphLegacy(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	_ = os.Chdir(dir)

	if err := os.WriteFile(".opsgraph.yaml", []byte("default_since: 10m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".oncallgraph.yaml", []byte("default_since: 45m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Since() != 10*time.Minute {
		t.Fatalf("since = %v, want 10m from .opsgraph.yaml", cfg.Since())
	}
}

func TestLoadLegacyOncallgraphYAML(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	_ = os.Chdir(dir)

	if err := os.WriteFile(".oncallgraph.yaml", []byte("default_since: 45m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Since() != 45*time.Minute {
		t.Fatalf("since = %v, want 45m from legacy file", cfg.Since())
	}
}

func TestLoadValid(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ok.yaml")
	content := `version: 1
default_since: 30m
services:
  checkout:
    aliases: ["checkout-api"]
    owner: payments
    depends_on: ["auth"]
owners:
  payments:
    name: Payments Team
`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Since() != 30*time.Minute {
		t.Fatalf("since = %v", cfg.Since())
	}
	svc, ok := cfg.Services["checkout"]
	if !ok || svc.Owner != "payments" || len(svc.DependsOn) != 1 {
		t.Fatalf("service parse: %+v", cfg.Services)
	}
	if cfg.Owners["payments"].Name != "Payments Team" {
		t.Fatalf("owner parse: %+v", cfg.Owners)
	}
	if cfg.AI.Model == "" || cfg.AI.OllamaURL == "" {
		t.Fatalf("ai defaults missing: %+v", cfg.AI)
	}
}
