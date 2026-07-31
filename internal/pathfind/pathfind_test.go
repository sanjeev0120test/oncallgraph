package pathfind_test

import (
	"testing"

	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/opsgraph/opsgraph/internal/pathfind"
)

func TestShortestPath(t *testing.T) {
	deps := []model.Dependency{
		{FromServiceID: "order", ToServiceID: "checkout"},
		{FromServiceID: "checkout", ToServiceID: "auth"},
		{FromServiceID: "checkout", ToServiceID: "redis"},
	}
	p, err := pathfind.Shortest(deps, "order", "auth")
	if err != nil {
		t.Fatal(err)
	}
	if p.Hops != 2 || len(p.Nodes) != 3 {
		t.Fatalf("got %+v", p)
	}
	if p.Nodes[0] != "order" || p.Nodes[1] != "checkout" || p.Nodes[2] != "auth" {
		t.Fatalf("nodes = %v", p.Nodes)
	}
}

func TestNoPath(t *testing.T) {
	_, err := pathfind.Shortest(nil, "a", "b")
	if err == nil {
		t.Fatal("expected error")
	}
}
