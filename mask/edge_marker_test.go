package mask

import "testing"

func TestEdgeAlignedRects(t *testing.T) {
	tests := []struct {
		in  []float64 // one moveTo + many lineTo coords
		out []float64 // output buffer (5x4)
	}{
		{ // small square
			in: []float64{1, 1, 1, 3, 3, 3, 3, 1, 1, 1},
			out: []float64 {
				0, 0, 0,  0, 0,
				0, 1, 0, -1, 0,
				0, 1, 0, -1, 0,
				0, 0, 0,  0, 0,
			},
		},
		{ // full canvas rect
			in: []float64{0, 0, 0, 4, 5, 4, 5, 0, 0, 0},
			out: []float64 {
				1, 0, 0, 0, 0,
				1, 0, 0, 0, 0,
				1, 0, 0, 0, 0,
				1, 0, 0, 0, 0,
			},
		},
		{ // large canvas square
			in: []float64{0, 0, 0, 4, 4, 4, 4, 0, 0, 0},
			out: []float64 {
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
			},
		},
		{ // large outside rect
			in: []float64{-5, 0, -5, 4, 4, 4, 4, 0, -5, 0},
			out: []float64 {
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
			},
		},
		{ // smaller outside rect
			in: []float64{-5, 1, -5, 3, 4, 3, 4, 1, -5, 1},
			out: []float64 {
				0, 0, 0, 0,  0,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				0, 0, 0, 0,  0,
			},
		},
		{ // shape
			in: []float64{0, 1, 0, 3, 1, 3, 1, 4, 2, 4, 2, 2, 3, 2, 3, 1, 0, 1},
			out: []float64 {
				0, 0,  0,  0,  0,
				1, 0,  0, -1,  0,
				1, 0, -1,  0,  0,
				0, 1, -1,  0,  0,
			},
		},
	}

	emarker := edgeMarker{}
	emarker.Buffer.Resize(5, 4)
	for n, test := range tests {
		emarker.MoveTo(test.in[0], test.in[1])
		for i := 2; i < len(test.in); i += 2 {
			emarker.LineTo(test.in[i], test.in[i + 1])
		}
		if !similarFloat64Slices(test.out, emarker.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.in, test.out, emarker.Buffer.Values)
		}
		emarker.Buffer.Clear()
	}
}

func TestEdgeAlignedTriangles(t *testing.T) {
	tests := []struct {
		in  []float64 // one moveTo + many lineTo coords
		out []float64 // output buffer (4x4)
	}{
		{ // right triangle
			in: []float64{0, 0, 0, 4, 4, 4, 0, 0},
			out: []float64 {
				0.5, -0.5,  0.0,  0.0,
				1.0, -0.5, -0.5,  0.0,
				1.0,  0.0, -0.5, -0.5,
				1.0,  0.0,    0, -0.5,
			},
		},
		{ // right triangle, alternative orientation (both shape and def. direction)
			in: []float64{0, 0, 4, 0, 0, 4, 0, 0},
			out: []float64 {
				-1.0, 0.0, 0.0, 0.5,
				-1.0, 0.0, 0.5, 0.5,
				-1.0, 0.5, 0.5, 0.0,
				-0.5, 0.5, 0.0, 0.0,
			},
		},
		{ // triangle with wide base
			in: []float64{0, 0, 2, 4, 4, 0, 0, 0},
			out: []float64 {
				0.75, 0.25, 0.0, -0.25,
				0.25, 0.75, 0.0, -0.75,
				0.00, 0.75, 0.0, -0.75,
				0.00, 0.25, 0.0, -0.25,
			},
		},
		{ // slimmer right triangle, tricky fill proportions
			in: []float64{0, 0, 0, 4, 3, 4, 0, 0},
			out: []float64 {
				 3/ 8., -3 / 8.,    0.0 ,   0.0 ,
				23/24., -19/24., -4 /24.,   0.0 ,
				  1.0 ,  -1/ 6., -19/24., -1/24.,
				  1.0 ,    0.0 , -3 / 8., -5/ 8.,
			},
		},
		{ // slimmer right triangle, alternative orientation
			in: []float64{0, 0, 3, 0, 0, 4, 0, 0},
			out: []float64 {
				  -1.0 ,   0.0 , 3 / 8., 5/ 8.,
				  -1.0 ,  1/ 6., 19/24., 1/24.,
				-23/24., 19/24., 4 /24.,  0.0 ,
				 -3/ 8., 3 / 8.,   0.0 ,  0.0 ,
			},
		},
	}

	emarker := edgeMarker{}
	emarker.Buffer.Resize(4, 4)
	for n, test := range tests {
		emarker.MoveTo(test.in[0], test.in[1])
		for i := 2; i < len(test.in); i += 2 {
			emarker.LineTo(test.in[i], test.in[i + 1])
		}
		if !similarFloat64Slices(test.out, emarker.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.in, test.out, emarker.Buffer.Values)
		}
		emarker.Buffer.Clear()
	}
}

func TestEdgeUnalignedRects(t *testing.T) {
	tests := []struct {
		in  []float64 // one moveTo + many lineTo coords
		out []float64 // output buffer (4x4)
	}{
		{ // shifted square
			in: []float64{0.5, 0, 0.5, 4, 2.5, 4, 2.5, 0, 0.5, 0},
			out: []float64 {
				0.5, 0.5, -0.5, -0.5,
				0.5, 0.5, -0.5, -0.5,
				0.5, 0.5, -0.5, -0.5,
				0.5, 0.5, -0.5, -0.5,
			},
		},
		{ // shifted square, in both axes
			in: []float64{0.5, 0.5, 0.5, 3.5, 2.5, 3.5, 2.5, 0.5, 0.5, 0.5},
			out: []float64 {
				0.25, 0.25, -0.25, -0.25,
				0.50, 0.50, -0.50, -0.50,
				0.50, 0.50, -0.50, -0.50,
				0.25, 0.25, -0.25, -0.25,
			},
		},
		{ // slightly shifted square
			in: []float64{0.2, 0, 0.2, 4, 2.2, 4, 2.2, 0, 0.2, 0},
			out: []float64 {
				0.8, 0.2, -0.8, -0.2,
				0.8, 0.2, -0.8, -0.2,
				0.8, 0.2, -0.8, -0.2,
				0.8, 0.2, -0.8, -0.2,
			},
		},
		{ // significantly shifted square
			in: []float64{0.8, 0, 0.8, 4, 2.8, 4, 2.8, 0, 0.8, 0},
			out: []float64 {
				0.2, 0.8, -0.2, -0.8,
				0.2, 0.8, -0.2, -0.8,
				0.2, 0.8, -0.2, -0.8,
				0.2, 0.8, -0.2, -0.8,
			},
		},
	}

	emarker := edgeMarker{}
	emarker.Buffer.Resize(4, 4)
	for n, test := range tests {
		emarker.MoveTo(test.in[0], test.in[1])
		for i := 2; i < len(test.in); i += 2 {
			emarker.LineTo(test.in[i], test.in[i + 1])
		}
		if !similarFloat64Slices(test.out, emarker.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.in, test.out, emarker.Buffer.Values)
		}
		emarker.Buffer.Clear()
	}
}

func TestEdgeSinglePixel(t *testing.T) {
	tests := []struct {
		in  []float64 // one moveTo + many lineTo coords
		out []float64 // output buffer (5x4)
	}{
		{ // pix square
			in: []float64{0, 0, 0, 1, 1, 1, 1, 0, 0, 0},
			out: []float64 { 1, -1 },
		},
		{ // half-pix square
			in: []float64{0.5, 0, 0.5, 1, 1, 1, 1, 0, 0.5, 0},
			out: []float64 { 0.5, -0.5 },
		},
	}

	emarker := edgeMarker{}
	emarker.Buffer.Resize(2, 1)
	for n, test := range tests {
		emarker.MoveTo(test.in[0], test.in[1])
		for i := 2; i < len(test.in); i += 2 {
			emarker.LineTo(test.in[i], test.in[i + 1])
		}
		if !similarFloat64Slices(test.out, emarker.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.in, test.out, emarker.Buffer.Values)
		}
		emarker.Buffer.Clear()
	}
}
