package impact_test

import (
	"testing"

	"github.com/sanjeev0120test/opsgraph/internal/impact"
	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestDownstreamImpact(t *testing.T) {
	svcs := []model.Service{{ID: "auth"}, {ID: "checkout"}, {ID: "order"}}
	deps := []model.Dependency{
		{FromServiceID: "checkout", ToServiceID: "auth"},
		{FromServiceID: "order", ToServiceID: "checkout"},
	}
	res := impact.Downstream("auth", svcs, deps)
	if len(res.Affected) != 2 {
		t.Fatalf("affected=%v", res.Affected)
	}
	if res.MaxDepth < 1 {
		t.Fatalf("max depth=%d", res.MaxDepth)
	}
}
