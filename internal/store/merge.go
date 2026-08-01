package store

import (
	"fmt"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// MergeFrom upserts all entities from src into s (no second network scrape).
// Live Prom/AM alert zombies in s are resolved against the live statuses present
// in src for those sources. Meta keys from src are copied last.
func (s *Store) MergeFrom(src *Store) error {
	if src == nil {
		return fmt.Errorf("merge store: nil source")
	}
	owners, err := src.ListOwners()
	if err != nil {
		return err
	}
	for _, o := range owners {
		if err := s.UpsertOwner(o); err != nil {
			return err
		}
	}
	svcs, err := src.ListServices()
	if err != nil {
		return err
	}
	for _, svc := range svcs {
		svc = mergeServiceRow(s, svc)
		if err := s.UpsertService(svc); err != nil {
			return err
		}
	}
	deps, err := src.ListAllDependencies()
	if err != nil {
		return err
	}
	for _, d := range deps {
		if err := s.UpsertDependency(d); err != nil {
			return err
		}
	}
	changes, err := src.ListAllChanges()
	if err != nil {
		return err
	}
	for _, c := range changes {
		if err := s.UpsertChange(c); err != nil {
			return err
		}
	}
	alerts, err := src.ListAllAlerts()
	if err != nil {
		return err
	}
	keepBySource := map[string]map[string]bool{}
	for _, a := range alerts {
		if err := s.UpsertAlert(a); err != nil {
			return err
		}
		if keepBySource[a.Source] == nil {
			keepBySource[a.Source] = map[string]bool{}
		}
		keepBySource[a.Source][a.ID] = true
	}
	for _, source := range []string{"prometheus", "alertmanager"} {
		if !srcHasConnector(src, source) {
			continue
		}
		keep := keepBySource[source]
		// Quiet empty scrape (connector ok, zero alerts) must not wipe dst firings.
		if len(keep) == 0 {
			continue
		}
		if _, err := s.ResolveActiveAlertsNotIn(source, keep); err != nil {
			return err
		}
	}
	for _, svc := range svcs {
		rb, err := src.GetRunbook(svc.ID)
		if err == nil {
			if err := s.UpsertRunbook(*rb); err != nil {
				return err
			}
		} else if err != ErrNotFound {
			return err
		}
	}
	evs, err := src.ListAllEvidence()
	if err != nil {
		return err
	}
	for _, e := range evs {
		if err := s.UpsertEvidence(e); err != nil {
			return err
		}
	}
	for _, key := range []string{"connector:prometheus", "connector:alertmanager"} {
		if v, ok, err := src.GetMeta(key); err != nil {
			return err
		} else if ok {
			if err := s.SetMeta(key, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func srcHasConnector(src *Store, source string) bool {
	v, ok, err := src.GetMeta("connector:" + source)
	return err == nil && ok && v == "ok"
}

// mergeServiceRow keeps richer destination health/sources when the scrape only
// carries config-seed unknowns (temp LiveIngest seed must not wipe k8s health).
func mergeServiceRow(dst *Store, svc model.Service) model.Service {
	existing, err := dst.GetService(svc.ID)
	if err != nil {
		return svc
	}
	if svc.Health == model.HealthUnknown && existing.Health != "" && existing.Health != model.HealthUnknown {
		svc.Health = existing.Health
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(existing.Sources)+len(svc.Sources))
	for _, src := range append(append([]string{}, existing.Sources...), svc.Sources...) {
		if src == "" || seen[src] {
			continue
		}
		seen[src] = true
		out = append(out, src)
	}
	svc.Sources = out
	if svc.Name == "" || svc.Name == svc.ID {
		if existing.Name != "" {
			svc.Name = existing.Name
		}
	}
	if svc.OwnerID == "" {
		svc.OwnerID = existing.OwnerID
	}
	return svc
}
