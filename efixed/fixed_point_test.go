//go:build test

package efixed

import "testing"
import "math/rand"
import "time"
import "math"

import "golang.org/x/image/math/fixed"

func TestFromFloat(t *testing.T) {
	tests := []struct {
		in   float64
		low  fixed.Int26_6
		high fixed.Int26_6
	}{
		{in:   0.0, low:    0, high:    0},
		{in:   1.0, low:   64, high:   64},
		{in:  -1.0, low:  -64, high:  -64},
		{in:   0.5, low:   32, high:   32},
		{in:  3.14, low:  201, high:  201},
		{in: -3.14, low: -201, high: -201},
		{in:  8.33, low:  533, high:  533},
		{in: 8.3359375, low: 533, high: 534},
		{in: 8.3359374, low: 533, high: 533},
		{in: 8.3359376, low: 534, high: 534},
		{in: -8.3359375, low: -534, high: -533},
		{in: -8.3359374, low: -533, high: -533},
		{in: -8.3359376, low: -534, high: -534},
		{in:  33554432, low: 2147483647, high: 2147483647},
		{in: -33554432, low: -2147483648, high: -2147483648},
		{in: -33554432.015625, low: -2147483648, high: -2147483648},
	}

	for i, test := range tests {
		low, high := FromFloat64(test.in)
		if low != test.low || high != test.high {
			str := "test #%d: in (%.6f) expected outs %d (%.6f) and %d (%.6f), but got %d (%.6f) and %d (%.6f)"
			t.Fatalf(str, i, test.in, test.low, ToFloat64(test.low), test.high, ToFloat64(test.high), low, ToFloat64(low), high, ToFloat64(high))
		}
	}

	// test expected panics
	for _, value := range []float64{ math.NaN(), math.Inf(1), math.Inf(-1), -33554432.015626, 33554432.0000001 } {
		func() {
			defer func(){ _ = recover() }()
			FromFloat64(value)
			t.Fatalf("expected %f to panic", value)
		}()
	}
}

func TestHalfRounding(t *testing.T) {
	halfUpTests := []struct {
		in fixed.Int26_6
		out fixed.Int26_6
	}{
		{in: 0, out: 0}, {in: 32, out: 64}, {in: 31, out: 0},
		{in: -32, out: 0}, {in: -31, out: 0}, {in: -33, out: -64},
		{in: FromFloat64RoundToZero(3.1416), out: FromFloat64RoundToZero(3.00)},
		{in: FromFloat64RoundToZero(-3.1416), out: FromFloat64RoundToZero(-3.00)},
		{in: FromFloat64RoundToZero(-3.9)  , out: FromFloat64RoundToZero(-4.00)},
		{in: FromFloat64RoundToZero(3.9)   , out: FromFloat64RoundToZero(4.00)},
		{in: FromFloat64RoundToZero(112.4) , out: FromFloat64RoundToZero(112.00)},
		{in: FromFloat64RoundToZero(112.5) , out: FromFloat64RoundToZero(113.00)},
		{in: FromFloat64RoundToZero(112.6) , out: FromFloat64RoundToZero(113.00)},
		{in: FromFloat64RoundToZero(-112.4), out: FromFloat64RoundToZero(-112.00)},
		{in: FromFloat64RoundToZero(-112.5), out: FromFloat64RoundToZero(-112.00)},
		{in: FromFloat64RoundToZero(-112.6), out: FromFloat64RoundToZero(-113.00)},
	}

	// round half up tests
	for i, test := range halfUpTests {
		got := RoundHalfUp(test.in)
		if got != test.out {
			str := "test #%d: in %d (%.6f) expected out %d (%.6f) but got %d (%.6f)"
			t.Fatalf(str, i, test.in, ToFloat64(test.in), test.out, ToFloat64(test.out), got, ToFloat64(got))
		}
	}

	// round half down tests (half up tests but with flipped sign)
	for i, test := range halfUpTests {
		got := RoundHalfDown(-test.in)
		if got != -test.out {
			str := "test #%d: in %d (%.6f) expected out %d (%.6f) but got %d (%.6f)"
			t.Fatalf(str, i, -test.in, ToFloat64(-test.in), -test.out, ToFloat64(-test.out), got, ToFloat64(got))
		}
	}

	// consistency test between round half up and round half down
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		value := fixed.Int26_6(rand.Int31n(1 << 26) - (1 << 25))
		up   := RoundHalfUp(-value)
		down := RoundHalfDown(value)
		if up != -down {
			str := "rand test: in %d (%.6f) caused discordant output, up with -in returned %d (%.6f), and down returned %d (%.6f)"
			t.Fatalf(str, i, value, ToFloat64(value), up, ToFloat64(up), down, ToFloat64(down))
		}
	}
}

func TestIntHalf(t *testing.T) {
	halfUpTests := []struct {
		in fixed.Int26_6
		out int
	}{
		{in: 0, out: 0}, {in: 32, out: 1}, {in: 31, out: 0},
		{in: -32, out: 0}, {in: -31, out: 0}, {in: -33, out: -1},
		{in: FromFloat64RoundToZero(3.1416), out: 3},
		{in: FromFloat64RoundToZero(-3.1416), out: -3},
		{in: FromFloat64RoundToZero(-3.9), out: -4},
		{in: FromFloat64RoundToZero(3.9), out: 4},
		{in: FromFloat64RoundToZero(112.4), out: 112},
		{in: FromFloat64RoundToZero(112.5), out: 113},
		{in: FromFloat64RoundToZero(112.6), out: 113},
		{in: FromFloat64RoundToZero(-112.4), out: -112},
		{in: FromFloat64RoundToZero(-112.5), out: -112},
		{in: FromFloat64RoundToZero(-112.6), out: -113},
	}

	// round half up tests
	for i, test := range halfUpTests {
		got := ToIntHalfUp(test.in)
		if got != test.out {
			str := "test #%d: in %d (%.6f) expected out %d but got %d"
			t.Fatalf(str, i, test.in, ToFloat64(test.in), test.out, got)
		}
	}

	// round half down tests (half up tests but with flipped sign)
	for i, test := range halfUpTests {
		got := ToIntHalfDown(-test.in)
		if got != -test.out {
			str := "test #%d: in %d (%.6f) expected out %d but got %d"
			t.Fatalf(str, i, test.in, ToFloat64(test.in), test.out, got)
		}
	}

	// consistency test between int half up and int half down
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		value := fixed.Int26_6(rand.Int31n(1 << 26) - (1 << 25))
		up   := ToIntHalfUp(value)
		down := ToIntHalfDown(-value)
		if -up != down {
			str := "rand test: in %d (%.6f) caused discordant output, up with in returned %d, and down with -in returned %d"
			t.Fatalf(str, i, value, ToFloat64(value), up, down)
		}
	}
}

func TestQuantizeFract(t *testing.T) {
	upTests := []struct {
		step uint8
		in  fixed.Int26_6
		out fixed.Int26_6
	}{
		{step: 1, in: 26, out: 26}, {step: 1, in: 27, out: 27}, {step: 1, in: 45, out: 45},
		{step: 2, in: 26, out: 26}, {step: 2, in: 27, out: 28}, {step: 2, in: 45, out: 46},
		{step: 3, in: 26, out: 27}, {step: 3, in: 27, out: 27}, {step: 3, in: 45, out: 45},
		{step: 4, in: 26, out: 28}, {step: 4, in: 27, out: 28}, {step: 4, in: 45, out: 44},
		{step: 5, in: 62, out: 60}, {step: 5, in: 63, out: 64}, {step: 5, in: 59, out: 60},
		{step: 5, in: 67, out: 69}, {step: 5, in: 66, out: 64},
		
		{step: 1, in: -26, out: -26}, {step: 1, in: -27, out: -27}, {step: 1, in: -45, out: -45},
		{step: 2, in: -26, out: -26}, {step: 2, in: -27, out: -26}, {step: 2, in: -45, out: -44},
		{step: 3, in: -26, out: -27}, {step: 3, in: -27, out: -27}, {step: 3, in: -45, out: -45},
		{step: 4, in: -26, out: -24}, {step: 4, in: -27, out: -28}, {step: 4, in: -45, out: -44},
		{step: 5, in: -62, out: -60}, {step: 5, in: -63, out: -64}, {step: 5, in: -59, out: -60},
		{step: 5, in: -67, out: -69}, {step: 5, in: -66, out: -64},
	}

	for i, test := range upTests {
		got := QuantizeFractUp(test.in, test.step)
		if got != test.out {
			str := "test #%d: with step %d, input %.6f (%d) expected out %.6f (%d) but got %.6f (%d"
			t.Fatalf(str, i, test.step, ToFloat64(test.in), test.in, ToFloat64(test.out), test.out, ToFloat64(got), got)
		}
	}

	// ---- down tests ----

	downTests := []struct {
		step uint8
		in  fixed.Int26_6
		out fixed.Int26_6
	}{
		{step: 1, in: 26, out: 26}, {step: 1, in: 27, out: 27}, {step: 1, in: 45, out: 45},
		{step: 2, in: 26, out: 26}, {step: 2, in: 27, out: 26}, {step: 2, in: 45, out: 44},
		{step: 3, in: 26, out: 27}, {step: 3, in: 27, out: 27}, {step: 3, in: 45, out: 45},
		{step: 4, in: 26, out: 24}, {step: 4, in: 27, out: 28}, {step: 4, in: 45, out: 44},
		{step: 5, in: 62, out: 60}, {step: 5, in: 63, out: 64}, {step: 5, in: 59, out: 60},
		{step: 5, in: 67, out: 69}, {step: 5, in: 66, out: 64},
		
		{step: 1, in: -26, out: -26}, {step: 1, in: -27, out: -27}, {step: 1, in: -45, out: -45},
		{step: 2, in: -26, out: -26}, {step: 2, in: -27, out: -28}, {step: 2, in: -45, out: -46},
		{step: 3, in: -26, out: -27}, {step: 3, in: -27, out: -27}, {step: 3, in: -45, out: -45},
		{step: 4, in: -26, out: -28}, {step: 4, in: -27, out: -28}, {step: 4, in: -45, out: -44},
		{step: 5, in: -62, out: -60}, {step: 5, in: -63, out: -64}, {step: 5, in: -59, out: -60},
		{step: 5, in: -67, out: -69}, {step: 5, in: -66, out: -64},
	}

	for i, test := range downTests {
		got := QuantizeFractDown(test.in, test.step)
		if got != test.out {
			str := "test #%d: with step %d, input %.6f (%d) expected out %.6f (%d) but got %.6f (%d"
			t.Fatalf(str, i, test.step, ToFloat64(test.in), test.in, ToFloat64(test.out), test.out, ToFloat64(got), got)
		}
	}
}
