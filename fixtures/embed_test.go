package fixtures

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbedMatchesDisk enforces PLAN's single-source claim: demo (embed) and
// --fixture / opsgraph test (disk) must see identical incident_checkout bytes.
func TestEmbedMatchesDisk(t *testing.T) {
	emb, err := CheckoutFS()
	if err != nil {
		t.Fatalf("CheckoutFS: %v", err)
	}
	diskRoot := "incident_checkout"
	err = fs.WalkDir(emb, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		embBytes, err := fs.ReadFile(emb, path)
		if err != nil {
			return err
		}
		diskPath := filepath.Join(diskRoot, filepath.FromSlash(path))
		diskBytes, err := os.ReadFile(diskPath)
		if err != nil {
			return err
		}
		if !bytes.Equal(embBytes, diskBytes) {
			t.Errorf("embed≠disk for %s (embed %d bytes, disk %d bytes)", path, len(embBytes), len(diskBytes))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("embed walk: %v", err)
	}
	err = filepath.WalkDir(diskRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(diskRoot, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if _, err := fs.Stat(emb, slash); err != nil {
			t.Errorf("disk file %s missing from embed", slash)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("disk walk: %v", err)
	}
}
