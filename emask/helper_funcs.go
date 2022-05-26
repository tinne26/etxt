package emask

import "image"
import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt/efixed"

// Given some glyph bounds and a fractional pixel position, it figures out
// what integer size must be used to fit the bounds, what normalization
// offset must be applied to keep the coordinates in the positive plane,
// and what final offset must be applied to the final mask to align its
// bounds to the glyph origin. This is used in NewContour functions.
func figureOutBounds(bounds fixed.Rectangle26_6, fract fixed.Point26_6) (image.Point, fixed.Point26_6, image.Point) {
	floorMinX := efixed.Floor(bounds.Min.X)
	floorMinY := efixed.Floor(bounds.Min.Y)
	var maskCorrection image.Point
	maskCorrection.X = int(floorMinX >> 6)
	maskCorrection.Y = int(floorMinY >> 6)

	var normOffset fixed.Point26_6
	normOffset.X = -floorMinX + fract.X
	normOffset.Y = -floorMinY + fract.Y
	width  := (bounds.Max.X + normOffset.X).Ceil()
	height := (bounds.Max.Y + normOffset.Y).Ceil()
	return image.Pt(width, height), normOffset, maskCorrection
}

// Around 9 times as fast as using a regular for loop.
// This can trivially be made generic, and can also be adapted
// to fill buffers with patterns (for example to fill
// images with a specific color).
func fastFillFloat64(buffer []float64, value float64) {
	if len(buffer) <= 24 { // no-copy case
		for i, _ := range buffer {
			buffer[i] = value
		}
	} else { // copy case
		for i, _ := range buffer[:16] {
			buffer[i] = value
		}
		for i := 16; i < len(buffer); i *= 2 {
			copy(buffer[i:], buffer[:i])
		}
	}
}

func fastFillUint8(buffer []uint8, value uint8) {
	if len(buffer) <= 24 { // no-copy case
		for i, _ := range buffer {
			buffer[i] = value
		}
	} else { // copy case
		for i, _ := range buffer[:16] {
			buffer[i] = value
		}
		for i := 16; i < len(buffer); i *= 2 {
			copy(buffer[i:], buffer[:i])
		}
	}
}

// linearly interpolate (ax, ay) and (bx, by) at the given t, which
// must be in [0, 1]
func lerp(ax, ay, bx, by float64, t float64) (float64, float64) {
	return interpolateAt(ax, bx, t), interpolateAt(ay, by, t)
}

// interpolate a and b at the given t, which must be in [0, 1]
func interpolateAt(a, b float64, t float64) float64 { return a + t*(b - a) }

// Given two points of a line, it returns its A, B and C
// coefficients from the form "Ax + By + C = 0".
func toLinearFormABC(ox, oy, fx, fy float64) (float64, float64, float64) {
	a, b, c := fy - oy, -(fx - ox), (fx - ox)*oy - (fy - oy)*ox
	return a, b, c
}

// given two line equations in Ax + By + C = 0 form, finds the
// intersecting point. If lines are parallel, the returned bool
// will be false and the returned point will always be (0, 0).
func intersectABC(a1, b1, c1, a2, b2, c2 float64) (float64, float64, bool) {
	// we have two line equations:
	// >> a1*x + b1*y + c1 = 0
	// >> a2*x + b2*y + c2 = 0
	// so we apply cramer's rule to solve the system:
	// x = (b2*c1 - b1*c2)/(b2*a1 - b1*a2)

	// first, though, make sure lines are not parallel...
	xdiv := b2*a1 - b1*a2
	if xdiv <= 0.00001 && xdiv >= -0.00001 { return 0, 0, false }

	// lines are not parallel, solve x and then y
	x := (b2*c1 - b1*c2)/xdiv
	if b1 != 0 {
		return x, (c1 - a1*x)/b1, true
	} else { // b2 != 0
		return x, (c2 - a2*x)/b2, true
	}
}

// given a line equation in Ax + By + C = 0 form and a point, finds
// the perpendicular ABC line equation that passes through the given
// point
func perpendicularABC(a, b, c, x, y float64) (float64, float64, float64) {
	// we have ax + by + c = 0, and we want to find dx + ey + f = 0...
	// we can use d = b, e = -a and f = -d*x - e*y
	d := b
	e := -a
	f := -d*x - e*y
	return d, e, f
}

func abs64(value float64) float64 {
	if value >= 0 { return value }
	return -value
}

func clampUnit64(value float64) float64 {
	if value <= 1.0 { return value }
	return 1.0
}
