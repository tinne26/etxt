package emask

import "math"
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

// If we had two line equations like this:
// >> a1*x + b1*y = c1
// >> a2*x + b2*y = c2
// We would apply cramer's rule to solve the system:
// >> x = (b2*c1 - b1*c2)/(b2*a1 - b1*a2)
// This function solves this system, but assuming c1 and c2 have
// a negative sign (ax + by + c = 0), and taking a precomputed
// xdiv = (b2*a1 - b1*a2) value. xdiv can only be zero if one
// of the lines is vertical and the other horizontal.
func shortCramer(xdiv, a1, b1, c1, a2, b2, c2 float64) (float64, float64) {
	// ... and we have to account for perpendicular cases
	if xdiv == 0 {
		if a1 == 0 {
			if b2 == 0 { return -c2/a2, -c1/b1 }
			panic("parallel lines")
		} else if b1 == 0 {
			if a2 == 0 { return -c1/a1, -c2/b2 }
			panic("parallel lines")
		}
		panic("parallel lines")
	}

	// actual application of cramer's rule
	x := (b2*-c1 - b1*-c2)/xdiv
	if b1 != 0 { return x, (-c1 - a1*x)/b1 }
	return x, (-c2 - a2*x)/b2
}

// given a line equation in Ax + By + C = 0 form and a point, finds
// the perpendicular ABC line equation that passes through the given
// point. the C is not in the parameters because it's not necessary
func perpendicularABC(a, b, x, y float64) (float64, float64, float64) {
	// we have ax + by + c = 0, and we want to find dx + ey + f = 0...
	// we can use d = b, e = -a and f = -d*x - e*y
	d := b
	e := -a
	f := -d*x - e*y
	return d, e, f
}

// given a line equation in the form Ax + By + C = 0, it returns
// C1 and C2 such that two new line equations can be created that
// are parallel to the original line, but at distance 'dist' from it
func parallelsAtDist(a, b, c float64, dist float64) (float64, float64) {
	var c1, c2 float64
	if a == 0 { // horizontal line
		y := -c/b
		c1 = -(y + dist)*b
		c2 = -(y - dist)*b
	} else if b == 0 { // vertical line
		x := -c/a
		c1 = -(x + dist)*a
		c2 = -(x - dist)*a
	} else {
		// We use the formula for the distance between a point and a line:
		// >> dist = |ax + by + c|/sqrt(a*a + b*b)
		// We assume x = 0 and find the two y possible values.
		// We use the points (0, y1) and (0, y2) to find the new c1 and c2.
		f := dist*math.Sqrt(a*a + b*b)
		y1 := (-c + f)/b
		y2 := (-c - f)/b
		c1 = -b*y1
		c2 = -b*y2
	}
	return c1, c2
}

// Given two paths, each defined by a pair of ax + by + c = 0 equations
// where only the c coefficient takes two different values, and the
// central starting point of the first path and the central ending point
// of the second, the intersecting quad inner and outer points are computed.
// TODO: this may be better off as a function from one segment to another,
//       to avoid all the argument passing...
func intersectPaths(aa, ab, ac1, ac2, ba, bb, bc1, bc2, pax, pay, pbx, pby float64) (float64, float64, float64, float64) {
	// find 4 intersection points
	// TODO: see how to handle NaNs here
	xdiv := bb*aa - ab*ba
	x11, y11 := shortCramer(xdiv, aa, ab, ac1, ba, bb, bc1)
	x12, y12 := shortCramer(xdiv, aa, ab, ac1, ba, bb, bc2)
	x21, y21 := shortCramer(xdiv, aa, ab, ac2, ba, bb, bc1)
	x22, y22 := shortCramer(xdiv, aa, ab, ac2, ba, bb, bc2)

	// find central intersection point
	jx := (x11 + x12 + x21 + x22)/4
	jy := (y11 + y12 + y21 + y22)/4

	// get linear equations from central point to line start/end points
	jaa, jab, jac := toLinearFormABC(pax, pay, jx, jy)
	jba, jbb, jbc := toLinearFormABC(pbx, pby, jx, jy)

	// determine which point among the 4 intersection points falls
	// at each side of ja & jb line equations to determine inner and
	// outer vertices
	boa := (jab*pby > -jaa*pbx - jac)
	bob := (jbb*pay > -jba*pax - jbc)
	var inX, inY, outX, outY float64
	for _, p := range []struct{x, y float64}{{x11, y11}, {x12, y12}, {x21, y21}, {x22, y22}} {
		jaCmp := (jab*p.y > -jaa*p.x - jac)
		jbCmp := (jbb*p.y > -jba*p.x - jbc)
		if (boa == jaCmp) == (bob == jbCmp) {
			if boa == jaCmp {
				inX, inY = p.x, p.y
			} else {
				outX, outY = p.x, p.y
			}
		}
	}
	return outX, outY, inX, inY
}

func abs64(value float64) float64 {
	if value >= 0 { return value }
	return -value
}

func clampUnit64(value float64) float64 {
	if value <= 1.0 { return value }
	return 1.0
}
