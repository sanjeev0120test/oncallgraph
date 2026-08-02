package ask

import (
	"testing"
	"time"
)

func TestHumanDurationAndWindowLabel(t *testing.T) {
	if got := humanDuration(60 * time.Minute); got != "60m" {
		t.Fatalf("humanDuration(60m)=%q", got)
	}
	if got := humanDuration(2 * time.Hour); got != "2h" {
		t.Fatalf("humanDuration(2h)=%q", got)
	}
	if got := windowLabel(10 * time.Minute); got != "10m (changes 60m)" {
		t.Fatalf("windowLabel(10m)=%q", got)
	}
	if got := windowLabel(time.Hour); got != "60m" {
		t.Fatalf("windowLabel(1h)=%q", got)
	}
}

func TestChangeLookbackFloor(t *testing.T) {
	if got := ChangeLookback(5 * time.Minute); got != 60*time.Minute {
		t.Fatalf("ChangeLookback(5m)=%v want 60m", got)
	}
	if got := ChangeLookback(2 * time.Hour); got != 2*time.Hour {
		t.Fatalf("ChangeLookback(2h)=%v", got)
	}
}
