//go:build gtxt

package emask

import "testing"

func TestBufferFillConvexQuad(t *testing.T) {
	tests := []struct{
		width int
		height int
		coords [8]float64 // ax, ay, bx, by, cx, cy, dx, dy (pairs in any order)
		out []float64
	}{
		{ // one-pixel sized quad
			width: 1, height: 1,
			coords: [8]float64{0, 0,   0, 1,   1, 0,   1, 1},
			out: []float64{1.0},
		},
		{ // two-pixel sized quad
			width: 2, height: 2,
			coords: [8]float64{0, 0,   0, 2,   2, 0,   2, 2},
			out: []float64{1.0, 1.0, 1.0, 1.0},
		},
		{ // half-pixel sized rect
			width: 1, height: 1,
			coords: [8]float64{0, 0,   0, 0.5,   1.0, 0,   1, 0.5},
			out: []float64{0.5},
		},
		{ // half-pixel sized rect (different orientation and shifted)
			width: 1, height: 2,
			coords: [8]float64{0.5, 0.5,   1, 0.5,   0.5, 1.5,   1, 1.5},
			out: []float64{0.25, 0.25},
		},
		{ // half-pixel triangle
			width: 1, height: 1,
			coords: [8]float64{0, 0,   0, 0,   1, 1,   0, 1},
			out: []float64{0.5},
		},
		{ // two-pixel triangle
			width: 2, height: 1,
			coords: [8]float64{0, 0,   0, 0,   2, 1,   0, 1},
			out: []float64{0.75, 0.25},
		},
		{ // trapeze
			width: 3, height: 3,
			coords: [8]float64{1, 2,   3, 2,   0, 3,   3, 3},
			out: []float64{0, 0, 0,   0, 0, 0,   0.5, 1, 1},
		},
		{ // flat top simple shape
			width: 1, height: 3,
			coords: [8]float64{0, 3,   1, 0,   1, 2,   0, 0},
			out: []float64{1, 1, 0.5},
		},
		{ // flat bottom simple shape
			width: 1, height: 3,
			coords: [8]float64{1, 0,   1, 3,   0, 3,   0, 1},
			out: []float64{0.5, 1, 1},
		},
		{ // trapeze with left triangle with flat top
			width: 3, height: 1,
			coords: [8]float64{1, 1,   0, 0,   3, 0,   3, 1},
			out: []float64{0.5, 1, 1},
		},
		{ // left-pointing isosceles
			width: 2, height: 1,
			coords: [8]float64{2, 0,   2, 1,   0.5, 0.5,   2, 0},
			out: []float64{0.08333333, 0.666666666},
		},
		{ // right-pointing isosceles
			width: 2, height: 1,
			coords: [8]float64{0, 0,   0, 1,   1.5, 0.5,   0, 0},
			out: []float64{0.666666666, 0.08333333},
		},
		{ // hard case, tilted trapeze /_/
			width: 3, height: 1,
			coords: [8]float64{0, 1,   2, 0,   3, 0,   1, 1},
			out: []float64{0.25, 0.5, 0.25},
		},
		{ // hard case, tilted trapeze \_\ (symmetric to previous test)
			width: 3, height: 1,
			coords: [8]float64{0, 0,   2, 1,   3, 1,   1, 0},
			out: []float64{0.25, 0.5, 0.25},
		},
		{ // diamond shape without flat top or bottom
			width: 2, height: 2,
			coords: [8]float64{1, 0,   0, 1,   2, 1,   1, 2},
			out: []float64{0.5, 0.5, 0.5, 0.5},
		},
		{ // unaligned diamond shape
			width: 2, height: 2,
			coords: [8]float64{1, 0.5,   0.5, 1,   1.5, 1,   1, 1.5},
			out: []float64{0.125, 0.125, 0.125, 0.125},
		},
		{ // diagonal shape crossing 4 pixels
			width: 2, height: 2,
			coords: [8]float64{0.15, 0.85,   0.85, 0.15,   1.15, 1.85,   1.85, 1.15},
			out: []float64{0.455, 0.245, 0.245, 0.455},
		},
	}

	buffer := &buffer{}
	for n, test := range tests {
		if len(test.out) != test.width*test.height {
			t.Fatalf("malformed test %d", n)
		}
		buffer.Resize(test.width, test.height)
		ax, ay := test.coords[0], test.coords[1]
		bx, by := test.coords[2], test.coords[3]
		cx, cy := test.coords[4], test.coords[5]
		dx, dy := test.coords[6], test.coords[7]
		buffer.FillConvexQuad(ax, ay, bx, by, cx, cy, dx, dy)
		if !similarFloat64Slices(test.out, buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.coords, test.out, buffer.Values)
		}
	}
}

func TestOutlinerCut(t *testing.T) {
	tests := []struct{
		thickness float64
		width int
		height int
		coords []float64 // the first pair is the first MoveTo, and
		                 // remaining pairs are used for LineTo commands
		out []float64
	}{
		// ---- single lines, axis-aligned ----
		{ // horizontal line
			thickness: 1.0, width: 2, height: 1,
			coords: []float64{0.0, 0.5, 2.0, 0.5},
			out: []float64{1.0, 1.0},
		},
		{ // vertical line
			thickness: 1.0, width: 1, height: 2,
			coords: []float64{0.5, 0.0, 0.5, 2.0},
			out: []float64{1.0, 1.0},
		},
		{ // wider horizontal line
			thickness: 2.0, width: 3, height: 2,
			coords: []float64{0.0, 1.0, 3.0, 1.0},
			out: []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
		},
		{ // wider vertical line
			thickness: 2.0, width: 2, height: 3,
			coords: []float64{1.0, 0.0, 1.0, 3.0},
			out: []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
		},

		// ---- single diagonal lines ----
		{ // 45 deg line
			thickness: 0.2, width: 1, height: 1,
			coords: []float64{0.1, 0.1, 0.9, 0.9},
			out: []float64{0.22627},
		},
		{ // some tilted line going through two vertical pixels
			thickness: 0.2, width: 1, height: 2,
			coords: []float64{0.25, 0.5, 0.75, 1.5},
			out: []float64{0.1118033, 0.1118033}, // total = 0.2236
		},
		{ // like the previous, but flipped 90 degs
			thickness: 0.2, width: 2, height: 1,
			coords: []float64{0.5, 0.25, 1.5, 0.75},
			out: []float64{0.1118033, 0.1118033}, // total = 0.2236
		},
		{ // diagonal going through 4 pixels
			thickness: 0.5, width: 2, height: 2,
			coords: []float64{0.5, 0.5, 1.5, 1.5},
			out: []float64{0.29105, 0.0625, 0.0625, 0.29105},
		},

		// ---- basic shapes ----
		{ // L-like shape
			thickness: 1.0, width: 3, height: 3,
			coords: []float64{0.5, 0.0, 0.5, 2.5, 3.0, 2.5},
			out: []float64{1.0, 0.0, 0.0, 1.0, 0.0, 0.0, 1.0, 1.0, 1.0},
		},
		{ // C-like shape
			thickness: 1.0, width: 3, height: 3,
			coords: []float64{3.0, 0.5, 0.5, 0.5, 0.5, 2.5, 3.0, 2.5},
			out: []float64{1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0, 1.0},
		},

		// ---- single lines from multiple segments ----
		{ // horizontal line lousily drawn in two segments
			thickness: 1.0, width: 2, height: 1,
			coords: []float64{0.0, 0.5,   1.0, 0.5,   2.0, 0.5},
			out: []float64{1.0, 1.0},
		},
		{ // vertical line lousily drawn in two segments
			thickness: 1.0, width: 1, height: 2,
			coords: []float64{0.5, 0.0,   0.5, 1.0,   0.5, 2.0},
			out: []float64{1.0, 1.0},
		},
		{ // diagonal line lousily drawn in two segments
			thickness: 1.0, width: 2, height: 2,
			coords: []float64{0.5, 0.5,   1.0, 1.0,   1.5, 1.5},
			out: []float64{0.457, 0.25, 0.25, 0.457},
		},
	}

	outliner := &outliner{}
	outliner.CurveSegmenter.SetThreshold(1/1024)
	outliner.CurveSegmenter.SetMaxSplits(8)
	for n, test := range tests {
		if len(test.out) != test.width*test.height {
			t.Fatalf("malformed test %d", n)
		}
		outliner.Buffer.Resize(test.width, test.height)
		outliner.MoveTo(test.coords[0], test.coords[1])
		outliner.SetThickness(test.thickness)
		for i := 2; i < len(test.coords); i += 2 {
			outliner.LineTo(test.coords[i], test.coords[i + 1])
		}
		outliner.CutPath()

		// clamp values bigger than 1 for proper result comparisons
		// for i := 0; i < len(outliner.Buffer.Values); i++ {
		// 	if outliner.Buffer.Values[i] > 1.0 {
		// 		outliner.Buffer.Values[i] = 1.0
		// 	}
		// }

		if !similarFloat64Slices(test.out, outliner.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.coords, test.out, outliner.Buffer.Values)
		}
	}
}
