package fract

import "testing"
import "math"

func TestFromFloat64(t *testing.T) {
	nzr := math.Copysign(0, -1) // negative zero
	tests := []struct {
		in   float64
		low  Unit
		high Unit
	}{
		{in: 0.0, low: 0, high: 0},
		{in: nzr, low: 0, high: 0},
		{in: 1.0, low: 64, high: 64},
		{in: -1.0, low: -64, high: -64},
		{in: 0.5, low: 32, high: 32},
		{in: 3.14, low: 201, high: 201},
		{in: -3.14, low: -201, high: -201},
		{in: 8.33, low: 533, high: 533},
		{in: 8.3359375, low: 533, high: 534},
		{in: 8.3359374, low: 533, high: 533},
		{in: 8.3359376, low: 534, high: 534},
		{in: -8.3359375, low: -534, high: -533},
		{in: -8.3359374, low: -533, high: -533},
		{in: -8.3359376, low: -534, high: -534},
		{in: MaxFloat64, low: MaxUnit, high: MaxUnit},
		{in: MinFloat64, low: MinUnit, high: MinUnit},
	}

	for i, test := range tests {
		low, high := FromFloat64Down(test.in), FromFloat64Up(test.in)
		if low != test.low || high != test.high {
			str := "test #%d: in (%f), expected outs %d (%f) and %d (%f), but got %d (%f) and %d (%f)"
			t.Fatalf(str, i, test.in, test.low, test.low.ToFloat64(), test.high, test.high.ToFloat64(), low, low.ToFloat64(), high, high.ToFloat64())
		}
		away := FromFloat64(test.in)
		if test.in >= 0 {
			if away != test.high {
				str := "test #%d: expected FromFloat64(%f) to return %d (%f), but got %d (%f) instead"
				t.Fatalf(str, i, test.in, test.high, test.high.ToFloat64(), away, away.ToFloat64())
			}
		} else {
			if away != test.low {
				str := "test #%d: expected FromFloat64(%f) to return %d (%f), but got %d (%f) instead"
				t.Fatalf(str, i, test.in, test.low, test.low.ToFloat64(), away, away.ToFloat64())
			}
		}
	}
}
