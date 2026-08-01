// Package store persists the incident graph in a pure-Go SQLite database.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver, no CGO
)

// Store wraps a single-connection SQLite database.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens (creating if needed) a persistent store under dataDir/state.db.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir %q: %w", dataDir, err)
	}
	return open(filepath.Join(dataDir, "state.db"))
}

// OpenTemp opens an isolated store in a temp directory. The returned cleanup
// closes the DB and removes the directory; use it for demo/test/--fixture runs.
func OpenTemp() (*Store, func(), error) {
	dir, err := os.MkdirTemp("", "opsgraph-")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp dir: %w", err)
	}
	s, err := open(filepath.Join(dir, "state.db"))
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, nil, err
	}
	cleanup := func() {
		_ = s.Close()
		_ = os.RemoveAll(dir)
	}
	return s, cleanup, nil
}

func open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite has a single writer; serialize all access through one connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	for _, pragma := range []string{"PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000"} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply %q: %w", pragma, err)
		}
	}
	s := &Store{db: db, path: dbPath}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// Path returns the on-disk SQLite path.
func (s *Store) Path() string { return s.path }

// SchemaVersion is the current PRAGMA user_version for this schema.
const SchemaVersion = 1

func (s *Store) initSchema() error {
	const schema = `
CREATE TABLE IF NOT EXISTS services (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	aliases TEXT,
	owner_id TEXT,
	health TEXT,
	labels TEXT,
	sources TEXT
);
CREATE TABLE IF NOT EXISTS owners (
	id TEXT PRIMARY KEY,
	name TEXT,
	team TEXT,
	email TEXT
);
CREATE TABLE IF NOT EXISTS changes (
	id TEXT PRIMARY KEY,
	service_id TEXT,
	at TEXT,
	type TEXT,
	summary TEXT,
	author TEXT,
	revision TEXT,
	source TEXT,
	evidence_id TEXT
);
CREATE TABLE IF NOT EXISTS dependencies (
	from_service_id TEXT,
	to_service_id TEXT,
	type TEXT,
	source TEXT,
	PRIMARY KEY (from_service_id, to_service_id, type)
);
CREATE TABLE IF NOT EXISTS alerts (
	id TEXT PRIMARY KEY,
	service_id TEXT,
	at TEXT,
	severity TEXT,
	name TEXT,
	status TEXT,
	summary TEXT,
	source TEXT,
	evidence_id TEXT
);
CREATE TABLE IF NOT EXISTS runbooks (
	service_id TEXT PRIMARY KEY,
	id TEXT,
	path TEXT,
	owner_id TEXT,
	steps TEXT
);
CREATE TABLE IF NOT EXISTS evidence (
	id TEXT PRIMARY KEY,
	source TEXT,
	at TEXT,
	kind TEXT,
	summary TEXT,
	raw_ref TEXT,
	service_id TEXT
);
CREATE INDEX IF NOT EXISTS idx_changes_service_at ON changes(service_id, at);
CREATE INDEX IF NOT EXISTS idx_alerts_service_status_at ON alerts(service_id, status, at);
CREATE INDEX IF NOT EXISTS idx_evidence_service ON evidence(service_id);
CREATE INDEX IF NOT EXISTS idx_deps_from ON dependencies(from_service_id);
CREATE INDEX IF NOT EXISTS idx_deps_to ON dependencies(to_service_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	// Best-effort column add for stores created before service_id existed.
	_, _ = s.db.Exec(`ALTER TABLE evidence ADD COLUMN service_id TEXT`)

	var ver int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&ver); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}
	if ver > SchemaVersion {
		return fmt.Errorf("database schema v%d is newer than this binary (supports v%d); upgrade opsgraph", ver, SchemaVersion)
	}
	// Future: apply migrations for ver < SchemaVersion here.
	if ver != SchemaVersion {
		if _, err := s.db.Exec(fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion)); err != nil {
			return fmt.Errorf("set user_version: %w", err)
		}
	}
	return nil
}

// UserVersion returns PRAGMA user_version.
func (s *Store) UserVersion() (int, error) {
	var v int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return 0, wrap("user_version", err)
	}
	return v, nil
}

// Reset deletes all entity rows so a subsequent ingest fully replaces prior state.
// Schema and user_version are preserved.
func (s *Store) Reset() error {
	tx, err := s.db.Begin()
	if err != nil {
		return wrap("begin reset", err)
	}
	for _, q := range []string{
		`DELETE FROM evidence`,
		`DELETE FROM runbooks`,
		`DELETE FROM alerts`,
		`DELETE FROM dependencies`,
		`DELETE FROM changes`,
		`DELETE FROM services`,
		`DELETE FROM owners`,
	} {
		if _, err := tx.Exec(q); err != nil {
			_ = tx.Rollback()
			return wrap("reset "+q, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrap("commit reset", err)
	}
	return nil
}

// --- helpers ---

func fmtTime(t time.Time) string {
	return t.UTC().Truncate(time.Second).Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	t, _ := parseTimeOK(s)
	return t
}

// parseTimeOK accepts RFC3339 and RFC3339Nano (and a few common variants
// written by connectors). Corrupt values yield ok=false and a zero time.
func parseTimeOK(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z07:00",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func fromJSON(s string, v any) {
	if s == "" {
		return
	}
	_ = json.Unmarshal([]byte(s), v)
}
