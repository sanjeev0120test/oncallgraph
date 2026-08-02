package store

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// Frozen schema fingerprint for SchemaVersion. When DDL changes intentionally:
// 1) bump SchemaVersion + add a migration path, 2) update both constants below.
const (
	frozenSchemaVersion = 2
	frozenSchemaSHA256  = "395325ad2c048c239973c290a2a05ed10478bcc0bc48dfae494985477a8689d9"
)

func TestSchemaDDLFingerprintCoupledToVersion(t *testing.T) {
	sum := sha256.Sum256([]byte(strings.TrimSpace(schemaDDL)))
	got := hex.EncodeToString(sum[:])
	if got != frozenSchemaSHA256 {
		t.Fatalf("schemaDDL changed (sha256=%s). Bump SchemaVersion (now %d), migrate, then update frozenSchemaSHA256+frozenSchemaVersion", got, SchemaVersion)
	}
	if SchemaVersion != frozenSchemaVersion {
		t.Fatalf("SchemaVersion=%d but fingerprint freeze still at v%d", SchemaVersion, frozenSchemaVersion)
	}
}
