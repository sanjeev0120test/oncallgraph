//go:build ignore

// check_fixtures validates golden fixture packs are complete and JSON-valid.
// Usage: go run ./scripts/check_fixtures.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type pack struct {
	dir      string
	required []string
}

func main() {
	packs := []pack{
		{
			dir: "fixtures/incident_checkout",
			required: []string{
				"meta.yaml",
				"services.yaml",
				"dependencies.yaml",
				"owners.yaml",
				"alerts.yaml",
				"changes.yaml",
				"expected/ask_checkout.json",
				"expected/ask_auth.json",
				"expected/verify_checkout.json",
				"expected/verify_auth.json",
			},
		},
		{
			dir: "fixtures/fleet_healthy",
			required: []string{
				"meta.yaml",
				"services.yaml",
				"dependencies.yaml",
				"owners.yaml",
				"alerts.yaml",
				"changes.yaml",
				"expected/ask_api.json",
				"expected/verify_api.json",
			},
		},
	}

	var errs []string
	for _, p := range packs {
		for _, rel := range p.required {
			path := filepath.Join(p.dir, filepath.FromSlash(rel))
			info, err := os.Stat(path)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: missing %s", p.dir, rel))
				continue
			}
			if info.IsDir() || info.Size() == 0 {
				errs = append(errs, fmt.Sprintf("%s: empty or not a file", path))
			}
		}
		expectedDir := filepath.Join(p.dir, "expected")
		entries, err := os.ReadDir(expectedDir)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", expectedDir, err))
			continue
		}
		jsonN := 0
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			jsonN++
			path := filepath.Join(expectedDir, e.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			if len(raw) == 0 {
				errs = append(errs, path+": empty golden")
				continue
			}
			if !json.Valid(raw) {
				errs = append(errs, path+": invalid JSON")
			}
		}
		if jsonN == 0 {
			errs = append(errs, expectedDir+": no golden JSON files")
		}
	}

	live := "fixtures/ci_live_k8s/.opsgraph.yaml"
	if _, err := os.Stat(live); err != nil {
		errs = append(errs, live+": required for live-connector smoke")
	}

	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "check_fixtures: fixture pack integrity failed:")
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, " ", e)
		}
		os.Exit(1)
	}
}
