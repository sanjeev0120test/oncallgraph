package store

import (
	"sort"

	"github.com/sanjeev0120test/oncallgraph/internal/model"
)

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
		v.At = parseTime(at)
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
		v.At = parseTime(at)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		fi, fj := model.AlertActive(out[i].Status), model.AlertActive(out[j].Status)
		if fi != fj {
			return fi
		}
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.After(out[j].At)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
