package version

import "testing"

func TestStringIncludesFields(t *testing.T) {
	Version = "v1.2.3"
	Commit = "abc"
	Date = "2026-07-31T12:00:00Z"
	got := String()
	want := "v1.2.3 (commit abc, built 2026-07-31T12:00:00Z)"
	if got != want {
		t.Fatalf("String()=%q want %q", got, want)
	}
}
