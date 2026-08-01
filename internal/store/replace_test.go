package store_test

import (
	"testing"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestReplaceFromFile(t *testing.T) {
	src, cleanupSrc, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupSrc)
	if err := src.UpsertService(model.Service{ID: "svc", Name: "svc", Health: model.HealthHealthy}); err != nil {
		t.Fatal(err)
	}
	dstDir := t.TempDir()
	dst, err := store.Open(dstDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dst.Close() })
	if err := dst.ReplaceFromFile(src.Path()); err != nil {
		t.Fatal(err)
	}
	svc, err := dst.GetService("svc")
	if err != nil || svc.ID != "svc" {
		t.Fatalf("replace lost service: %+v err=%v", svc, err)
	}
}
