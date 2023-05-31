package mask

import "image"

import "github.com/tinne26/etxt/fract"

// Given the glyph bounds and an origin position indicating the subpixel
// positioning (only lowest bits will be taken into account), it returns
// the bounding integer width and heights, the normalization offset to be
// applied to keep the coordinates in the positive plane, and the final
// offset to be applied on the final mask to align its bounds to the glyph
// origin. This is used in Rasterize() functions.
func figureOutBounds(bounds fract.Rect, origin fract.Point) (int, int, fract.Point, image.Point) {
	floorMinX := bounds.Min.X.Floor()
	floorMinY := bounds.Min.Y.Floor()
	var maskCorrection image.Point
	maskCorrection.X = floorMinX.ToIntFloor()
	maskCorrection.Y = floorMinY.ToIntFloor()

	var normOffset fract.Point
	normOffset.X = -floorMinX + origin.X.FractShift()
	normOffset.Y = -floorMinY + origin.Y.FractShift()
	width  := (bounds.Max.X + normOffset.X).Ceil()
	height := (bounds.Max.Y + normOffset.Y).Ceil()
	return width.ToIntFloor(), height.ToIntFloor(), normOffset, maskCorrection
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

func abs64(value float64) float64 {
	if value >= 0 { return value }
	return -value
}

func clampUnit64(value float64) float64 {
	if value <= 1.0 { return value }
	return 1.0
}
