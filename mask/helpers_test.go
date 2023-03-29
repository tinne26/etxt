package mask

// Helper functions for testing.

import "math"
import "math/rand"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

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

func randomTriangle(rng *rand.Rand, w, h int) sfnt.Segments {
	fsw, fsh := float64(w)*64, float64(h)*64
	segments := make([]sfnt.Segment, 0, 2)
	startX, startY := fixed.Int26_6(fsw/2), fixed.Int26_6(fsh/16)
	segments = moveTo(segments, startX, startY)
	segments = lineTo(segments, startX, fixed.Int26_6(fsh - fsh/16))
	cx, cy := fixed.Int26_6(rng.Float64()*fsw), fixed.Int26_6(rng.Float64()*fsh)
	segments = lineTo(segments, cx, cy)
	segments = lineTo(segments, startX, startY)
	return sfnt.Segments(segments)
}

func randomQuad(rng *rand.Rand, w, h int) sfnt.Segments {
	fsw, fsh := float64(w)*64, float64(h)*64
	segments := make([]sfnt.Segment, 0, 2)
	startX, startY := fixed.Int26_6(fsw/2), fixed.Int26_6(fsh/16)
	segments = moveTo(segments, startX, startY)
	segments = lineTo(segments, startX, fixed.Int26_6(fsh - fsh/16))
	cx, cy := fixed.Int26_6(rng.Float64()*fsw), fixed.Int26_6(rng.Float64()*fsh)
	segments = quadTo(segments, cx, cy, startX, startY)
	return sfnt.Segments(segments)
}

func randomSegments(rng *rand.Rand, lines, w, h int) sfnt.Segments {
	fsw, fsh := float64(w)*64, float64(h)*64
	var makeXY = func() (fixed.Int26_6, fixed.Int26_6) {
		return fixed.Int26_6(rng.Float64()*fsw), fixed.Int26_6(rng.Float64()*fsh)
	}

	// actually generate the segments
	startX, startY := makeXY()
	segments := make([]sfnt.Segment, 0, lines + 1)
	segments = moveTo(segments, startX, startY)
	for i := 0; i < lines; i++ {
		x, y := makeXY()
		switch rng.Intn(3) {
		case 0: // LineTo
			segments = lineTo(segments, x, y)
		case 1: // QuadTo
			cx, cy := makeXY()	
			segments = quadTo(segments, cx, cy, x, y)
		case 2: // CubeTo
			cx1, cy1 := makeXY()
			cx2, cy2 := makeXY()
			segments = cubeTo(segments, cx1, cy1, cx2, cy2, x, y)
		default:
			panic("unexpected case")
		}
	}
	segments = lineTo(segments, startX, startY)
	return sfnt.Segments(segments)
}

func polySegments(coords []float64) sfnt.Segments {
	if len(coords) % 2 != 0 {
		panic("number of coordinates must be even")
	}
	if len(coords) < 6 {
		panic("number of coordinates must be at least 6 (three points)")
	}

	var tofx = func(x float64) fixed.Int26_6 { return fixed.Int26_6(x*64) }
	segments := make([]sfnt.Segment, 0, len(coords)/2 + 1)
	segments = moveTo(segments, tofx(coords[0]), tofx(coords[1]))
	for i := 2; i < len(coords); i += 2 {
		x := coords[i + 0]
		y := coords[i + 1]
		segments = lineTo(segments, tofx(x), tofx(y))
	}
	segments = lineTo(segments, tofx(coords[0]), tofx(coords[1]))
	return sfnt.Segments(segments)
}

func newSegment(op sfnt.SegmentOp, x1, y1, x2, y2, x3, y3 fixed.Int26_6) sfnt.Segment {
	return sfnt.Segment { Op: op, Args: [3]fixed.Point26_6 {
			fixed.Point26_6{x1, y1}, fixed.Point26_6{x2, y2}, fixed.Point26_6{x3, y3},
		},
	}
}

func moveTo(segs []sfnt.Segment, x, y fixed.Int26_6) []sfnt.Segment {
	return append(segs, newSegment(sfnt.SegmentOpMoveTo, x, y, 0, 0, 0, 0))
}

func lineTo(segs []sfnt.Segment, x, y fixed.Int26_6) []sfnt.Segment {
	return append(segs, newSegment(sfnt.SegmentOpLineTo, x, y, 0, 0, 0, 0))
}

func quadTo(segs []sfnt.Segment, cx, cy, x, y fixed.Int26_6) []sfnt.Segment {
	return append(segs, newSegment(sfnt.SegmentOpQuadTo, cx, cy, x, y, 0, 0))
}

func cubeTo(segs []sfnt.Segment, cx1, cy1, cx2, cy2, x, y fixed.Int26_6) []sfnt.Segment {
	return append(segs, newSegment(sfnt.SegmentOpCubeTo, cx1, cy1, cx2, cy2, x, y))
}
