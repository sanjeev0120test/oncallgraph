package store

import "fmt"

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
		switch a.Status {
		case "firing", "pending", "suppressed":
			if keepBySource[a.Source] == nil {
				keepBySource[a.Source] = map[string]bool{}
			}
			keepBySource[a.Source][a.ID] = true
		}
	}
	for _, source := range []string{"prometheus", "alertmanager"} {
		if keep, ok := keepBySource[source]; ok || srcHasConnector(src, source) {
			if keep == nil {
				keep = map[string]bool{}
			}
			if _, err := s.ResolveActiveAlertsNotIn(source, keep); err != nil {
				return err
			}
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
