// Package impact computes recursive downstream blast impact trees.
package impact

import (
	"sort"

	"github.com/sanjeev0120test/oncallgraph/internal/model"
)

// Node is one service in the impact tree.
type Node struct {
	ID       string `json:"id"`
	Health   string `json:"health,omitempty"`
	Depth    int    `json:"depth"`
	Children []Node `json:"children,omitempty"`
}

// Result is the full downstream impact of a service outage.
type Result struct {
	Root     string   `json:"root"`
	Affected []string `json:"affected"`
	MaxDepth int      `json:"max_depth"`
	Tree     Node     `json:"tree"`
}

// Downstream builds a recursive tree of services that depend (transitively) on root.
// Edge direction: From depends on To, so children of X are services whose To==X.
func Downstream(root string, services []model.Service, deps []model.Dependency) Result {
	health := map[string]string{}
	for _, s := range services {
		health[s.ID] = s.Health
	}
	children := map[string][]string{}
	for _, d := range deps {
		children[d.ToServiceID] = append(children[d.ToServiceID], d.FromServiceID)
	}
	for k := range children {
		sort.Strings(children[k])
	}

	seen := map[string]bool{root: true}
	var affected []string
	maxDepth := 0

	var walk func(id string, depth int) Node
	walk = func(id string, depth int) Node {
		if depth > maxDepth {
			maxDepth = depth
		}
		n := Node{ID: id, Health: health[id], Depth: depth}
		for _, child := range children[id] {
			if seen[child] {
				continue
			}
			seen[child] = true
			affected = append(affected, child)
			n.Children = append(n.Children, walk(child, depth+1))
		}
		return n
	}

	tree := walk(root, 0)
	sort.Strings(affected)
	return Result{Root: root, Affected: affected, MaxDepth: maxDepth, Tree: tree}
}
