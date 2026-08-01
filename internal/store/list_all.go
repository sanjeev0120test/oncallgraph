package store

import (
	"sort"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// ListAllEvidence returns every evidence row, newest-first.
func (s *Store) ListAllEvidence() ([]model.Evidence, error) {
	rows, err := s.db.Query(`
SELECT id,source,at,kind,summary,raw_ref,service_id FROM evidence ORDER BY at DESC, id`)
	if err != nil {
		return nil, wrap("list all evidence", err)
	}
	defer rows.Close()
	var out []model.Evidence
	for rows.Next() {
		var v model.Evidence
		var at string
		if err := rows.Scan(&v.ID, &v.Source, &at, &v.Kind, &v.Summary, &v.RawRef, &v.ServiceID); err != nil {
			return nil, wrap("scan evidence", err)
		}
		parsed, perr := requireTime(at, "evidence", v.ID)
		if perr != nil {
			return nil, perr
		}
		v.At = parsed
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListAllChanges returns all changes newest-first (deterministic by at,id).
func (s *Store) ListAllChanges() ([]model.Change, error) {
	rows, err := s.db.Query(`
SELECT id,service_id,at,type,summary,author,revision,source,evidence_id
FROM changes ORDER BY at DESC, id`)
	if err != nil {
		return nil, wrap("list all changes", err)
	}
	defer rows.Close()
	var out []model.Change
	for rows.Next() {
		var v model.Change
		var at string
		if err := rows.Scan(&v.ID, &v.ServiceID, &at, &v.Type, &v.Summary, &v.Author, &v.Revision, &v.Source, &v.EvidenceID); err != nil {
			return nil, wrap("scan change", err)
		}
		parsed, perr := requireTime(at, "change", v.ID)
		if perr != nil {
			return nil, perr
		}
		v.At = parsed
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListAllAlerts returns all alerts, firing first then newest.
func (s *Store) ListAllAlerts() ([]model.Alert, error) {
	rows, err := s.db.Query(`
SELECT id,service_id,at,severity,name,status,summary,source,evidence_id FROM alerts`)
	if err != nil {
		return nil, wrap("list all alerts", err)
	}
	defer rows.Close()
	var out []model.Alert
	for rows.Next() {
		var v model.Alert
		var at string
		if err := rows.Scan(&v.ID, &v.ServiceID, &at, &v.Severity, &v.Name, &v.Status, &v.Summary, &v.Source, &v.EvidenceID); err != nil {
			return nil, wrap("scan alert", err)
		}
		parsed, perr := requireTime(at, "alert", v.ID)
		if perr != nil {
			return nil, perr
		}
		v.At = parsed
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		return alertSortLess(out[i], out[j])
	})
	return out, nil
}
