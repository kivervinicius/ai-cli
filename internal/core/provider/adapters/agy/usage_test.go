package agy

import (
	"math"
	"testing"
)

func TestLegacyQuotaUsesConsumedPercentage(t *testing.T) {
	for _, tc := range []struct{ consumed, remaining float64 }{{0, 100}, {90.15, 9.85}, {100, 0}, {-5, 100}, {120, 0}} {
		if got := agyRemaining(tc.consumed); math.Abs(got-tc.remaining) > 0.001 {
			t.Errorf("agyRemaining(%v)=%v, want %v", tc.consumed, got, tc.remaining)
		}
	}
	if got := agyUsed(90.15); got != 90.15 {
		t.Fatalf("agyUsed=%v", got)
	}
}
