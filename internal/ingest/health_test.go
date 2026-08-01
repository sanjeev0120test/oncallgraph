package ingest

import "testing"

func TestMergeHealthKeepsWorseAppState(t *testing.T) {
	cases := []struct {
		cur, in, want string
	}{
		{"", "healthy", "healthy"},
		{"unknown", "degraded", "degraded"},
		{"degraded", "healthy", "degraded"},
		{"unhealthy", "healthy", "unhealthy"},
		{"healthy", "degraded", "degraded"},
		{"degraded", "unhealthy", "unhealthy"},
		{"healthy", "healthy", "healthy"},
	}
	for _, tc := range cases {
		got := mergeHealth(tc.cur, tc.in)
		if got != tc.want {
			t.Fatalf("mergeHealth(%q,%q)=%q want %q", tc.cur, tc.in, got, tc.want)
		}
	}
}
