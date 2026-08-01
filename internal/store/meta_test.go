package store

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestAsOfRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if _, ok, err := s.AsOf(); err != nil || ok {
		t.Fatalf("empty as_of: ok=%v err=%v", ok, err)
	}
	want := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.SetAsOf(want); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.AsOf()
	if err != nil || !ok || !got.Equal(want) {
		t.Fatalf("AsOf = %v ok=%v err=%v", got, ok, err)
	}
}

func TestFindAliasCollisions(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpsertService(model.Service{ID: "a", Name: "a", Aliases: []string{"shared"}, Health: model.HealthUnknown}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertService(model.Service{ID: "b", Name: "b", Aliases: []string{"shared"}, Health: model.HealthUnknown}); err != nil {
		t.Fatal(err)
	}
	hits, err := s.FindAliasCollisions()
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected collision")
	}
}
