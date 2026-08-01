package runbook_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/runbook"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestDeployAgeRejectsNonPositiveDuration(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	v := runbook.NewVerifier(s, now)
	res, err := v.Verify(model.Runbook{
		ServiceID: "checkout",
		Steps:     []model.RunbookStep{{Number: 1, Text: "age", Check: "deploy_age_lt:0s"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Steps[0].Status != model.StatusError {
		t.Fatalf("status=%s want error", res.Steps[0].Status)
	}
}

func TestDeployAgeFutureIsStale(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertChange(model.Change{
		ID: "c1", ServiceID: "checkout", At: now.Add(time.Hour), Type: "deploy",
		Summary: "future", Source: "fixture", EvidenceID: "ev1",
	}); err != nil {
		t.Fatal(err)
	}
	v := runbook.NewVerifier(s, now)
	res, err := v.Verify(model.Runbook{
		ServiceID: "checkout",
		Steps:     []model.RunbookStep{{Number: 1, Text: "age", Check: "deploy_age_lt:30m"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Steps[0].Status != model.StatusStale {
		t.Fatalf("status=%s want stale", res.Steps[0].Status)
	}
	if !strings.Contains(res.Steps[0].Message, "future") {
		t.Fatalf("message=%q", res.Steps[0].Message)
	}
}

func TestDeployAgeIgnoresCommits(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.UpsertService(model.Service{ID: "svc", Name: "svc", Health: model.HealthHealthy}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertChange(model.Change{
		ID: "c1", ServiceID: "svc", At: now.Add(-5 * time.Minute), Type: "commit",
		Summary: "fresh commit", Source: "git",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertChange(model.Change{
		ID: "d1", ServiceID: "svc", At: now.Add(-2 * time.Hour), Type: "deploy",
		Summary: "old deploy", Source: "fixture", EvidenceID: "ev-d1",
	}); err != nil {
		t.Fatal(err)
	}
	rb := model.Runbook{
		ID: "rb", ServiceID: "svc", Path: "x.md",
		Steps: []model.RunbookStep{
			{Number: 1, Text: "recent deploy?", Check: "deploy_age_lt:60m"},
		},
	}
	res, err := runbook.NewVerifier(s, now).Verify(rb)
	if err != nil {
		t.Fatal(err)
	}
	if res.Steps[0].Status != model.StatusStale {
		t.Fatalf("status = %q msg=%q; commit must not satisfy deploy_age_lt", res.Steps[0].Status, res.Steps[0].Message)
	}
	if res.Steps[0].EvidenceID != "ev-d1" {
		t.Fatalf("evidence = %q, want ev-d1 (the deploy)", res.Steps[0].EvidenceID)
	}
}
