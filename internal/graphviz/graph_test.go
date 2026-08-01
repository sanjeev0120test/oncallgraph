package graphviz

import (
	"strings"
	"testing"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

func TestMermaidEscapesQuotesAndIsStable(t *testing.T) {
	svcs := []model.Service{
		{ID: "b-svc", Health: `degraded "x"`},
		{ID: "a-svc", Health: "healthy"},
	}
	deps := []model.Dependency{
		{FromServiceID: "b-svc", ToServiceID: "a-svc", Type: `http "edge"`},
	}
	a := Mermaid(svcs, deps)
	b := Mermaid([]model.Service{svcs[1], svcs[0]}, deps)
	if a != b {
		t.Fatalf("mermaid not deterministic:\n%s\n---\n%s", a, b)
	}
	if strings.Contains(a, `"`) && strings.Count(a, `"`)%2 != 0 {
		// Node labels use quotes; inner quotes must be escaped to single quotes.
	}
	if strings.Contains(a, `degraded "x"`) {
		t.Fatalf("unescaped quote in label: %s", a)
	}
	if !strings.Contains(a, "degraded 'x'") {
		t.Fatalf("expected escaped health label: %s", a)
	}
	if !strings.Contains(a, "http 'edge'") {
		t.Fatalf("expected escaped edge label: %s", a)
	}
}
