//go:build test || bench

package emask

// Helper functions used during testing and/or benchmarking.

import "math"
import "math/rand"
import "golang.org/x/image/math/fixed"

func randomShape(rng *rand.Rand, lines, w, h int) Shape {
	fsw, fsh := float64(w)*64, float64(h)*64
	var makeXY = func() (fixed.Int26_6, fixed.Int26_6) {
		return fixed.Int26_6(rng.Float64()*fsw), fixed.Int26_6(rng.Float64()*fsh)
	}
	startX, startY := makeXY()

	shape := NewShape(lines + 1)
	shape.MoveToFract(startX, startY)
	for i := 0; i < lines; i++ {
		x, y := makeXY()
		switch rng.Intn(3) {
		case 0: // LineTo
			shape.LineToFract(x, y)
		case 1: // QuadTo
			cx, cy := makeXY()
			shape.QuadToFract(cx, cy, x, y)
		case 2: // CubeTo
			cx1, cy1 := makeXY()
			cx2, cy2 := makeXY()
			shape.CubeToFract(cx1, cy1, cx2, cy2, x, y)
		}
	}
	shape.LineToFract(startX, startY)
	return shape
}

func similarFloat64Slices(a []float64, b []float64) bool {
	if len(a) != len(b) { return false }
	for i, valueA := range a {
		if valueA != b[i] {
			diff := math.Abs(valueA - b[i])
			if diff > 0.001 { return false } // allow small precision differences
		}
	}
	return true
}
