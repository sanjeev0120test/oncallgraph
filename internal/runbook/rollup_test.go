package runbook

import (
	"testing"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestRollupManualOnly(t *testing.T) {
	got := rollup([]model.StepVerifyResult{
		{Status: model.StatusManual},
		{Status: model.StatusManual},
	})
	if got != model.StatusManual {
		t.Fatalf("manual-only rollup = %q, want manual", got)
	}
}

func TestRollupEmptyIsManual(t *testing.T) {
	if got := rollup(nil); got != model.StatusManual {
		t.Fatalf("empty rollup = %q", got)
	}
}

func TestRollupPassWithManual(t *testing.T) {
	got := rollup([]model.StepVerifyResult{
		{Status: model.StatusPass},
		{Status: model.StatusManual},
	})
	if got != model.StatusPass {
		t.Fatalf("pass+manual = %q, want pass", got)
	}
}
