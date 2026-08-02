package main

import "testing"

func TestSanitizeFileStem(t *testing.T) {
	cases := map[string]string{
		"checkout":     "checkout",
		"checkout-api": "checkout-api",
		"a/b c":        "a_b_c",
		"@@@":          "service",
		"":             "service",
	}
	for in, want := range cases {
		if got := sanitizeFileStem(in); got != want {
			t.Fatalf("sanitizeFileStem(%q)=%q want %q", in, got, want)
		}
	}
}
