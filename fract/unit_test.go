//go:build test

package fract

import "testing"
import "math"

func TestToFloat64(t *testing.T) {
	tests := []struct {
		in  Unit
		out float64
	}{
		{0, 0}, {64, 1}, {32, 0.5}, {-32, -0.5},
		{1, 1.0/64.0}, {2, 2.0/64.0}, {-2, -2.0/64.0},
		{3, 3.0/64.0}, {63, 63.0/64.0}, {96, 1.5},
		{MinUnit, MinFloat64}, {MaxUnit, MaxFloat64},
	}

	for i, test := range tests {
		out := test.in.ToFloat64()
		if out != test.out {
			str := "test #%d: in %d expected out %f, but got %f"
			t.Fatalf(str, i, test.in, test.out, out)
		}
	}
}

func TestIsWhole(t *testing.T) {
	tests := []struct {
		in  Unit
		out bool
	}{
		{0, true}, {1, false}, {-1, false}, {-32, false}, {32, false},
		{64, true}, {-64, true}, {-128, true}, {128, true}, {-95, false},
		{18, false},
	}

	for i, test := range tests {
		out := test.in.IsWhole()
		if out != test.out {
			str := "test #%d: in %d (%f) expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, out)
		}
	}
}

func TestFract(t *testing.T) {
	tests := []struct {
		in  Unit
		out Unit
	}{
		{0, 0}, {32, 32}, {64, 0}, {31, 31}, {63, 63},
		{127, 63}, {65, 1}, {96, 32},
		{-32, -32}, {-1, -1}, {-31, -31}, {-33, -33},
		{-64, 0}, {-128, 0}, {-65, -1},
	}

	for i, test := range tests {
		out := test.in.Fract()
		if out != test.out {
			str := "test #%d: in %d (%f) expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, out)
		}
		_, fract := math.Modf(test.in.ToFloat64())
		if fract != out.ToFloat64() { panic("bad test") }
	}
}

func TestToIntFloor(t *testing.T) {
	tests := []struct {
		in  Unit
		out int
	}{
		{   0,  0}, { 32,  0}, {  96,  1}, {  64,  1},
		{  65,  1}, { 63,  0}, { -64, -1}, { -65, -2},
		{ -63, -1}, {-96, -2}, {-127, -2}, {-128, -2},
		{-129, -3}, {127,  1}, { 129,  2},
	}

	for i, test := range tests {
		out := test.in.ToIntFloor()
		if out != test.out {
			str := "test #%d: in %d (%f) expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, out)
		}
	}
}

func TestToIntCeil(t *testing.T) {
	tests := []struct {
		in  Unit
		out int
	}{
		{   0,  0}, { 32,  1}, {  96,  2}, {  64,  1},
		{  65,  2}, { 63,  1}, { -64, -1}, { -65, -1},
		{ -63,  0}, {-96, -1}, {-127, -1}, {-128, -2},
		{-129, -2}, {127,  2}, { 129,  3},
	}

	for i, test := range tests {
		out := test.in.ToIntCeil()
		if out != test.out {
			str := "test #%d: in %d (%f) expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, out)
		}
	}
}

func TestToIntToward(t *testing.T) {
	tests := []struct {
		in  Unit
		ref int
		out int
	}{
		{0, +1, 0}, {31, +1, 1}, {32, +1, 1}, {33, +1, 1}, {96, +1, 1}, {64, +1, 1},
		{0, -1, 0}, {31, -1, 0}, {32, -1, 0}, {33, -1, 0}, {96, -1, 1}, {64, -1, 1},
		{-31, +1,  0}, {-32, +1,  0}, {-33, +1,  0}, {-96, +1, -1}, {-64, +1, -1},
		{-31, -1, -1}, {-32, -1, -1}, {-33, -1, -1}, {-96, -1, -1}, {-64, -1, -1},
		{95, 2, 2}, {-95, -2, -2}, {-128, -2, -2}, {-128, 500, -2}, {-127, 500, -1},
		{-129, 500, -2},
	}

	for i, test := range tests {
		out := test.in.ToIntToward(test.ref)
		if out != test.out {
			str := "test #%d: in %d (%f), towards %d, expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.ref, test.out, out)
		}
	}
}

func TestToIntAway(t *testing.T) {
	tests := []struct {
		in  Unit
		ref int
		out int
	}{
		{0, +1, 0}, {31, +1, 0}, {32, +1, 0}, {33, +1, 0}, {96, +1, 2}, {64, +1, 1},
		{0, -1, 0}, {31, -1, 1}, {32, -1, 1}, {33, -1, 1}, {96, -1, 2}, {64, -1, 1},
		{-31, +1, -1}, {-32, +1, -1}, {-33, +1, -1}, {-96, +1, -2}, {-64, +1, -1},
		{-31, -1,  0}, {-32, -1,  0}, {-33, -1,  0}, {-96, -1, -2}, {-64, -1, -1},
		{95, 2, 1}, {-95, -2, -1}, {-128, -2, -2}, {-128, 500, -2}, {-127, 500, -2},
		{-129, 500, -3}, {65, -1, 2},
	}

	for i, test := range tests {
		out := test.in.ToIntAway(test.ref)
		if out != test.out {
			str := "test #%d: in %d (%f), away %d, expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.ref, test.out, out)
		}
	}
}

func TestToIntHalfDown(t *testing.T) {
	tests := []struct {
		in  Unit
		out int
	}{
		{0, 0}, {64, 1}, {-64, -1}, {128, 2}, {-128, -2},
		{32, 0}, {31, 0}, {33, 1}, {63, 1}, {64 + 32, 1}, {64 + 33, 2}, {64 + 31, 1},
		{-1, 0}, {-32, -1}, {-31, 0}, {-33, -1}, {-65, -1},
		{-64 - 33, -2}, {-64 - 32, -2}, {-64 - 31, -1},
	}

	for i, test := range tests {
		out := test.in.ToIntHalfDown()
		if out != test.out {
			str := "test #%d: in %d (%f), expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, out)
		}
	}
}

func TestToIntHalfUp(t *testing.T) {
	tests := []struct {
		in  Unit
		out int
	}{
		{0, 0}, {64, 1}, {-64, -1}, {128, 2}, {-128, -2},
		{32, 1}, {31, 0}, {33, 1}, {63, 1}, {64 + 32, 2}, {64 + 33, 2}, {64 + 31, 1},
		{-1, 0}, {-32, 0}, {-31, 0}, {-33, -1}, {-65, -1},
		{-64 - 33, -2}, {-64 - 32, -1}, {-64 - 31, -1},
	}

	for i, test := range tests {
		out := test.in.ToIntHalfUp()
		if out != test.out {
			str := "test #%d: in %d (%f), expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, out)
		}
		if out != test.in.ToInt() {
			str := "test #%d: ToIntHalfUp() != ToInt() (with in %d (%f))"
			t.Fatalf(str, i, test.in, test.in.ToFloat64())
		}
	}
}

func TestToIntHalfToward(t *testing.T) {
	tests := []struct {
		in  Unit
		ref int
		out int
	}{
		{0,  42, 0}, {64,  42, 1}, {-64,  42, -1}, {128,  42, 2}, {-128,  42, -2},
		{0, -42, 0}, {64, -42, 1}, {-64, -42, -1}, {128, -42, 2}, {-128, -42, -2},
		{32, 42, 1}, {31, 42, 0}, {33, 42, 1}, {32, 0, 0}, {31, 0, 0}, {33, 0, 1},
		{-32, -42, -1}, {-31, -42, 0}, {-33, -42, -1}, {-32, 0, 0}, {-31, 0, 0}, {-33, 0, -1},
	}

	for i, test := range tests {
		out := test.in.ToIntHalfToward(test.ref)
		if out != test.out {
			str := "test #%d: in %d (%f), toward %d, expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.ref, test.out, out)
		}
	}
}

func TestToIntHalfAway(t *testing.T) {
	tests := []struct {
		in  Unit
		ref int
		out int
	}{
		{0,  42, 0}, {64,  42, 1}, {-64,  42, -1}, {128,  42, 2}, {-128,  42, -2},
		{0, -42, 0}, {64, -42, 1}, {-64, -42, -1}, {128, -42, 2}, {-128, -42, -2},
		{32, 42, 0}, {31, 42, 0}, {33, 42, 1}, {32, 0, 1}, {31, 0, 0}, {33, 0, 1},
		{-32, -42, 0}, {-31, -42, 0}, {-33, -42, -1}, {-32, 0, -1}, {-31, 0, 0}, {-33, 0, -1},
	}

	for i, test := range tests {
		out := test.in.ToIntHalfAway(test.ref)
		if out != test.out {
			str := "test #%d: in %d (%f), away %d, expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.ref, test.out, out)
		}
	}
}

func TestFloor(t *testing.T) {
	tests := []struct {
		in  Unit
		out Unit
	}{
		{   0,   0}, { 32,  0}, {  96,  64}, {  64, 64},
		{  65,  64}, { 63,  0}, { -64, -64}, { -65, -128},
		{ -63, -64}, {-96, -128}, {-127, -128}, {-128, -128},
		{-129, -192}, {127,  64}, { 129,  128},
	}

	for i, test := range tests {
		out := test.in.Floor()
		if out != test.out {
			str := "test #%d: in %d (%f) expected out %d (%f), but got %d (%f)"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, test.out.ToFloat64(), out, out.ToFloat64())
		}
	}
}

func TestCeil(t *testing.T) {
	tests := []struct {
		in  Unit
		out Unit
	}{
		{   0,   0}, { 32, 64}, { 96, 128}, { 64,  64},
		{  65, 128}, { 63, 64}, {-64, -64}, {-65, -64},
		{ -63,    0}, {-96, -64}, {-127, -64}, {-128, -128},
		{-129, -128}, {127, 128}, { 129, 192},
	}

	for i, test := range tests {
		out := test.in.Ceil()
		if out != test.out {
			str := "test #%d: in %d (%f) expected out %d (%f), but got %d (%f)"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, test.out.ToFloat64(), out, out.ToFloat64())
		}
	}
}

func TestToward(t *testing.T) {
	tests := []struct {
		in  Unit
		ref int
		out Unit
	}{
		{0, +1, 0}, {31, +1, 64}, {32, +1, 64}, {33, +1, 64}, {96, +1, 64}, {64, +1, 64},
		{0, -1, 0}, {31, -1, 0}, {32, -1, 0}, {33, -1, 0}, {96, -1, 64}, {64, -1, 64},
		{-31, +1,  0}, {-32, +1,  0}, {-33, +1,  0}, {-96, +1, -64}, {-64, +1, -64},
		{-31, -1, -64}, {-32, -1, -64}, {-33, -1, -64}, {-96, -1, -64}, {-64, -1, -64},
		{95, 2, 128}, {-95, -2, -128}, {-128, -2, -128}, {-128, 500, -128}, {-127, 500, -64},
		{-129, 500, -128},
	}

	for i, test := range tests {
		out := test.in.Toward(test.ref)
		if out != test.out {
			str := "test #%d: in %d (%f), towards %d, expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.ref, test.out, out)
		}
	}
}

func TestAway(t *testing.T) {
	tests := []struct {
		in  Unit
		ref int
		out Unit
	}{
		{0, +1, 0}, {31, +1, 0}, {32, +1, 0}, {33, +1, 0}, {96, +1, 128}, {64, +1, 64},
		{0, -1, 0}, {31, -1, 64}, {32, -1, 64}, {33, -1, 64}, {96, -1, 128}, {64, -1, 64},
		{-31, +1, -64}, {-32, +1, -64}, {-33, +1, -64}, {-96, +1, -128}, {-64, +1, -64},
		{-31, -1,  0}, {-32, -1,  0}, {-33, -1,  0}, {-96, -1, -128}, {-64, -1, -64},
		{95, 2, 64}, {-95, -2, -64}, {-128, -2, -128}, {-128, 500, -128}, {-127, 500, -128},
		{-129, 500, -192}, {65, -1, 128},
	}

	for i, test := range tests {
		out := test.in.Away(test.ref)
		if out != test.out {
			str := "test #%d: in %d (%f), away %d, expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.ref, test.out, out)
		}
	}
}

func TestHalfDown(t *testing.T) {
	tests := []struct {
		in  Unit
		out Unit
	}{
		{0, 0}, {64, 64}, {-64, -64}, {128, 128}, {-128, -128},
		{32, 0}, {31, 0}, {33, 64}, {63, 64}, {64 + 32, 64}, {64 + 33, 128}, {64 + 31, 64},
		{-1, 0}, {-32, -64}, {-31, 0}, {-33, -64}, {-65, -64},
		{-64 - 33, -128}, {-64 - 32, -128}, {-64 - 31, -64},
	}

	for i, test := range tests {
		out := test.in.HalfDown()
		if out != test.out {
			str := "test #%d: in %d (%f), expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, out)
		}
	}
}

func TestHalfUp(t *testing.T) {
	tests := []struct {
		in  Unit
		out Unit
	}{
		{0, 0}, {64, 64}, {-64, -64}, {128, 128}, {-128, -128},
		{32, 64}, {31, 0}, {33, 64}, {63, 64}, {64 + 32, 128}, {64 + 33, 128}, {64 + 31, 64},
		{-1, 0}, {-32, 0}, {-31, 0}, {-33, -64}, {-65, -64},
		{-64 - 33, -128}, {-64 - 32, -64}, {-64 - 31, -64},
	}

	for i, test := range tests {
		out := test.in.HalfUp()
		if out != test.out {
			str := "test #%d: in %d (%f), expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.out, out)
		}
	}
}

func TestHalfToward(t *testing.T) {
	tests := []struct {
		in  Unit
		ref int
		out Unit
	}{
		{0,  42, 0}, {64,  42, 64}, {-64,  42, -64}, {128,  42, 128}, {-128,  42, -128},
		{0, -42, 0}, {64, -42, 64}, {-64, -42, -64}, {128, -42, 128}, {-128, -42, -128},
		{32, 42, 64}, {31, 42, 0}, {33, 42, 64}, {32, 0, 0}, {31, 0, 0}, {33, 0, 64},
		{-32, -42, -64}, {-31, -42, 0}, {-33, -42, -64}, {-32, 0, 0}, {-31, 0, 0}, {-33, 0, -64},
	}

	for i, test := range tests {
		out := test.in.HalfToward(test.ref)
		if out != test.out {
			str := "test #%d: in %d (%f), toward %d, expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.ref, test.out, out)
		}
	}
}

func TestHalfAway(t *testing.T) {
	tests := []struct {
		in  Unit
		ref int
		out Unit
	}{
		{0,  42, 0}, {64,  42, 64}, {-64,  42, -64}, {128,  42, 128}, {-128,  42, -128},
		{0, -42, 0}, {64, -42, 64}, {-64, -42, -64}, {128, -42, 128}, {-128, -42, -128},
		{32, 42, 0}, {31, 42, 0}, {33, 42, 64}, {32, 0, 64}, {31, 0, 0}, {33, 0, 64},
		{-32, -42, 0}, {-31, -42, 0}, {-33, -42, -64}, {-32, 0, -64}, {-31, 0, 0}, {-33, 0, -64},
	}

	for i, test := range tests {
		out := test.in.HalfAway(test.ref)
		if out != test.out {
			str := "test #%d: in %d (%f), away %d, expected out %d, but got %d"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.ref, test.out, out)
		}
	}
}

func TestMul(t *testing.T) {
	tests := []struct {
		in  Unit
		mul Unit
		out float64
	}{
		{0, 0, 0}, {0, 35, 0}, {-1125, 0, 0},
		{64, 182, 182/64.0}, {222, 64, 222/64.0}, 
		{64, 64, 1}, {64, -64, -1}, {64, 128, 2}, {128, -64, -2},
		{64, 32, 0.5}, {-64, -32, 0.5}, {32, -64, -0.5}, 
		{32, 32, 1/4.0}, {-32, -32, 1/4.0}, {32, -32, -1/4.0}, {-32, 32, -1/4.0},
		{64*3, 32, 1.5}, {64*2 + 2, 32, 1.0 + 1/64.0}, {64*3, -32, -1.5}, {-64*2 - 2, 32, -1.0 - 1/64.0},

		// some of the tricky inexact cases where the +32 makes a difference
		{-95, 31, -0.718750}, {-94, 30, -0.687500}, {-93, 29, -0.656250},
		{-92, 28, -0.625000}, {-91, 27, -0.593750}, {-87, 23, -0.484375},
		{-84, 20, -0.406250}, {-82, 18, -0.359375}, {-78, 14, -0.265625},
	}

	for i, test := range tests {
		out := test.in.Mul(test.mul).ToFloat64()
		if out != test.out {
			str := "test #%d: in %d (%f) * %d (%f), expected out %f, but got %f"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.mul, test.mul.ToFloat64(), test.out, out)
		}
	}
}

func TestQuantizeUp(t *testing.T) {
	tests := []struct {
		step Unit
		in  Unit
		out Unit
	}{
		{step: 1, in: 26, out: 26}, {step: 1, in: 27, out: 27}, {step: 1, in: 45, out: 45},
		{step: 2, in: 26, out: 26}, {step: 2, in: 27, out: 28}, {step: 2, in: 45, out: 46},
		{step: 3, in: 26, out: 27}, {step: 3, in: 27, out: 27}, {step: 3, in: 45, out: 45},
		{step: 4, in: 26, out: 28}, {step: 4, in: 27, out: 28}, {step: 4, in: 45, out: 44},
		{step: 5, in: 62, out: 60}, {step: 5, in: 63, out: 64}, {step: 5, in: 59, out: 60},
		{step: 5, in: 67, out: 69}, {step: 5, in: 66, out: 64},
		
		// full unit by unit consistency test sequence
		{3, 67, 67}, {3, 66, 67}, {3, 65, 64}, {3, 64, 64}, {3, 63, 63}, {3, 62, 63}, {3, 61, 60},
		{3, 60, 60}, {3, 59, 60}, {3, 58, 57}, {3, 57, 57},
		{3, 3, 3}, {3, 2, 3}, {3, 1, 0}, {3, 0, 0}, {3, -1, -1}, {3, -2, -1}, {3, -3, -4}, {3, -4, -4},
		{3, -5, -4}, {3, -6, -7}, {3, -7, -7},
		{3, -64, -64}, {3, -63, -64}, {3, -62, -61}, {3, -61, -61}, {3, -65, -65},

		// even tie
		{2, 66, 66}, {2, 65, 66}, {2, 64, 64}, {2, 63, 64}, {2, 62, 62}, {2, 61, 62}, 

		{step: 1, in: -26, out: -26}, {step: 1, in: -27, out: -27}, {step: 1, in: -45, out: -45},
		{step: 2, in: -26, out: -26}, {step: 2, in: -27, out: -26}, {step: 2, in: -45, out: -44},
		{step: 3, in: -26, out: -25}, {step: 3, in: -27, out: -28}, {step: 3, in: -45, out: -46},
		{step: 4, in: -26, out: -24}, {step: 4, in: -27, out: -28}, {step: 4, in: -45, out: -44},
		{step: 5, in: -62, out: -64}, {step: 5, in: -63, out: -64}, {step: 5, in: -59, out: -59},
	}

	for i, test := range tests {
		out := test.in.QuantizeUp(test.step)
		if out != test.out {
			str := "test #%d: in %d (%f), step %d, expected out %f (%d), but got %f (%d)"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.step, test.out.ToFloat64(), test.out, out.ToFloat64(), out)
		}
		mod := (out & 0x3F) % test.step
		if mod != 0 {
			str := "test #%d: in %d (%f), step %d, out = %d (%f), fractBits %d %% step %d == %d (!= 0)"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.step, out, out.ToFloat64(), (out & 0x3F), test.step, mod)
		}
	}
}

func TestQuantizeDown(t *testing.T) {
	tests := []struct {
		step Unit
		in  Unit
		out Unit
	}{
		{step: 1, in: 26, out: 26}, {step: 1, in: 27, out: 27}, {step: 1, in: 45, out: 45},
		{step: 2, in: 26, out: 26}, {step: 2, in: 27, out: 26}, {step: 2, in: 45, out: 44},
		{step: 3, in: 26, out: 27}, {step: 3, in: 27, out: 27}, {step: 3, in: 45, out: 45},
		{step: 4, in: 26, out: 24}, {step: 4, in: 27, out: 28}, {step: 4, in: 45, out: 44},
		{step: 5, in: 62, out: 60}, {step: 5, in: 63, out: 64}, {step: 5, in: 59, out: 60},
		{step: 5, in: 67, out: 69}, {step: 5, in: 66, out: 64},
		
		// full unit by unit consistency test sequence
		{3, 67, 67}, {3, 66, 67}, {3, 65, 64}, {3, 64, 64}, {3, 63, 63}, {3, 62, 63}, {3, 61, 60},
		{3, 60, 60}, {3, 59, 60}, {3, 58, 57}, {3, 57, 57},
		{3, 3, 3}, {3, 2, 3}, {3, 1, 0}, {3, 0, 0}, {3, -1, -1}, {3, -2, -1}, {3, -3, -4}, {3, -4, -4},
		{3, -5, -4}, {3, -6, -7}, {3, -7, -7},
		{3, -64, -64}, {3, -63, -64}, {3, -62, -61}, {3, -61, -61}, {3, -65, -65},

		// even tie
		{2, 66, 66}, {2, 65, 64}, {2, 64, 64}, {2, 63, 62}, {2, 62, 62}, {2, 61, 60}, 

		{step: 1, in: -26, out: -26}, {step: 1, in: -27, out: -27}, {step: 1, in: -45, out: -45},
		{step: 2, in: -26, out: -26}, {step: 2, in: -27, out: -28}, {step: 2, in: -45, out: -46},
		{step: 3, in: -26, out: -25}, {step: 3, in: -27, out: -28}, {step: 3, in: -45, out: -46},
		{step: 4, in: -26, out: -28}, {step: 4, in: -27, out: -28}, {step: 4, in: -45, out: -44},
		{step: 5, in: -62, out: -64}, {step: 5, in: -63, out: -64}, {step: 5, in: -59, out: -59},
	}

	for i, test := range tests {
		out := test.in.QuantizeDown(test.step)
		if out != test.out {
			str := "test #%d: in %d (%f), step %d, expected out %f (%d), but got %f (%d)"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.step, test.out.ToFloat64(), test.out, out.ToFloat64(), out)
		}
		mod := (out & 0x3F) % test.step
		if mod != 0 {
			str := "test #%d: in %d (%f), step %d, out = %d (%f), fractBits %d %% step %d == %d (!= 0)"
			t.Fatalf(str, i, test.in, test.in.ToFloat64(), test.step, out, out.ToFloat64(), (out & 0x3F), test.step, mod)
		}
	}
}
