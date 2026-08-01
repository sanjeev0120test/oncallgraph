package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestFindFiringAlertNameOnly(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertAlert(model.Alert{
		ID: "a1", ServiceID: "checkout", At: now, Name: "CheckoutErrorRateHigh",
		Status: "firing", Source: "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.FindFiringAlert("checkout"); err != nil || ok {
		t.Fatalf("service id must not match alert name: ok=%v err=%v", ok, err)
	}
	al, ok, err := s.FindFiringAlert("CheckoutErrorRateHigh")
	if err != nil || !ok || al.ID != "a1" {
		t.Fatalf("name match failed: ok=%v al=%+v err=%v", ok, al, err)
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
