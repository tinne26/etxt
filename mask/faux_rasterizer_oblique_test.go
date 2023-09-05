package mask

import "testing"

import "github.com/tinne26/etxt/fract"

func TestFauxMinusOneSkew(t *testing.T) {
	// minus one skew is represented with a zero, and it's easy to mess up initialization
	rast := FauxRasterizer{}
	rast.SetSkewFactor(-1)
	skew := rast.GetSkewFactor()
	if skew != -1.0 {
		t.Fatalf("expected skew to be %f, got %f", -1.0, skew)
	}
}

func TestFauxOblique(t *testing.T) {
	tests := []struct{
		skew float32
		in []float64
		out []uint8
	}{
		{
			skew: 1.0,
			in: []float64{ 0, -1, /**/ 1, -1, /**/ 1, 1, /**/ 0, 1 },
			out: []uint8{ 0, 128, 128, /**/ 128, 128, 0 },
		},
		{
			skew: -1.0,
			in: []float64{ 0, -1, /**/ 1, -1, /**/ 1, 1, /**/ 0, 1 },
			out: []uint8{ 128, 128, 0, /**/ 0, 128, 128 },
		},
		{
			skew: 1.0,
			in: []float64{ 0, -2, /**/ 1, -2, /**/ 1, 2, /**/ 0, 2 },
			out: []uint8{ 0, 0, 0, 128, 128, /**/ 0, 0, 128, 128, 0, /**/ 0, 128, 128, 0, 0, /**/ 128, 128, 0, 0, 0 },
		},
	}

	rast := FauxRasterizer{}
	for i, test := range tests {
		rast.SetSkewFactor(test.skew)
		skew := rast.GetSkewFactor()
		if skew != test.skew {
			t.Fatalf("imprecise internal skew, expected %f, got %f", test.skew, skew)
		}
		segments := polySegments(test.in)
		mask, err := rast.Rasterize(segments, fract.Point{})
		if err != nil { t.Fatalf("unexpected error: %s", err) }
		if !eqSliceUint8(mask.Pix, test.out) {
			exportTest("oblique_fail.png", mask)
			t.Fatalf("test #%d mistmatch: expected %v, got %v", i, test.out, mask.Pix)
		}
	}
}
