package model

import "testing"

func TestAlertActive(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"firing", true},
		{"pending", true},
		{"resolved", false},
		{"suppressed", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := AlertActive(tc.status); got != tc.want {
			t.Fatalf("AlertActive(%q)=%v want %v", tc.status, got, tc.want)
		}
	}
}

func TestHealthConstants(t *testing.T) {
	for _, h := range []string{HealthHealthy, HealthDegraded, HealthUnhealthy, HealthUnknown} {
		if h == "" {
			t.Fatal("empty health constant")
		}
	}
}
