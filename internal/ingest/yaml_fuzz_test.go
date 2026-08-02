package ingest

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func FuzzServicesYAML(f *testing.F) {
	f.Add([]byte("services:\n  - id: api\n    name: API\n    health: healthy\n"))
	f.Add([]byte(""))
	f.Add([]byte("services: [1, 2, {"))
	f.Add([]byte("services:\n  - id: !!binary abc\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		var out fxServices
		// Must never panic on untrusted fixture YAML; parse errors are fine.
		_ = yaml.Unmarshal(data, &out)
	})
}
