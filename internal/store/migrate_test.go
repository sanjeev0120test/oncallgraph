package store

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSchemaRejectsUnsupportedUserVersion(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 999`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	_, err = Open(dir)
	if err == nil {
		t.Fatal("expected Open to reject future schema version")
	}
	if !strings.Contains(err.Error(), "newer than this binary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFreshStoreStampsCurrentSchema(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ver, err := s.UserVersion()
	if err != nil {
		t.Fatal(err)
	}
	if ver != SchemaVersion {
		t.Fatalf("user_version=%d want %d", ver, SchemaVersion)
	}
}

func TestMigrateV1ToV2(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE services (id TEXT PRIMARY KEY, name TEXT NOT NULL, aliases TEXT, owner_id TEXT, health TEXT, labels TEXT, sources TEXT);
CREATE TABLE owners (id TEXT PRIMARY KEY, name TEXT, team TEXT, email TEXT);
CREATE TABLE changes (id TEXT PRIMARY KEY, service_id TEXT, at TEXT, type TEXT, summary TEXT, author TEXT, revision TEXT, source TEXT, evidence_id TEXT);
CREATE TABLE dependencies (from_service_id TEXT, to_service_id TEXT, type TEXT, source TEXT, PRIMARY KEY (from_service_id, to_service_id, type));
CREATE TABLE alerts (id TEXT PRIMARY KEY, service_id TEXT, at TEXT, severity TEXT, name TEXT, status TEXT, summary TEXT, source TEXT, evidence_id TEXT);
CREATE TABLE runbooks (service_id TEXT PRIMARY KEY, id TEXT, path TEXT, owner_id TEXT, steps TEXT);
CREATE TABLE evidence (id TEXT PRIMARY KEY, source TEXT, at TEXT, kind TEXT, summary TEXT, raw_ref TEXT, service_id TEXT);
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
PRAGMA user_version = 1;
`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open v1 store: %v", err)
	}
	defer s.Close()
	ver, err := s.UserVersion()
	if err != nil {
		t.Fatal(err)
	}
	if ver != 2 {
		t.Fatalf("user_version=%d want 2 after migrate", ver)
	}
}

func TestSchemaRejectsUnknownLegacyVersion(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = -1`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	_, err = Open(dir)
	if err == nil {
		t.Fatal("expected Open to reject unknown legacy schema")
	}
	if !strings.Contains(err.Error(), "needs migration") {
		t.Fatalf("unexpected error: %v", err)
	}
}
