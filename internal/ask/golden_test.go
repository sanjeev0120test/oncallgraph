package ask_test

import (
	"bytes"
	"io/fs"
	"testing"
	"time"

	"github.com/opsgraph/opsgraph/fixtures"
	"github.com/opsgraph/opsgraph/internal/ask"
	"github.com/opsgraph/opsgraph/internal/ingest"
	"github.com/opsgraph/opsgraph/internal/output"
	"github.com/opsgraph/opsgraph/internal/runbook"
	"github.com/opsgraph/opsgraph/internal/store"
)

// TestGoldensMatch enforces byte-identical output against the checked-in golden
// files across every OS in CI (embedded pack includes expected/).
func TestGoldensMatch(t *testing.T) {
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

	verifier := runbook.NewVerifier(s, now)
	for _, svc := range []string{"checkout", "auth"} {
		res, err := ask.Ask(s, svc, ask.Options{Since: time.Hour, Now: now, WithRunbook: true})
		if err != nil {
			t.Fatalf("ask %s: %v", svc, err)
		}
		assertGolden(t, fsys, "expected/ask_"+svc+".json", res)

		vr, err := verifier.VerifyService(svc)
		if err != nil {
			t.Fatalf("verify %s: %v", svc, err)
		}
		assertGolden(t, fsys, "expected/verify_"+svc+".json", vr)
	}
}

func assertGolden(t *testing.T, fsys fs.FS, name string, v any) {
	t.Helper()
	var buf bytes.Buffer
	if err := output.JSON(&buf, v); err != nil {
		t.Fatal(err)
	}
	want, err := fs.ReadFile(fsys, name)
	if err != nil {
		t.Fatalf("read golden %s: %v (run: opsgraph test ./fixtures/incident_checkout --update)", name, err)
	}
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(want, buf.Bytes()) {
		t.Fatalf("golden %s mismatch; regenerate with:\n  go run ./cmd/opsgraph test ./fixtures/incident_checkout --update", name)
	}
}
