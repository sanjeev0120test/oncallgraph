package store

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// ErrNotFound is returned when a lookup finds nothing.
var ErrNotFound = errors.New("not found")

// ErrAmbiguous is returned when a name/alias matches more than one service.
var ErrAmbiguous = errors.New("ambiguous service match")

// --- upserts (idempotent by primary key) ---

// UpsertService inserts or updates a service.
func (s *Store) UpsertService(v model.Service) error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("upsert service: empty id")
	}
	_, err := s.db.Exec(`
INSERT INTO services (id,name,aliases,owner_id,health,labels,sources)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
	name=excluded.name, aliases=excluded.aliases, owner_id=excluded.owner_id,
	health=excluded.health, labels=excluded.labels, sources=excluded.sources`,
		v.ID, v.Name, toJSON(v.Aliases), v.OwnerID, v.Health, toJSON(v.Labels), toJSON(v.Sources))
	return wrap("upsert service", err)
}

// UpsertOwner inserts or updates an owner.
func (s *Store) UpsertOwner(v model.Owner) error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("upsert owner: empty id")
	}
	_, err := s.db.Exec(`
INSERT INTO owners (id,name,team,email) VALUES (?,?,?,?)
ON CONFLICT(id) DO UPDATE SET name=excluded.name, team=excluded.team, email=excluded.email`,
		v.ID, v.Name, v.Team, v.Email)
	return wrap("upsert owner", err)
}

// UpsertChange inserts or updates a change.
func (s *Store) UpsertChange(v model.Change) error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("upsert change: empty id")
	}
	_, err := s.db.Exec(`
INSERT INTO changes (id,service_id,at,type,summary,author,revision,source,evidence_id)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
	service_id=excluded.service_id, at=excluded.at, type=excluded.type,
	summary=excluded.summary, author=excluded.author, revision=excluded.revision,
	source=excluded.source, evidence_id=excluded.evidence_id`,
		v.ID, v.ServiceID, fmtTime(v.At), v.Type, v.Summary, v.Author, v.Revision, v.Source, v.EvidenceID)
	return wrap("upsert change", err)
}

// UpsertDependency inserts or updates a dependency edge.
func (s *Store) UpsertDependency(v model.Dependency) error {
	if strings.TrimSpace(v.FromServiceID) == "" || strings.TrimSpace(v.ToServiceID) == "" {
		return fmt.Errorf("upsert dependency: empty from/to service id")
	}
	if v.Type == "" {
		v.Type = "unknown"
	}
	_, err := s.db.Exec(`
INSERT INTO dependencies (from_service_id,to_service_id,type,source)
VALUES (?,?,?,?)
ON CONFLICT(from_service_id,to_service_id,type) DO UPDATE SET source=excluded.source`,
		v.FromServiceID, v.ToServiceID, v.Type, v.Source)
	return wrap("upsert dependency", err)
}

// UpsertAlert inserts or updates an alert.
func (s *Store) UpsertAlert(v model.Alert) error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("upsert alert: empty id")
	}
	_, err := s.db.Exec(`
INSERT INTO alerts (id,service_id,at,severity,name,status,summary,source,evidence_id)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
	service_id=excluded.service_id, at=excluded.at, severity=excluded.severity,
	name=excluded.name, status=excluded.status, summary=excluded.summary,
	source=excluded.source, evidence_id=excluded.evidence_id`,
		v.ID, v.ServiceID, fmtTime(v.At), v.Severity, v.Name, v.Status, v.Summary, v.Source, v.EvidenceID)
	return wrap("upsert alert", err)
}

// UpsertRunbook inserts or updates a runbook (keyed by service).
func (s *Store) UpsertRunbook(v model.Runbook) error {
	if strings.TrimSpace(v.ServiceID) == "" {
		return fmt.Errorf("upsert runbook: empty service id")
	}
	_, err := s.db.Exec(`
INSERT INTO runbooks (service_id,id,path,owner_id,steps) VALUES (?,?,?,?,?)
ON CONFLICT(service_id) DO UPDATE SET
	id=excluded.id, path=excluded.path, owner_id=excluded.owner_id, steps=excluded.steps`,
		v.ServiceID, v.ID, v.Path, v.OwnerID, toJSON(v.Steps))
	return wrap("upsert runbook", err)
}

// UpsertEvidence inserts or updates evidence.
func (s *Store) UpsertEvidence(v model.Evidence) error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("upsert evidence: empty id")
	}
	_, err := s.db.Exec(`
INSERT INTO evidence (id,source,at,kind,summary,raw_ref,service_id) VALUES (?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
	source=excluded.source, at=excluded.at, kind=excluded.kind,
	summary=excluded.summary, raw_ref=excluded.raw_ref, service_id=excluded.service_id`,
		v.ID, v.Source, fmtTime(v.At), v.Kind, v.Summary, v.RawRef, v.ServiceID)
	return wrap("upsert evidence", err)
}

// --- queries ---

// ListServices returns all services sorted by id.
func (s *Store) ListServices() ([]model.Service, error) {
	rows, err := s.db.Query(`SELECT id,name,aliases,owner_id,health,labels,sources FROM services ORDER BY id`)
	if err != nil {
		return nil, wrap("list services", err)
	}
	defer rows.Close()
	var out []model.Service
	for rows.Next() {
		var v model.Service
		var aliases, labels, sources string
		if err := rows.Scan(&v.ID, &v.Name, &aliases, &v.OwnerID, &v.Health, &labels, &sources); err != nil {
			return nil, wrap("scan service", err)
		}
		fromJSON(aliases, &v.Aliases)
		fromJSON(labels, &v.Labels)
		fromJSON(sources, &v.Sources)
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetService returns a service by exact id.
func (s *Store) GetService(id string) (*model.Service, error) {
	all, err := s.ListServices()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, ErrNotFound
}

// GetServiceByNameOrAlias resolves a service by id, name, or alias (case-insensitive).
func (s *Store) GetServiceByNameOrAlias(nameOrAlias string) (*model.Service, error) {
	all, err := s.ListServices()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(nameOrAlias))
	var matches []model.Service
	for i := range all {
		hit := strings.ToLower(all[i].ID) == q || strings.ToLower(all[i].Name) == q
		if !hit {
			for _, a := range all[i].Aliases {
				if strings.ToLower(a) == q {
					hit = true
					break
				}
			}
		}
		if hit {
			matches = append(matches, all[i])
		}
	}
	switch len(matches) {
	case 0:
		return nil, ErrNotFound
	case 1:
		return &matches[0], nil
	default:
		ids := make([]string, len(matches))
		for i := range matches {
			ids[i] = matches[i].ID
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("%w: %q matches %s", ErrAmbiguous, nameOrAlias, strings.Join(ids, ", "))
	}
}

// GetOwner returns an owner by id.
func (s *Store) GetOwner(id string) (*model.Owner, error) {
	var v model.Owner
	err := s.db.QueryRow(`SELECT id,name,team,email FROM owners WHERE id=?`, id).
		Scan(&v.ID, &v.Name, &v.Team, &v.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, wrap("get owner", err)
	}
	return &v, nil
}

// ListChanges returns changes for a service at/after since, newest first.
func (s *Store) ListChanges(serviceID string, since time.Time) ([]model.Change, error) {
	rows, err := s.db.Query(`
SELECT id,service_id,at,type,summary,author,revision,source,evidence_id
FROM changes WHERE service_id=? AND at>=? ORDER BY at DESC, id`,
		serviceID, fmtTime(since))
	if err != nil {
		return nil, wrap("list changes", err)
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

// ListAlerts returns alerts for a service. Active (firing/pending) alerts are
// always included regardless of since — their StartsAt may predate the window
// while they are still paging. Resolved/historical alerts respect since.
// Results are ordered firing/pending first, then newest.
func (s *Store) ListAlerts(serviceID string, since time.Time) ([]model.Alert, error) {
	rows, err := s.db.Query(`
SELECT id,service_id,at,severity,name,status,summary,source,evidence_id
FROM alerts WHERE service_id=? AND (status IN ('firing','pending') OR at>=?)`,
		serviceID, fmtTime(since))
	if err != nil {
		return nil, wrap("list alerts", err)
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
	// active (firing/pending) first, then newest, then id for stability.
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

// ListDependencies returns all edges touching the service (either direction).
func (s *Store) ListDependencies(serviceID string) ([]model.Dependency, error) {
	rows, err := s.db.Query(`
SELECT from_service_id,to_service_id,type,source FROM dependencies
WHERE from_service_id=? OR to_service_id=?
ORDER BY from_service_id, to_service_id, type`, serviceID, serviceID)
	if err != nil {
		return nil, wrap("list dependencies", err)
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

// GetRunbook returns the runbook for a service, or ErrNotFound.
func (s *Store) GetRunbook(serviceID string) (*model.Runbook, error) {
	var v model.Runbook
	var steps string
	err := s.db.QueryRow(`SELECT service_id,id,path,owner_id,steps FROM runbooks WHERE service_id=?`, serviceID).
		Scan(&v.ServiceID, &v.ID, &v.Path, &v.OwnerID, &steps)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, wrap("get runbook", err)
	}
	fromJSON(steps, &v.Steps)
	return &v, nil
}

// ListEvidence returns evidence for the given ids, sorted by (at, id).
func (s *Store) ListEvidence(ids []string) ([]model.Evidence, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.Query(`SELECT id,source,at,kind,summary,raw_ref,service_id FROM evidence WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, wrap("list evidence", err)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.Before(out[j].At)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// ListEvidenceForService returns evidence rows tagged with the given service id.
func (s *Store) ListEvidenceForService(serviceID string) ([]model.Evidence, error) {
	rows, err := s.db.Query(
		`SELECT id,source,at,kind,summary,raw_ref,service_id FROM evidence WHERE service_id = ? ORDER BY at, id`,
		serviceID,
	)
	if err != nil {
		return nil, wrap("list evidence for service", err)
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

// LatestChange returns the newest change for a service (any type), if any.
func (s *Store) LatestChange(serviceID string) (*model.Change, bool, error) {
	changes, err := s.ListChanges(serviceID, time.Unix(0, 0).UTC())
	if err != nil {
		return nil, false, err
	}
	if len(changes) == 0 {
		return nil, false, nil
	}
	return &changes[0], true, nil
}

// LatestDeployOrRollout returns the newest deploy or rollout for a service at
// or before asOf. Future rows (clock skew) are skipped so they cannot hide a
// real prior deploy. Used by deploy_age_* runbook checks.
func (s *Store) LatestDeployOrRollout(serviceID string, asOf time.Time) (*model.Change, bool, error) {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	rows, err := s.db.Query(`
SELECT id,service_id,at,type,summary,author,revision,source,evidence_id
FROM changes WHERE service_id=? AND type IN ('deploy','rollout') AND at<=?
ORDER BY at DESC, id LIMIT 1`, serviceID, fmtTime(asOf))
	if err != nil {
		return nil, false, wrap("latest deploy/rollout", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, rows.Err()
	}
	var v model.Change
	var at string
	if err := rows.Scan(&v.ID, &v.ServiceID, &at, &v.Type, &v.Summary, &v.Author, &v.Revision, &v.Source, &v.EvidenceID); err != nil {
		return nil, false, wrap("scan deploy/rollout", err)
	}
	parsed, perr := requireTime(at, "change", v.ID)
	if perr != nil {
		return nil, false, perr
	}
	v.At = parsed
	return &v, true, nil
}

// ResolveActiveAlertsNotIn marks firing/pending alerts from source as resolved
// when their id is not in keep. Used after Prometheus/Alertmanager scrapes so
// absent series do not remain zombies. keep may be empty (resolve all active
// for that source). Returns the number of rows updated.
func (s *Store) ResolveActiveAlertsNotIn(source string, keep map[string]bool) (int, error) {
	rows, err := s.db.Query(`
SELECT id FROM alerts
WHERE source=? AND status IN ('firing','pending')`, source)
	if err != nil {
		return 0, wrap("list active alerts for resolve", err)
	}
	defer rows.Close()
	var stale []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, wrap("scan active alert id", err)
		}
		if keep != nil && keep[id] {
			continue
		}
		stale = append(stale, id)
	}
	if err := rows.Err(); err != nil {
		return 0, wrap("iterate active alerts", err)
	}
	n := 0
	for _, id := range stale {
		res, err := s.db.Exec(`UPDATE alerts SET status='resolved' WHERE id=?`, id)
		if err != nil {
			return n, wrap("resolve alert "+id, err)
		}
		aff, _ := res.RowsAffected()
		n += int(aff)
	}
	return n, nil
}

// FindFiringAlert returns the newest active (firing/pending) alert matching an
// alert name for the given service. Name-only matches across other services are
// rejected so alert_firing:X cannot false-pass from a sibling service's page.
func (s *Store) FindFiringAlert(serviceID, name string) (*model.Alert, bool, error) {
	serviceID = strings.TrimSpace(serviceID)
	name = strings.TrimSpace(name)
	if serviceID == "" || name == "" {
		return nil, false, nil
	}
	return s.findFiringAlertBy(`service_id=? AND name=?`, serviceID, name)
}

// FindRolloutEvidence finds rollout evidence for a deployment name. Matches
// raw_ref (preferred), legacy id ev-k8s-rollout-<name>, or namespaced
// ev-k8s-rollout-<ns>-<name>. Suffix-only LIKE matches are rejected.
func (s *Store) FindRolloutEvidence(name string) (*model.Evidence, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false, nil
	}
	rows, err := s.db.Query(`
SELECT id,source,at,kind,summary,raw_ref,service_id FROM evidence
WHERE kind='rollout' AND (raw_ref=? OR id LIKE 'ev-k8s-rollout-%')
ORDER BY id`, name)
	if err != nil {
		return nil, false, wrap("find rollout evidence", err)
	}
	defer rows.Close()
	var matches []model.Evidence
	for rows.Next() {
		var v model.Evidence
		var at string
		if err := rows.Scan(&v.ID, &v.Source, &at, &v.Kind, &v.Summary, &v.RawRef, &v.ServiceID); err != nil {
			return nil, false, wrap("scan rollout evidence", err)
		}
		if v.RawRef != name && !rolloutIDMatches(v.ID, name) {
			continue
		}
		parsed, perr := requireTime(at, "evidence", v.ID)
		if perr != nil {
			return nil, false, perr
		}
		v.At = parsed
		matches = append(matches, v)
	}
	if err := rows.Err(); err != nil {
		return nil, false, wrap("iterate rollout evidence", err)
	}
	switch len(matches) {
	case 0:
		return nil, false, nil
	case 1:
		return &matches[0], true, nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.ID
		}
		return nil, false, fmt.Errorf("%w: deployment %q matches multiple rollouts: %s", ErrAmbiguous, name, strings.Join(ids, ", "))
	}
}

func rolloutIDMatches(id, name string) bool {
	const prefix = "ev-k8s-rollout-"
	if !strings.HasPrefix(id, prefix) {
		return false
	}
	suf := strings.TrimPrefix(id, prefix)
	return suf == name || strings.HasSuffix(suf, "-"+name)
}

func (s *Store) findFiringAlertBy(pred string, args ...any) (*model.Alert, bool, error) {
	rows, err := s.db.Query(`
SELECT id,service_id,at,severity,name,status,summary,source,evidence_id
FROM alerts WHERE status IN ('firing','pending') AND (`+pred+`)
ORDER BY CASE status WHEN 'firing' THEN 0 ELSE 1 END, at DESC, id LIMIT 1`, args...)
	if err != nil {
		return nil, false, wrap("find firing alert", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, rows.Err()
	}
	var v model.Alert
	var at string
	if err := rows.Scan(&v.ID, &v.ServiceID, &at, &v.Severity, &v.Name, &v.Status, &v.Summary, &v.Source, &v.EvidenceID); err != nil {
		return nil, false, wrap("scan firing alert", err)
	}
	parsed, perr := requireTime(at, "alert", v.ID)
	if perr != nil {
		return nil, false, perr
	}
	v.At = parsed
	return &v, true, nil
}

// Counts returns row counts per table (for `opsgraph status`).
func (s *Store) Counts() (map[string]int, error) {
	tables := []string{"services", "owners", "changes", "dependencies", "alerts", "runbooks", "evidence"}
	out := make(map[string]int, len(tables))
	for _, t := range tables {
		var n int
		if err := s.db.QueryRow(`SELECT count(*) FROM ` + t).Scan(&n); err != nil {
			return nil, wrap("count "+t, err)
		}
		out[t] = n
	}
	return out, nil
}

func wrap(what string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", what, err)
}
