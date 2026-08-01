package store

import "testing"

func TestParseTimeOKLayouts(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"2026-07-31T12:00:00Z", true},
		{"2026-07-31T12:00:00.123456789Z", true},
		{"2026-07-31T12:00:00+00:00", true},
		{"not-a-time", false},
		{"", false},
	}
	for _, tc := range cases {
		_, ok := parseTimeOK(tc.in)
		if ok != tc.want {
			t.Fatalf("parseTimeOK(%q)=%v want %v", tc.in, ok, tc.want)
		}
	}
}
