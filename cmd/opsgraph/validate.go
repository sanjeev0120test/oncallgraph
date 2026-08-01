package main

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ingest"
	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newValidateFixtureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-fixture <dir>",
		Short: "Validate a fixture pack can be ingested and has required files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			required := []string{
				"meta.yaml", "services.yaml", "owners.yaml", "changes.yaml",
				"dependencies.yaml", "alerts.yaml",
			}
			var missing []string
			for _, f := range required {
				if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
					missing = append(missing, f)
				}
			}
			if len(missing) > 0 {
				return fail(1, "missing required files: %v", missing)
			}
			if _, err := os.Stat(filepath.Join(dir, "runbooks")); err != nil {
				cmd.PrintErrln("warning: no runbooks/ directory (opsgraph test may skip verify goldens)")
			}
			if matches, _ := filepath.Glob(filepath.Join(dir, "expected", "*.json")); len(matches) == 0 {
				cmd.PrintErrln("warning: no expected/*.json goldens (opsgraph test has nothing to compare)")
			}

			// Referential check against declared services.yaml IDs (before stub synthesis).
			declared, err := declaredServiceIDs(filepath.Join(dir, "services.yaml"))
			if err != nil {
				return fail(1, "services.yaml: %v", err)
			}
			depPairs, err := declaredDependencies(filepath.Join(dir, "dependencies.yaml"))
			if err != nil {
				return fail(1, "dependencies.yaml: %v", err)
			}
			var synthTargets []string
			for _, d := range depPairs {
				if !declared[d.from] {
					return fail(1, "dependency from undeclared service %q (add it to services.yaml)", d.from)
				}
				if !declared[d.to] {
					synthTargets = append(synthTargets, d.to)
					cmd.PrintErrf("warning: dependency target %q not in services.yaml (will be synthesized as unknown)\n", d.to)
				}
			}

			s, cleanup, err := store.OpenTemp()
			if err != nil {
				return fail(2, "%v", err)
			}
			defer cleanup()
			now, err := ingest.IngestFixtureDir(s, dir)
			if err != nil {
				return fail(1, "ingest failed: %v", err)
			}
			if now.IsZero() {
				return fail(1, "fixture meta.yaml must set a non-zero now: timestamp for determinism")
			}
			counts, err := s.Counts()
			if err != nil {
				return fail(2, "%v", err)
			}
			if counts["services"] == 0 {
				return fail(1, "fixture has zero services after ingest")
			}
			// Synthesized dependency targets must exist after ingest (blast-radius completeness).
			for _, id := range uniqSorted(synthTargets) {
				svc, err := s.GetService(id)
				if err != nil {
					return fail(1, "dependency target %q was not synthesized after ingest: %v", id, err)
				}
				if svc.Health != model.HealthUnknown {
					cmd.PrintErrf("warning: synthesized service %q has health %q (expected unknown)\n", id, svc.Health)
				}
			}
			svcs, err := s.ListServices()
			if err != nil {
				return fail(2, "%v", err)
			}
			known := map[string]bool{}
			for _, svc := range svcs {
				known[svc.ID] = true
			}
			changes, err := s.ListAllChanges()
			if err != nil {
				return fail(2, "%v", err)
			}
			for _, c := range changes {
				if c.ServiceID != "" && !known[c.ServiceID] {
					return fail(1, "change %q references unknown service %q", c.ID, c.ServiceID)
				}
			}
			alerts, err := s.ListAllAlerts()
			if err != nil {
				return fail(2, "%v", err)
			}
			for _, a := range alerts {
				if a.ServiceID != "" && !known[a.ServiceID] {
					return fail(1, "alert %q references unknown service %q", a.ID, a.ServiceID)
				}
			}
			cmd.Printf("ok - fixture %q valid (now=%s)\n", dir, now.UTC().Format(time.RFC3339))
			for _, k := range []string{"services", "owners", "changes", "dependencies", "alerts", "runbooks", "evidence"} {
				cmd.Printf("  %-13s %d\n", k+":", counts[k])
			}
			return nil
		},
	}
}

type depPair struct{ from, to string }

func declaredServiceIDs(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f struct {
		Services []struct {
			ID string `yaml:"id"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, s := range f.Services {
		if s.ID != "" {
			out[s.ID] = true
		}
	}
	return out, nil
}

func declaredDependencies(path string) ([]depPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f struct {
		Dependencies []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		} `yaml:"dependencies"`
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	var out []depPair
	for _, d := range f.Dependencies {
		out = append(out, depPair{from: d.From, to: d.To})
	}
	return out, nil
}

func uniqSorted(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
