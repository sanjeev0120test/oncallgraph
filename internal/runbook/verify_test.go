package runbook_test

import (
	"testing"
	"time"

	"github.com/opsgraph/opsgraph/fixtures"
	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/opsgraph/opsgraph/internal/runbook"
	"github.com/opsgraph/opsgraph/internal/store"
)

func loadFixture(t *testing.T) (*store.Store, time.Time) {
	t.Helper()
	fsys, err := fixtures.CheckoutFS()
	if err != nil {
		t.Fatal(err)
	}
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	now, err := ingest.IngestFixtureFS(s, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return s, now
}

func TestVerifyCheckoutRunbookIsStale(t *testing.T) {
	s, now := loadFixture(t)
	res, err := runbook.NewVerifier(s, now).VerifyService("checkout")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.StatusStale {
		t.Fatalf("checkout runbook status = %q, want stale", res.Status)
	}
	want := []string{model.StatusPass, model.StatusStale, model.StatusManual}
	if len(res.Steps) != 3 {
		t.Fatalf("want 3 steps, got %d", len(res.Steps))
	}
	for i, w := range want {
		if res.Steps[i].Status != w {
			t.Fatalf("step %d status = %q, want %q (msg=%q)", i+1, res.Steps[i].Status, w, res.Steps[i].Message)
		}
	}
	// The passing deploy_age check should cite evidence.
	if res.Steps[0].EvidenceID == "" {
		t.Fatal("deploy_age step should cite evidence")
	}
}

func TestVerifyAuthRunbookPasses(t *testing.T) {
	s, now := loadFixture(t)
	res, err := runbook.NewVerifier(s, now).VerifyService("auth")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.StatusPass {
		t.Fatalf("auth runbook status = %q, want pass", res.Status)
	}
}

func TestVerifyMissingRunbook(t *testing.T) {
	s, now := loadFixture(t)
	res, err := runbook.NewVerifier(s, now).VerifyService("order")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.StatusMissing {
		t.Fatalf("order runbook status = %q, want missing", res.Status)
	}
}
