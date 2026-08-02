package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestDemoMatchesFixtureAskJSON proves embed/demo wiring matches disk --fixture
// ask for the same service/window (product path, not just file bytes).
func TestDemoMatchesFixtureAskJSON(t *testing.T) {
	fx := fixtureDir(t)
	demoOut, _, code := runRoot(t, "demo", "--format", "json")
	if code != 0 {
		t.Fatalf("demo json exit = %d", code)
	}
	askOut, _, code := runRoot(t, "ask", "checkout", "--fixture", fx, "--since", "1h", "--format", "json")
	if code != 0 {
		t.Fatalf("ask fixture json exit = %d", code)
	}
	if !bytes.Equal([]byte(demoOut), []byte(askOut)) {
		// Pretty-print a short hint; full dumps are large.
		var demoObj, askObj map[string]json.RawMessage
		_ = json.Unmarshal([]byte(demoOut), &demoObj)
		_ = json.Unmarshal([]byte(askOut), &askObj)
		t.Fatalf("demo JSON ≠ ask --fixture JSON (demo keys=%d ask keys=%d demo_len=%d ask_len=%d)",
			len(demoObj), len(askObj), len(demoOut), len(askOut))
	}
}
