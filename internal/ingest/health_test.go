package ingest

import "testing"

func TestMergeHealthKeepsWorseAppState(t *testing.T) {
	cases := []struct {
		cur, in string
		sources []string
		want    string
	}{
		{"", "healthy", nil, "healthy"},
		{"unknown", "degraded", nil, "degraded"},
		{"degraded", "healthy", nil, "degraded"}, // app-level must not be cleared by replicas
		{"unhealthy", "healthy", nil, "unhealthy"},
		{"healthy", "degraded", nil, "degraded"},
		{"degraded", "unhealthy", nil, "unhealthy"},
		{"healthy", "healthy", nil, "healthy"},
		{"degraded", "healthy", []string{"kubernetes"}, "healthy"}, // k8s recovery wins
	}
	for _, tc := range cases {
		got := mergeHealth(tc.cur, tc.in, tc.sources)
		if got != tc.want {
			t.Fatalf("mergeHealth(%q,%q,%v)=%q want %q", tc.cur, tc.in, tc.sources, got, tc.want)
		}
	}
}
