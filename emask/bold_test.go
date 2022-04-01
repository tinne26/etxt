//go:build gtxt

package emask

import "testing"

func TestFauxBoldWhole(t *testing.T) {
	tests := []struct{
		w int
		in []uint8
		out []uint8
	}{
		{w: 0, in: []uint8{0, 100,  0, 100, 0}, out: []uint8{0, 100,   0, 100,   0}},
		{w: 1, in: []uint8{0, 100,  0, 100, 0}, out: []uint8{0, 100, 100, 100, 100}},
		{w: 1, in: []uint8{0, 100, 50, 100, 0}, out: []uint8{0, 100, 100, 100, 100}},
		{w: 1, in: []uint8{0, 9, 0, 0, 9, 0, 0}, out: []uint8{0, 9, 9, 0, 9, 9, 0}},
		{w: 2, in: []uint8{5, 0, 0, 0, 5, 0, 0}, out: []uint8{5, 5, 5, 0, 5, 5, 5}},
		{w: 2, in: []uint8{0, 5, 6, 4, 0, 0}, out: []uint8{0, 5, 6, 6, 6, 4}},
		{w: 1, in: []uint8{9, 8, 7, 8, 9, 0}, out: []uint8{9, 9, 8, 8, 9, 9}},
		{w: 1, in: []uint8{9, 1, 1, 2, 9, 0}, out: []uint8{9, 9, 1, 2, 9, 9}},
		{w: 3, // real tests be like...
			in : []uint8{0, 0, 0, 0, 0, 200, 255, 235, 250, 230,  97,   2,   0,  0, 0, 0, 0, 78, 249, 255, 252, 111, 251, 251, 148,  19,   0,   0,  0, 0, 0, 28, 214, 255, 255, 120,   0,   0,   0, 0, 0},
			out: []uint8{0, 0, 0, 0, 0, 200, 255, 255, 255, 255, 250, 250, 230, 97, 2, 0, 0, 78, 249, 255, 255, 255, 255, 252, 251, 251, 251, 148, 19, 0, 0, 28, 214, 255, 255, 255, 255, 255, 120, 0, 0},
		},
		{w: 8,
			in : []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 250, 255, 255, 255, 152,  74,  47,  69, 102, 169, 240, 254, 170, 102, 245, 252,  91,   0,   0,   0,   0,   0,   0,   0,  0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			out: []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 250, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 254, 254, 254, 254, 254, 254, 254, 254, 252, 252, 252, 252, 91, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	rast := FauxRasterizer{}
	for i, test := range tests {
		rast.SetExtraWidth(float64(test.w))
		out := make([]uint8, len(test.in))
		for n, value := range test.in { out[n] = value }
		rast.applyRowExtraWidth(out, out, 0, 999)
		if !eqSliceUint8(out, test.out) {
			t.Fatalf("test#%d: in %v (+%d), expected %v, got %v", i, test.in, test.w, test.out, out)
		}
	}
}

func TestFauxBoldFract(t *testing.T) {
	tests := []struct{
		w float64
		in []uint8
		out []uint8
	}{
		{w: 0.5, in: []uint8{0, 100,  0, 100, 0}, out: []uint8{0, 100,  50, 100,  50}},
		{w: 0.5, in: []uint8{0, 100, 50,  25, 0}, out: []uint8{0, 100,  75,  37,  12}},
		{w: 0.5, in: []uint8{0, 100, 50, 100, 0}, out: []uint8{0, 100,  75, 100,  50}},
		{w: 0.5, in: []uint8{0, 100,  0,  50, 0}, out: []uint8{0, 100,  50,  50,  25}},
		{w: 0.125, in: []uint8{100, 0,  0,  100, 0}, out: []uint8{100, 12,  0,  100,  12}},
		{w: 0.625, in: []uint8{100, 0,  0,  100, 0}, out: []uint8{100, 62,  0,  100,  62}},
		{w: 1.5, in: []uint8{100, 0,  0,  100, 0, 0}, out: []uint8{100, 100, 50,  100,  100, 50}},
		{w: 1.5, in: []uint8{0, 50, 0, 50, 0, 100, 0, 0}, out: []uint8{0, 50, 50, 50, 50, 100, 100, 50}},
		{w: 8.5, in: []uint8{2, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, out: []uint8{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 0, 0}},
	}

	rast := FauxRasterizer{}
	for i, test := range tests {
		rast.SetExtraWidth(test.w)
		out := make([]uint8, len(test.in))
		for n, value := range test.in { out[n] = value }
		rast.applyRowExtraWidth(out, out, 0, 999)
		if !eqSliceUint8(out, test.out) {
			t.Fatalf("test#%d: in %v (+%0.1f), expected %v, got %v", i, test.in, test.w, test.out, out)
		}
	}
}

func eqSliceUint8(a, b []uint8) bool {
	if len(a) != len(b) { return false }
	for i, valueA := range a {
		if valueA != b[i] { return false }
	}
	return true
}
