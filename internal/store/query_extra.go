package store

import (
	"database/sql"
	"errors"
	"sort"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// ListOwners returns all owners sorted by id.
func (s *Store) ListOwners() ([]model.Owner, error) {
	rows, err := s.db.Query(`SELECT id,name,team,email FROM owners ORDER BY id`)
	if err != nil {
		return nil, wrap("list owners", err)
	}
	defer rows.Close()
	var out []model.Owner
	for rows.Next() {
		var v model.Owner
		if err := rows.Scan(&v.ID, &v.Name, &v.Team, &v.Email); err != nil {
			return nil, wrap("scan owner", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetEvidence returns one evidence row by id.
func (s *Store) GetEvidence(id string) (*model.Evidence, error) {
	var v model.Evidence
	var at string
	err := s.db.QueryRow(
		`SELECT id,source,at,kind,summary,raw_ref,service_id FROM evidence WHERE id=?`, id,
	).Scan(&v.ID, &v.Source, &at, &v.Kind, &v.Summary, &v.RawRef, &v.ServiceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, wrap("get evidence", err)
	}
	parsed, perr := requireTime(at, "evidence", v.ID)
	if perr != nil {
		return nil, perr
	}
	v.At = parsed
	return &v, nil
}

// ListAllDependencies returns every dependency edge, sorted.
func (s *Store) ListAllDependencies() ([]model.Dependency, error) {
	rows, err := s.db.Query(`
SELECT from_service_id,to_service_id,type,source FROM dependencies
ORDER BY from_service_id, to_service_id, type`)
	if err != nil {
		return nil, wrap("list all dependencies", err)
	}
	defer rows.Close()
	var out []model.Dependency
	for rows.Next() {
		var v model.Dependency
		if err := rows.Scan(&v.FromServiceID, &v.ToServiceID, &v.Type, &v.Source); err != nil {
			return nil, wrap("scan dependency", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListServicesByOwner returns services owned by ownerID.
func (s *Store) ListServicesByOwner(ownerID string) ([]model.Service, error) {
	all, err := s.ListServices()
	if err != nil {
		return nil, err
	}
	var out []model.Service
	for _, svc := range all {
		if svc.OwnerID == ownerID {
			out = append(out, svc)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
