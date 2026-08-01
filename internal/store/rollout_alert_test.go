package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestFindFiringAlertServiceScoped(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertAlert(model.Alert{
		ID: "a1", ServiceID: "payments", At: now, Name: "HighErrorRate",
		Status: "firing", Source: "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.FindFiringAlert("checkout", "HighErrorRate"); err != nil || ok {
		t.Fatalf("sibling service must not match: ok=%v err=%v", ok, err)
	}
	al, ok, err := s.FindFiringAlert("payments", "HighErrorRate")
	if err != nil || !ok || al.ID != "a1" {
		t.Fatalf("same-service name match failed: ok=%v al=%+v err=%v", ok, al, err)
	}
	if _, ok, err := s.FindFiringAlert("payments", "payments"); err != nil || ok {
		t.Fatalf("service id must not match alert name: ok=%v err=%v", ok, err)
	}
}

func TestFindRolloutEvidenceNoSuffixFalsePositive(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertEvidence(model.Evidence{
		ID: "ev-k8s-rollout-prod-mysvc", Source: "kubernetes", At: now,
		Kind: "rollout", Summary: "rollout mysvc", RawRef: "mysvc", ServiceID: "mysvc",
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.FindRolloutEvidence("svc"); err != nil || ok {
		t.Fatalf("suffix false-positive: ok=%v err=%v", ok, err)
	}
	ev, ok, err := s.FindRolloutEvidence("mysvc")
	if err != nil || !ok || ev.ID != "ev-k8s-rollout-prod-mysvc" {
		t.Fatalf("expected mysvc rollout: ok=%v ev=%+v err=%v", ok, ev, err)
	}
}

func TestFindRolloutEvidenceAmbiguous(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{"ev-k8s-rollout-a-api", "ev-k8s-rollout-b-api"} {
		if err := s.UpsertEvidence(model.Evidence{
			ID: id, Source: "kubernetes", At: now, Kind: "rollout",
			Summary: "rollout api", RawRef: "api", ServiceID: "api",
		}); err != nil {
			t.Fatal(err)
		}
	}
	_, _, err = s.FindRolloutEvidence("api")
	if !errors.Is(err, store.ErrAmbiguous) {
		t.Fatalf("want ErrAmbiguous, got %v", err)
	}
}

func TestUpsertRejectsEmptyIDs(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	if err := s.UpsertAlert(model.Alert{ID: " ", ServiceID: "x"}); err == nil {
		t.Fatal("expected empty alert id error")
	}
	if err := s.UpsertEvidence(model.Evidence{ID: ""}); err == nil {
		t.Fatal("expected empty evidence id error")
	}
	if err := s.UpsertDependency(model.Dependency{FromServiceID: "a", ToServiceID: ""}); err == nil {
		t.Fatal("expected empty dependency error")
	}
}

func TestLatestDeployOrRolloutSkipsFuture(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertChange(model.Change{
		ID: "future", ServiceID: "checkout", At: now.Add(time.Hour), Type: "deploy",
		Summary: "future", Source: "fixture", EvidenceID: "ev-f",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertChange(model.Change{
		ID: "past", ServiceID: "checkout", At: now.Add(-10 * time.Minute), Type: "deploy",
		Summary: "past", Source: "fixture", EvidenceID: "ev-p",
	}); err != nil {
		t.Fatal(err)
	}
	ch, ok, err := s.LatestDeployOrRollout("checkout", now)
	if err != nil || !ok || ch.ID != "past" {
		t.Fatalf("want past deploy, got ok=%v ch=%+v err=%v", ok, ch, err)
	}
}
