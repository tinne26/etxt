package emask

import "math"

// edgeMarker implements a simplified and more readable version of the
// algorithm used by vector.Rasterizer, providing access to the result of
// the first rasterization step. The final accumulation process can be
// seen in EdgeMarkerRasterizer's Rasterize() method too.
//
// The algorithms are documented and contextualized here:
// >> https://github.com/tinne26/etxt/blob/main/docs/rasterize-outlines.md
//
// The zero value is usable, but curves will not be segmented smoothly.
// You should manually SetCurveThreshold() and SetMaxCurveSplits() to
// the desired values.
type edgeMarker struct {
	x float64 // current drawing point position
	y float64 // current drawing point position
	Width int  // canvas width, in pixels
	Height int // canvas height, in pixels
	Buffer []float64 // be very careful if you touch this directly.
	// ^ Negative values are used for counter-clockwise segments,
	//   positive values are used for clockwise segments.

	curveThreshold float32 // threshold to decide if a segment approximates
	                       // a bézier curve well enough or we should split
	maxCurveSplits uint8   // a cutoff for curve segmentation
}

// Sets a new Width and Height and resizes the underlying buffer if
// necessary. The buffer contents are cleared too.
func (self *edgeMarker) Resize(width, height int) {
	if width <= 0 || height <= 0 { panic("width or height <= 0") }
	self.Width  = width
	self.Height = height
	totalLen := width*height
	if len(self.Buffer) == totalLen {
		// nothing
	} else if len(self.Buffer) > totalLen {
		self.Buffer = self.Buffer[0 : totalLen]
	} else { // len(self.Buffer) < totalLen
		if cap(self.Buffer) >= totalLen {
			self.Buffer = self.Buffer[0 : totalLen]
		} else {
			self.Buffer = make([]float64, totalLen)
			return // stop before ClearBuffer()
		}
	}

	self.ClearBuffer()
}

// Fills the internal buffer with zeros.
func (self *edgeMarker) ClearBuffer() {
	fastFillFloat64(self.Buffer, 0)
}

// Moves the current position to the given coordinates.
func (self *edgeMarker) MoveTo(x, y float64) {
	self.x = x
	self.y = y
}

// Creates a straight boundary from the current position to the given
// target and moves the current position to the new one.
//
// While the 'LineTo' name is used to stay consistent with other similar
// interfaces, don't think in terms of "drawing lines"; we are defining
// the boundaries of an outline.
func (self *edgeMarker) LineTo(x, y float64) {
	// This method is the core of edgeMarker. Additional context
	// and explanations are available at docs/rasterize-outlines.md.

	// changes in y equal or below this threshold are considered 0.
	// this is bigger than 0 in order to account for floating point
	// division unstability
	const HorizontalityThreshold = 0.000001

	// make sure to set the new current position at the end
	defer self.MoveTo(x, y)

	// get position increases in both axes
	deltaX := x - self.x
	deltaY := y - self.y

	// if the y doesn't change, we are marking an horizontal boundary...
	// but horizontal boundaries don't have to be marked
	if math.Abs(deltaY) <= HorizontalityThreshold { return }
	xAdvancePerY := deltaX/deltaY

	// mark boundaries for every pixel that we pass through
	for {
		// get next whole position in the current direction
		nextX := nextWholeCoord(self.x, deltaX)
		nextY := nextWholeCoord(self.y, deltaY)

		// check if we reached targets and clamp
		atHorzTarget := hasReachedTarget(nextX, x, deltaX)
		atVertTarget := hasReachedTarget(nextY, y, deltaY)
		if atHorzTarget { nextX = x }
		if atVertTarget { nextY = y }

		// find distances to next coords
		horzAdvance := nextX - self.x
		vertAdvance := nextY - self.y

		// determine which whole coordinate we reach first
		// with the current line direction and position
		altHorzAdvance := xAdvancePerY*vertAdvance
		if math.Abs(altHorzAdvance) <= math.Abs(horzAdvance) {
			// reach vertical whole coord first
			horzAdvance = altHorzAdvance
		} else {
			// reach horizontal whole coord first
			// (notice that here xAdvancePerY can't be 0)
			vertAdvance = horzAdvance/xAdvancePerY
		}

		// mark the boundary segment traversing the vertical axis at
		// the current pixel
		self.markBoundary(self.x, self.y, horzAdvance, vertAdvance)

		// update current position *(note 1)
		self.x += horzAdvance
		self.y += vertAdvance

		// return if we reached the final position
		if atHorzTarget && atVertTarget { return }
	}

	// *note 1: precision won't cause trouble here. Since we calculated
	//          horizontal and vertical advances from a whole position,
	//          once we re-add that difference we will reach the whole
	//          position again if that's what has to happen.
	// *note 2: notice that there are many optimizations that are not
	//          being applied here. for example, vector.Rasterizer treats
	//          the buffer as a continuous space instead of using rows
	//          as boundaries, which allows for faster buffer accumulation
	//          at a later stage (also using SIMD instructions). Many
	//          other small optimizations are possible if you don't mind
	//          hurting readability. using float32 instead of float64 can
	//          also speed up things. allocations could also be reduced.
	//          see also CubeTo for curve optimizations.
}

// Creates a boundary from the current position to the given target
// as a quadratic Bézier curve through the given control point and
// moves the current position to the new one.
func (self *edgeMarker) QuadTo(ctrlX, ctrlY, fx, fy float64) {
	self.recursiveQuadTo(ctrlX, ctrlY, fx, fy, 0)
}

func (self *edgeMarker) recursiveQuadTo(ctrlX, ctrlY, fx, fy float64, depth uint8) {
	if depth >= self.maxCurveSplits || self.withinThreshold(self.x, self.y, fx, fy, ctrlX, ctrlY) {
		self.LineTo(fx, fy)
		return
	}

	ocx, ocy := lerp(self.x, self.y, ctrlX, ctrlY, 0.5) // origin to control
	cfx, cfy := lerp(ctrlX, ctrlY, fx, fy, 0.5)         // control to end
	ix , iy  := lerp(ocx, ocy, cfx, cfy, 0.5)           // interpolated point
	self.recursiveQuadTo(ocx, ocy, ix, iy, depth + 1)
	self.recursiveQuadTo(cfx, cfy, fx, fy, depth + 1)
}

// Creates a boundary from the current position to the given target
// as a cubic Bézier curve through the given control points and
// moves the current position to the new one.
func (self *edgeMarker) CubeTo(cx1, cy1, cx2, cy2, fx, fy float64) {
	// performance notes: reducing to 1 split can cut rasterization
	// times in 15%. vector.Rasterizer's approach is also more
	// direct and could slightly cut rasterization time.
	self.recursiveCubeTo(cx1, cy1, cx2, cy2, fx, fy, 0)
}

func (self *edgeMarker) recursiveCubeTo(cx1, cy1, cx2, cy2, fx, fy float64, depth uint8) {
	if depth >= self.maxCurveSplits || (self.withinThreshold(self.x, self.y, cx2, cy2, cx1, cy1) && self.withinThreshold(cx1, cy1, fx, fy, cx2, cy2)) {
		self.LineTo(fx, fy)
		return
	}

	oc1x , oc1y  := lerp(self.x, self.y, cx1, cy1, 0.5) // origin to control 1
	c1c2x, c1c2y := lerp(cx1, cy1, cx2, cy2, 0.5)       // control 1 to control 2
	c2fx , c2fy  := lerp(cx2, cy2, fx, fy, 0.5)         // control 2 to end
	iox  , ioy   := lerp(oc1x, oc1y, c1c2x, c1c2y, 0.5) // first interpolation from origin
	ifx  , ify   := lerp(c1c2x, c1c2y, c2fx, c2fy, 0.5) // second interpolation to end
	ix   , iy    := lerp(iox, ioy, ifx, ify, 0.5)       // cubic interpolation
	self.recursiveCubeTo(oc1x, oc1y, iox, ioy, ix, iy, depth + 1)
	self.recursiveCubeTo(ifx, ify, c2fx, c2fy, fx, fy, depth + 1)
}

// Sets the threshold distance to use when splitting Bézier curves into
// linear segments. If a linear segment misses the curve by more than
// the threshold value, the curve will be split. Otherwise, the linear
// segment will be used to approximate it. The default value is 0.3.
//
// Values very close to zero could prevent the algorithm from converging
// due to floating point instability, but MaxCurveSplits will prevent
// infinite looping anyway.
func (self *edgeMarker) SetCurveThreshold(dist float32) {
	self.curveThreshold = dist
}

// Sets the maximum amount of times a curve can be recursively split
// into subsegments while trying to approximate it (QuadTo / CubeTo).
//
// The maximum number of segments that will approximate a curve is
// 2^maxCurveSplits. The default value of maxCurveSplits is 6.
//
// This value is typically used as a cutoff to prevent low curve thresholds
// from making the curve splitting process too slow, but it can also be used
// creatively to get jaggy results instead of smooth curves.
//
// Values outside the [0, 255] range will be silently clamped.
func (self *edgeMarker) SetMaxCurveSplits(maxCurveSplits int) {
	if maxCurveSplits < 0 {
		self.maxCurveSplits = 0
	} else if maxCurveSplits > 255 {
		self.maxCurveSplits = 255
	} else {
		self.maxCurveSplits = uint8(maxCurveSplits)
	}
}

// --- helper functions ---

func (self *edgeMarker) markBoundary(x, y, horzAdvance, vertAdvance float64) {
	// find the pixel position on which we have to mark the boundary
	col := intFloorOfSegment(x, horzAdvance)
	row := intFloorOfSegment(y, vertAdvance)

	// stop if going outside bounds (except for negative
	// x coords, which have to be applied anyway as they
	// would accumulate)
	if row < 0 || row >= self.Height { return }
	if col >= self.Width { return }

	// to mark the boundary, we have to see how much we have moved vertically,
	// and accumulate that change into the relevant pixel(s). the vertAdvance
	// tells us how much total change we have, but we also have to interpolate
	// the change *through* the current pixel if it's not fully filled or
	// unfilled exactly at the pixel boundary, marking the boundary through
	// two pixels instead of only one

	// edge case with negative columns. in this case, the whole change
	// is applied to the first column of the affected row
	if col < 0 {
		self.Buffer[row*self.Width] += vertAdvance
		return
	}

	// determine total and partial changes
	totalChange := vertAdvance
	var partialChange float64
	if horzAdvance >= 0 {
		partialChange = (1 - (x - math.Floor(x) + horzAdvance/2))*vertAdvance
	} else { // horzAdvance < 0
		partialChange = (math.Ceil(x) - x - horzAdvance/2)*vertAdvance
	}

	// set the accumulator values
	self.Buffer[row*self.Width + col] += partialChange
	if col + 1 < self.Width {
		self.Buffer[row*self.Width + col + 1] += (totalChange - partialChange)
	}
}

func (self *edgeMarker) withinThreshold(ox, oy, fx, fy, px, py float64) bool {
	// https://en.wikipedia.org/wiki/Distance_from_a_point_to_a_line#Line_defined_by_an_equation
	// dist = |a*x + b*y + c| / sqrt(a^2 + b^2)
	a, b, c := toLinearFormABC(ox, oy, fx, fy)
	n := a*px + b*py + c
	return n*n <= float64(self.curveThreshold)*float64(self.curveThreshold)*(a*a + b*b)
}

func hasReachedTarget(current float64, limit float64, deltaSign float64) bool {
	if deltaSign >= 0 { return current >= limit }
	return current <= limit
}

func nextWholeCoord(position float64, deltaSign float64) float64 {
	if deltaSign == 0 { return position } // this works for *our* context
	if deltaSign > 0 {
		ceil := math.Ceil(position)
		if ceil != position { return ceil }
		return ceil + 1.0
	} else { // deltaSign < 0
		floor := math.Floor(position)
		if floor != position { return floor }
		return floor - 1
	}
}

func intFloorOfSegment(start, advance float64) int {
	floor := math.Floor(start)
	if advance >= 0 { return int(floor) }
	if floor != start { return int(floor) }
	return int(floor) - 1
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
