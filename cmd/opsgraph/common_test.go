package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sanjeev0120test/opsgraph/internal/config"
)

func TestLiveConnectorsEnabledRequiresUsableConfig(t *testing.T) {
	if liveConnectorsEnabled(nil) {
		t.Fatal("nil config")
	}
	cfg := config.Default()
	if liveConnectorsEnabled(cfg) {
		t.Fatal("defaults should not prefer live (k8s off, no URLs)")
	}
	cfg.Connectors.Kubernetes.Enabled = true
	if liveConnectorsEnabled(cfg) {
		t.Fatal("k8s enabled without snapshot must not prefer live")
	}
	cfg.Connectors.Kubernetes.Snapshot = "k8s/deployments.yaml"
	if !liveConnectorsEnabled(cfg) {
		t.Fatal("k8s with snapshot should prefer live")
	}
	cfg2 := config.Default()
	cfg2.Connectors.Prometheus.Enabled = true
	if liveConnectorsEnabled(cfg2) {
		t.Fatal("prom enabled without url must not prefer live")
	}
	cfg2.Connectors.Prometheus.URL = "http://127.0.0.1:9090"
	if !liveConnectorsEnabled(cfg2) {
		t.Fatal("prom with url should prefer live")
	}
}

func TestResolveDataDirConfigRelative(t *testing.T) {
	cfg := config.Default()
	cfg.DataDir = "mystore"
	got := resolveDataDir("", cfg, filepath.Join("cfg", "dir"))
	want := filepath.Clean(filepath.Join("cfg", "dir", "mystore"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	abs := "/var/opsgraph"
	if runtime.GOOS == "windows" {
		abs = `C:\opsgraph-data`
	}
	cfg.DataDir = abs
	if got := resolveDataDir("", cfg, "ignored"); got != abs {
		t.Fatalf("abs data_dir: got %q", got)
	}
	if got := resolveDataDir("flag-data", cfg, "cfg"); got != "flag-data" {
		t.Fatalf("flag wins: got %q", got)
	}
}
