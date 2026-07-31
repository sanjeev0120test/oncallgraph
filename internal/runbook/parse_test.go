package runbook_test

import (
	"testing"

	"github.com/sanjeev0120test/opsgraph/internal/runbook"
)

func TestParseOpsgraphChecks(t *testing.T) {
	md := []byte(`---
service: checkout
---

1. Prefer the canonical annotation.
<!-- opsgraph:check=deploy_age_lt:60m -->

2. Manual step.
<!-- opsgraph:check=manual -->
`)
	rb, fm, err := runbook.Parse(md, "runbooks/checkout.md")
	if err != nil {
		t.Fatal(err)
	}
	if fm.Service != "checkout" {
		t.Fatalf("front matter service = %q", fm.Service)
	}
	if len(rb.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(rb.Steps))
	}
	if rb.Steps[0].Check != "deploy_age_lt:60m" {
		t.Fatalf("step1 check = %q", rb.Steps[0].Check)
	}
	if rb.Steps[1].Check != "manual" {
		t.Fatalf("step2 check = %q", rb.Steps[1].Check)
	}
}
