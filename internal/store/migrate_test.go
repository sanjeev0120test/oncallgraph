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
