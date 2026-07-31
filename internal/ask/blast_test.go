package ask_test

import (
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestBlastRadiusDirectionAndSynthesize(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)

	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.UpsertService(model.Service{ID: "a", Name: "a", Health: model.HealthHealthy, Sources: []string{"test"}}))
	must(s.UpsertService(model.Service{ID: "c", Name: "c", Health: model.HealthHealthy, Sources: []string{"test"}}))
	// A depends on B (missing -> synthesize); C depends on A.
	must(s.UpsertDependency(model.Dependency{FromServiceID: "a", ToServiceID: "b", Type: "http", Source: "test"}))
	must(s.UpsertDependency(model.Dependency{FromServiceID: "c", ToServiceID: "a", Type: "http", Source: "test"}))

	res, err := ask.Ask(s, "a", ask.Options{Since: time.Hour, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Upstream) != 1 || res.Upstream[0].ID != "b" || res.Upstream[0].Health != model.HealthUnknown {
		t.Fatalf("upstream = %+v, want synthesized b/unknown", res.Upstream)
	}
	if len(res.Upstream[0].Sources) != 1 || res.Upstream[0].Sources[0] != "dependency" {
		t.Fatalf("synthesized sources = %v", res.Upstream[0].Sources)
	}
	if len(res.Downstream) != 1 || res.Downstream[0].ID != "c" {
		t.Fatalf("downstream = %+v, want [c]", res.Downstream)
	}
}
