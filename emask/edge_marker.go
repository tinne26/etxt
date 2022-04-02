package emask

import "math"

// An alternative to vector.Rasterizer that provides access to the result
// of the first rasterization step, which is to mark the outline boundaries
// that cross the raster image vertically.
//
// The point of this EdgeMarker is to provide a more focused and readable
// alternative to vector.Rasterizer that users of etxt can adapt if
// desired (e.g: higher-quality rasterization effects, alternative scan
// directions, sparse representations, fixed point implementations, etc).
// It's still a very low-level tool that almost no user of etxt will be
// interested in touching. The docs include an [explanation] of the
// algorithms and code.
//
// NOTICE: this code still has some issues that make the results not great.
//
// [explanation]: https://github.com/tinne26/etxt/blob/main/docs/rasterize-outlines.md
type EdgeMarker struct {
	x float64 // current drawing point position
	y float64 // current drawing point position
	Width int  // canvas width, in pixels
	Height int // canvas height, in pixels
	Buffer []float64 // be *very* careful if you touch this directly.
	// ^ Negative values are used for counter-clockwise segments,
	//   positive values are used for clockwise segments.

	curveThreshold float64 // expressed as threshold^2
	maxCurveSplits int // prevent infinite loops
	initFlags uint8 // stores whether curveThreshold and maxCurveSplits
	                // have been initialized with bits 0b01 and 0x10
}

// Sets a new Width and Height and resizes the underlying buffer if
// necessary. The buffer contents are cleared too.
func (self *EdgeMarker) Resize(width, height int) {
	// check init flags
	if self.initFlags & 1 == 0 { self.SetCurveThreshold(0.3) }
	if self.initFlags & 2 == 0 { self.SetMaxCurveSplits(8)  }

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
func (self *EdgeMarker) ClearBuffer() {
	fastFillFloat64(self.Buffer, 0)
}

// Moves the current position to the given coordinates.
func (self *EdgeMarker) MoveTo(x, y float64) {
	self.x = x
	self.y = y
}

// Creates a straight boundary from the current position to the given
// target and moves the current position to the new one.
//
// While the 'LineTo' name is used to stay consistent with other similar
// interfaces, don't think in terms of "drawing lines"; we are defining
// the boundaries of an outline.
func (self *EdgeMarker) LineTo(x, y float64) {
	// TODO: this implementation is very clear and intended to be as educative
	//       as possible. that said, while there's nothing terribly inefficient
	//       in it and I generally expect other parts of the code to be more
	//       critical in terms of performance, I'd have to write an optimized
	//       version and benchmark to see how much I actually left on the table.

	// changes in y equal or below this threshold are considered 0.
	// this is bigger than 0 in order to account for floating point
	// division unstability
	const HorizontalityThreshold = 0.000001

	// when we are done, set the new current position
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
}

// Creates a boundary from the current position to the given target
// as a quadratic Bézier curve through the given control point and
// moves the current position to the new one.
func (self *EdgeMarker) QuadTo(ctrlX, ctrlY, x, y float64) {
	// o and f are the start and end points of the curve
	ox, oy, fx, fy := self.x, self.y, x, y

	// create a slice to store curve targets and make the
	// algorithm iterative instead of recursive
	type curveTarget struct{ x, y, t float64 }
	nextTargets := make([]curveTarget, 0, self.maxCurveSplits + 1)
	nextTargets  = append(nextTargets, curveTarget{ x, y, 1.0 })

	// during the process we need line equations in ABC form
	// in order to compute distances between a line and a point,
	// and we can reuse some of them, so we keep helper vars
	a, b, c, a2b2 := toLinearFormABC(ox, oy, fx, fy)

	// keep splitting the curve until segments are within the
	// "flatness" threshold and we can draw them as straight lines
	tReached := 0.0 // our progress in segmenting the curve
	splitBudget := self.maxCurveSplits
	for {
		// get next target and interpolate curve at the current t
		target := nextTargets[len(nextTargets) - 1]
		t := (target.t - tReached)/2
		ix, iy := interpQuadBezier(ox, oy, fx, fy, ctrlX, ctrlY, t)

		// see if the interpolated point is within the configurable
		// threshold. if it is, we use a straight line
		if splitBudget <= 0 || self.withinCurveThreshold(a, b, c, a2b2, ix, iy) {
			lx, ly := self.x, self.y // (memorize, needed later)
			self.LineTo(target.x, target.y) // self x/y is advanced with this too
			if len(nextTargets) == 1 { return } // last target reached, stop

			// update our variables for the next iteration
			nextTargets = nextTargets[:len(nextTargets) - 1]
			tReached    = t // increase reached t
			a, b, c, a2b2 = toLinearFormABC(lx, ly, target.x, target.y)
			splitBudget += 1
		} else { // sub-split required
			nextTargets = append(nextTargets, curveTarget{ ix, iy, t })
			splitBudget -= 1
		}
	}
}

// Creates a boundary from the current position to the given target
// as a cubic Bézier curve through the given control points and
// moves the current position to the new one.
func (self *EdgeMarker) CubeTo(cx1, cy1, cx2, cy2, x, y float64) {
	// go and read QuadTo implementation. this is the same, but
	// without documentation and using interpCubeBezier. that's it.
	ox, oy, fx, fy := self.x, self.y, x, y
	type curveTarget struct{ x, y, t float64 }
	nextTargets := make([]curveTarget, 0, self.maxCurveSplits + 1)
	nextTargets  = append(nextTargets, curveTarget{ x, y, 1.0 })
	a, b, c, a2b2 := toLinearFormABC(ox, oy, fx, fy)

	tReached := 0.0
	splitBudget := self.maxCurveSplits
	for {
		target := nextTargets[len(nextTargets) - 1]
		t := (target.t - tReached)/2
		ix, iy := interpCubeBezier(ox, oy, fx, fy, cx1, cy1, cx2, cy2, t)
		if splitBudget <= 0 || self.withinCurveThreshold(a, b, c, a2b2, ix, iy) {
			lx, ly := self.x, self.y
			self.LineTo(target.x, target.y)
			if len(nextTargets) == 1 { return }
			nextTargets = nextTargets[:len(nextTargets) - 1]
			tReached    = t
			a, b, c, a2b2 = toLinearFormABC(lx, ly, target.x, target.y)
			splitBudget += 1
		} else { // sub-split
			nextTargets = append(nextTargets, curveTarget{ ix, iy, t })
			splitBudget -= 1
		}
	}
}

// Sets the threshold distance to use when splitting Bézier curves into
// linear segments. If a linear segment misses the curve by more than
// the threshold value, the curve will be split. Otherwise, the linear
// segment will be used to approximate it. The default value is 0.3.
//
// If the threshold is too low (typically <= 0.1), the algorithms
// may enter infinite loops only prevented by MaxCurveSplits.
func (self *EdgeMarker) SetCurveThreshold(dist float64) {
	self.curveThreshold = dist*dist // we store it squared already
	self.initFlags |= 1
}

// Sets the maximum amount of curve splits. When thresholds are low,
// they may not be enough to prevent the algorithms from entering
// infinite curve splitting loops, so setting a hard max is necessary.
//
// The default value is 8.
func (self *EdgeMarker) SetMaxCurveSplits(maxCurveSplits int) {
	self.maxCurveSplits = maxCurveSplits
	self.initFlags |= 2
}

// --- helper functions ---

func (self *EdgeMarker) markBoundary(x, y, horzAdvance, vertAdvance float64) {
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

// given the A, B and C coefficients of a line equation in ABC form
// and a point, and returns true if the given line and the point (px, py)
// are close enough (configurable threshold)
func (self *EdgeMarker) withinCurveThreshold(a, b, c, a2b2, px, py float64) bool {
	// https://en.wikipedia.org/wiki/Distance_from_a_point_to_a_line#Line_defined_by_an_equation
	// dist = |a*x + b*y + c| / sqrt(a^2 + b^2)
	n := a*px + b*py + c
	return n*n <= self.curveThreshold*a2b2
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

// interpolate the quadratic bézier curve at the given t [0, 1].
// see https://youtu.be/YATikPP2q70?t=205 for a visual explanation
func interpQuadBezier(ax, ay, bx, by, ctrlX, ctrlY, t float64) (float64, float64) {
	acx, acy := lerp(ax, ay, ctrlX, ctrlY, t)
	cbx, cby := lerp(ctrlX, ctrlY, bx, by, t)
	return lerp(acx, acy, cbx, cby, t)
}

func interpCubeBezier(ax, ay, bx, by, cx1, cy1, cx2, cy2, t float64) (float64, float64) {
	ac2x, ac2y := interpQuadBezier(ax, ay, cx2, cy2, cx1, cy1, t)
	c1bx, c1by := interpQuadBezier(cx1, cy1, bx, by, cx2, cy2, t)
	return lerp(ac2x, ac2y, c1bx, c1by, t)
}

func toLinearFormABC(ox, oy, fx, fy float64) (float64, float64, float64, float64) {
	a, b, c := fy - oy, -(fx - ox), (fx - ox)*oy - (fy - oy)*ox
	return a, b, c, a*a + b*b
}

// Around 9 times as fast as using a regular for loop.
// This can trivially made generic, and can also be adapted
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
