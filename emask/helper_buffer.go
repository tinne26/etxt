package emask

import "math"

// A common buffer implementation shared by edge_marker and outliner.
type buffer struct {
	Width  int // canvas width, in pixels
	Height int // canvas height, in pixels
	Values []float64
	// ^ Negative values are used for counter-clockwise segments,
	//   positive values are used for clockwise segments.
}

// Sets a new Width and Height and resizes the underlying buffer if
// necessary. The buffer contents are cleared too.
func (self *buffer) Resize(width, height int) {
	if width <= 0 || height <= 0 { panic("width or height <= 0") }
	self.Width  = width
	self.Height = height
	totalLen := width*height
	if len(self.Values) == totalLen {
		// nothing
	} else if len(self.Values) > totalLen {
		self.Values = self.Values[0 : totalLen]
	} else { // len(self.Values) < totalLen
		if cap(self.Values) >= totalLen {
			self.Values = self.Values[0 : totalLen]
		} else {
			self.Values = make([]float64, totalLen)
			return // stop before ClearBuffer()
		}
	}

	self.Clear()
}

// Fills the internal buffer with zeros.
func (self *buffer) Clear() {
	fastFillFloat64(self.Values, 0)
}

// Performs the boundary change accumulation operation storing
// the results into the given buffer. Used with the edge marker
// rasterizer.
func (self *buffer) AccumulateUint8(buffer []uint8) {
	if len(buffer) != self.Width*self.Height {
		panic("uint8 buffer has wrong length")
	}

	index := 0
	for y := 0; y < self.Height; y++ {
		accumulator := float64(0)
		accUint8    := uint8(0)
		for x := 0; x < self.Width; x++ {
			value := self.Values[index]
			if value != 0 { // small optimization
				accumulator += value
				accUint8 = uint8(clampUnit64(abs64(accumulator))*255)
			}
			buffer[index] = accUint8
			index += 1
		}
	}
}

// Fill a convex quadrilateral polygon whose bounds are defined by
// the points given as parameters. The points don't need to follow
// any particular order, but must define a convex quadrilateral
// (triangles and lines also work as they are like quadrilaterals
// with one or two sides collapsed, which the algorithm handles ok).
//
// Keep in mind that values that fall outside the buffer area (e.g.:
// negative values) will be clipped.
func (self *buffer) FillConvexQuad(ax, ay, bx, by, cx, cy, dx, dy float64) {
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
		blx, brx := sort2f64(pts[0].x, pts[1].x)
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
		self.FillAlignedQuad(pts[0].y, pts[3].y, tlx, trx, blx, brx) // fill trapeze
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
		// TODO: this probably works for the general case anyway
	}
}

// Fills a quadrilateral defined by the given coordinates, where the
// top and bottom sides are perpendicular to the Y axis (which makes
// the quadrilateral a trapezoid).
// Notice: this method is extremely slow and unoptimized, but it's
//         already confusing enough as it is, so I don't intend to
//         make it faster soon. If you want to implement faster
//         outlining algorithms, go ahead yourself, there's much
//         room for improvement.
func (self *buffer) FillAlignedQuad(ty, by, tlx, trx, blx, brx float64) {
	// assert validity of arguments
	if ty  > by  { panic("ty > by") }
	if tlx > trx { panic("tlx > trx") }
	if blx > brx { panic("blx > brx") }

	// clip coordinates to buffer dimensions
	if by <= 0 { return }
	if ty >= float64(self.Height) { return }
	if ty < 0 { ty = 0 }
	if by > float64(self.Height) { by = float64(self.Height) }
	if tlx < 0 { tlx = 0 }
	if trx < 0 { trx = 0 }
	if blx < 0 { blx = 0 }
	if brx < 0 { brx = 0 }
	if tlx > float64(self.Width) { tlx = float64(self.Width) }
	if trx > float64(self.Width) { trx = float64(self.Width) }
	if blx > float64(self.Width) { blx = float64(self.Width) }
	if brx > float64(self.Width) { brx = float64(self.Width) }

	// check if clipping transformed the quad into an empty area
	if ty == by { return }
	if tlx == trx && blx == brx { return } // line

	// prepare x advance deltas
	dlx := (blx - tlx)/(by - ty) // left delta per y
	drx := (brx - trx)/(by - ty) // right delta per y

	// lousily iterate each pixel
	for {
		// get next top y position
		nextTy := nextPxCeil(ty)
		if nextTy > by { nextTy = by }

		// get row start index
		rowStartIndex := basePxId(nextTy)*self.Width

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

		// iterate the row
		localTlx, localBlx := tlx, blxRow
		for {
			// increase x positions
			nextTlx, nextBlx := math.Ceil(localTlx), math.Ceil(localBlx)
			if nextTlx >= localTlx && nextBlx >= localBlx {
				nextTlx += 1
				nextBlx += 1
			}
			if nextTlx > trx    { nextTlx = trx    }
			if nextBlx > brxRow { nextBlx = brxRow }

			if trx < 0 || brxRow < 0 {
				panic(brxRow)
			}

			if nextTlx > float64(self.Width) { panic(nextTlx) }

			// draw pixel
			value := computePixelAlpha(ty, nextTy, localTlx, nextTlx, localBlx, nextBlx)
			self.Values[rowStartIndex + basePxId(nextTlx)] += value

			// update x variables
			if nextTlx == trx && nextBlx == brxRow {
				tlx, trx = blxRow, brxRow
				break
			}
			localTlx, localBlx = nextTlx, nextBlx
		}

		// update variables for next iteration
		if nextTy == by { break }
		ty = nextTy
	}
}

// given coordinates for a trapezoid inside a pixel, figure out how much
// of the pixel does the quad fill. this is simply computing the area
// of a trapezoid, which is (a + b)*h/2, where h is the height, a is the
// length of the top side, and b the length of the bottom side
func computePixelAlpha(ty, by, tlx, trx, blx, brx float64) float64 {
	a := trx - tlx
	b := brx - blx
	h := by - ty
	return (a + b)*h/2.0
}

func basePxId(value float64) int {
	floor := math.Floor(value)
	if floor != value { return int(floor) }
	if floor == 0 { return 0 }
	return int(floor) - 1
}

func nextPxCeil(value float64) float64 {
	ceil := math.Ceil(value)
	if ceil != value { return ceil }
	return ceil + 1.0
}

// TODO: remove unused code
// handle cases of lines and triangles
// IDEA: if three points in a row share x or y, they constitute a triangle.
//       if four points in a row share x or y, they constitute a line.
//       even if they don't share x or y, they may still constitute a line
//       if tlx == trx && blx == brx
// if pts[0].x == pts[1].x || pts[2].x == pts[3].x {
// 	if pts[0].x == pts[2].x || pts[1].x == pts[3].x {
// 		if pts[0].x == pts[3].x { return } // vertical line
// 		self.FillTrapeze(pts[0].y, pts[3].y, pts[0].x, pts[0].x, pts[3].x, pts[3].x)
// 		return // triangle with vertical side
// 	}
// }
// if pts[0].y == pts[1].y || pts[2].y == pts[3].y {
// 	if pts[0].x == pts[2].x || pts[1].x == pts[3].x {
// 		if pts[0].x == pts[3].x { return } // vertical line
// 		self.FillTrapeze(pts[0].y, pts[3].y, pts[0].x, pts[0].x, pts[3].x, pts[3].x)
// 		return // triangle with vertical side
// 	}
// }

// alternative accumulation implementation... slightly faster in 1.18
// but more fragile to compiler changes. for example, just removing
// the goto and using start = index directly slows down
// the whole rasterization program around 15% despite the
// apparent logical equivalence of the programs.
// 		rowEnd := index + self.Width
// 		accumulator := float64(0)
// 		accUint8 := uint8(0)
// again:
// 		start := index
// 		for index < rowEnd {
// 			if self.Values[index] == 0 {
// 				index += 1
// 			} else {
// 				if start != index {
// 					fastFillUint8(buffer[start : index], accUint8)
// 				}
// 				accumulator += self.Values[index]
// 				accUint8 = uint8(clampUnit64(abs64(accumulator))*255)
// 				buffer[index] = accUint8
// 				index += 1
// 				goto again
// 			}
// 		}
// 		if start != index {
// 			fastFillUint8(buffer[start : index], accUint8)
// 		}

// // actually attempt to draw
// tyCeil , tyHasFractPart := ceilCrossing(ty)
// byFloor, byHasFractPart := floorCrossing(by)
// // draw fractional top part
// if tyHasFractPart {
// 	if tyCeil > byFloor {
// 		// edge case with no vertical pixel crossing (only a
// 		// single row is affected, and only partially)
//
// 		return
// 	} else {
//
// 	}
// }

// // draw main part of the quad
// // TODO...
// for y := int(tyCeil); y < int(byFloor); y++ {
// 	rowStart := y*self.Width
//
// 	leftCeil, leftHasFractPart := ceilCrossing(tlx)
// 	rightFloor, rightHasFractPart := floorCrossing(trx)
// 	tlxNext, trxNext := tlx + dlx, trx + drx
// 	// TODO: there can be two fractional parts, or even many more... whoopsies.
// 	//       I need to loop from left top to left bottom.
// 	//       maybe have fractLeftFloor, fractLeftCeil
// 	if leftHasFractPart {
// 		// TODO...
// 	}
// 	ilc, irf := int(leftCeil), int(rightFloor)
// 	if ilc < irf {
// 		fastFillFloat64(self.Values[rowStart + ilc : rowStart + irf], 1.0)
// 	}
// 	if rightHasFractPart {
// 		// TODO...
// 	}
// }
//
// // TODO: I can also try a very very raw approach where I do directly
// //       pixel by pixel with "full" calculations, more like edgeMarker.
// //       Basically, find next x/y, pass 4 coords and solve the individual
// //       pixel or something...
//
//
// // draw fractional bottom part
// if byHasFractPart {
// 	// TODO...
// }
//
// func floorCrossing(value float64) (float64, bool) {
// 	floor := math.Floor(value)
// 	if floor == start { return floor, false }
// 	return floor, true
// }
//
// func ceilCrossing(value float64) (float64, bool) {
// 	ceil := math.Ceil(value)
// 	if ceil == start { return ceil, false }
// 	return ceil, true
// }
