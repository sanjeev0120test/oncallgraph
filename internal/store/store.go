// Package store persists the incident graph in a pure-Go SQLite database.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	raw_ref TEXT
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

// --- helpers ---

func fmtTime(t time.Time) string {
	return t.UTC().Truncate(time.Second).Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
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
