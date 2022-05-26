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
	Buffer buffer
	CurveSegmenter curveSegmenter
}

func (self *edgeMarker) X() float64 { return self.x }
func (self *edgeMarker) Y() float64 { return self.y }

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
	//          hurting readability. for example, figuring out advance
	//          directions and applying them directly instead of depending
	//          on functions like nextWholeCoord. using float32 instead of
	//          float64 can also speed up things.
}

// Creates a boundary from the current position to the given target
// as a quadratic Bézier curve through the given control point and
// moves the current position to the new one.
func (self *edgeMarker) QuadTo(ctrlX, ctrlY, fx, fy float64) {
	self.CurveSegmenter.TraceQuad(self.LineTo, self.x, self.y, ctrlX, ctrlY, fx, fy)
}

// Creates a boundary from the current position to the given target
// as a cubic Bézier curve through the given control points and
// moves the current position to the new one.
func (self *edgeMarker) CubeTo(cx1, cy1, cx2, cy2, fx, fy float64) {
	self.CurveSegmenter.TraceCube(self.LineTo, self.x, self.y, cx1, cy1, cx2, cy2, fx, fy)
}

// --- helper functions ---

func (self *edgeMarker) markBoundary(x, y, horzAdvance, vertAdvance float64) {
	// find the pixel position on which we have to mark the boundary
	col := intFloorOfSegment(x, horzAdvance)
	row := intFloorOfSegment(y, vertAdvance)

	// stop if going outside bounds (except for negative
	// x coords, which have to be applied anyway as they
	// would accumulate)
	if row < 0 || row >= self.Buffer.Height { return }
	if col >= self.Buffer.Width { return }

	// to mark the boundary, we have to see how much we have moved vertically,
	// and accumulate that change into the relevant pixel(s). the vertAdvance
	// tells us how much total change we have, but we also have to interpolate
	// the change *through* the current pixel if it's not fully filled or
	// unfilled exactly at the pixel boundary, marking the boundary through
	// two pixels instead of only one

	// edge case with negative columns. in this case, the whole change
	// is applied to the first column of the affected row
	if col < 0 {
		self.Buffer.Values[row*self.Buffer.Width] += vertAdvance
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
	self.Buffer.Values[row*self.Buffer.Width + col] += partialChange
	if col + 1 < self.Buffer.Width {
		self.Buffer.Values[row*self.Buffer.Width + col + 1] += (totalChange - partialChange)
	}
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
