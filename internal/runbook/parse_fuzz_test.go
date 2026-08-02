package runbook

import (
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add([]byte("---\nservice: checkout\nowner: payments\n---\n\n# x\n\n1. Step\n<!-- opsgraph:check=manual -->\n"))
	f.Add([]byte(""))
	f.Add([]byte("---\nnot: yaml: [[\n---\nbody"))
	f.Add([]byte("1. only a step\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		// Must never panic on untrusted runbook bytes; error is fine.
		_, _, _ = Parse(data, "fuzz.md")
	})
}
