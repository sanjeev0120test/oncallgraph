// Package pathfind finds dependency paths between services.
package pathfind

import (
	"fmt"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// Path is an ordered list of service IDs from start to goal along depends-on edges.
type Path struct {
	Nodes []string `json:"nodes"`
	Hops  int      `json:"hops"`
}

// Shortest finds the shortest depends-on path from → to (BFS).
// Direction follows dependency edges: walker moves From → To.
func Shortest(deps []model.Dependency, from, to string) (Path, error) {
	if from == to {
		return Path{Nodes: []string{from}, Hops: 0}, nil
	}
	adj := map[string][]string{}
	for _, d := range deps {
		adj[d.FromServiceID] = append(adj[d.FromServiceID], d.ToServiceID)
	}
	type item struct {
		id   string
		path []string
	}
	q := []item{{from, []string{from}}}
	seen := map[string]bool{from: true}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, next := range adj[cur.id] {
			if seen[next] {
				continue
			}
			np := append(append([]string{}, cur.path...), next)
			if next == to {
				return Path{Nodes: np, Hops: len(np) - 1}, nil
			}
			seen[next] = true
			q = append(q, item{next, np})
		}
	}
	return Path{}, fmt.Errorf("no dependency path from %q to %q", from, to)
}
