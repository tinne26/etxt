package emask

import "math"

// This file contains methods that implement a "fill quadrilateral"
// operation used with the outliner rasterizer. This is a CPU-heavy
// process, fiddly, slow and annoying. It could be done with an
// uint8 buffer, it could be much more optimized and many other
// things. Whatever.

// Fill a convex quadrilateral polygon whose bounds are defined by
// the points given as parameters. The points don't need to follow
// any particular order, but must define a convex quadrilateral
// (triangles and lines also work as they are like quadrilaterals
// with one or two sides collapsed, which the algorithm handles ok).
//
// Values outside the buffer's bounds will be clipped. (TODO: UNIMPLEMENTED)
//
// TODO: this could also be done on the GPU and the algorithms would be easier.
func (self *buffer) FillConvexQuad(ax, ay, bx, by, cx, cy, dx, dy float64) {
	// the first part is all about clipping
	// TODO: clip directly at the outliner stage..? clip on both places..?

	// clip polygon
	if ay < 0 {
		panic("clipping unimplemented")
		// TODO: recursive calls after splitting...
		//self.uncheckedFillConvexQuad(ax, ay, bx, by, cx, cy, dx, dy)
		//self.uncheckedFillConvexQuad(ax, ay, bx, by, cx, cy, dx, dy)
	} else if by > float64(self.Height) {
		// two calls
		panic("clipping unimplemented")
	} else if ax < 0 {
		panic("clipping unimplemented")
	} else if bx < 0 {
		panic("clipping unimplemented")
	} else if cx < 0 {
		panic("clipping unimplemented")
	} else if dx < 0 {
		panic("clipping unimplemented")
	} else if ax > float64(self.Width) {
		panic("clipping unimplemented")
	} else if bx > float64(self.Width) {
		panic("clipping unimplemented")
	} else if cx > float64(self.Width) {
		panic("clipping unimplemented")
	} else if dx > float64(self.Width) {
		panic("clipping unimplemented")
	} else { // no clipping required, nice
		self.uncheckedFillConvexQuad(ax, ay, bx, by, cx, cy, dx, dy)
	}
}

// Precondition: vertices must be all inside the working area. The call
//               will panic otherwise.
func (self *buffer) uncheckedFillConvexQuad(ax, ay, bx, by, cx, cy, dx, dy float64) {
	if math.IsNaN(ax) || math.IsNaN(ay) || math.IsNaN(bx) || math.IsNaN(by) || math.IsNaN(cx) || math.IsNaN(cy) || math.IsNaN(dx) || math.IsNaN(dy) {
		panic("nan")
	}

	// sort points so they go from smallest y to biggest y
	type point struct { x, y float64 }
	pts := [4]point{point{ax, ay}, point{bx, by}, point{cx, cy}, point{dx, dy}}
	if pts[0].y > pts[3].y { pts[0], pts[3] = pts[3], pts[0] }
	if pts[0].y > pts[1].y { pts[0], pts[1] = pts[1], pts[0] }
	if pts[2].y > pts[3].y { pts[2], pts[3] = pts[3], pts[2] }
	if pts[1].y > pts[2].y {
		pts[1], pts[2] = pts[2], pts[1]
		if pts[0].y > pts[1].y { pts[0], pts[1] = pts[1], pts[0] }
		if pts[2].y > pts[3].y { pts[2], pts[3] = pts[3], pts[2] }
	}

	// define some local helper functions
	sort2f64   := func(a, b float64) (float64, float64) { if a <= b { return a, b } else { return b, a }}
	leftVertId := func(a, b int) int { if pts[a].x <= pts[b].x { return a } else { return b } }

	// since the quadrilateral is convex, we know that points 0 and 1 are
	// connected and that points 2 and 3 are also connected. What we don't
	// know is whether point 0 also connects to 2 or 3, and same for 1.
	// find it out as this is necessary later in most cases.
	pt0Conn := leftVertId(2, 3) // set pt0Pair to the bottom left vert id
	if pts[0].x < pts[1].x { // if 0 on the left, 0 connects to bottom left
		// bottom left was correct
	} else if pts[0].x > pts[1].x { // if 0 on the right, 0 connects to bottom right
		pt0Conn = 5 - pt0Conn // if pair was 2, set to 3, if it was 3, set to 2
	} else if pts[0].x < pts[2].x { // 0.x == 1.x, both bottom verts are on one side
		pt0Conn = 5 - pt0Conn // if pair was 2, set to 3, if it was 3, set to 2
	} // else { /* bottom left was correct */ }

	// subdivide the polygon in one, two or three parts
	// (the code may seem both confusing and repetitive. don't get
	// too hung up, try to understand each case one by one and follow
	// what's happening geometrically... vertex indices are tricky)
	flatTop    := (pts[0].y == pts[1].y)
	flatBottom := (pts[2].y == pts[3].y)
	if flatTop && flatBottom { // quad can be drawn as a single trapeze
		tlx, trx := sort2f64(pts[0].x, pts[1].x)
		blx, brx := sort2f64(pts[2].x, pts[3].x)
		self.FillAlignedQuad(pts[0].y, pts[3].y, tlx, trx, blx, brx)
	} else if flatTop { // quad can be drawn with a trapeze and a triangle
		tlx, trx := sort2f64(pts[0].x, pts[1].x)

		// to get the first trapeze we need to intersect lines 0-3 or
		// 1-3 (whichever would form a triangle with vert 2) with an
		// horizontal line going through pts[2].y
		vertIdOpp2 := 3 - pt0Conn // vertex id opposite to vert 2
		ia, ib, ic := toLinearFormABC(pts[vertIdOpp2].x, pts[vertIdOpp2].y, pts[3].x, pts[3].y)
		ix := -(ic + ib*pts[2].y)/ia // ax + by + c = 0, then x = (-c - by)/a
		blx, brx := sort2f64(pts[2].x, ix)
		self.FillAlignedQuad(pts[0].y, pts[2].y, tlx, trx, blx, brx) // fill trapeze
		self.FillAlignedQuad(pts[2].y, pts[3].y, blx, brx, pts[3].x, pts[3].x) // fill bottom triangle
		// ...remaining code is barely documented as it doesn't introduce any new ideas
	} else if flatBottom { // quad can be drawn with a triangle and a trapeze
		ia, ib, ic := toLinearFormABC(pts[0].x, pts[0].y, pts[pt0Conn].x, pts[pt0Conn].y)
		ix := -(ic + ib*pts[1].y)/ia
		blx, brx := sort2f64(pts[1].x, ix)
		self.FillAlignedQuad(pts[0].y, pts[1].y, pts[0].x, pts[0].x, blx, brx) // fill top triangle
		tlx, trx := blx, brx
		blx, brx  = sort2f64(pts[2].x, pts[3].x)
		self.FillAlignedQuad(pts[1].y, pts[3].y, tlx, trx, blx, brx) // fill bottom trapeze
	} else { // quad is drawn with a triangle, a trapeze and then yet another triangle
		// notice: this could be the only case, as it works for the general case,
		//         but having separate cases improves performance in X% (TODO: benchmark)
		ia, ib, ic := toLinearFormABC(pts[0].x, pts[0].y, pts[pt0Conn].x, pts[pt0Conn].y)
		ix := -(ic + ib*pts[1].y)/ia // ax + by + c = 0, then x = (-c - by)/a
		blx, brx := sort2f64(pts[1].x, ix)
		self.FillAlignedQuad(pts[0].y, pts[1].y, pts[0].x, pts[0].x, blx, brx) // fill top triangle
		vertIdOpp2 := 3 - pt0Conn // vertex id opposite to vert 2
		ia, ib, ic = toLinearFormABC(pts[vertIdOpp2].x, pts[vertIdOpp2].y, pts[3].x, pts[3].y)
		ix  = -(ic + ib*pts[2].y)/ia
		tlx, trx := blx, brx
		blx, brx  = sort2f64(pts[2].x, ix)
		self.FillAlignedQuad(pts[1].y, pts[2].y, tlx, trx, blx, brx) // fill trapeze
		self.FillAlignedQuad(pts[2].y, pts[3].y, blx, brx, pts[3].x, pts[3].x) // fill bottom triangle
	}
}

// Fills a quadrilateral defined by the given coordinates, where the
// top and bottom sides are perpendicular to the Y axis (which makes
// the quadrilateral a trapezoid, with flat top and bottom sides).
func (self *buffer) FillAlignedQuad(ty, by, tlx, trx, blx, brx float64) {
	// assert validity of arguments order
	if ty  > by  { panic("ty > by") }
	if tlx > trx { panic("tlx > trx") }
	if blx > brx { panic("blx > brx") }

	// early return cases
	if ty == by { return }
	if tlx == trx && blx == brx { return } // line, no area

	// prepare x advance deltas
	dy  := by - ty
	dlx := (blx - tlx)/dy // left delta per y
	drx := (brx - trx)/dy // right delta per y
	dly := dy/math.Abs(tlx - blx)
	dry := dy/math.Abs(trx - brx)

	// lousily iterate each row
	for {
		// get next top y position
		nextTy := math.Floor(ty + 1)
		if nextTy > by { nextTy = by }

		// prepare next bottom x coords for iterating the row
		blxRow := tlx + (nextTy - ty)*dlx
		brxRow := trx + (nextTy - ty)*drx

		// corrections to blxRow and brxRow that may happen at
		// most once due to floating point precision errors
		if dlx > 0 {
			if blxRow > blx { blxRow = blx }
		} else if blxRow < blx {
			blxRow = blx
		}
		if drx > 0 {
			if brxRow > brx { brxRow = brx }
		} else if brxRow < brx {
			brxRow = brx
		}

		// fill the row
		self.fillRow(ty, nextTy, tlx, trx, blxRow, brxRow, dly, dry)
		tlx, trx = blxRow, brxRow

		// update variables for next iteration
		if nextTy == by { break }
		ty = nextTy
	}
}

func (self *buffer) fillRow(ty, by, tlx, trx, blx, brx float64, dly, dry float64) {
	baseRowIndex := int(math.Floor(ty))*self.Width
	olx, orx := math.Max(tlx, blx), math.Min(trx, brx)
	if olx <= orx {
		// overlap case, center is a rect
		left, right := tlx, trx
		if tlx > blx { // left triangle with flat bottom
			self.fillRowRectTriangle(by, -dly, blx, tlx, baseRowIndex)
		} else if blx > tlx { // left triangle with flat top
			left = blx
			self.fillRowRectTriangle(ty, dly, tlx, blx, baseRowIndex)
		}

		if trx > brx { // right triangle with flat top
			right = brx
			self.fillRowRectTriangleRTL(ty, dry, trx, brx, baseRowIndex)
		} else if brx > trx { // right triangle with flat bottom
			self.fillRowRectTriangleRTL(by, -dry, brx, trx, baseRowIndex)
		}

		if left != right {
			self.fillRowRect(ty, by, left, right, baseRowIndex)
		}
	} else { // tilted quad or triangle, but at least one part is flat
		// non-overlap case, center can be a tilted quad
		var qleft, qright float64 // quad left, quad right
		qlty, qlby, qrty, qrby := ty, by, ty, by // quad left top y, quad left bottom y, etc.
		if tlx > blx { // left triangle with flat bottom and right triangle with flat top case
			qleft, qright = brx, tlx
			qlty = self.fillRowRectTriangle(by, -dly, blx, brx, baseRowIndex)
			qrby = self.fillRowRectTriangleRTL(ty, dry, trx, tlx, baseRowIndex)
			if qrby > by { qrby = by }
			if qlty < ty { qlty = ty }
		} else if blx > tlx { // left triangle with flat top and right triangle with flat bottom case
			qleft, qright = trx, blx
			qlby = self.fillRowRectTriangle(ty, dly, tlx, trx, baseRowIndex)
			qrty = self.fillRowRectTriangleRTL(by, -dry, brx, blx, baseRowIndex)
			if qlby > by { qlby = by }
			if qrty < ty { qrty = ty }
		} else {
			panic("unexpected tlx == blx")
		}

		if qleft != qright {
			self.fillRowAlignedQuad(qleft, qright, qlty, qlby, qrty, qrby, dly, dry, baseRowIndex)
		}
	}
}

func (self *buffer) fillRowRectTriangle(startY, yChange, left, right float64, baseRowIndex int) float64 {
	alignedLeft  := math.Ceil(left)
	alignedRight := math.Floor(right)

	if alignedLeft > alignedRight { // single pixel special case
		xdiff := right - left
		ydiff := xdiff*yChange
		self.Values[baseRowIndex + int(math.Floor(left))] += math.Abs(ydiff)*xdiff/2.0
		return startY + ydiff
	}

	y := startY
	if left != alignedLeft { // fractional left part
		xdiff := alignedLeft - left
		ydiff := xdiff*yChange
		self.Values[baseRowIndex + int(math.Floor(left))] += math.Abs(ydiff)*xdiff/2.0
		y += ydiff
	}

	partialChange := math.Abs(yChange)/2.0
	for x := int(alignedLeft); x < int(alignedRight); x++ {
		self.Values[baseRowIndex + x] += partialChange + math.Abs(startY - y)
		y += yChange
	}

	if right != alignedRight { // fractional right part
		xdiff := right - alignedRight
		ydiff := xdiff*yChange
		self.Values[baseRowIndex + int(alignedRight)] += math.Abs(ydiff)*xdiff/2.0 + xdiff*math.Abs(startY - y)
		y += ydiff
	}

	return y
}

func (self *buffer) fillRowRectTriangleRTL(startY, yChange, right, left float64, baseRowIndex int) float64 {
	alignedLeft  := math.Ceil(left)
	alignedRight := math.Floor(right)

	if alignedLeft > alignedRight { // single pixel special case
		xdiff := right - left
		ydiff := xdiff*yChange
		self.Values[baseRowIndex + int(math.Floor(left))] += math.Abs(ydiff)*xdiff/2.0
		return startY + ydiff
	}

	y := startY
	if right != alignedRight {
		xdiff := right - alignedRight
		ydiff := xdiff*yChange
		self.Values[baseRowIndex + int(alignedRight)] += math.Abs(ydiff)*xdiff/2.0
		y += ydiff
	}

	partialChange := math.Abs(yChange)/2.0
	for x := int(alignedRight) - 1; x >= int(alignedLeft); x-- {
		self.Values[baseRowIndex + x] += partialChange + math.Abs(startY - y)
		y += yChange
	}

	if left != alignedLeft { // fractional left part
		xdiff := alignedLeft - left
		ydiff := xdiff*yChange
		self.Values[baseRowIndex + int(math.Floor(left))] += math.Abs(ydiff)*xdiff/2.0 + xdiff*math.Abs(startY - y)
		y += ydiff
	}

	return y
}

func (self *buffer) fillRowRect(ty, by, left, right float64, baseRowIndex int) {
	alignedLeft  := math.Ceil(left)
	alignedRight := math.Floor(right)
	dy := by - ty

	if alignedLeft > alignedRight { // single pixel special case
		self.Values[baseRowIndex + int(math.Floor(left))] += dy*(right - left)
		return
	}

	if left != alignedLeft { // fractional left part
		xdiff := alignedLeft - left
		self.Values[baseRowIndex + int(math.Floor(left))] += dy*xdiff
	}

	if alignedLeft != alignedRight {
		if dy == 1 {
			si, fi := baseRowIndex + int(alignedLeft), baseRowIndex + int(alignedRight)
			fastFillFloat64(self.Values[si : fi], 1.0)
		} else {
			for x := int(alignedLeft); x < int(alignedRight); x++ {
				self.Values[baseRowIndex + x] += dy
			}
		}
	}

	if right != alignedRight { // fractional left part
		xdiff := right - alignedRight
		self.Values[baseRowIndex + int(alignedRight)] += dy*xdiff
	}
}

func (self *buffer) fillRowAlignedQuad(left, right, tly, bly, try, bry, dly, dry float64, baseRowIndex int) {
	// figure out orientation
	var dty, dby float64 // delta top y, delta bottom y
	if try < tly || bry < bly { // moving upwards, negative
		dty = -dly
		dby = -dry
	} else { // moving downwards
		dty = dry
		dby = dly
	}

	alignedLeft  := math.Ceil(left)
	alignedRight := math.Floor(right)

	if alignedLeft > alignedRight { // single pixel special case
		ab := (bly - tly) + (bry - try)
		self.Values[baseRowIndex + int(math.Floor(left))] += ab*(right - left)/2
		return
	}

	if left != alignedLeft { // fractional left part
		xdiff := alignedLeft - left
		newTly, newBly := tly + dty*xdiff, bly + dby*xdiff
		ab := (bly - tly) + (newBly - newTly)
		self.Values[baseRowIndex + int(math.Floor(left))] += ab*xdiff/2
		tly, bly = newTly, newBly

	}

	// main loop
	if dly == dry { // optimized version when dly == dry
		partialChange := bly - tly // simplified from ((bly - tly) + (bly+dly - bry+dry))*1/2
		for x := int(alignedLeft); x < int(alignedRight); x++ {
			self.Values[baseRowIndex + x] += partialChange
		}
		iters := (alignedRight - alignedLeft)
		tly, bly = tly + dty*iters, bly + dby*iters
	} else {
		for x := int(alignedLeft); x < int(alignedRight); x++ {
			// TODO: optimize expression once tests are working, I
			//       doubt the compiler will catch this otherwise
			newTly, newBly := tly + dty, bly + dby
			ab := (bly - tly) + (newBly - newTly)
			self.Values[baseRowIndex + x] += ab/2
			tly, bly = newTly, newBly
		}
	}

	if right != alignedRight { // fractional right part
		xdiff := right - alignedRight
		ab := (bly - tly) + (bry - try)
		self.Values[baseRowIndex + int(alignedRight)] += ab*xdiff/2
	}
}
