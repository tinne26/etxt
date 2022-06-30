package emask

type outlineSegment struct {
	// starting position
	ox float64
	oy float64
	fx float64
	fy float64

	// coefficients for line equations in the form Ax + By + C = 0
	a  float64 // a1 and a2 are the same, lines are parallel
	b  float64 // b1 and b2 are the same, lines are parallel
	c1 float64
	c2 float64
}

func (self *outlineSegment) Fill(buffer *buffer, prevSegment, nextSegment *outlineSegment) {//, maxOutDist float64) {
	oxOut, oyOut, oxIn, oyIn, oxx, oxy := prevSegment.intersect(self)
	fxOut, fyOut, fxIn, fyIn, fxx, fxy := self.intersect(nextSegment)

	// TODO: would still need further triangle clipping for thickness
	//       shorter than segment len cases
	_, _ = oxx, oxy
	_, _ = fxx, fxy

	// if outer points get too far away, clamp them
	// TODO: this clamping must also be mirrored on CutHead and CutTail
	//       once I confirm it's working...
	// TODO: the mirroring is unlikely to be ok here.
	// TODO: if this starts getting more complex, consider reimplementing
	//       CutHead and CutTail as Fills with synthetic outlineSegments.
	// TODO: I should have a clearer idea of the whole, complete strategy
	//       before I go into any other bs

	// maxOutDist2 := maxOutDist*maxOutDist
	// oDist2 := dist2(self.ox, self.oy, oxOut, oyOut)
	// if oDist2 > maxOutDist2 {
	// 	factor := math.Sqrt(oDist2)/maxOutDist
	// 	oxOut, oyOut = lerp(self.ox, self.oy, oxOut, oyOut, factor)
	// 	buffer.FillConvexQuad(oxOut, oyOut, oxOut, oyOut, oxIn, oyIn, oxx, oxy)
	// 	oxOut, oyOut = oxx, oxy
	// }
	// fDist2 := dist2(self.fx, self.fy, fxOut, fyOut)
	// if fDist2 > maxOutDist2 {
	// 	factor := math.Sqrt(fDist2)/maxOutDist
	// 	fxOut, fyOut = lerp(self.fx, self.fy, fxOut, fyOut, factor)
	// 	buffer.FillConvexQuad(fxOut, fyOut, fxOut, fyOut, fxIn, fyIn, fxx, fxy)
	// 	fxOut, fyOut = fxx, fxy
	// }

	// main quad fill
	buffer.FillConvexQuad(oxOut, oyOut, oxIn, oyIn, fxOut, fyOut, fxIn, fyIn)
}

func (self *outlineSegment) CutHead(buffer *buffer, prevSegment *outlineSegment) {//, maxOutDist float64) {
	oxOut, oyOut, oxIn, oyIn, oxx, oxy := prevSegment.intersect(self)
	_, _ = oxx, oxy
	// maxOutDist2 := maxOutDist*maxOutDist
	// oDist2 := dist2(self.ox, self.oy, oxOut, oyOut)
	// if oDist2 > maxOutDist2 { // clamping for the outer point
	// 	factor := math.Sqrt(oDist2)/maxOutDist
	// 	oxOut, oyOut = lerp(self.ox, self.oy, oxOut, oyOut, factor)
	// 	buffer.FillConvexQuad(oxOut, oyOut, oxOut, oyOut, oxIn, oyIn, oxx, oxy)
	// 	oxOut, oyOut = oxx, oxy
	// }

	a, b, oc := perpendicularABC(self.a, self.b, self.fx, self.fy)
	xdiv := a*self.b - self.a*b
	ox1, oy1 := shortCramer(xdiv, a, b, oc, self.a, self.b, self.c1)
	ox2, oy2 := shortCramer(xdiv, a, b, oc, self.a, self.b, self.c2)
	buffer.FillConvexQuad(oxOut, oyOut, oxIn, oyIn, ox1, oy1, ox2, oy2)
}

func (self *outlineSegment) CutTail(buffer *buffer, nextSegment *outlineSegment) {//, maxOutDist float64) {
	a, b, oc := perpendicularABC(self.a, self.b, self.ox, self.oy)
	xdiv := a*self.b - self.a*b
	ox1, oy1 := shortCramer(xdiv, a, b, oc, self.a, self.b, self.c1)
	ox2, oy2 := shortCramer(xdiv, a, b, oc, self.a, self.b, self.c2)
	fxOut, fyOut, fxIn, fyIn, fxx, fxy := self.intersect(nextSegment)

	_, _ = fxx, fxy
	// maxOutDist2 := maxOutDist*maxOutDist
	// fDist2 := dist2(self.fx, self.fy, fxOut, fyOut)
	// if fDist2 > maxOutDist2 { // clamping for the outer point
	// 	factor := math.Sqrt(fDist2)/maxOutDist
	// 	fxOut, fyOut = lerp(self.fx, self.fy, fxOut, fyOut, factor)
	// 	buffer.FillConvexQuad(fxOut, fyOut, fxOut, fyOut, fxIn, fyIn, fxx, fxy)
	// 	fxOut, fyOut = fxx, fxy
	// }

	buffer.FillConvexQuad(ox1, oy1, ox2, oy2, fxOut, fyOut, fxIn, fyIn)
}

func (self *outlineSegment) Cut(buffer *buffer) {
	a, b, oc := perpendicularABC(self.a, self.b, self.ox, self.oy)
	xdiv := a*self.b - self.a*b
	ox1, oy1 := shortCramer(xdiv, a, b, oc, self.a, self.b, self.c1)
	ox2, oy2 := shortCramer(xdiv, a, b, oc, self.a, self.b, self.c2)
	fc := -(a*self.fx + b*self.fy) // ax + by + c = 0
	fx1, fy1 := shortCramer(xdiv, a, b, fc, self.a, self.b, self.c1)
	fx2, fy2 := shortCramer(xdiv, a, b, fc, self.a, self.b, self.c2)
	buffer.FillConvexQuad(ox1, oy1, ox2, oy2, fx1, fy1, fx2, fy2)
}

// Intersects two outline segments and determines the inner and outer
// points at which they intersect. The returned values are outer vertex
// x and outer vertex y, inner vertex x and inner vertex y, and the
// other intersecting vertex on self before outer x/y.
func (self *outlineSegment) intersect(other *outlineSegment) (float64, float64, float64, float64, float64, float64) {
	// find 4 intersection points
	xdiv := self.a*other.b - other.a*self.b
	x11, y11 := shortCramer(xdiv, self.a, self.b, self.c1, other.a, other.b, other.c1)
	x12, y12 := shortCramer(xdiv, self.a, self.b, self.c1, other.a, other.b, other.c2)
	x21, y21 := shortCramer(xdiv, self.a, self.b, self.c2, other.a, other.b, other.c1)
	x22, y22 := shortCramer(xdiv, self.a, self.b, self.c2, other.a, other.b, other.c2)

	// determine which point among the 4 intersection points falls
	// at each side of the line equations to determine inner and
	// outer vertices
	ac, bc := -(self.a*self.fx + self.b*self.fy), -(other.a*self.fx + other.b*self.fy)
	boa := (self.b*other.fy > -self.a*other.fx - ac)
	bob := (other.b*self.oy > -other.a*self.ox - bc)
	var inX, inY, outX, outY float64
	var auxX, auxY float64
	for _, pt := range []struct{x, y float64}{{x11, y11}, {x12, y12}, {x21, y21}, {x22, y22}} {
		jaCmp := (self.b*pt.y  > -self.a*pt.x  - ac)
		jbCmp := (other.b*pt.y > -other.a*pt.x - bc)
		if (boa == jaCmp) == (bob == jbCmp) {
			if boa == jaCmp {
				inX, inY = pt.x, pt.y
			} else {
				outX, outY = pt.x, pt.y
			}
		} else if boa != jaCmp {
			auxX, auxY = pt.x, pt.y
		}
	}
	return outX, outY, inX, inY, auxX, auxY
}
