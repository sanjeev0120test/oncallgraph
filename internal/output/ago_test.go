package output

import (
	"testing"
	"time"
)

func TestAgoHandlesSkewAndMissing(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if got := ago(now, now.Add(time.Minute)); got != "0s" {
		t.Fatalf("future then: %q", got)
	}
	if got := ago(now, time.Time{}); got != "?" {
		t.Fatalf("zero then: %q", got)
	}
	if got := ago(now, now.Add(-90*time.Second)); got != "1m" {
		t.Fatalf("90s: %q", got)
	}
}
