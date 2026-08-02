package main

import "testing"

func TestAlertAlwaysVisible(t *testing.T) {
	cases := map[string]bool{
		"firing":     true,
		"pending":    true,
		"suppressed": true,
		"resolved":   false,
		"":           false,
	}
	for status, want := range cases {
		if got := alertAlwaysVisible(status); got != want {
			t.Fatalf("alertAlwaysVisible(%q)=%v want %v", status, got, want)
		}
	}
}
